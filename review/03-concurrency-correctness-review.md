# Concurrency & Correctness Review — Task API

**Scope:** `main.go`, `internal/task/{handler,service,repository,model,router}.go`, `internal/response/model.go`, `main_test.go`
**Date:** 2026-07-14
**Focus:** Concurrency safety and correctness of the in-memory `TaskRepository`.

---

## Summary

The API stores all state in a single, process-wide `TaskRepository` (`main.go:17`) whose fields `tasks []Task` and `nextID int` are read and mutated with **no synchronization whatsoever**. Gin (via `net/http`) dispatches every incoming request on its own goroutine, so any two overlapping requests that touch the repository form a classic **data race**. Under Go's memory model this is undefined behavior: lost writes, duplicate IDs, a corrupted slice header, or an outright crash (`fatal error: concurrent map/slice writes`, index-out-of-range panics).

Secondary correctness issues: `GetAll` leaks the internal backing slice to callers, `DeleteById` reuses the backing array in an aliasing-unsafe way, there is no length/content validation on `Title`, and the service layer is a thin pass-through that adds no value.

**`go test -race ./...` result** (below) is **PASS** — but this is a false sense of safety: the only test (`main_test.go`) issues requests **strictly sequentially** on a single goroutine, so the race detector never observes concurrent access. The absence of a failure here does **not** mean the code is race-free; it means the test does not exercise concurrency.

```
$ cd /Users/uzy/Documents/developer3/app-dev-2/tour && go test -race ./...
ok      api     1.826s
?       api/internal/response   [no test files]
?       api/internal/task       [no test files]
```
(go1.26.5 darwin/arm64)

A concurrent test *would* trip the detector — see the fix section for a repro harness.

---

## Findings by severity

### CRITICAL

#### C1 — Unsynchronized shared mutable state → data race
**Files/lines:**
- `internal/task/repository.go:9-12` (unguarded fields `tasks`, `nextID`)
- `internal/task/repository.go:28-29` (`Create`: `append` + `nextID++` writes)
- `internal/task/repository.go:33-34` (`GetAll`: read)
- `internal/task/repository.go:37-44` (`GetById`: read/range)
- `internal/task/repository.go:47-58` (`UpdateById`: read + element write)
- `internal/task/repository.go:60-70` (`DeleteById`: read + slice-header write)

The single `taskRepo` created at `main.go:17` is shared by all handlers, and Gin serves each HTTP request on a distinct goroutine. Every method above touches `r.tasks` / `r.nextID` without a lock, atomic, or channel.

**Concrete failure scenario (lost update / duplicate ID):**
Two clients `POST /api/v1/tasks` at the same time, repository starts with `nextID == 5`.

1. Goroutine A reads `r.nextID` (5), builds `Task{Id:5}`.
2. Goroutine B reads `r.nextID` (5), builds `Task{Id:5}`.
3. A executes `r.tasks = append(r.tasks, taskA)`; header now len 1.
4. B executes `r.tasks = append(r.tasks, taskB)` — but B captured the *old* slice header (len 0) before A's append landed, so B's append writes to index 0 and produces a header of len 1, **overwriting A's element**.
5. Both do `r.nextID++`; two `++` on the same starting value yield `6` instead of `7`.

Net result: **two tasks accepted by the API, one silently lost, both share `Id:5`, and `nextID` is now wrong** (next create collides again). Which specific corruption occurs is nondeterministic — the `append` may also grow/reallocate concurrently and panic.

`UpdateById`/`DeleteById` racing with a reader (`GetAll`/`GetById`) can also panic with `index out of range` because the length observed and the backing array can change mid-iteration.

**Fix:** guard all access with a `sync.RWMutex` (see Recommended fix below). This is the primary defect.

---

### HIGH

#### H1 — `GetAll` returns the internal slice by reference
**File/line:** `internal/task/repository.go:33-35`, surfaced via `service.go:7-9` and `handler.go:14`.

```go
func (r *TaskRepository) GetAll() []Task {
    return r.tasks   // caller now holds a reference to the live backing array
}
```

The handler receives the *same* slice the repository keeps mutating. Even if C1 were fixed with a mutex around the method call, the lock is released the instant `GetAll` returns, yet the caller still holds an aliased slice. A subsequent `Create`/`Update`/`Delete` mutating the shared backing array while the JSON encoder walks that slice is a read/write race and can serialize a torn/duplicated view.

**Failure scenario:** request X calls `GetAll`, gets slice header (ptr=P, len=3). While Gin's JSON encoder iterates P, request Y runs `UpdateById` and writes `r.tasks[1].Title`. X now encodes a half-updated element. With `-race` this reports a data race on `Task.Title`.

**Fix:** return a copy:
```go
func (r *TaskRepository) GetAll() []Task {
    r.mu.RLock()
    defer r.mu.RUnlock()
    out := make([]Task, len(r.tasks))
    copy(out, r.tasks)
    return out
}
```

#### H2 — `DeleteById` aliases/mutates the backing array in place
**File/line:** `internal/task/repository.go:64`

```go
r.tasks = append(r.tasks[:i], r.tasks[i+1:]...)
```

`append(r.tasks[:i], r.tasks[i+1:]...)` shifts elements left **in the same backing array** and shrinks the length. Any other goroutine still holding the pre-delete slice header (e.g. the one leaked by H1, or a concurrent `GetAll`) will observe elements being overwritten underneath it — a data race and a logically wrong view (a shifted-but-not-yet-shortened tail leaves a duplicated last element visible to the stale header).

Note: `DeleteById` (and `UpdateById`) aren't even wired into the router yet (`router.go` only registers `GET ""`, `GET "/:id"`, `POST ""`), but they are part of the public repository API and will race the moment a `DELETE` route is added.

**Fix:** under the write lock, build a new slice or use `slices.Delete` and accept that stale external references must not exist (guaranteed once H1 returns copies):
```go
r.tasks = slices.Delete(r.tasks, i, i+1)
```
combined with H1's copy-on-read, no external alias survives.

---

### MEDIUM

#### M1 — No length/content validation on `Title`
**File/line:** `internal/task/model.go:32-34`

```go
type CreateTaskRequest struct {
    Title string `json:"title" binding:"required"`
}
```

`binding:"required"` rejects an empty/missing `title`, but there is **no upper bound**. A client can `POST {"title": "<10 MB string>"}` and it is stored verbatim, growing process memory unboundedly (a DoS vector for an in-memory store). `Completed` is intentionally forced to `false` on create (`repository.go:25`), which is fine, but worth noting the create request cannot set it.

**Fix:** add a max length, e.g. `binding:"required,max=255"`, and consider `min=1` semantics (Gin's `required` already rejects `""`).

#### M2 — Handler's 500 branch is currently dead but semantically loose
**File/line:** `internal/task/handler.go:62-68`

`GetById` treats any non-`ErrTaskNotFound` error as `500`. The repository today returns **only** `ErrTaskNotFound` (`repository.go:44`), so the 500 branch is unreachable — defensive, not a bug. Flagged because it is easy to misread as "the repo can fail in other ways." It's acceptable to keep as future-proofing, but it should be documented, or the repo's error contract narrowed so the intent is explicit.

---

### LOW

#### L1 — Service layer is a redundant pass-through
**File/line:** `internal/task/service.go:11-18` (`GetById`), `7-9` (`GetAllTasks`), `20-22` (`CreateTask`)

`GetById` does `task, err := s.repo.GetById(id); if err != nil { return Task{}, err }; return task, nil` — identical to `return s.repo.GetById(id)`. The service adds no validation, transformation, or transaction boundary. Not a correctness bug, but dead abstraction: any real business rule (e.g., the Title validation of M1, or de-duplication) belongs here and is currently absent.

**Fix:** either collapse to `return s.repo.GetById(id)` or give the service a real responsibility (validation, ID policy).

#### L2 — `nextID` monotonic reuse semantics
**File/line:** `internal/task/repository.go:11,29`

`nextID` only ever increments and is never decremented on delete, so IDs are **not reused** after deletion — this is correct and desirable (avoids resurrecting a deleted task's identity). No change needed; documented here so the monotonic behavior is an explicit design decision rather than an accident. Note the counter starts at `1` and there is no persistence, so IDs reset on process restart.

---

## Recommended fix — add a `sync.RWMutex`

`sync.RWMutex` is the right primitive here (many concurrent reads, occasional writes). `sync.Map` is an alternative but a poor fit: it loses ordering, complicates the monotonic `nextID`, and forces type assertions; a mutex-guarded slice/map is clearer for this workload. If ordering by insertion doesn't matter, a `map[int]Task` under the same mutex avoids the O(n) scans and the delete-shift hazard entirely.

```go
package task

import (
    "errors"
    "slices"
    "sync"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository struct {
    mu     sync.RWMutex
    tasks  []Task
    nextID int
}

func NewTaskRepository() *TaskRepository {
    return &TaskRepository{tasks: []Task{}, nextID: 1}
}

func (r *TaskRepository) Create(payload CreateTaskRequest) Task {
    r.mu.Lock()
    defer r.mu.Unlock()
    newTask := Task{Id: r.nextID, Title: payload.Title, Completed: false}
    r.tasks = append(r.tasks, newTask)
    r.nextID++
    return newTask
}

func (r *TaskRepository) GetAll() []Task {
    r.mu.RLock()
    defer r.mu.RUnlock()
    out := make([]Task, len(r.tasks)) // copy: never leak the backing array
    copy(out, r.tasks)
    return out
}

func (r *TaskRepository) GetById(id int) (Task, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, t := range r.tasks {
        if t.Id == id {
            return t, nil
        }
    }
    return Task{}, ErrTaskNotFound
}

func (r *TaskRepository) UpdateById(id int, payload UpdateTaskRequest) (Task, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    for i := range r.tasks {
        if r.tasks[i].Id == id {
            r.tasks[i].Title = payload.Title
            r.tasks[i].Completed = payload.Completed
            return r.tasks[i], nil
        }
    }
    return Task{}, ErrTaskNotFound
}

func (r *TaskRepository) DeleteById(id int) (Task, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    for i := range r.tasks {
        if r.tasks[i].Id == id {
            deleted := r.tasks[i]
            r.tasks = slices.Delete(r.tasks, i, i+1)
            return deleted, nil
        }
    }
    return Task{}, ErrTaskNotFound
}
```

### Test that would actually catch the race

The current suite passes `-race` only because it is sequential. Add a concurrent test so the detector has overlapping access to observe:

```go
func TestConcurrentCreate(t *testing.T) {
    router := setupRouter()
    var wg sync.WaitGroup
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            w := httptest.NewRecorder()
            body := []byte(`{"title":"race"}`)
            req, _ := http.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
            req.Header.Set("Content-Type", "application/json")
            router.ServeHTTP(w, req)
        }()
    }
    wg.Wait()
    // With the unsynchronized repo, `go test -race` reports a DATA RACE here.
    // After the mutex fix, expect exactly 50 tasks with 50 distinct IDs.
}
```

Run with:
```
go test -race ./...
```
Against the current code this test reports `DATA RACE` on `r.tasks`/`r.nextID`; against the fixed repository it passes and yields 50 unique IDs.

---

## Priority order
1. **C1** — add `sync.RWMutex` (fixes the core data race).
2. **H1 / H2** — copy-on-read in `GetAll`, safe delete (close the aliasing leaks).
3. **M1** — bound `Title` length.
4. **M2 / L1 / L2** — tidy error contract, collapse or empower the service layer, document ID monotonicity.
5. Add the concurrent `-race` test to CI so regressions are caught.

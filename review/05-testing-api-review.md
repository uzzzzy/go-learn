# Testing & API Completeness Review — Go/Gin Task API

## Summary

The task API is a small, cleanly-layered Gin service (handler → service → repository)
backed by an in-memory store. The layering and the recently-introduced `Repository`/`Service`
interfaces are good foundations, but the **test suite and the routed API surface are both
significantly incomplete**.

Testing is limited to a *single* happy-path integration test (`main_test.go`) exercising
Create → GetAll → GetById as three order-dependent subtests that share one router. There are
no unit tests for the service or repository, no error-path coverage (400/404/500), and no
table-driven tests. Overall the `task` package — where all the business logic lives — has
**0.0% statement coverage**.

On completeness, the repository implements full CRUD (`UpdateById`, `DeleteById`), but the
router never registers `PUT`/`PATCH`/`DELETE`, so those methods are dead, untested, and
unreachable. There is also an interface inconsistency: the repo exposes update/delete, the
`Service` interface omits them, and the router omits them again. No pagination, filtering, or
sorting exists on the list endpoint, and the `/health` endpoint lives outside `/api/v1` and is
untested.

## Coverage Results

Command: `go test -cover ./...`

| Package | Result | Coverage | Notes |
|---|---|---|---|
| `api` (root / `main.go`, `main_test.go`) | ok | **60.0%** | Only the routes hit by the one integration test; `health` and error branches uncovered |
| `api/internal/response` | no test files | — | Pure types; low risk |
| `api/internal/task` | (built) | **0.0%** | Handler, service, repository — **all core logic untested** |

The 60% figure is misleading: it reflects only the wiring in `main.go`, not the `task`
package, which carries the actual behavior and sits at zero.

## Testing Gaps by Severity

### Critical — Core logic has 0% coverage

The `task` package (handler, service, repository) has no unit tests at all. Every validation
branch, error path, and CRUD method is unverified. Start with fast, dependency-free repository
and service unit tests.

Repository unit test (table-driven, covers GetById hit/miss and Create):

```go
package task

import "testing"

func TestRepository_CreateAndGetById(t *testing.T) {
	repo := NewTaskRepository()
	created := repo.Create(CreateTaskRequest{Title: "write tests"})

	if created.Id != 1 || created.Title != "write tests" || created.Completed {
		t.Fatalf("unexpected created task: %+v", created)
	}

	tests := []struct {
		name    string
		id      int
		wantErr error
	}{
		{"found", 1, nil},
		{"missing", 999, ErrTaskNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := repo.GetById(tc.id)
			if err != tc.wantErr {
				t.Fatalf("GetById(%d) err = %v, want %v", tc.id, err, tc.wantErr)
			}
		})
	}
}
```

### Critical — Update/Delete repository methods are entirely untested

`UpdateById` and `DeleteById` contain the trickiest logic (slice mutation via
`append(r.tasks[:i], r.tasks[i+1:]...)`) yet have zero coverage and no route to exercise them.

```go
func TestRepository_UpdateAndDelete(t *testing.T) {
	repo := NewTaskRepository()
	repo.Create(CreateTaskRequest{Title: "a"})
	repo.Create(CreateTaskRequest{Title: "b"})

	updated, err := repo.UpdateById(1, UpdateTaskRequest{Title: "a2", Completed: true})
	if err != nil || updated.Title != "a2" || !updated.Completed {
		t.Fatalf("update failed: %+v err=%v", updated, err)
	}

	deleted, err := repo.DeleteById(2)
	if err != nil || deleted.Id != 2 {
		t.Fatalf("delete failed: %+v err=%v", deleted, err)
	}
	if _, err := repo.GetById(2); err != ErrTaskNotFound {
		t.Fatalf("expected task 2 gone, got err=%v", err)
	}

	// error paths
	if _, err := repo.UpdateById(999, UpdateTaskRequest{Title: "x"}); err != ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound on update, got %v", err)
	}
	if _, err := repo.DeleteById(999); err != ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound on delete, got %v", err)
	}
}
```

### High — No HTTP error-path tests (400 / 404 / 500)

The handler has explicit branches for invalid JSON / missing title (400), non-numeric ID
(400), and not-found (404) — none are tested. These are the branches most likely to regress.
A table-driven HTTP test keeps each router isolated (fixes the ordering problem below):

```go
func TestGetTask_ErrorPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		wantCode int
	}{
		{"non-numeric id", "GET", "/api/v1/tasks/abc", "", http.StatusBadRequest},
		{"not found", "GET", "/api/v1/tasks/999", "", http.StatusNotFound},
		{"create missing title", "POST", "/api/v1/tasks", `{}`, http.StatusBadRequest},
		{"create invalid json", "POST", "/api/v1/tasks", `{`, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := setupRouter() // fresh router per subtest — no shared state
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("%s: got %d, want %d (body=%s)", tc.name, w.Code, tc.wantCode, w.Body)
			}
		})
	}
}
```

### High — Service layer untested; no use of mock Repository

The `Service` interface takes a `Repository` interface, which makes it trivially mockable, but
no test exercises `TaskService` in isolation (e.g. verifying error propagation from
`GetById`). Add a tiny fake repo to assert the service forwards errors unchanged.

### Medium — Order-dependent subtests sharing one router

`TestTaskWorkflow` builds one `router := setupRouter()` and the `Get By Id` subtest depends on
`createdTaskID` set by the `Create` subtest. `Get All Task` asserts `len == 1`, which only
holds because of prior state and execution order. If subtests are reordered, run in isolation,
or parallelized, they break. Prefer per-test fresh routers plus one deliberate end-to-end flow
test if a workflow test is genuinely wanted.

### Medium — `/health` endpoint untested

`health` in `main.go` is never asserted. A one-line test guards the ops/liveness contract:

```go
func TestHealth(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("health: got %d, want 200", w.Code)
	}
}
```

### Low — No concurrency test on the in-memory store

`TaskRepository` mutates `tasks`/`nextID` without a mutex. Not a test-quality issue per se, but
a `-race` test creating tasks concurrently would surface the unsynchronized access that a real
Gin server (concurrent requests) will hit.

## API Completeness

### Missing / unreachable endpoints

The repository already supports these operations; the router does not expose them. Handlers
for update and delete do not yet exist and must be added alongside the routes.

| Method | Path | Handler (to add) | Backing repo method | Status |
|---|---|---|---|---|
| `PUT` (or `PATCH`) | `/api/v1/tasks/:id` | `TaskHandler.UpdateTask` (missing) | `UpdateById` ✅ exists | **Not routed** |
| `DELETE` | `/api/v1/tasks/:id` | `TaskHandler.DeleteTask` (missing) | `DeleteById` ✅ exists | **Not routed** |

Current routes registered in `internal/task/router.go`:

- `GET /api/v1/tasks` → `GetTasks`
- `GET /api/v1/tasks/:id` → `GetTask`
- `POST /api/v1/tasks` → `CreateTask`

### Interface inconsistency (CRUD not wired end-to-end)

There is a three-way mismatch that should be reconciled:

- `Repository` interface (`model.go`): **full CRUD** — includes `UpdateById`, `DeleteById`.
- `Service` interface (`model.go`): **omits** update/delete — only `GetAllTasks`, `GetById`,
  `CreateTask`. `TaskService` (`service.go`) likewise has no `UpdateTask`/`DeleteTask` methods.
- Router (`router.go`): **omits** update/delete routes and handlers.

Net effect: the repository's update/delete code is dead. Either add
`UpdateTask`/`DeleteTask` to the `Service` interface + `TaskService` + handlers + routes to
complete CRUD, or remove the unused repository methods if update/delete is out of scope.
Leaving them half-wired is the worst option (dead, untested code).

### Other gaps

- **No pagination / filtering / sorting** on `GET /api/v1/tasks`. As the store grows, the list
  endpoint returns everything unbounded. Consider `?limit=&offset=` and `?completed=` filtering.
- **`/health` outside the API version group.** It is registered on the root router
  (`main.go`), not under `/api/v1`. That is a defensible convention for liveness probes, but it
  is inconsistent and untested — document the choice and add a test.
- **No 405 handling / method contract.** Unrouted methods on `/tasks` fall through to Gin's
  default 404 rather than 405 Method Not Allowed; fine for now but worth noting once
  update/delete land.

## Prioritized Action List

1. **Add repository unit tests** (table-driven) covering `Create`, `GetById` hit/miss,
   `UpdateById` success + not-found, `DeleteById` success + not-found. Highest value, zero
   dependencies — lifts the `task` package off 0% immediately. (Critical)
2. **Add handler HTTP error-path tests**: invalid JSON 400, missing title 400, non-numeric id
   400, not-found 404 — each with a fresh `setupRouter()`. (High)
3. **Decide CRUD scope and reconcile the interfaces.** Either wire `PUT`/`PATCH` +
   `DELETE` end-to-end (Service interface → `TaskService` methods → handlers → routes) *with*
   tests, or delete the unused repo methods. (High)
4. **Add service-layer unit tests** using a fake `Repository` to verify error propagation.
   (High)
5. **Fix `TestTaskWorkflow` order-dependence**: isolate subtests with per-test routers; keep at
   most one intentional end-to-end flow test. (Medium)
6. **Add a `/health` test** and document why it sits outside `/api/v1`. (Medium)
7. **Add pagination/filtering** to the list endpoint (`?limit`, `?offset`, `?completed`), with
   tests. (Medium)
8. **Add a `-race` concurrency test** and guard the in-memory store with a mutex if it stays.
   (Low)

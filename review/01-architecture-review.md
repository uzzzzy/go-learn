# Architecture & Design Review — Task Management REST API (`api`)

## Summary

This is a small, cleanly written Go/Gin task-management API that already gets the fundamentals right: it applies a Handler → Service → Repository layering, depends on interfaces rather than concrete types (`Service`, `Repository`), uses a generic `ApiResponse[T]` envelope for consistent output, and has an integration test covering the happy path. The dependency direction is correct — handlers know nothing about storage, and the service is decoupled from Gin.

That said, the design carries several architectural rough edges that will bite as the project grows. The most consequential are: (1) the DI wiring is split awkwardly across `main.go` and `router.go`, hiding the service/handler construction inside a routing function; (2) `model.go` is a catch-all that mixes DTOs, the domain entity, interfaces, and concrete struct definitions in one file; (3) the `Repository` interface exposes `UpdateById`/`DeleteById` that no route, handler, or service ever calls — dead code behind a live interface; (4) the in-memory repository is not concurrency-safe and does not model the abstraction a real datastore needs (no `context.Context`, no error on writes); and (5) there is no configuration layer and no graceful shutdown, so the server cannot be operated cleanly. The `go.mod` also carries `go.mongodb.org/mongo-driver/v2` as a genuinely unused dependency (`quic-go` is a legitimate transitive dep of Gin via HTTP/3 and should stay).

None of these are correctness bugs in the current happy path, but they are structural decisions worth fixing before the codebase accumulates more surface area.

---

## Findings by Severity

### Critical

None. The application compiles, layers correctly, and serves its implemented routes.

---

### High

#### H1 — In-memory repository is not concurrency-safe
**File:** `internal/task/repository.go:9-70`

`TaskRepository` mutates `r.tasks` and `r.nextID` (`Create`, `UpdateById`, `DeleteById`) with no synchronization. Gin serves every request in its own goroutine, so two concurrent `POST /tasks` requests race on the `append` and the `nextID++`, which can corrupt the slice, produce duplicate IDs, or lose writes. This is latent today only because the test drives requests sequentially.

**Recommendation:** Guard the repository state with a `sync.RWMutex` (read-lock reads, write-lock mutations), or, better, treat the in-memory implementation as a throwaway and hide it behind the `Repository` interface so it can be swapped for a real store. At minimum, add a mutex now — it is a two-line fix that removes a real data race.

#### H2 — Repository abstraction will not scale to a real datastore
**File:** `internal/task/model.go:3-9`, `internal/task/repository.go:21-45`

The `Repository` interface signatures assume synchronous, infallible in-memory access:
- `Create(payload CreateTaskRequest) Task` returns no `error` — a database insert can always fail.
- No method accepts a `context.Context`, so request cancellation, deadlines, and tracing cannot propagate to the data layer.
- `GetAll() []Task` returns the entire table with no pagination/limit — fine for a slice, unworkable against a database.

When this is pointed at Postgres/Mongo/etc., every one of these signatures must change, cascading into the service and handler. Designing the interface for the eventual datastore now avoids a churn-heavy refactor later.

**Recommendation:** Evolve the interface toward `Create(ctx context.Context, payload CreateTaskRequest) (Task, error)`, thread `context.Context` from the Gin handler (`c.Request.Context()`) through service to repository, and add pagination parameters to `GetAll`. Also decouple the storage row from the wire DTO — passing `CreateTaskRequest` (an HTTP-layer DTO) directly into the repository leaks a transport concern into the persistence boundary; the service should map DTO → domain entity before calling the repo.

---

### Medium

#### M1 — Dead code behind the `Repository` interface (`UpdateById` / `DeleteById`)
**File:** `internal/task/model.go:7-8`, `internal/task/repository.go:47-70`, `internal/task/router.go:9-14`

`Repository` declares `UpdateById` and `DeleteById`, and `TaskRepository` fully implements them, but:
- The `Service` interface (`model.go:11-15`) exposes only `GetAllTasks`, `GetById`, `CreateTask`.
- `TaskService` (`service.go`) has no update/delete methods.
- No route is registered for PUT/PATCH/DELETE (`router.go` wires only GET, GET/:id, POST).

So roughly a third of the repository is unreachable code reachable only through an interface method nobody calls. This is misleading (it implies update/delete are supported) and untested.

**Recommendation:** Either (a) complete the vertical slice — add `UpdateTask`/`DeleteTask` to the `Service` interface and `TaskService`, add handlers, and register `PUT /tasks/:id` and `DELETE /tasks/:id` — or (b) remove `UpdateById`/`DeleteById` from both the interface and the implementation until they are needed. Do not leave them half-wired. Given the `UpdateTaskRequest` DTO already exists (`model.go:36-39`) and is otherwise unused, option (a) appears to be the intended direction.

#### M2 — DI wiring is split and partially hidden inside routing
**File:** `main.go:17-18`, `internal/task/router.go:5-8`

Object construction is fragmented: `main.go` builds the *repository*, then `RegisterRouters` *also* constructs the service and handler as a side effect of registering routes. This conflates two responsibilities — dependency assembly and route mapping — and means the composition root is not in one place. It also makes it awkward to inject a pre-built service (e.g. a mock, or one wrapped with middleware/decorators) in tests, and forces `RegisterRouters` to know how to build the whole chain.

**Recommendation:** Assemble the full dependency graph in the composition root (`main.go` or a dedicated `wire`/`app` package): build repo → service → handler there, and have `RegisterRouters(rg, handler)` accept the finished handler and do nothing but map routes. This keeps DI in one place and makes the router a pure routing concern.

#### M3 — `model.go` is a mixed-concern catch-all
**File:** `internal/task/model.go:1-40`

This one file holds four distinct kinds of declaration:
- Domain entity: `Task` (lines 26-30)
- Transport DTOs: `CreateTaskRequest`, `UpdateTaskRequest` (lines 32-39)
- Abstraction contracts: `Repository`, `Service` interfaces (lines 3-15)
- Concrete component structs: `TaskHandler`, `TaskService` (lines 17-23)

Bundling the entity with DTOs, and interfaces with the concrete structs that implement/consume them, blurs the boundaries the layering is trying to establish. `TaskHandler`'s struct definition living apart from its methods in `handler.go` (and likewise `TaskService`) also makes each component harder to read as a unit.

**Recommendation:** Split by concern: keep `Task` (and later, DTOs) in `model.go`; move the `TaskHandler` struct next to its methods in `handler.go` and `TaskService` next to its methods in `service.go`. Interface *placement* is otherwise good — defining `Repository`/`Service` in the consuming `task` package follows the Go idiom of "accept interfaces where you use them" — but they would read more clearly in a small `interfaces.go` (or co-located with their consumers) than mixed in with entity/DTO types.

#### M4 — No configuration layer
**File:** `main.go:25-29`

The server relies entirely on `router.Run()`, which defaults to `:8080` and reads only the `PORT` env var implicitly. There is no config struct, no way to set read/write timeouts, no environment-driven settings (log level, datastore DSN, etc.). As soon as a real datastore or per-environment behavior is added, configuration will have to be retrofitted through the whole startup path.

**Recommendation:** Introduce a small `config` package that loads from environment (and/or flags) into a typed `Config` struct — at minimum HTTP address and timeouts — and pass it into the composition root. This also sets up cleanly for the datastore DSN in H2.

#### M5 — No graceful shutdown
**File:** `main.go:25-29`

`router.Run()` blocks and offers no lifecycle control: on `SIGINT`/`SIGTERM` the process dies immediately, dropping in-flight requests and skipping any cleanup (which will matter once there are DB connections to close). It also hard-codes `http.Server` defaults with no timeouts, leaving the server exposed to slow-client resource exhaustion.

**Recommendation:** Construct an explicit `*http.Server` with `ReadTimeout`/`WriteTimeout`/`IdleTimeout`, run `ListenAndServe` in a goroutine, and block on a `signal.NotifyContext`; on signal, call `srv.Shutdown(ctx)` with a bounded timeout. This is the standard Go production startup pattern and pairs naturally with M4.

---

### Low

#### L1 — Unused dependency: `mongo-driver/v2`
**File:** `go.mod:35`

`go.mongodb.org/mongo-driver/v2 v2.5.0` is listed as an indirect dependency but is imported nowhere in the module (`grep -rn "mongo" --include="*.go"` returns nothing). It is not a transitive dependency of Gin or testify — it appears to be a leftover from exploratory work. It bloats the dependency graph and `go.sum` for no benefit.

**Recommendation:** Run `go mod tidy`; it should drop this entry. (For contrast, `github.com/quic-go/quic-go` and `qpack` on lines 31-32 are *legitimate* transitive deps pulled in by Gin's HTTP/3 support and should remain — do not remove them.)

#### L2 — Internal service error masking is a no-op today but risks over-broad 500s
**File:** `internal/task/handler.go:52-68`, `internal/task/service.go:11-18`

`TaskService.GetById` forwards the repository error unchanged, and the handler maps anything that is not `ErrTaskNotFound` to `500`. Today the repo only ever returns `ErrTaskNotFound` or `nil`, so this branch is unreachable — but it establishes a pattern where any future sentinel error from the repo silently becomes a `500`. As the error surface grows (validation, conflict, etc.) this will misclassify client errors as server errors.

**Recommendation:** As more error types appear, map them explicitly (e.g. an errors-to-HTTP translation helper) rather than defaulting everything to `500`. Not urgent at current scope.

#### L3 — Health endpoint bypasses the API grouping and envelope conventions
**File:** `main.go:20`, `main.go:31-37`

`/health` is registered at the root while all real endpoints live under `/api/v1`, and it is defined as a free function in `package main` rather than alongside other handlers. This is a minor inconsistency; it is defensible to keep health outside the versioned API, but the placement mixes handler logic into the composition root.

**Recommendation:** Optionally move health/readiness handlers into a small `health` package (or a system routes group) for consistency. Low priority.

#### L4 — `_ = createdTaskID` no-op in test
**File:** `main_test.go:80`

`createdTaskID` is already used by the "Get By Id" subtest, so the trailing `_ = createdTaskID` is a redundant no-op left over from earlier iteration. Harmless but noise.

**Recommendation:** Remove line 80.

---

## Prioritized Recommendations

1. **Add a mutex to the in-memory repository (H1).** Removes a real data race; two-line change.
2. **Resolve the dead update/delete code (M1).** Either complete the PUT/DELETE slice (interface → service → handler → route) or remove the unused repo methods — the `UpdateTaskRequest` DTO suggests completion was intended.
3. **Consolidate the composition root and simplify the router (M2).** Build repo→service→handler in one place; make `RegisterRouters` take a finished handler.
4. **Design the `Repository` interface for a real datastore (H2).** Add `context.Context`, return `error` from writes, add pagination, and map DTO→entity in the service rather than passing DTOs to the repo.
5. **Add configuration + graceful shutdown (M4, M5).** Typed `Config` from env, explicit `http.Server` with timeouts, signal-driven `Shutdown`.
6. **Split `model.go` by concern (M3).** Move component structs next to their methods; keep entity/DTOs (and optionally interfaces) in focused files.
7. **`go mod tidy` to drop `mongo-driver/v2` (L1).**
8. **Housekeeping (L2–L4):** explicit error mapping as the error surface grows, optional health-package extraction, and delete the redundant test no-op.

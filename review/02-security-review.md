# Security Review: Go/Gin Task-Management REST API

**Target:** `/Users/uzy/Documents/developer3/app-dev-2/tour`
**Stack:** Go, Gin, in-memory store (`[]Task` slice)
**Reviewed:** `main.go`, `internal/task/{handler,service,repository,model,router}.go`, `internal/response/model.go`, `main_test.go`
**Date:** 2026-07-14

---

## Summary

This is a small, unauthenticated CRUD API backed by an in-process slice. There are no application-level security controls: no authentication/authorization, no transport security (TLS), no input size or length limits, no request body caps, no rate limiting, no server timeouts, no security response headers, and no CORS policy. Several of these compound into denial-of-service (DoS) exposure, and a couple of paths leak internal error strings to clients.

The single most impactful gaps are the **complete absence of authentication/authorization**, the **unbounded input and response sizes** (memory-exhaustion DoS), and the **lack of server-level timeouts and body limits**. None of these are theoretical; each is directly reachable by an anonymous caller on the default `:8080` listener bound to all interfaces.

Because the store is a mutating in-memory slice guarded by no synchronization, there is also a data-race concern under concurrent load (correctness/availability), noted below as it has security (availability/integrity) implications.

Severity counts: **Critical 2, High 4, Medium 5, Low 4.**

> Note: `router.go` only wires `GET /tasks`, `GET /tasks/:id`, and `POST /tasks`. `UpdateById`/`DeleteById` exist in the repository but are not exposed. Findings focus on reachable surface, with unexposed code flagged where relevant.

---

## Findings by Severity

### CRITICAL

#### C1 — No authentication or authorization on any endpoint
**Location:** `internal/task/router.go:9-14`, `main.go:15-22`
**Threat/Impact:** Every route (`GET /api/v1/tasks`, `GET /api/v1/tasks/:id`, `POST /api/v1/tasks`) is fully anonymous. Any network peer that can reach the listener can enumerate and create tasks without limit. Combined with the all-interfaces bind (see H4) and no rate limiting (see H3), this is an open, abusable write endpoint. There is no notion of a resource owner, so even if auth were added later, the data model (`Task` in `model.go:26-30`) has no owner field to enforce per-user authorization.
**Remediation:**
- Put authenticated routes behind middleware. Minimal bearer-token/JWT gate:
```go
func AuthRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        tok := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
        if !validToken(tok) { // constant-time compare / JWT verify
            c.AbortWithStatusJSON(http.StatusUnauthorized, response.ApiResponse[any]{
                Status: response.StatusFailed, Error: "unauthorized",
            })
            return
        }
        c.Next()
    }
}
// main.go
apiGroup := router.Group("/api/v1", AuthRequired())
```
- Add an owner/tenant field to `Task` and scope reads/writes to the authenticated principal (authorization), so users cannot read or mutate others' tasks.

#### C2 — Unbounded request body and unbounded field length (memory-exhaustion DoS)
**Location:** `internal/task/handler.go:25` (`c.ShouldBindJSON(&input)`), `internal/task/model.go:32-39` (`Title` has only `binding:"required"`, no `max`)
**Threat/Impact:** `ShouldBindJSON` reads the entire request body with no `http.MaxBytesReader` cap, and `Title` has no maximum length. A single request with a multi-hundred-MB JSON string is fully buffered into memory and then stored permanently in the in-memory slice (`repository.go:21-31`). Repeating this (no rate limit, no auth) drives the process to OOM and takes down the service. Because stored tasks are never evicted and `GetAll` returns everything (see H2), each oversized create also permanently inflates the response payload.
**Remediation:**
- Cap the body at the HTTP layer before binding:
```go
func BodyLimit(max int64) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
        c.Next()
    }
}
// apply e.g. BodyLimit(64 * 1024) to write routes
```
- Add length bounds to the model:
```go
type CreateTaskRequest struct {
    Title string `json:"title" binding:"required,max=256"`
}
type UpdateTaskRequest struct {
    Title     string `json:"title" binding:"required,max=256"`
    Completed bool   `json:"completed"`
}
```
- Consider a hard cap on total stored tasks (reject creates past a ceiling) since the store never shrinks.

---

### HIGH

#### H1 — No server timeouts (Slowloris / slow-read DoS)
**Location:** `main.go:28` (`router.Run()`)
**Threat/Impact:** `router.Run()` uses `http.ListenAndServe` with a zero-value `http.Server`, meaning `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, and `IdleTimeout` are all unset (infinite). A handful of slow-sending clients can hold connections open indefinitely (Slowloris), exhausting connection/goroutine resources with negligible attacker cost. There are also no per-request `context` deadlines.
**Remediation:** Replace `router.Run()` with an explicitly configured server:
```go
srv := &http.Server{
    Addr:              "127.0.0.1:8080",
    Handler:           router,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      15 * time.Second,
    IdleTimeout:       60 * time.Second,
    MaxHeaderBytes:    1 << 20,
}
log.Fatal(srv.ListenAndServe())
```

#### H2 — Unbounded `GetAll` with no pagination
**Location:** `internal/task/handler.go:13-20`, `service.go:7-9`, `repository.go:33-35`
**Threat/Impact:** `GET /tasks` serializes and returns the entire task slice with no `limit`/`offset`. Combined with unbounded creates (C2) and no eviction, response size grows without bound; a large store makes each list call expensive (CPU for JSON marshaling, memory for the buffered response), amplifying DoS and increasing latency for all callers.
**Remediation:** Add pagination with a hard server-side max page size:
```go
// parse ?limit= & ?offset=, clamp limit to e.g. [1,100]
tasks := h.service.GetPaged(offset, limit)
```
Return total count separately; never return the full collection unbounded.

#### H3 — No rate limiting / abuse controls
**Location:** `main.go:12-23` (no limiter middleware anywhere)
**Threat/Impact:** Nothing throttles request volume. Anonymous callers can create tasks or hammer endpoints in a tight loop, amplifying every DoS finding above and enabling cheap resource exhaustion of the single-process in-memory store.
**Remediation:** Add IP/token-based rate limiting middleware (e.g. `golang.org/x/time/rate` per-client, or a reverse proxy / API gateway). Apply stricter limits to the write route.

#### H4 — Listener binds to all interfaces by default
**Location:** `main.go:28` (`router.Run()` defaults to `:8080` = `0.0.0.0:8080`)
**Threat/Impact:** With no address argument, Gin binds `0.0.0.0`, exposing the unauthenticated API on every network interface (LAN, container network, potentially public if the host is exposed). This maximizes the blast radius of C1/C2/H1-H3.
**Remediation:** Bind explicitly to a trusted interface (e.g. `127.0.0.1:8080` behind a reverse proxy that terminates TLS and auth), and make the bind address configurable via env/flag. See H1 snippet (`Addr: "127.0.0.1:8080"`).

---

### MEDIUM

#### M1 — Internal error strings returned to clients
**Location:** `internal/task/handler.go:62-68` (500 path returns `err.Error()`), `handler.go:25-31` (bind error returns `err.Error()`)
**Threat/Impact:** The 500 branch echoes the raw error message to the client, and the JSON-bind failure returns Gin/validator internals (field names, type mismatch details, struct info). This is information disclosure that can help an attacker learn internal structure and error conditions. (The `ErrTaskNotFound` message at `handler.go:54-59` is a safe, intentional sentinel.)
**Remediation:** Return a generic message to the client and log the detail server-side:
```go
if err != nil {
    log.Printf("GetById(%d): %v", id, err) // server log only
    c.JSON(http.StatusInternalServerError, response.ApiResponse[any]{
        Status: response.StatusFailed, Error: "internal error",
    })
    return
}
```
For bind errors, return a fixed `"invalid request body"` rather than `err.Error()`.

#### M2 — No security response headers
**Location:** `main.go:12-23` (no header middleware)
**Threat/Impact:** Responses carry no `X-Content-Type-Options: nosniff`, `X-Frame-Options`/`Content-Security-Policy`, `Strict-Transport-Security`, `Cache-Control`, or `Referrer-Policy`. While this is a JSON API, missing `nosniff` and framing/CSP headers weaken defense-in-depth if any response is ever rendered in a browser context.
**Remediation:** Add a small middleware setting at minimum:
```go
c.Header("X-Content-Type-Options", "nosniff")
c.Header("X-Frame-Options", "DENY")
c.Header("Cache-Control", "no-store")
// HSTS once TLS is in place
```

#### M3 — No TLS / transport encryption
**Location:** `main.go:28` (`router.Run()` serves plain HTTP)
**Threat/Impact:** All traffic (including any future auth tokens) travels in cleartext, open to interception and tampering on the network path. Directly compounds C1 once auth is added.
**Remediation:** Terminate TLS at the app (`srv.ListenAndServeTLS`) or, more commonly, at a reverse proxy/load balancer in front of a loopback-bound app (H4). Enforce HTTPS and add HSTS (M2).

#### M4 — No CORS policy defined
**Location:** `main.go:12-23` (no CORS middleware)
**Threat/Impact:** No explicit CORS configuration. Gin sends no `Access-Control-Allow-Origin` by default, so browser cross-origin reads are blocked — but the absence of an intentional policy means this is unmanaged. If a permissive policy is later bolted on carelessly (e.g. reflecting origin + credentials), it becomes a vulnerability. Flagged so a deliberate, least-privilege policy is chosen.
**Remediation:** Define an explicit allowlist-based CORS policy (specific origins, methods, headers; never `*` with credentials) using a maintained middleware (`github.com/gin-contrib/cors`).

#### M5 — Concurrent access to shared mutable store without synchronization (data race → integrity/availability)
**Location:** `internal/task/repository.go:9-31` (`tasks []Task`, `nextID int` mutated in `Create`/`UpdateById`/`DeleteById` with no mutex)
**Threat/Impact:** A single `TaskRepository` is shared across all requests (`main.go:17-18`). Gin serves requests concurrently, so simultaneous `Create` calls race on `r.tasks`/`r.nextID` (`append` + `nextID++` are not atomic). This can corrupt the slice, produce duplicate/lost IDs, or panic (slice growth race) — an availability and data-integrity issue reachable simply by concurrent clients. Run with `-race` to confirm.
**Remediation:** Guard all store access with a `sync.RWMutex` (read lock for `GetAll`/`GetById`, write lock for `Create`/`Update`/`Delete`), or move to a store that is safe for concurrent use.
```go
type TaskRepository struct {
    mu     sync.RWMutex
    tasks  []Task
    nextID int
}
```

---

### LOW

#### L1 — Negative / non-positive IDs accepted by parser
**Location:** `internal/task/handler.go:41-50` (`strconv.Atoi` accepts `-5`, `0`)
**Threat/Impact:** `strconv.Atoi` accepts negative and zero values; these pass validation and fall through to a linear scan that returns `ErrTaskNotFound`. Low impact today (no negative IDs are ever assigned), but it wastes a full-slice scan and signals missing input validation. `nextID` starts at 1, so IDs are always positive.
**Remediation:** Reject `id < 1` explicitly after parsing:
```go
if err != nil || id < 1 {
    c.JSON(http.StatusBadRequest, response.ApiResponse[any]{
        Status: response.StatusFailed, Error: "Invalid ID"})
    return
}
```

#### L2 — `gin.Default()` default logger may leak data and is not production-tuned
**Location:** `main.go:13`
**Threat/Impact:** `gin.Default()` attaches the default Logger and Recovery middleware. The Logger writes request lines (method, path, latency, client IP) to stdout by default — acceptable, but request paths can contain sensitive query data and logs are unstructured/unbounded. Recovery is good (prevents panic crashes) but by default does not print the stack to the client (safe) only when `gin.Mode()` is release; in debug mode Gin is more verbose. Ensure `GIN_MODE=release` in production.
**Remediation:** Use `gin.New()` with explicitly chosen middleware (structured logger that redacts sensitive fields, Recovery), and set release mode:
```go
gin.SetMode(gin.ReleaseMode)
router := gin.New()
router.Use(gin.Recovery(), structuredLogger())
```

#### L3 — No request `Content-Type` enforcement on writes
**Location:** `internal/task/handler.go:22-31`
**Threat/Impact:** `ShouldBindJSON` will attempt to parse regardless of `Content-Type`. Minor; primarily a robustness/clarity issue and mild CSRF-surface consideration for browser clients (though auth is absent entirely). 
**Remediation:** Prefer `c.ShouldBindWith(&input, binding.JSON)` guarded by a `Content-Type: application/json` check, and reject otherwise.

#### L4 — Unexposed mutating repository methods lack the same controls
**Location:** `internal/task/repository.go:47-70` (`UpdateById`, `DeleteById`), not wired in `router.go`
**Threat/Impact:** Not currently reachable (no routes), so no live risk — but when exposed they will need the same auth (C1), authorization/ownership, body limits and length validation (C2), concurrency guards (M5), and generic error handling (M1). `DeleteById` also performs `append(tasks[:i], tasks[i+1:]...)` which mutates the backing array — must be under the write lock.
**Remediation:** When routing these, apply auth middleware + `UpdateTaskRequest` bounds + mutex; treat delete as a state-changing action requiring authorization.

---

## Prioritized Remediation Checklist

1. **[C1]** Add authentication middleware to `/api/v1` and an owner field + authorization checks on tasks.
2. **[C2]** Add `http.MaxBytesReader` body cap on writes and `max=` length validators on `Title` (both request structs).
3. **[H4]** Bind the listener to `127.0.0.1` (configurable), behind a proxy — stop defaulting to `0.0.0.0:8080`.
4. **[H1]** Replace `router.Run()` with an `http.Server` that sets Read/ReadHeader/Write/Idle timeouts and `MaxHeaderBytes`.
5. **[H3]** Add rate limiting (per-IP/token), stricter on the write route.
6. **[H2]** Add mandatory pagination with a clamped max page size to `GET /tasks`.
7. **[M5]** Guard the in-memory store with a `sync.RWMutex`; verify with `go test -race`.
8. **[M1]** Stop returning `err.Error()` to clients (500 and bind paths); log detail server-side, return generic messages.
9. **[M3]** Terminate TLS (proxy or app) and enforce HTTPS.
10. **[M2]** Add security headers middleware (`nosniff`, framing, `Cache-Control`, HSTS after TLS).
11. **[M4]** Define an explicit least-privilege CORS policy (allowlist, no `*`+credentials).
12. **[L2]** Set `gin.ReleaseMode`, use `gin.New()` with a redacting structured logger + Recovery.
13. **[L1]** Reject `id < 1` in `GetTask`.
14. **[L3]** Enforce `Content-Type: application/json` on writes.
15. **[L4]** Apply all of the above to `UpdateById`/`DeleteById` before exposing them.

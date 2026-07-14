# Checklist per Kategori → Prioritas

Diurutkan **prioritas** (P0 Blocker → P1 Penting → P2 Kualitas → P3 Skala/hardening) di dalam tiap kategori.

Catatan: beberapa item lintas-kategori (mis. data race) muncul di satu kategori "rumah" utamanya agar tidak dobel. Kode temuan sumber dicantumkan di tiap item.

Progress total: `3 / 41`

---

## 🏛️ Arsitektur & Desain

**P1**
- [ ] Putuskan scope CRUD & rekonsiliasi 3-arah (Repository punya Update/Delete; Service & router tidak): *(A-M1)*
  - [ ] (a) Lengkapi: `Service` interface + `TaskService` + handler + route `PUT`/`PATCH`/`DELETE`, **atau**
  - [ ] (b) Hapus `UpdateById`/`DeleteById` sampai dibutuhkan. `internal/task/{model,repository,router}.go`
- [ ] `go mod tidy` — buang `mongo-driver/v2` (biarkan `quic-go`). `go.mod`. *(A-L1)*

**P2**
- [ ] Konsolidasi composition root: bangun repo→service→handler di satu tempat; `RegisterRouters` hanya map route. `main.go:17-18`, `internal/task/router.go:5-8`. *(A-M2)*
- [ ] Pecah `model.go` → `model.go` (entity) / `dto.go` / interface didekatkan ke consumer. *(A-M3)*

**P3**
- [ ] Propagasi `context.Context` di method Service/Repository. *(A-H2)*
- [ ] Evolusikan signature Repository untuk datastore nyata: `Create(...) (Task, error)`, map DTO→entity di service. *(A-H2)*
- [ ] Tambah config layer (typed `Config` dari env). `main.go`. *(A-M4)*
- [ ] Graceful shutdown: `signal.NotifyContext` + `srv.Shutdown(ctx)`. `main.go:25-29`. *(A-M5)*
- [ ] Map error→HTTP eksplisit saat error surface bertambah. `internal/task/handler.go:62-68`. *(A-L2)*

---

## 🔒 Keamanan

**P0**
- [x] Batasi ukuran request body dengan `http.MaxBytesReader` di route tulis. `internal/middleware/request.go`, dipasang di `POST /tasks` (`internal/task/router.go:17`); handler balas `413` via `isBodyTooLarge` (`internal/task/handler.go:39`). *(S-C2)*
- [ ] Tambah `max=` pada `Title` (`binding:"required,max=256"`). `internal/task/model.go:32-39`. *(S-C2)*
- [ ] Ganti `router.Run()` dengan `http.Server` bertimeout (anti-Slowloris). `main.go:28`. *(S-H1)*
- [ ] Bind ke `127.0.0.1` (configurable), bukan `0.0.0.0`. `main.go:28`. *(S-H4)*
- [ ] Berhenti kembalikan `err.Error()` mentah ke klien; log di server, balas generik. `internal/task/handler.go:25-31, 62-68`. *(S-M1)*

**P1**
- [ ] Tambah paginasi ke `GET /tasks` (`?limit=&offset=`, clamp max). *(S-H2)*
- [ ] Tambah rate limiting (per-IP/token, ketat di route tulis). `main.go`. *(S-H3)*

**P3**
- [ ] Auth/authz: middleware bearer/JWT di `/api/v1` + field owner pada `Task`. *(S-C1)*
- [ ] TLS/HTTPS (app atau reverse proxy) + HSTS. `main.go:28`. *(S-M3)*
- [ ] Security headers middleware (`nosniff`, `X-Frame-Options`, `Cache-Control`). *(S-M2)*
- [ ] CORS eksplisit (allowlist; jangan `*`+credentials). *(S-M4)*
- [ ] `gin.New()` + `gin.ReleaseMode` + logger terstruktur ter-redaksi. `main.go:13`. *(S-L2)*
- [ ] Tolak `id < 1` eksplisit di `GetTask`. `internal/task/handler.go:41-50`. *(S-L1)*
- [ ] Enforce `Content-Type: application/json` pada route tulis. `internal/task/handler.go:22-31`. *(S-L3)*
- [ ] Cap jumlah total task tersimpan. *(S-C2 catatan)*

---

## ⚙️ Konkurensi & Korektness

**P0**
- [x] Tambah `sync.RWMutex` ke `TaskRepository` (read/write lock sesuai method). `internal/task/repository.go:9-70`. *(C1)*
- [x] `GetAll` kembalikan salinan (`make`+`copy`), jangan bocorkan backing array. `internal/task/repository.go:33-35`. *(C-H1)*
- [ ] `DeleteById` pakai `slices.Delete` di bawah write-lock. `internal/task/repository.go:64`. *(C-H2)*

**P3**
- [ ] Kolaps atau isi service layer (`GetById` pass-through murni). `internal/task/service.go:11-18`. *(C-L1)*
- [ ] Pertimbangkan `map[int]Task` untuk hilangkan O(n) scan & bahaya delete-shift. `internal/task/repository.go`. *(C-recommended)*

---

## 🧹 Kualitas Kode & Idiom Go

**P1**
- [ ] `gofmt -w internal/task/model.go internal/task/service.go` (dua file gagal format). *(Q-H1)*

**P2**
- [ ] Hilangkan stutter penamaan: `TaskRepository`→`Repository`, dst; `NewTaskX`→`NewX`. `internal/task/*`, `main.go:17-18`. *(Q-M1)*
- [ ] Perbaiki `omitempty` pada generic `Data T` (pakai `*T` atau drop tag). `internal/response/model.go:12`. *(Q-M3)*
- [ ] Tambah helper response (`OK/Success/Fail`), hapus duplikasi envelope (5×). `internal/task/handler.go`. *(Q-L3)*
- [ ] Health handler pakai konstanta `StatusSuccess`, bukan `"ok"`. `main.go:31-37`. *(Q-L3)*
- [ ] Tambah doc comment ke semua identifier exported. *(Q-L1)*
- [ ] Rename `RegisterRouters` → `RegisterRoutes`. `internal/task/router.go:5`. *(Q-L2)*
- [ ] Adopsi `golangci-lint` + gate CI. *(Q-tooling)*

**P3**
- [ ] Tangani error dari start server (jangan buang return `Run()`/`ListenAndServe()`). `main.go:28`. *(Q-M4)*
  > *(overlap dengan P0 Security H1 — kerjakan sekaligus saat migrasi ke `http.Server`)*

---

## 🧪 Testing & Kelengkapan API

**P1**
- [ ] Unit test repository (table-driven): `Create`, `GetById` hit/miss, `UpdateById`/`DeleteById` sukses+not-found. `internal/task/repository_test.go`. *(T-Critical)*
- [ ] Unit test service dengan fake `Repository` (propagasi error). `internal/task/service_test.go`. *(T-High)*
- [ ] HTTP error-path test: 400 invalid JSON / missing title / non-numeric id, 404 not-found; router segar per subtest. *(T-High)*
- [ ] Concurrent `-race` test (mis. 50 goroutine POST) + `go test -race ./...` di CI. *(T-Low / C)*

**P2**
- [ ] Rapikan `main_test.go`: hapus `_ = createdTaskID`; subtest jangan order-dependent. `main_test.go:80, 17-81`. *(T-Medium, Q-L5/L6)*
- [ ] Test `/health` + dokumentasikan posisinya di luar `/api/v1`. *(T-Medium)*

**P3**
- [ ] Paginasi/filter di list endpoint (`?limit`, `?offset`, `?completed`) + test. *(T-Medium)*
  > *(overlap dengan Security H2 / Arsitektur H2)*

---

### Ringkasan
| Kategori | P0 | P1 | P2 | P3 |
|----------|----|----|----|----|
| Arsitektur | – | 2 | 2 | 5 |
| Keamanan | 5 | 2 | – | 8 |
| Konkurensi | 3 | – | – | 2 |
| Kualitas Kode | – | 1 | 7 | 1 |
| Testing | – | 4 | 2 | 1 |

> Item bertanda *overlap* sengaja ditaruh di kategori rumahnya; kerjakan bersamaan dengan item terkait di kategori lain. Detail lengkap di [01](01-architecture-review.md)–[05](05-testing-api-review.md).

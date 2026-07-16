# Deep Review — Ringkasan Eksekutif

**Proyek:** `api` — Task-management REST API (Go 1.26 + Gin, in-memory store)
**Tanggal review:** 2026-07-14
**Metode:** 5 agent review paralel, masing-masing satu dimensi. Status build: `go test ./...` PASS, `go vet ./...` bersih.

## Daftar Laporan

| # | Dimensi | File |
|---|---------|------|
| 01 | Arsitektur & Desain | [01-architecture-review.md](01-architecture-review.md) |
| 02 | Keamanan | [02-security-review.md](02-security-review.md) |
| 03 | Konkurensi & Korektness | [03-concurrency-correctness-review.md](03-concurrency-correctness-review.md) |
| 04 | Kualitas Kode & Idiom Go | [04-code-quality-review.md](04-code-quality-review.md) |
| 05 | Testing & Kelengkapan API | [05-testing-api-review.md](05-testing-api-review.md) |

## Temuan Lintas-Laporan (muncul di beberapa review)

Isu yang dikonfirmasi oleh lebih dari satu agent — prioritas tertinggi:

1. **Data race pada in-memory repository** (03, 01, 02) — `TaskRepository.tasks`/`nextID` dimutasi tanpa mutex; Gin melayani tiap request di goroutine terpisah. Perbaikan: `sync.RWMutex`.
2. **`UpdateById` / `DeleteById` = dead code** (01, 05) — diimplementasikan di Repository tapi tidak pernah di-route dan tidak ada di interface `Service`. Tidak ada endpoint PUT/PATCH/DELETE. `UpdateTaskRequest` DTO juga tak terpakai.
3. **Tidak ada batas ukuran input** (02, 03) — `ShouldBindJSON` tanpa `MaxBytesReader`, `Title` tanpa `max`, `GetAll` tanpa paginasi → risiko memory-exhaustion DoS.
4. **Kebocoran detail internal** (02, 04) — `err.Error()` mentah dikembalikan ke klien di [handler.go](internal/task/handler.go).

## Ringkasan per Dimensi

- **Arsitektur:** Layering Handler→Service→Repository + DI berbasis interface sudah baik; wiring terpecah canggung antara `main.go` dan `router.go`; `model.go` catch-all (entity + DTO + interface + struct). Tak ada `context.Context`, config layer, atau graceful shutdown. `mongo-driver/v2` tak terpakai → `go mod tidy`.
- **Keamanan:** 2 Critical (tanpa auth/authz; body tak dibatasi → DoS), 4 High, 5 Medium, 4 Low. Checklist 15 item.
- **Konkurensi:** Data race nyata; `DeleteById` memutasi backing array; `GetAll` membocorkan slice internal. `go test -race` tidak menangkapnya karena test tidak konkuren.
- **Kualitas kode:** `gofmt -l` menandai `model.go` & `service.go` (blank line); stutter nama `task.TaskRepository`, error `router.Run()` diabaikan, `omitempty` generik menyesatkan, tanpa doc comment, tanpa unit test di `internal/task`. Saran: `golangci-lint`.
- **Testing:** Coverage `internal/task` = 0%, root 60% (menyesatkan). Hanya satu test happy-path yang order-dependent. Perlu unit + error-path + table-driven tests.

## Rekomendasi Prioritas (Milestone Belajar Go)

Berikut adalah urutan pengerjaan yang dirancang secara bertahap dari pemahaman dasar hingga konsep produksi/arsitektur Go:

### Milestone 1 — Tooling, Idiom, & Konvensi Go (Dasar)
*Fokus pada penulisan kode Go yang idiomatik dan penggunaan tooling standar.*
- [x] Pecah `model.go` → `model.go` (entity) / `dto.go` / `interfaces.go` (sebagian). *(A-M3)*
- [x] Format kode yang tidak rapi (`model.go` & `service.go`) menggunakan `gofmt`. *(Q-H1)*
- [x] `go mod tidy` — `mongo-driver/v2` tetap ada karena di-import secara indirect oleh `gin`. *(A-L1)*
- [x] Format kode (`gofmt -w .`) dan jalankan linter (`golangci-lint`) secara menyeluruh. *(Q-tooling)*
- [i] Hilangkan stutter penamaan: `TaskRepository`→`Repository`, `NewTaskRepository`→`NewRepository` pada package `task`. *(Q-M1)*
- [x] Ubah nama `RegisterRouters` → `RegisterRoutes` di `router.go`. *(Q-L2)*
- [ ] Perbaiki tag `omitempty` pada generic `Data T` (pakai `*T` atau hapus tag). *(Q-M3)*
- [x] Batasi bind server ke `127.0.0.1` (hanya lokal) saat development, bukan `0.0.0.0`. *(S-H4)*
- [ ] *Practical Task:* Implementasikan interface `fmt.Stringer` pada struct `Task` untuk menghasilkan output log yang informatif secara otomatis saat mencetak objek task.
- [ ] *Practical Task:* Buat target kustom di `Makefile` atau shell script (`lint.sh`) untuk menjalankan `gofmt` dan `golangci-lint` secara berurutan.

### Milestone 2 — HTTP Handler & Validasi API Go (Menengah)
*Fokus pada penanganan request HTTP, error handling, dan response formatting.*
- [x] Batasi ukuran request body dengan `http.MaxBytesReader` di route tulis. *(S-C2)*
- [x] Tambah `max=` pada `Title` (`binding:"required,max=256"`). *(S-C2)*
- [ ] Putuskan scope CRUD: Lengkapi route/handler `PUT`/`PATCH`/`DELETE` (lengkapi `Service` interface, impl repo, handler, router). *(A-M1)*
- [ ] Berhenti kembalikan `err.Error()` mentah ke klien; log di server dan balas error generik. *(S-M1)*
- [ ] Tambah helper response (`OK`/`Success`/`Fail`) untuk merapikan handler & hapus duplikasi envelope. *(Q-L3)*
- [ ] Tolak `id < 1` secara eksplisit di `GetTask` dan pastikan header `Content-Type: application/json`. *(S-L1, S-L3)*
- [ ] Ubah health handler menggunakan konstanta `StatusSuccess` dibanding string `"ok"`. *(Q-L3)*
- [ ] *Practical Task:* Buat Middleware kustom sederhana untuk mencatat logs HTTP (method, path, status, latency) secara manual ke stdout.
- [ ] *Practical Task:* Tambahkan field `description` pada request DTO dan Entity, lengkap dengan validasi Gin binding minimal 10 karakter.

### Milestone 3 — Unit Testing di Go (Korektness)
*Fokus pada gaya pengujian Go yang idiomatik (table-driven test) dan isolation.*
- [ ] Buat unit test repository dengan metode table-driven (`Create`, `GetById`, `UpdateById`/`DeleteById`). *(T-Critical)*
- [ ] Buat unit test service menggunakan fake `Repository` untuk pengujian isolasi & propagasi error. *(T-High)*
- [ ] Buat HTTP error-path test untuk router/handler (bad request, invalid JSON, missing fields, not found). *(T-High)*
- [ ] Rapikan `main_test.go` agar subtest tidak order-dependent. *(T-Medium)*
- [ ] Tambahkan unit test untuk `/health`. *(T-Medium)*
- [ ] *Practical Task:* Buat struct fake/mock untuk `Repository` secara manual (tanpa library generator) untuk mensimulasikan database error saat testing.
- [ ] *Practical Task:* Jalankan `go test -coverprofile=coverage.out` dan buat script otomatis untuk membuka HTML coverage via browser dengan `go tool cover -html`.

### Milestone 4 — Struktur Data & Konkurensi Go (Lanjutan)
*Fokus pada safety dan optimasi konkurensi di Go.*
- [x] Tambah `sync.RWMutex` ke `TaskRepository` (lindungi mutasi data dari race condition). *(C1)*
- [x] `GetAll` kembalikan salinan (copy) slice, jangan bocorkan backing array asli. *(C-H1)*
- [ ] Gunakan `slices.Delete` di `DeleteById` di bawah perlindungan write-lock. *(C-H2)*
- [ ] Ubah penyimpanan repository dari slice `[]Task` menjadi `map[int]Task` untuk lookup O(1). *(C-recommended)*
- [ ] Sederhanakan service layer jika method hanya berupa pass-through murni. *(C-L1)*
- [ ] Buat concurrent `-race` test dengan goroutine konkuren, lalu jalankan `go test -race ./...`. *(T-Low / C)*
- [ ] *Practical Task:* Tulis fungsi benchmark (`BenchmarkRepository`) untuk membandingkan performa baca-tulis repositori berbasis slice vs map.
- [ ] *Practical Task:* Gunakan `sync.Once` untuk memastikan inisialisasi state awal database in-memory hanya terjadi sekali saat dipanggil dari goroutine konkuren.

### Milestone 5 — Arsitektur & Hardening Sistem (Kesiapan Produksi)
*Fokus pada penyusunan dependency injection, life-cycle server, dan manajemen request.*
- [x] Ganti `router.Run()` dengan `http.Server` bertimeout (anti-Slowloris). *(S-H1)*
- [ ] Integrasikan composition root: inisialisasi repo, service, handler di satu tempat (misal `main.go`). *(A-M2)*
- [ ] Propagasikan `context.Context` dari handler turun ke Service dan Repository. *(A-H2)*
- [ ] Tambahkan konfigurasi port & timeout menggunakan config layer typed `Config` yang dibaca dari env. *(A-M4)*
- [ ] Terapkan Graceful Shutdown menggunakan `signal.NotifyContext` + `srv.Shutdown(ctx)`. *(A-M5)*
- [ ] Tambahkan paginasi (`limit` & `offset` query params) pada `GET /tasks`. *(S-H2)*
- [ ] Tambahkan rate limiting middleware pada server API. *(S-H3)*
- [ ] *Practical Task:* Implementasikan middleware CORS kustom secara manual (mengatur header `Access-Control-Allow-Origin`, `Methods`, dan `Headers` pada response) tanpa library eksternal.
- [ ] *Practical Task:* Buat shell script sederhana (`smoke-test.sh`) untuk otomatis menguji fungsionalitas API utama (POST, GET, PUT, DELETE) menggunakan `curl` pasca server berjalan.

### Milestone 6 — Keamanan API & Produksi Lanjutan
*Fokus pada pengerasan keamanan sistem web.*
- [ ] Tambahkan Autentikasi/Autorisasi (middleware bearer/JWT) & field owner di `Task`. *(S-C1)*
- [ ] Terapkan TLS/HTTPS + HSTS. *(S-M3)*
- [ ] Tambahkan security headers middleware (`nosniff`, `X-Frame-Options`, `Cache-Control`). *(S-M2)*
- [ ] Konfigurasi CORS secara eksplisit (jangan `*` dengan credentials). *(S-M4)*
- [ ] Gunakan `gin.New()` + `gin.ReleaseMode` + structured logging (slog/zap). *(S-L2)*
- [ ] Batasi total jumlah task tersimpan (cap in-memory storage capacity). *(S-C2)*
- [ ] *Practical Task:* Buat middleware autentikasi JWT kustom untuk mengekstrak token dari header `Authorization: Bearer <token>`, memvalidasi claims, dan menyimpan data `userID` ke dalam `gin.Context`.
- [ ] *Practical Task:* Terapkan middleware Rate Limiter sederhana menggunakan token bucket algorithm berbasis IP client.

> Catatan: semua temuan bersifat rekomendasi. Tidak ada file sumber yang dimodifikasi selama review ini.

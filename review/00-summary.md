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

## Rekomendasi Prioritas (gabungan)

**P0 — sebelum apa pun dianggap produksi**
- [x] Tambah `sync.RWMutex` ke `TaskRepository`.
- [x] Batasi ukuran body (`MaxBytesReader`) di route tulis.
- [x] Tambah `binding:"max=..."` pada `Title`.
- [ ] Hentikan pengembalian `err.Error()` mentah ke klien.
- [x] Tambah server timeouts (ganti `router.Run()` dengan `http.Server` terkonfigurasi).

**P1 — kelengkapan & konsistensi**
- [ ] Ekspos PUT/PATCH/DELETE atau hapus method repo yang tak terpakai (putuskan salah satu).
- [ ] Paginasi `GetAll`.
- [ ] Tambah unit test + error-path test untuk `internal/task`.
- [ ] `go mod tidy` (buang `mongo-driver/v2`).

**P2 — kualitas & kesiapan skala**
- [ ] Propagasi `context.Context` ke Service/Repository.
- [x] Pecah `model.go` → `model.go` / `dto.go` / `interfaces.go` (sebagian, stutter penamaan belum).
- [ ] `gofmt -w .`, integrasikan `golangci-lint` di CI.
- [ ] Graceful shutdown + config layer (port/timeouts dari env).

> Catatan: semua temuan bersifat rekomendasi. Tidak ada file sumber yang dimodifikasi selama review ini.

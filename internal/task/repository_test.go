package task

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// TestConcurrentCreate memastikan Create aman dipanggil banyak goroutine
// sekaligus: jumlah task benar dan tidak ada ID duplikat / hilang.
func TestConcurrentCreate(t *testing.T) {
	repo := NewTaskRepository()

	const goroutines = 100
	t.Logf("menjalankan %d goroutine Create() bersamaan", goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			repo.Create(CreateTaskRequest{Title: fmt.Sprintf("task-%d", n)})
		}(i)
	}
	wg.Wait()

	tasks := repo.GetAll()
	t.Logf("total task setelah semua goroutine selesai: %d (harap %d)", len(tasks), goroutines)
	if len(tasks) != goroutines {
		t.Fatalf("expected %d tasks, got %d", goroutines, len(tasks))
	}

	seen := make(map[int]bool, goroutines)
	for _, task := range tasks {
		if seen[task.Id] {
			t.Fatalf("duplicate id detected: %d", task.Id)
		}
		seen[task.Id] = true
	}
	t.Logf("OK: %d ID unik, tidak ada duplikat", len(seen))
}

// TestConcurrentReadWrite menjalankan semua operasi repository secara
// bersamaan. Tujuannya menyulut race detector (`go test -race`) jika ada
// akses tak terlindungi ke state internal.
func TestConcurrentReadWrite(t *testing.T) {
	repo := NewTaskRepository()

	// Seed data awal supaya operasi read/update/delete punya sasaran.
	const seed = 50
	for i := 0; i < seed; i++ {
		repo.Create(CreateTaskRequest{Title: fmt.Sprintf("seed-%d", i)})
	}
	t.Logf("seed %d task selesai", seed)

	// Hitung tiap operasi biar kelihatan bebannya di log.
	var creates, reads, gets, updates, deletes int64

	const workers = 50
	t.Logf("menjalankan %d worker x 5 operasi = %d goroutine bersamaan", workers, workers*5)

	var wg sync.WaitGroup
	wg.Add(workers * 5)

	for i := 0; i < workers; i++ {
		id := (i % seed) + 1

		go func(n int) {
			defer wg.Done()
			repo.Create(CreateTaskRequest{Title: fmt.Sprintf("new-%d", n)})
			atomic.AddInt64(&creates, 1)
		}(i)

		go func() {
			defer wg.Done()
			_ = repo.GetAll()
			atomic.AddInt64(&reads, 1)
		}()

		go func(id int) {
			defer wg.Done()
			_, _ = repo.GetById(id)
			atomic.AddInt64(&gets, 1)
		}(id)

		go func(id int) {
			defer wg.Done()
			_, _ = repo.UpdateById(id, UpdateTaskRequest{Title: "updated", Completed: true})
			atomic.AddInt64(&updates, 1)
		}(id)

		go func(id int) {
			defer wg.Done()
			_, _ = repo.DeleteById(id)
			atomic.AddInt64(&deletes, 1)
		}(id)
	}

	wg.Wait()

	t.Logf("operasi selesai -> create=%d getAll=%d getById=%d update=%d delete=%d",
		creates, reads, gets, updates, deletes)
	t.Logf("state akhir: %d task (tanpa race)", len(repo.GetAll()))
}

// TestConcurrentDelete memastikan tiap task hanya bisa dihapus tepat sekali:
// walau banyak goroutine berebut menghapus id yang sama, jumlah sukses harus
// sama persis dengan jumlah task yang ada.
func TestConcurrentDelete(t *testing.T) {
	repo := NewTaskRepository()

	const total = 100
	for i := 0; i < total; i++ {
		repo.Create(CreateTaskRequest{Title: fmt.Sprintf("task-%d", i)})
	}
	t.Logf("seed %d task, tiap id diperebutkan 2 goroutine untuk dihapus", total)

	var (
		wg        sync.WaitGroup
		successes int64
	)

	// Dua goroutine berebut menghapus tiap id (1..total).
	for id := 1; id <= total; id++ {
		for dup := 0; dup < 2; dup++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				if _, err := repo.DeleteById(id); err == nil {
					atomic.AddInt64(&successes, 1)
				}
			}(id)
		}
	}
	wg.Wait()

	t.Logf("delete sukses=%d (harap %d), task tersisa=%d", successes, total, len(repo.GetAll()))
	if successes != total {
		t.Fatalf("expected exactly %d successful deletes, got %d", total, successes)
	}
	if remaining := repo.GetAll(); len(remaining) != 0 {
		t.Fatalf("expected repository empty, got %d tasks", len(remaining))
	}
	t.Logf("OK: tiap task dihapus tepat sekali, repository kosong")
}

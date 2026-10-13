package memory

import (
	"sync"
	"testing"

	"meta-automation/internal/domain/job"
)

func TestJobStoreSaveAndGet(t *testing.T) {
	store := NewJobStore()

	task, err := job.New("job-001", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	store.Save(task.Snapshot())

	snapshot, found := store.Get(task.ID)
	if !found {
		t.Fatal("Job tidak ditemukan")
	}

	if snapshot.Status != job.StatusQueued {
		t.Errorf("status = %q, ingin queued", snapshot.Status)
	}

	// Perubahan salinan tidak boleh mengubah data di store.
	snapshot.Name = "Diubah"

	stored, _ := store.Get(task.ID)
	if stored.Name != "WABA Percobaan" {
		t.Error("data store berubah melalui salinan")
	}

	// Simpan snapshot terbaru setelah status berubah.
	if err := task.Start(); err != nil {
		t.Fatal(err)
	}

	store.Save(task.Snapshot())

	updated, found := store.Get(task.ID)
	if !found {
		t.Fatal("Job tidak ditemukan setelah diperbarui")
	}

	if updated.Status != job.StatusRunning {
		t.Errorf("status = %q, ingin running", updated.Status)
	}
}

func TestJobStoreGetMissingJob(t *testing.T) {
	store := NewJobStore()

	_, found := store.Get("tidak-ada")
	if found {
		t.Error("Job yang tidak ada seharusnya tidak ditemukan")
	}
}

func TestInsertRejectsDuplicateID(t *testing.T) {
	store := NewJobStore()

	first := job.Snapshot{
		ID:   "job-001",
		Name: "WABA Pertama",
	}

	if !store.Insert(first) {
		t.Fatal("Insert pertama seharusnya berhasil")
	}

	second := job.Snapshot{
		ID:   "job-001",
		Name: "WABA Kedua",
	}

	if store.Insert(second) {
		t.Fatal("Insert ID duplikat harus ditolak")
	}

	snapshot, found := store.Get(first.ID)
	if !found || snapshot.Name != first.Name {
		t.Error("data pertama tidak dipertahankan")
	}
}

func TestInsertAllowsOnlyOneConcurrentCreation(t *testing.T) {
	store := NewJobStore()

	start := make(chan struct{})
	results := make(chan bool, 2)

	var workers sync.WaitGroup
	workers.Add(2)

	for i := 0; i < 2; i++ {
		go func() {
			defer workers.Done()

			// Kedua goroutine menunggu sinyal mulai yang sama.
			<-start

			results <- store.Insert(job.Snapshot{
				ID:   "job-001",
				Name: "WABA Percobaan",
			})
		}()
	}

	close(start)
	workers.Wait()
	close(results)

	successes := 0
	for inserted := range results {
		if inserted {
			successes++
		}
	}

	if successes != 1 {
		t.Errorf(
			"jumlah Insert berhasil = %d, ingin 1",
			successes,
		)
	}

	if _, found := store.Get("job-001"); !found {
		t.Error("Job tidak tersimpan")
	}
}

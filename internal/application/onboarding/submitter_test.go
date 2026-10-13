package onboarding

import (
	"context"
	"errors"
	"testing"

	"meta-automation/internal/domain/job"
)

type fakeJobQueue struct {
	err   error
	calls int
}

func (q *fakeJobQueue) Enqueue(
	ctx context.Context,
	task *job.Job,
) error {
	q.calls++
	return q.err
}

func TestSubmitCreatesAndEnqueuesJob(t *testing.T) {
	store := newFakeJobStore()
	service := NewService(
		&fakeBrowser{},
		store,
		fakeIDGenerator{id: "job-001"},
	)
	queue := &fakeJobQueue{}
	submitter := NewSubmitter(service, queue)

	snapshot, err := submitter.Submit(
		context.Background(),
		"123456",
		"WABA Percobaan",
	)
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.ID != "job-001" {
		t.Errorf("ID = %q, ingin job-001", snapshot.ID)
	}

	if snapshot.Status != job.StatusQueued {
		t.Errorf("status = %q, ingin queued", snapshot.Status)
	}

	if queue.calls != 1 {
		t.Errorf("Enqueue dipanggil %d kali, ingin 1", queue.calls)
	}

	if _, found := service.Get(snapshot.ID); !found {
		t.Error("Job tidak tersimpan")
	}
}

func TestSubmitCancelsJobWhenQueueRejectsIt(t *testing.T) {
	store := newFakeJobStore()
	service := NewService(
		&fakeBrowser{},
		store,
		fakeIDGenerator{id: "job-001"},
	)
	queue := &fakeJobQueue{err: ErrQueueFull}
	submitter := NewSubmitter(service, queue)

	_, err := submitter.Submit(
		context.Background(),
		"123456",
		"WABA Percobaan",
	)

	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("error = %v, ingin ErrQueueFull", err)
	}

	snapshot, found := service.Get("job-001")
	if !found {
		t.Fatal("snapshot tidak ditemukan")
	}

	if snapshot.Status != job.StatusCancelled {
		t.Errorf(
			"status = %q, ingin %q",
			snapshot.Status,
			job.StatusCancelled,
		)
	}
}

func TestSubmitRejectsInvalidInput(t *testing.T) {
	store := newFakeJobStore()
	service := NewService(
		&fakeBrowser{},
		store,
		fakeIDGenerator{id: "job-001"},
	)
	queue := &fakeJobQueue{}
	submitter := NewSubmitter(service, queue)

	snapshot, err := submitter.Submit(
		context.Background(),
		"123456",
		"   ",
	)

	if err == nil {
		t.Fatal("nama kosong harus ditolak")
	}

	if snapshot.ID != "" {
		t.Error("permintaan gagal tidak boleh mengembalikan ID Job")
	}

	if queue.calls != 0 {
		t.Error("permintaan invalid tidak boleh masuk antrean")
	}

	if len(store.snapshots) != 0 {
		t.Error("permintaan invalid tidak boleh menyimpan Job")
	}
}

func TestSubmitRejectsCancelledContext(t *testing.T) {
	store := newFakeJobStore()
	service := NewService(
		&fakeBrowser{},
		store,
		fakeIDGenerator{id: "job-001"},
	)
	queue := &fakeJobQueue{}
	submitter := NewSubmitter(service, queue)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancel()

	snapshot, err := submitter.Submit(
		ctx,
		"123456",
		"WABA Percobaan",
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, ingin context.Canceled", err)
	}

	if snapshot.ID != "" {
		t.Error("permintaan dibatalkan tidak boleh mengembalikan ID Job")
	}

	if queue.calls != 0 {
		t.Error("permintaan dibatalkan tidak boleh masuk antrean")
	}

	if len(store.snapshots) != 0 {
		t.Error("permintaan dibatalkan tidak boleh menyimpan Job")
	}
}

package onboarding

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"meta-automation/internal/domain/job"
)

func TestQueueProcessesJob(t *testing.T) {
	browser := &fakeBrowser{}
	store := newFakeJobStore()

	service := NewService(
		browser,
		store,
		fakeIDGenerator{id: "job-001"},
	)
	queue := NewQueue(service)

	task, err := service.Create("123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	if err := queue.Enqueue(context.Background(), task); err != nil {
		t.Fatal(err)
	}

	if browser.calls != 0 {
		t.Fatal("Enqueue tidak boleh langsung menjalankan browser")
	}

	if err := queue.processNext(context.Background()); err != nil {
		t.Fatalf("ProcessNext() gagal: %v", err)
	}

	if browser.calls != 1 {
		t.Errorf("browser dipanggil %d kali, ingin 1", browser.calls)
	}

	snapshot, found := service.Get("job-001")
	if !found {
		t.Fatal("snapshot Job tidak ditemukan")
	}

	if snapshot.Status != job.StatusAwaitingManualAction {
		t.Errorf(
			"status = %q, ingin %q",
			snapshot.Status,
			job.StatusAwaitingManualAction,
		)
	}
}

func TestQueueStopsWaitingWhenContextIsCancelled(t *testing.T) {
	service := NewService(
		&fakeBrowser{},
		newFakeJobStore(),
		fakeIDGenerator{id: "job-001"},
	)
	queue := NewQueue(service)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := queue.processNext(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, ingin context.Canceled", err)
	}
}

type sequenceRunner struct {
	calls     int
	secondRun chan struct{}
}

func (r *sequenceRunner) Run(
	ctx context.Context,
	task *job.Job,
) error {
	r.calls++

	if r.calls == 1 {
		return errors.New("job pertama gagal")
	}

	close(r.secondRun)
	return nil
}

func TestWorkerContinuesAfterJobFailure(t *testing.T) {
	runner := &sequenceRunner{
		secondRun: make(chan struct{}),
	}
	queue := NewQueue(runner)

	for _, id := range []string{"job-001", "job-002"} {
		task, err := job.New(id, "123456", "WABA Percobaan")
		if err != nil {
			t.Fatal(err)
		}

		if err := queue.Enqueue(context.Background(), task); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobErrors := make(chan error, 2)
	result := make(chan error, 1)

	go func() {
		result <- queue.Run(ctx, func(err error) {
			jobErrors <- err
		})
	}()

	select {
	case <-runner.secondRun:
	case <-time.After(2 * time.Second):
		t.Fatal("worker tidak melanjutkan job kedua")
	}

	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error worker = %v, ingin context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker tidak berhenti")
	}

	select {
	case <-jobErrors:
	default:
		t.Error("error job pertama tidak dilaporkan")
	}

	if runner.calls != 2 {
		t.Errorf("runner dipanggil %d kali, ingin 2", runner.calls)
	}
}

func TestQueueRejectsSecondWorker(t *testing.T) {
	browser := &blockingBrowser{
		started: make(chan struct{}),
	}

	service := NewService(
		browser,
		newFakeJobStore(),
		fakeIDGenerator{id: "job-001"},
	)
	queue := NewQueue(service)

	task, err := service.Create("123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	if err := queue.Enqueue(context.Background(), task); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstResult := make(chan error, 1)

	go func() {
		firstResult <- queue.Run(ctx, func(err error) {})
	}()

	// Tunggu sampai worker pertama menjalankan browser.
	select {
	case <-browser.started:
	case <-time.After(2 * time.Second):
		t.Fatal("worker pertama tidak mulai")
	}

	// Worker pertama masih aktif, jadi worker kedua harus ditolak.
	secondErr := queue.Run(ctx, func(err error) {})
	if secondErr == nil {
		t.Error("worker kedua seharusnya ditolak")
	}

	cancel()

	select {
	case err := <-firstResult:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf(
				"error worker pertama = %v, ingin context.Canceled",
				err,
			)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker pertama tidak berhenti")
	}
}

func TestQueueRejectsJobAfterClose(t *testing.T) {
	service := NewService(
		&fakeBrowser{},
		newFakeJobStore(),
		fakeIDGenerator{id: "job-001"},
	)
	queue := NewQueue(service)

	task, err := job.New("job-001", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	queue.Close()
	queue.Close()

	err = queue.Enqueue(context.Background(), task)

	if !errors.Is(err, ErrQueueClosed) {
		t.Fatalf("error = %v, ingin ErrQueueClosed", err)
	}

	if task.Status() != job.StatusQueued {
		t.Error("Job yang ditolak belum menjadi milik antrean")
	}
}

func TestQueueRejectsJobWhenFull(t *testing.T) {
	service := NewService(
		&fakeBrowser{},
		newFakeJobStore(),
		fakeIDGenerator{id: "unused"},
	)
	queue := NewQueue(service)

	for i := 0; i < 10; i++ {
		task, err := job.New(
			fmt.Sprintf("job-%d", i),
			"123456",
			"WABA Percobaan",
		)
		if err != nil {
			t.Fatal(err)
		}

		if err := queue.Enqueue(context.Background(), task); err != nil {
			t.Fatal(err)
		}
	}

	extra, err := job.New("job-extra", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	err = queue.Enqueue(context.Background(), extra)

	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("error = %v, ingin ErrQueueFull", err)
	}
}

func TestDrainCancelsPendingJobAndSavesSnapshot(t *testing.T) {
	browser := &fakeBrowser{}
	store := newFakeJobStore()

	service := NewService(
		browser,
		store,
		fakeIDGenerator{id: "job-001"},
	)
	queue := NewQueue(service)

	task, err := service.Create("123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	if err := queue.Enqueue(context.Background(), task); err != nil {
		t.Fatal(err)
	}

	queue.Close()

	pending, err := queue.Drain()
	if err != nil {
		t.Fatal(err)
	}

	if len(pending) != 1 {
		t.Fatalf("sisa antrean = %d, ingin 1", len(pending))
	}

	for _, task := range pending {
		if err := service.CancelQueued(task); err != nil {
			t.Fatal(err)
		}
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

	if browser.calls != 0 {
		t.Error("Job antre tidak boleh menjalankan browser saat shutdown")
	}

	remaining, err := queue.Drain()
	if err != nil {
		t.Fatal(err)
	}

	if len(remaining) != 0 {
		t.Error("antrean seharusnya sudah kosong")
	}
}

func TestDrainRejectsOpenQueue(t *testing.T) {
	service := NewService(
		&fakeBrowser{},
		newFakeJobStore(),
		fakeIDGenerator{id: "unused"},
	)
	queue := NewQueue(service)

	if _, err := queue.Drain(); err == nil {
		t.Error("antrean terbuka tidak boleh dikosongkan")
	}
}

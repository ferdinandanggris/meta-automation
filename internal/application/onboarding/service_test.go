package onboarding

import (
	"context"
	"errors"
	"meta-automation/internal/domain/job"
	"testing"
	"time"
)

type fakeBrowser struct {
	jobID      string
	businessID string
	name       string
	calls      int
	err        error
}

func (b *fakeBrowser) PrepareWABA(
	ctx context.Context,
	jobID, businessID, name string,
) error {
	b.jobID = jobID
	b.businessID = businessID
	b.name = name
	b.calls++
	return b.err
}

func TestRunPreparesWABAAndAwaitsManualAction(t *testing.T) {
	browser := &fakeBrowser{}
	service := NewService(browser, newFakeJobStore(), fakeIDGenerator{id: "job-001"})

	task, err := job.New("job-001", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Run(context.Background(), task); err != nil {
		t.Fatalf("Run() gagal: %v", err)
	}

	if browser.calls != 1 {
		t.Errorf("browser dipanggil %d kali, ingin 1", browser.calls)
	}

	if browser.businessID != task.BusinessID {
		t.Errorf("business ID yang dikirim ke browser tidak sesuai")
	}

	if browser.name != task.Name {
		t.Errorf("name yang dikirim ke browser tidak sesuai")
	}

	if browser.jobID != task.ID {
		t.Error("job ID yang dikirim ke browser tidak sesuai")
	}

	if task.Status() != job.StatusAwaitingManualAction {
		t.Errorf("status %q, ingin %q", task.Status(), job.StatusAwaitingManualAction)
	}
}

func TestRunMarksJobFailedWhenBrowserFails(t *testing.T) {
	browserErr := errors.New("tombol Tambahkan tidak ditemukan")
	browser := &fakeBrowser{err: browserErr}
	service := NewService(browser, newFakeJobStore(), fakeIDGenerator{id: "job-001"})

	task, err := job.New("job-001", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	err = service.Run(context.Background(), task)

	if !errors.Is(err, browserErr) {
		t.Fatalf("error = %v, ingin error browser", err)
	}

	if task.Status() != job.StatusFailed {
		t.Errorf(
			"status = %q, ingin %q",
			task.Status(),
			job.StatusFailed,
		)
	}
}

func TestRunRejectsAlreadyProcessedJob(t *testing.T) {
	browser := &fakeBrowser{}
	service := NewService(browser, newFakeJobStore(), fakeIDGenerator{id: "job-001"})

	task, err := job.New("job-001", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Run(context.Background(), task); err != nil {
		t.Fatalf("Run() gagal: %v", err)
	}

	if err := service.Run(context.Background(), task); err == nil {
		t.Fatal("menjalankan ulang job harus ditolak")
	}

	if browser.calls != 1 {
		t.Errorf("browser dipanggil %d kali, ingin tetap 1", browser.calls)
	}

	if task.Status() != job.StatusAwaitingManualAction {
		t.Error("percobaan ulang tidak boleh mengubah status job")
	}
}

func TestRunCancelsBeforeStartingBrowser(t *testing.T) {
	browser := &fakeBrowser{}
	service := NewService(browser, newFakeJobStore(), fakeIDGenerator{id: "job-001"})

	task, err := job.New("job-001", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = service.Run(ctx, task)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, ingin context.Canceled", err)
	}

	if browser.calls != 0 {
		t.Error("browser tidak boleh dipanggil")
	}

	if task.Status() != job.StatusCancelled {
		t.Errorf("status = %q, ingin %q", task.Status(), job.StatusCancelled)
	}
}

func TestRunCancelsWhenBrowserReturnsCancellation(t *testing.T) {
	browser := &fakeBrowser{err: context.Canceled}
	service := NewService(browser, newFakeJobStore(), fakeIDGenerator{id: "job-001"})

	task, err := job.New("job-001", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	err = service.Run(context.Background(), task)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, ingin context.Canceled", err)
	}

	if task.Status() != job.StatusCancelled {
		t.Errorf("status = %q, ingin %q", task.Status(), job.StatusCancelled)
	}
}

type blockingBrowser struct {
	started chan struct{}
}

func (b *blockingBrowser) PrepareWABA(
	ctx context.Context,
	jobID, businessID, name string,
) error {
	close(b.started)

	<-ctx.Done()
	return ctx.Err()
}

func TestRunStopsWhenContextIsCancelled(t *testing.T) {
	browser := &blockingBrowser{
		started: make(chan struct{}),
	}

	service := NewService(browser, newFakeJobStore(), fakeIDGenerator{id: "job-001"})

	task, err := job.New("job-001", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)

	go func() {
		result <- service.Run(ctx, task)
	}()

	select {
	case <-browser.started:
	case <-time.After(2 * time.Second):
		t.Fatal("browser tidak mulai")
	}

	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, ingin context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("proses tidak berhenti setelah dibatalkan")
	}

	if task.Status() != job.StatusCancelled {
		t.Errorf("status = %q, ingin %q", task.Status(), job.StatusCancelled)
	}
}

type cancellingBrowser struct {
	cancel context.CancelFunc
}

func (b *cancellingBrowser) PrepareWABA(
	ctx context.Context,
	jobID, businessID, name string,
) error {
	b.cancel()
	return nil
}

func TestRunChecksCancellationAfterBrowserReturns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	browser := &cancellingBrowser{cancel: cancel}
	service := NewService(browser, newFakeJobStore(), fakeIDGenerator{id: "job-001"})

	task, err := job.New("job-001", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	err = service.Run(ctx, task)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, ingin context.Canceled", err)
	}

	if task.Status() != job.StatusCancelled {
		t.Errorf(
			"status = %q, ingin %q",
			task.Status(),
			job.StatusCancelled,
		)
	}
}

type fakeJobStore struct {
	snapshots map[string]job.Snapshot
}

func newFakeJobStore() *fakeJobStore {
	return &fakeJobStore{
		snapshots: make(map[string]job.Snapshot),
	}
}

func (s *fakeJobStore) Save(snapshot job.Snapshot) {
	s.snapshots[snapshot.ID] = snapshot
}

func (s *fakeJobStore) Get(id string) (job.Snapshot, bool) {
	snapshot, found := s.snapshots[id]
	return snapshot, found
}

func (s *fakeJobStore) Insert(snapshot job.Snapshot) bool {
	if _, found := s.snapshots[snapshot.ID]; found {
		return false
	}

	s.snapshots[snapshot.ID] = snapshot
	return true
}

func TestCreateRejectsDuplicateID(t *testing.T) {
	store := newFakeJobStore()
	service := NewService(
		&fakeBrowser{},
		store,
		fakeIDGenerator{id: "job-001"},
	)

	if _, err := service.Create("123456", "WABA Pertama"); err != nil {
		t.Fatal(err)
	}

	task, err := service.Create("654321", "WABA Kedua")

	if !errors.Is(err, ErrJobAlreadyExists) {
		t.Fatalf("error = %v, ingin ErrJobAlreadyExists", err)
	}

	if task != nil {
		t.Error("Job duplikat seharusnya nil")
	}

	snapshot, found := store.Get("job-001")
	if !found {
		t.Fatal("Job pertama hilang")
	}

	if snapshot.Name != "WABA Pertama" {
		t.Error("Job pertama tertimpa")
	}
}

func TestRunSavesFinalSnapshot(t *testing.T) {
	browser := &fakeBrowser{}
	store := newFakeJobStore()
	service := NewService(browser, store, fakeIDGenerator{id: "job-001"})

	task, err := job.New("job-001", "123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Run(context.Background(), task); err != nil {
		t.Fatal(err)
	}

	snapshot, found := store.Get(task.ID)
	if !found {
		t.Fatal("snapshot Job tidak tersimpan")
	}

	if snapshot.Status != job.StatusAwaitingManualAction {
		t.Errorf(
			"status tersimpan = %q, ingin %q",
			snapshot.Status,
			job.StatusAwaitingManualAction,
		)
	}
}

func TestRunSavesSnapshotWhenBrowserStops(t *testing.T) {
	tests := []struct {
		name       string
		browserErr error
		wantStatus job.Status
	}{
		{
			name:       "browser gagal",
			browserErr: errors.New("tombol tidak ditemukan"),
			wantStatus: job.StatusFailed,
		},
		{
			name:       "proses dibatalkan",
			browserErr: context.Canceled,
			wantStatus: job.StatusCancelled,
		},
		{
			name:       "browser timeout",
			browserErr: context.DeadlineExceeded,
			wantStatus: job.StatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			browser := &fakeBrowser{err: tt.browserErr}
			store := newFakeJobStore()
			service := NewService(browser, store, fakeIDGenerator{id: "job-001"})

			task, err := service.Create("123456", "WABA Percobaan")
			if err != nil {
				t.Fatal(err)
			}

			err = service.Run(context.Background(), task)

			if !errors.Is(err, tt.browserErr) {
				t.Fatalf(
					"error = %v, ingin %v",
					err,
					tt.browserErr,
				)
			}

			snapshot, found := store.Get(task.ID)
			if !found {
				t.Fatal("snapshot akhir tidak tersimpan")
			}

			if snapshot.Status != tt.wantStatus {
				t.Errorf(
					"status tersimpan = %q, ingin %q",
					snapshot.Status,
					tt.wantStatus,
				)
			}
		})
	}
}

func TestCreateSavesQueuedJob(t *testing.T) {
	store := newFakeJobStore()
	service := NewService(&fakeBrowser{}, store, fakeIDGenerator{id: "job-001"})

	task, err := service.Create("123456", "WABA Percobaan")
	if err != nil {
		t.Fatalf("Create() gagal: %v", err)
	}

	snapshot, found := store.Get(task.ID)
	if !found {
		t.Fatal("Job baru tidak tersimpan")
	}

	if snapshot.Status != job.StatusQueued {
		t.Errorf(
			"status = %q, ingin %q",
			snapshot.Status,
			job.StatusQueued,
		)
	}

	if snapshot.BusinessID != "123456" {
		t.Error("business ID tersimpan tidak sesuai")
	}

	if snapshot.Name != "WABA Percobaan" {
		t.Error("nama tersimpan tidak sesuai")
	}
}

func TestCreateDoesNotSaveInvalidJob(t *testing.T) {
	store := newFakeJobStore()
	service := NewService(&fakeBrowser{}, store, fakeIDGenerator{id: "job-001"})

	task, err := service.Create("123456", "   ")

	if err == nil {
		t.Fatal("nama kosong harus ditolak")
	}

	if task != nil {
		t.Error("Job seharusnya nil")
	}

	if _, found := store.Get("job-001"); found {
		t.Error("Job tidak valid tidak boleh tersimpan")
	}
}

func TestGetReturnsCreatedJob(t *testing.T) {
	store := newFakeJobStore()
	service := NewService(&fakeBrowser{}, store, fakeIDGenerator{id: "job-001"})

	_, err := service.Create("123456", "WABA Percobaan")
	if err != nil {
		t.Fatal(err)
	}

	snapshot, found := service.Get("job-001")
	if !found {
		t.Fatal("Job tidak ditemukan")
	}

	if snapshot.ID != "job-001" {
		t.Errorf("ID = %q, ingin job-001", snapshot.ID)
	}

	if snapshot.Status != job.StatusQueued {
		t.Errorf(
			"status = %q, ingin %q",
			snapshot.Status,
			job.StatusQueued,
		)
	}
}

func TestGetReturnsFalseForMissingJob(t *testing.T) {
	service := NewService(&fakeBrowser{}, newFakeJobStore(), fakeIDGenerator{id: "job-001"})

	_, found := service.Get("tidak-ada")
	if found {
		t.Error("Job yang tidak ada seharusnya tidak ditemukan")
	}
}

type fakeIDGenerator struct {
	id  string
	err error
}

func (g fakeIDGenerator) Generate() (string, error) {
	return g.id, g.err
}

func TestCreateDoesNotSaveWhenIDGenerationFails(t *testing.T) {
	generatorErr := errors.New("generator gagal")
	store := newFakeJobStore()

	service := NewService(
		&fakeBrowser{},
		store,
		fakeIDGenerator{err: generatorErr},
	)

	task, err := service.Create("123456", "WABA Percobaan")

	if !errors.Is(err, generatorErr) {
		t.Fatalf("error = %v, ingin error generator", err)
	}

	if task != nil {
		t.Error("Job seharusnya nil")
	}

	if len(store.snapshots) != 0 {
		t.Error("tidak boleh ada snapshot tersimpan")
	}
}

func TestCreateReturnsValidationError(t *testing.T) {
	service := NewService(
		&fakeBrowser{},
		newFakeJobStore(),
		fakeIDGenerator{id: "job-001"},
	)

	_, err := service.Create("123456", "   ")

	var validationErr *job.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %v, ingin ValidationError", err)
	}

	if validationErr.Field != "name" {
		t.Errorf(
			"field = %q, ingin name",
			validationErr.Field,
		)
	}
}

package onboarding

import (
	"context"
	"errors"
	"meta-automation/internal/domain/job"
	"testing"
)

type fakeBrowser struct {
	businessID string
	name       string
	calls      int
	err        error
}

func (b *fakeBrowser) PrepareWABA(
	ctx context.Context,
	businessID, name string,
) error {
	b.businessID = businessID
	b.name = name
	b.calls++
	return b.err
}

func TestRunPreparesWABAAndAwaitsManualAction(t *testing.T) {
	browser := &fakeBrowser{}
	service := NewService(browser)

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

	if task.Status() != job.StatusAwaitingManualAction {
		t.Errorf("status %q, ingin %q", task.Status(), job.StatusAwaitingManualAction)
	}
}

func TestRunMarksJobFailedWhenBrowserFails(t *testing.T) {
	browserErr := errors.New("tombol Tambahkan tidak ditemukan")
	browser := &fakeBrowser{err: browserErr}
	service := NewService(browser)

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

package onboarding

import (
	"context"
	"errors"
	"sync"
	"testing"

	"meta-automation/internal/domain/job"
)

type fakeSubmission struct {
	calls int
	err   error
}

func (s *fakeSubmission) Submit(
	ctx context.Context,
	businessID, name string,
) (job.Snapshot, error) {
	s.calls++
	if s.err != nil {
		return job.Snapshot{}, s.err
	}

	return job.Snapshot{
		ID:         "job-001",
		BusinessID: businessID,
		Name:       name,
		Status:     job.StatusQueued,
	}, nil
}

func TestIdempotentSubmitterReusesAcceptedJob(t *testing.T) {
	next := &fakeSubmission{}
	submitter := NewIdempotentSubmitter(next)

	first, err := submitter.Submit(
		context.Background(),
		"request-001",
		"123456",
		"WABA Percobaan",
	)
	if err != nil {
		t.Fatal(err)
	}

	second, err := submitter.Submit(
		context.Background(),
		"request-001",
		"123456",
		"WABA Percobaan",
	)
	if err != nil {
		t.Fatal(err)
	}

	if first.ID != second.ID {
		t.Error("retry harus mengembalikan ID yang sama")
	}

	if next.calls != 1 {
		t.Errorf("Submit dipanggil %d kali, ingin 1", next.calls)
	}
}

func TestIdempotentSubmitterRejectsDifferentInput(t *testing.T) {
	next := &fakeSubmission{}
	submitter := NewIdempotentSubmitter(next)

	_, err := submitter.Submit(
		context.Background(),
		"request-001",
		"123456",
		"WABA Pertama",
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = submitter.Submit(
		context.Background(),
		"request-001",
		"123456",
		"WABA Kedua",
	)

	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, ingin ErrIdempotencyConflict", err)
	}

	if next.calls != 1 {
		t.Error("request konflik tidak boleh membuat Job lagi")
	}
}

func TestIdempotentSubmitterAllowsRetryAfterFailure(t *testing.T) {
	next := &fakeSubmission{err: ErrQueueFull}
	submitter := NewIdempotentSubmitter(next)

	_, err := submitter.Submit(
		context.Background(),
		"request-001",
		"123456",
		"WABA Percobaan",
	)

	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("error = %v, ingin ErrQueueFull", err)
	}

	// Simulasikan antrean sudah dapat menerima Job.
	next.err = nil

	snapshot, err := submitter.Submit(
		context.Background(),
		"request-001",
		"123456",
		"WABA Percobaan",
	)
	if err != nil {
		t.Fatalf("retry gagal: %v", err)
	}

	if snapshot.ID != "job-001" {
		t.Errorf("ID = %q, ingin job-001", snapshot.ID)
	}

	if next.calls != 2 {
		t.Errorf("Submit dipanggil %d kali, ingin 2", next.calls)
	}
}

func TestIdempotentSubmitterHandlesConcurrentRetries(t *testing.T) {
	next := &fakeSubmission{}
	submitter := NewIdempotentSubmitter(next)

	type result struct {
		snapshot job.Snapshot
		err      error
	}

	start := make(chan struct{})
	results := make(chan result, 2)

	var workers sync.WaitGroup
	workers.Add(2)

	for i := 0; i < 2; i++ {
		go func() {
			defer workers.Done()

			<-start

			snapshot, err := submitter.Submit(
				context.Background(),
				"request-001",
				"123456",
				"WABA Percobaan",
			)

			results <- result{
				snapshot: snapshot,
				err:      err,
			}
		}()
	}

	close(start)
	workers.Wait()
	close(results)

	for got := range results {
		if got.err != nil {
			t.Errorf("Submit gagal: %v", got.err)
			continue
		}

		if got.snapshot.ID != "job-001" {
			t.Errorf("ID = %q, ingin job-001", got.snapshot.ID)
		}
	}

	if next.calls != 1 {
		t.Errorf(
			"Submit dipanggil %d kali, ingin hanya 1",
			next.calls,
		)
	}
}

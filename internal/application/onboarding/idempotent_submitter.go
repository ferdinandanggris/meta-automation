package onboarding

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"meta-automation/internal/domain/job"
)

var (
	ErrInvalidIdempotencyKey = errors.New("idempotency key wajib diisi dan maksimal 128 byte")
	ErrIdempotencyConflict   = errors.New("idempotency key sudah digunakan dengan input berbeda")
)

type Submission interface {
	Submit(
		ctx context.Context,
		businessID, name string,
	) (job.Snapshot, error)
}

type submissionRecord struct {
	businessID string
	name       string
	snapshot   job.Snapshot
}

type IdempotentSubmitter struct {
	next Submission

	mu      sync.Mutex
	records map[string]submissionRecord
}

func NewIdempotentSubmitter(next Submission) *IdempotentSubmitter {
	return &IdempotentSubmitter{
		next:    next,
		records: make(map[string]submissionRecord),
	}
}

func (s *IdempotentSubmitter) Submit(
	ctx context.Context,
	key, businessID, name string,
) (job.Snapshot, error) {
	key = strings.TrimSpace(key)
	businessID = strings.TrimSpace(businessID)
	name = strings.TrimSpace(name)

	if key == "" || len(key) > 128 {
		return job.Snapshot{}, ErrInvalidIdempotencyKey
	}

	if err := ctx.Err(); err != nil {
		return job.Snapshot{}, fmt.Errorf("terima request: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Context bisa dibatalkan ketika menunggu lock.
	if err := ctx.Err(); err != nil {
		return job.Snapshot{}, fmt.Errorf("terima request: %w", err)
	}

	if previous, found := s.records[key]; found {
		if previous.businessID != businessID || previous.name != name {
			return job.Snapshot{}, ErrIdempotencyConflict
		}

		return previous.snapshot, nil
	}

	snapshot, err := s.next.Submit(ctx, businessID, name)
	if err != nil {
		return job.Snapshot{}, err
	}

	s.records[key] = submissionRecord{
		businessID: businessID,
		name:       name,
		snapshot:   snapshot,
	}

	return snapshot, nil
}

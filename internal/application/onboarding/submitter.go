package onboarding

import (
	"context"
	"fmt"

	"meta-automation/internal/domain/job"
)

type JobQueue interface {
	Enqueue(ctx context.Context, task *job.Job) error
}

type Submitter struct {
	service *Service
	queue   JobQueue
}

func NewSubmitter(service *Service, queue JobQueue) *Submitter {
	return &Submitter{
		service: service,
		queue:   queue,
	}
}

func (s *Submitter) Submit(
	ctx context.Context,
	businessID, name string,
) (job.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return job.Snapshot{}, fmt.Errorf("terima permintaan job: %w", err)
	}

	task, err := s.service.Create(businessID, name)
	if err != nil {
		return job.Snapshot{}, err
	}

	// Ambil snapshot sebelum kepemilikan Job diserahkan ke worker.
	snapshot := task.Snapshot()

	if err := s.queue.Enqueue(ctx, task); err != nil {
		if cancelErr := s.service.CancelQueued(task); cancelErr != nil {
			return job.Snapshot{}, fmt.Errorf(
				"antrean menolak job (%v); pembatalan gagal: %w",
				err,
				cancelErr,
			)
		}

		return job.Snapshot{}, fmt.Errorf("antrekan job: %w", err)
	}

	return snapshot, nil
}

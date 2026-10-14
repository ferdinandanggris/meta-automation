package onboarding

import (
	"context"
	"errors"
	"fmt"
	"meta-automation/internal/domain/job"
)

type Browser interface {
	PrepareWABA(
		ctx context.Context,
		jobID, businessID, name string,
	) error
}

type Service struct {
	browser Browser
	store   JobStore
	ids     IDGenerator
}

var ErrJobAlreadyExists = errors.New("job ID sudah ada")

func NewService(
	browser Browser,
	store JobStore,
	ids IDGenerator,
) *Service {
	return &Service{
		browser: browser,
		store:   store,
		ids:     ids,
	}
}
func (s *Service) Run(ctx context.Context, task *job.Job) error {

	defer func() {
		s.store.Save(task.Snapshot())
	}()

	if err := ctx.Err(); err != nil {
		if transitionErr := task.Cancel(); transitionErr != nil {
			return fmt.Errorf("batalkan job: %w", transitionErr)
		}

		return fmt.Errorf("proses dibatalkan sebelum mulai: %w", err)
	}

	if err := task.Start(); err != nil {
		return fmt.Errorf("mulai job: %w", err)
	}

	s.store.Save(task.Snapshot())

	if err := s.browser.PrepareWABA(
		ctx,
		task.ID,
		task.BusinessID,
		task.Name,
	); err != nil {
		if errors.Is(err, context.Canceled) {
			if transitionErr := task.Cancel(); transitionErr != nil {
				return fmt.Errorf("batalkan job %w", transitionErr)
			}

			return fmt.Errorf("proses dibatalkan: %w", err)
		}

		if transitionErr := task.Fail(); transitionErr != nil {
			return fmt.Errorf(
				"siapkan WABA gagal (%v); ubah status gagal: %w",
				err,
				transitionErr,
			)
		}

		return fmt.Errorf("siapkan WABA: %w", err)
	}

	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			if transitionErr := task.Cancel(); transitionErr != nil {
				return fmt.Errorf("batalkan job: %w", transitionErr)
			}
		} else {
			if transitionErr := task.Fail(); transitionErr != nil {
				return fmt.Errorf("ubah status gagal: %w", transitionErr)
			}
		}

		return fmt.Errorf("proses berhenti setelah persiapan: %w", err)
	}

	if err := task.AwaitManualAction(); err != nil {
		return fmt.Errorf("menunggu tindakan manual: %w", err)
	}

	return nil
}

func (s *Service) Create(
	businessID, name string,
) (*job.Job, error) {
	id, err := s.ids.Generate()
	if err != nil {
		return nil, fmt.Errorf("buat ID job: %w", err)
	}

	task, err := job.New(id, businessID, name)
	if err != nil {
		return nil, fmt.Errorf("buat job: %w", err)
	}

	if !s.store.Insert(task.Snapshot()) {
		return nil, ErrJobAlreadyExists
	}

	return task, nil
}

func (s *Service) Get(id string) (job.Snapshot, bool) {
	return s.store.Get(id)
}

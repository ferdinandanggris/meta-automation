package onboarding

import (
	"context"
	"fmt"
	"meta-automation/internal/domain/job"
)

type Browser interface {
	PrepareWABA(ctx context.Context, businessID, name string) error
}

type Service struct {
	browser Browser
}

func NewService(browser Browser) *Service {
	return &Service{
		browser: browser,
	}
}

func (s *Service) Run(ctx context.Context, task *job.Job) error {
	if err := task.Start(); err != nil {
		return fmt.Errorf("mulai job: %w", err)
	}

	if err := s.browser.PrepareWABA(
		ctx,
		task.BusinessID,
		task.Name,
	); err != nil {
		if transitionErr := task.Fail(); transitionErr != nil {
			return fmt.Errorf(
				"siapkan WABA gagal (%v); ubah status gagal: %w",
				err,
				transitionErr,
			)
		}

		return fmt.Errorf("siapkan WABA: %w", err)
	}

	if err := task.AwaitManualAction(); err != nil {
		return fmt.Errorf("menunggu tindakan manual: %w", err)
	}

	return nil
}

package browser

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Session) PrepareWABA(
	ctx context.Context,
	jobID, businessID, name string,
) (resultErr error) {

	if err := ctx.Err(); err != nil {
		return err
	}

	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return fmt.Errorf("job ID wajib diisi")
	}

	if err := s.beginPreparation(); err != nil {
		return err
	}

	defer func() {
		s.endPreparation(resultErr == nil)
	}()

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("nama WABA wajib diisi")
	}

	target, err := buildWABASettingsURL(businessID)
	if err != nil {
		return err
	}

	processCtx, cancelProcess := context.WithTimeout(
		ctx,
		2*time.Minute,
	)
	defer cancelProcess()

	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{
			name: "buka pengaturan WABA",
			run: func(ctx context.Context) error {
				return s.OpenContext(ctx, target)
			},
		},
		{
			name: "klik Tambahkan",
			run: func(ctx context.Context) error {
				detection, err := s.WaitForAddButton(ctx)
				if err != nil {
					return err
				}

				if detection.Count != 1 {
					return fmt.Errorf(
						"ditemukan %d kandidat Tambahkan",
						detection.Count,
					)
				}

				return s.ClickAddButton(ctx)
			},
		},
		{
			name: "klik pilihan akun baru",
			run: func(ctx context.Context) error {
				detection, err := s.WaitForCreateButton(ctx)
				if err != nil {
					return err
				}

				if detection.Count != 1 {
					return fmt.Errorf(
						"ditemukan %d pilihan akun baru",
						detection.Count,
					)
				}

				return s.ClickCreateButton(ctx)
			},
		},
		{
			name: "isi nama WABA",
			run: func(ctx context.Context) error {
				detection, err := s.WaitForNameInput(ctx)
				if err != nil {
					return err
				}

				if detection.Count != 1 {
					return fmt.Errorf(
						"ditemukan %d input nama",
						detection.Count,
					)
				}

				return s.FillNameInput(ctx, name)
			},
		},
		{
			name: "cari Lanjutkan",
			run: func(ctx context.Context) error {
				detection, err := s.WaitForContinueButton(ctx)
				if err != nil {
					return err
				}

				if detection.Count != 1 {
					return fmt.Errorf(
						"ditemukan %d tombol Lanjutkan",
						detection.Count,
					)
				}

				return nil
			},
		},
	}

	for _, step := range steps {
		if err := processCtx.Err(); err != nil {
			return fmt.Errorf("persiapan WABA dihentikan: %w", err)
		}

		stepCtx, cancelStep := context.WithTimeout(
			processCtx,
			30*time.Second,
		)

		err := step.run(stepCtx)

		// Ambil error context sebelum cancelStep dipanggil.
		contextErr := stepCtx.Err()
		cancelStep()

		if contextErr != nil {
			return fmt.Errorf("%s: %w", step.name, contextErr)
		}

		if err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}

	return nil
}

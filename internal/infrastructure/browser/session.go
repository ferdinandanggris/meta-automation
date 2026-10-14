package browser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"github.com/chromedp/chromedp"
)

type Session struct {
	ctx             context.Context
	cancelBrowser   context.CancelFunc
	cancelAllocator context.CancelFunc
	mu                   sync.Mutex
	preparing            bool
	awaitingManualAction bool
}

func NewSession(
	parent context.Context,
	executablePath, profileDir string,
) (*Session, error) {
	if executablePath == "" || profileDir == "" {
		return nil, errors.New("lokasi browser dan profil wajib diisi")
	}

	profilePath, err := filepath.Abs(profileDir)
	if err != nil {
		return nil, fmt.Errorf("tentukan lokasi profil: %w", err)
	}

	if err := os.MkdirAll(profilePath, 0700); err != nil {
		return nil, fmt.Errorf("buat folder profil: %w", err)
	}

	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(
		parent,
		chromedp.ExecPath(executablePath),
		chromedp.UserDataDir(profilePath),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false),
		chromedp.WindowSize(1100, 750),
	)

	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)

	return &Session{
		ctx:             browserCtx,
		cancelBrowser:   cancelBrowser,
		cancelAllocator: cancelAllocator,
	}, nil
}

func (s *Session) Open(url string) error {
	if err := chromedp.Do(
		s.ctx,
		chromedp.Navigate(url),
	); err != nil {
		return fmt.Errorf("buka halaman browser: %w", err)
	}

	return nil
}

func (s *Session) Close() error {
	err := chromedp.Cancel(s.ctx)

	s.cancelBrowser()
	s.cancelAllocator()

	return err
}

func (s *Session) Start() error {
	if err := chromedp.Do(s.ctx); err != nil {
		return fmt.Errorf("mulai browser: %w", err)
	}

	return nil
}

func (s *Session) OpenContext(
	ctx context.Context,
	target string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	evalCtx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	stop := context.AfterFunc(ctx, cancel)
	defer stop()

	if err := chromedp.Do(
		evalCtx,
		chromedp.Navigate(target),
	); err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return fmt.Errorf("navigasi dihentikan: %w", contextErr)
		}

		return fmt.Errorf("buka halaman: %w", err)
	}

	return nil
}
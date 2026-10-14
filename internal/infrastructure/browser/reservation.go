package browser

import "errors"

var (
	ErrBrowserBusy         = errors.New("browser sedang menjalankan persiapan")
	ErrManualActionPending = errors.New("browser masih menunggu tindakan manual")
	ErrNoManualAction      = errors.New("tidak ada tindakan manual yang menunggu")
)

func (s *Session) beginPreparation() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.preparing {
		return ErrBrowserBusy
	}

	if s.awaitingManualAction {
		return ErrManualActionPending
	}

	s.preparing = true
	return nil
}

func (s *Session) endPreparation(success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.preparing = false
	s.awaitingManualAction = success
}

func (s *Session) ReleaseManualAction() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.preparing {
		return ErrBrowserBusy
	}

	if !s.awaitingManualAction {
		return ErrNoManualAction
	}

	s.awaitingManualAction = false
	return nil
}
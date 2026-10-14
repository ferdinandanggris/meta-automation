package browser

import (
	"errors"
	"testing"
)

func TestSessionProtectsPendingManualAction(t *testing.T) {
	session := &Session{}

	if err := session.beginPreparation(); err != nil {
		t.Fatal(err)
	}

	if err := session.beginPreparation(); !errors.Is(err, ErrBrowserBusy) {
		t.Fatalf("error = %v, ingin ErrBrowserBusy", err)
	}

	if err := session.ReleaseManualAction(); !errors.Is(err, ErrBrowserBusy) {
		t.Fatalf("release saat persiapan = %v, ingin ErrBrowserBusy", err)
	}

	session.endPreparation(true)

	if err := session.beginPreparation(); !errors.Is(err, ErrManualActionPending) {
		t.Fatalf("error = %v, ingin ErrManualActionPending", err)
	}

	if err := session.ReleaseManualAction(); err != nil {
		t.Fatal(err)
	}

	if err := session.beginPreparation(); err != nil {
		t.Fatalf("persiapan baru ditolak: %v", err)
	}

	session.endPreparation(false)
}

func TestFailedPreparationReleasesSession(t *testing.T) {
	session := &Session{}

	if err := session.beginPreparation(); err != nil {
		t.Fatal(err)
	}

	session.endPreparation(false)

	if err := session.beginPreparation(); err != nil {
		t.Fatalf("sesi tidak dilepas setelah gagal: %v", err)
	}

	session.endPreparation(false)
}
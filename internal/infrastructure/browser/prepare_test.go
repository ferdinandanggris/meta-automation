package browser

import (
	"context"
	"testing"
)

func TestPrepareWABARejectsEmptyJobIDBeforeReservingBrowser(t *testing.T) {
	session := &Session{}

	if err := session.PrepareWABA(context.Background(), "   ", "123456", "WABA Percobaan"); err == nil {
		t.Fatal("job ID kosong harus ditolak")
	}

	if session.preparing || session.awaitingManualAction {
		t.Error("job ID invalid tidak boleh mereservasi browser")
	}
}

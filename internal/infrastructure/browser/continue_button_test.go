package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestDetectContinueButtonDoesNotClick(t *testing.T) {
	executable := os.Getenv("META_BROWSER_PATH")
	if executable == "" {
		t.Skip("META_BROWSER_PATH belum diisi")
	}

	page := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")

			_, _ = w.Write([]byte(`
				<!doctype html>
				<html>
				<body>
					<script>window.continueClicks = 0;</script>

					<button onclick="window.continueClicks++">
						Continue
					</button>

					<div role="dialog">
						<div role="button"
						     tabindex="0"
						     onclick="window.continueClicks++">
							<span>Lanjutkan</span>
						</div>
					</div>
				</body>
				</html>
			`))
		},
	))
	defer page.Close()

	session, err := NewSession(
		context.Background(),
		executable,
		t.TempDir(),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := session.Close(); err != nil {
			t.Errorf("tutup browser: %v", err)
		}
	}()

	if err := session.Open(page.URL); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(session.ctx, 5*time.Second)
	defer cancel()

	detection, err := session.DetectContinueButton(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if detection.Count != 1 {
		t.Fatalf("jumlah tombol = %d, ingin 1", detection.Count)
	}

	clicks, err := chromedp.Run(
		ctx,
		chromedp.Evaluate[int]("window.continueClicks"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if clicks != 0 {
		t.Errorf("jumlah klik = %d, ingin 0", clicks)
	}
}
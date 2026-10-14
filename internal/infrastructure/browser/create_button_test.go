package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"strings"
	"github.com/chromedp/chromedp"
)

func TestDetectCreateButtonFindsGridCellWithDescription(t *testing.T) {
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
					<div role="gridcell"
					     tabindex="0"
					     aria-labelledby="dynamic-title">
						<div>
							<div id="dynamic-title">
								Buat akun WhatsApp baru
							</div>
							<div>
								Create an account to connect to
								the WhatsApp Business Platform
							</div>
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

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	detection, err := session.DetectCreateButton(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if detection.Count != 1 {
		t.Fatalf("jumlah kandidat = %d, ingin 1", detection.Count)
	}

	if len(detection.Labels) != 1 {
		t.Fatalf("jumlah label = %d, ingin 1", len(detection.Labels))
	}

	if got := strings.TrimSpace(detection.Labels[0]); got != "Buat akun WhatsApp baru" {
		t.Errorf("label = %q, ingin Buat akun WhatsApp baru", got)
	}
}

func TestClickCreateButtonClicksGridCell(t *testing.T) {
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
					<script>window.clickCount = 0;</script>

					<div role="gridcell"
					     tabindex="0"
					     onclick="window.clickCount++">
						<div>Buat akun WhatsApp baru</div>
						<div>Deskripsi tambahan</div>
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

	if err := session.ClickCreateButton(ctx); err != nil {
		t.Fatalf("klik gagal: %v", err)
	}

	count, err := chromedp.Run(
		ctx,
		chromedp.Evaluate[int]("window.clickCount"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Errorf("jumlah klik = %d, ingin 1", count)
	}
}

func TestClickCreateButtonRejectsUnsafeCandidates(t *testing.T) {
	executable := os.Getenv("META_BROWSER_PATH")
	if executable == "" {
		t.Skip("META_BROWSER_PATH belum diisi")
	}

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

	tests := []struct {
		name string
		html string
	}{
		{
			name: "tidak ditemukan",
			html: `<button onclick="window.clickCount++">
				Pilihan lain
			</button>`,
		},
		{
			name: "nonaktif",
			html: `
				<div role="gridcell"
				     tabindex="0"
				     aria-disabled="true"
				     onclick="window.clickCount++">
					<div>Buat akun WhatsApp baru</div>
				</div>
			`,
		},
		{
			name: "ambigu",
			html: `
				<div role="gridcell"
				     tabindex="0"
				     onclick="window.clickCount++">
					<div>Buat akun WhatsApp baru</div>
				</div>
				<div role="gridcell"
				     tabindex="0"
				     onclick="window.clickCount++">
					<div>Buat akun WhatsApp baru</div>
				</div>
			`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set(
						"Content-Type",
						"text/html; charset=utf-8",
					)

					_, _ = w.Write([]byte(
						`<!doctype html><html><body>
						<script>window.clickCount = 0;</script>` +
							tt.html +
							`</body></html>`,
					))
				},
			))
			defer page.Close()

			if err := session.Open(page.URL); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(
				session.ctx,
				5*time.Second,
			)
			defer cancel()

			if err := session.ClickCreateButton(ctx); err == nil {
				t.Fatal("klik seharusnya ditolak")
			}

			count, err := chromedp.Run(
				ctx,
				chromedp.Evaluate[int]("window.clickCount"),
			)
			if err != nil {
				t.Fatal(err)
			}

			if count != 0 {
				t.Errorf("jumlah klik = %d, ingin 0", count)
			}
		})
	}
}
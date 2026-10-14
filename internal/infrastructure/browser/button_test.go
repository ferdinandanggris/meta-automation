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

func TestDetectAddButtonFindsLabelledButton(t *testing.T) {
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
					<span id="dynamic-label">Tambahkan</span>
					<div role="button"
					     aria-labelledby="dynamic-label"
					     tabindex="0">
						<span>Ikon</span>
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

	detection, err := session.WaitForAddButton(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if detection.Count != 1 {
		t.Fatalf("jumlah tombol = %d, ingin 1", detection.Count)
	}

	if len(detection.Labels) != 1 ||
		detection.Labels[0] != "Tambahkan" {
		t.Errorf("label tidak sesuai: %v", detection.Labels)
	}
}

func TestDetectAddButtonFiltersCandidates(t *testing.T) {
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
		name      string
		html      string
		wantCount int
	}{
		{
			name:      "tombol aktif",
			html:      `<button>Tambahkan</button>`,
			wantCount: 1,
		},
		{
			name:      "label bahasa Inggris",
			html:      `<button>Add</button>`,
			wantCount: 1,
		},
		{
			name:      "display none",
			html:      `<button style="display:none">Tambahkan</button>`,
			wantCount: 0,
		},
		{
			name:      "visibility hidden",
			html:      `<button style="visibility:hidden">Tambahkan</button>`,
			wantCount: 0,
		},
		{
			name:      "disabled",
			html:      `<button disabled>Tambahkan</button>`,
			wantCount: 0,
		},
		{
			name:      "aria disabled",
			html:      `<div role="button" aria-disabled="true">Tambahkan</div>`,
			wantCount: 0,
		},
		{
			name:      "sedang sibuk",
			html:      `<button aria-busy="true">Tambahkan</button>`,
			wantCount: 0,
		},
		{
			name:      "ambigu",
			html:      `<button>Tambahkan</button><button>Add</button>`,
			wantCount: 2,
		},
		{
			name:      "teks bukan kecocokan tepat",
			html:      `<button>Tambahkan pengguna</button>`,
			wantCount: 0,
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
						"<!doctype html><html><body>" +
							tt.html +
							"</body></html>",
					))
				},
			))
			defer page.Close()

			if err := session.Open(page.URL); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			defer cancel()

			detection, err := session.DetectAddButton(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if detection.Count != tt.wantCount {
				t.Errorf(
					"jumlah tombol = %d, ingin %d",
					detection.Count,
					tt.wantCount,
				)
			}
		})
	}
}

func TestClickAddButtonClicksSingleCandidate(t *testing.T) {
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
					<button onclick="window.clickCount++">
						Tambahkan
					</button>
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

	if err := session.ClickAddButton(ctx); err != nil {
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

func TestClickAddButtonRejectsUnsafeCandidates(t *testing.T) {
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
				Tombol lain
			</button>`,
		},
		{
			name: "ambigu",
			html: `
				<button onclick="window.clickCount++">Tambahkan</button>
				<button onclick="window.clickCount++">Add</button>
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

			if err := session.ClickAddButton(ctx); err == nil {
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
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

func TestDetectNameInputIgnoresSearchAndCategory(t *testing.T) {
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
					<label for="search">Search</label>
					<input id="search"
					       type="text"
					       placeholder="Search by phone number">

					<div role="dialog">
						<span id="name-label">
							Nama tampilan WhatsApp Business
						</span>
						<input type="text"
						       role="combobox"
						       aria-labelledby="name-label"
						       value="Family Pulsa">

						<span id="category-label">Kategori</span>
						<div role="combobox"
						     aria-labelledby="category-label">
							Belanja dan Ritel
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

	detection, err := session.DetectNameInput(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if detection.Count != 1 {
		t.Fatalf("jumlah input = %d, ingin 1", detection.Count)
	}

	if len(detection.Labels) != 1 {
		t.Fatalf("jumlah label = %d, ingin 1", len(detection.Labels))
	}

	if got := strings.TrimSpace(detection.Labels[0]); got != "Nama tampilan WhatsApp Business" {
		t.Errorf("label tidak sesuai: %q", got)
	}
}

func TestFillNameInputChangesOnlyName(t *testing.T) {
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
					<script>
						window.inputEvents = 0;
						window.changeEvents = 0;
						window.continueClicks = 0;
					</script>

					<input id="search"
					       type="text"
					       value="search awal">

					<div role="dialog">
						<span id="name-label">
							Nama tampilan WhatsApp Business
						</span>

						<input id="name"
						       type="text"
						       role="combobox"
						       aria-labelledby="name-label"
						       value="Family Pulsa"
						       oninput="window.inputEvents++"
						       onchange="window.changeEvents++">

						<button onclick="window.continueClicks++">
							Lanjutkan
						</button>
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

	ctx, cancel := context.WithTimeout(session.ctx, 10*time.Second)
	defer cancel()

	expectedName := `WABA "Percobaan"`

	if err := session.FillNameInput(ctx, expectedName); err != nil {
		t.Fatalf("pengisian nama gagal: %v", err)
	}

	type pageState struct {
		Name           string `json:"name"`
		Search         string `json:"search"`
		InputEvents    int    `json:"inputEvents"`
		ChangeEvents   int    `json:"changeEvents"`
		ContinueClicks int    `json:"continueClicks"`
	}

	state, err := chromedp.Run(
		ctx,
		chromedp.Evaluate[pageState](`({
			name: document.getElementById("name").value,
			search: document.getElementById("search").value,
			inputEvents: window.inputEvents,
			changeEvents: window.changeEvents,
			continueClicks: window.continueClicks
		})`),
	)
	if err != nil {
		t.Fatal(err)
	}

	if state.Name != expectedName {
		t.Errorf("nama = %q, ingin %q", state.Name, expectedName)
	}

	if state.Search != "search awal" {
		t.Error("input Search ikut berubah")
	}

	if state.InputEvents == 0 || state.ChangeEvents == 0 {
		t.Error("event input dan change harus terkirim")
	}

	if state.ContinueClicks != 0 {
		t.Error("Lanjutkan tidak boleh diklik")
	}
}

func TestFillNameInputRejectsUnsafeCandidates(t *testing.T) {
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
		name   string
		inputs string
	}{
		{
			name: "tidak ditemukan",
			inputs: `<input type="text" value="awal">`,
		},
		{
			name: "disabled",
			inputs: `<input type="text" role="combobox"
				aria-labelledby="name-label" value="awal" disabled>`,
		},
		{
			name: "readonly",
			inputs: `<input type="text" role="combobox"
				aria-labelledby="name-label" value="awal" readonly>`,
		},
		{
			name: "ambigu",
			inputs: `
				<input type="text" role="combobox"
					aria-labelledby="name-label" value="awal">
				<input type="text" role="combobox"
					aria-labelledby="name-label" value="awal">
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
						<script>
							window.inputEvents = 0;
							document.addEventListener("input", () => {
								window.inputEvents++;
							});
						</script>
						<div role="dialog">
							<span id="name-label">
								Nama tampilan WhatsApp Business
							</span>` +
							tt.inputs +
							`</div></body></html>`,
					))
				},
			))
			defer page.Close()

			if err := session.Open(page.URL); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(
				session.ctx,
				10*time.Second,
			)
			defer cancel()

			if err := session.FillNameInput(ctx, "WABA Diubah"); err == nil {
				t.Fatal("pengisian seharusnya ditolak")
			}

			type pageState struct {
				Values      []string `json:"values"`
				InputEvents int      `json:"inputEvents"`
			}

			state, err := chromedp.Run(
				ctx,
				chromedp.Evaluate[pageState](`({
					values: Array.from(
						document.querySelectorAll("input")
					).map(input => input.value),
					inputEvents: window.inputEvents
				})`),
			)
			if err != nil {
				t.Fatal(err)
			}

			for _, value := range state.Values {
				if value != "awal" {
					t.Errorf("nilai input berubah: %q", value)
				}
			}

			if state.InputEvents != 0 {
				t.Error("pengisian yang ditolak tidak boleh mengirim event input")
			}
		})
	}
}
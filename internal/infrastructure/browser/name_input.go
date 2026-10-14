package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"encoding/json"
	"strings"
)

type InputDetection struct {
	Count  int      `json:"count"`
	Labels []string `json:"labels"`
}

const findNameInputsJS = `(() => {
	const normalize = value =>
		(value || "").replace(/\s+/g, " ").trim().toLowerCase();

	const labelledText = element =>
		(element.getAttribute("aria-labelledby") || "")
			.split(/\s+/)
			.filter(Boolean)
			.map(id => element.ownerDocument.getElementById(id))
			.filter(Boolean)
			.map(label => label.textContent)
			.join(" ");

	const labels = [
		"nama tampilan whatsapp business",
		"whatsapp business display name"
	];

	return Array.from(
		document.querySelectorAll(
			'[role="dialog"] input[type="text"][role="combobox"]'
		)
	).filter(element => {
		const style = getComputedStyle(element);

		const visible =
			element.getClientRects().length > 0 &&
			style.display !== "none" &&
			style.visibility !== "hidden";

		const enabled =
			!element.disabled &&
			!element.readOnly &&
			element.getAttribute("aria-disabled") !== "true" &&
			element.getAttribute("aria-busy") !== "true";

		const label =
			labelledText(element) ||
			element.getAttribute("aria-label");

		return visible && enabled && labels.includes(normalize(label));
	});
})()`

func (s *Session) DetectNameInput(
	ctx context.Context,
) (InputDetection, error) {
	if err := ctx.Err(); err != nil {
		return InputDetection{}, err
	}

	script := `(() => {
		const candidates = ` + findNameInputsJS + `;

		return {
			count: candidates.length,
			labels: candidates.map(element =>
				(element.getAttribute("aria-labelledby") || "")
					.split(/\s+/)
					.filter(Boolean)
					.map(id => element.ownerDocument.getElementById(id))
					.filter(Boolean)
					.map(label => label.textContent)
					.join(" ") ||
				element.getAttribute("aria-label")
			)
		};
	})()`

	evalCtx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	stop := context.AfterFunc(ctx, cancel)
	defer stop()

	result, err := chromedp.Run(
		evalCtx,
		chromedp.Evaluate[InputDetection](script),
	)
	if err != nil {
		return InputDetection{}, fmt.Errorf("cari input nama WABA: %w", err)
	}

	return result, nil
}

func (s *Session) WaitForNameInput(
	ctx context.Context,
) (InputDetection, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return InputDetection{}, fmt.Errorf("menunggu input nama WABA: %w", err)
		}

		result, err := s.DetectNameInput(ctx)
		if err != nil {
			return InputDetection{}, err
		}

		if result.Count > 0 {
			return result, nil
		}

		select {
		case <-ctx.Done():
			return InputDetection{}, fmt.Errorf(
				"menunggu input nama WABA: %w",
				ctx.Err(),
			)
		case <-ticker.C:
		}
	}
}

func (s *Session) FillNameInput(
	ctx context.Context,
	name string,
) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("nama WABA wajib diisi")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	encodedName, err := json.Marshal(name)
	if err != nil {
		return fmt.Errorf("encode nama WABA: %w", err)
	}

	script := `(() => {
		const expected = ` + string(encodedName) + `;
		const candidates = ` + findNameInputsJS + `;

		if (candidates.length === 0) return "missing";
		if (candidates.length > 1) return "ambiguous";

		const input = candidates[0];
		const inputPrototype =
			input.ownerDocument.defaultView.HTMLInputElement.prototype;

		const setter = Object.getOwnPropertyDescriptor(
			inputPrototype,
			"value"
		).set;

		input.focus();
		setter.call(input, expected);

		input.dispatchEvent(new Event("input", { bubbles: true }));
		input.dispatchEvent(new Event("change", { bubbles: true }));
		input.blur();

		return "filled";
	})()`

	evalCtx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	stop := context.AfterFunc(ctx, cancel)
	defer stop()

	result, err := chromedp.Run(
		evalCtx,
		chromedp.Evaluate[string](script),
	)
	if err != nil {
		return fmt.Errorf("isi nama WABA: %w", err)
	}

	switch result {
	case "missing":
		return fmt.Errorf("input nama WABA tidak ditemukan")
	case "ambiguous":
		return fmt.Errorf("lebih dari satu input nama WABA ditemukan")
	case "filled":
		// Lanjutkan ke pemeriksaan nilai.
	default:
		return fmt.Errorf("hasil pengisian tidak dikenal: %q", result)
	}

	verifyScript := `(() => {
		const expected = ` + string(encodedName) + `;
		const candidates = ` + findNameInputsJS + `;

		return candidates.length === 1 &&
			candidates[0].value === expected;
	})()`

	_, err = chromedp.Run(
		evalCtx,
		chromedp.Poll[bool](
			verifyScript,
			chromedp.WithPollingInterval(500*time.Millisecond),
			chromedp.WithPollingTimeout(5*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("verifikasi nilai nama WABA: %w", err)
	}

	return nil
}
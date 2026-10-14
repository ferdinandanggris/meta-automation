package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

const findCreateButtonsJS = `(() => {
	const normalize = value =>
		(value || "").replace(/\s+/g, " ").trim().toLowerCase();

	const titles = [
		"buat akun whatsapp business baru",
		"buat akun whatsapp bisnis baru",
		"buat akun whatsapp baru",
		"create a new whatsapp business account",
		"create a new whatsapp account",
		"create a new whatsapp business",
		"create a new whatsapp"
	];

	const matches = value => titles.includes(normalize(value));

	const interactiveSelector =
		'button, [role="button"], [role="menuitem"], ' +
		'[role="gridcell"][tabindex="0"]';

	const candidates = new Set();

	for (const title of document.querySelectorAll("div, span, b, button")) {
		if (!matches(title.textContent)) continue;

		// Pilih elemen judul paling dalam, bukan pembungkus judul.
		if (Array.from(title.children).some(child =>
			matches(child.textContent)
		)) continue;

		const element = title.closest(interactiveSelector);
		if (!element) continue;

		const titleStyle = getComputedStyle(title);
		const style = getComputedStyle(element);

		const visible =
			title.getClientRects().length > 0 &&
			titleStyle.display !== "none" &&
			titleStyle.visibility !== "hidden" &&
			element.getClientRects().length > 0 &&
			style.display !== "none" &&
			style.visibility !== "hidden";

		const enabled =
			!element.disabled &&
			element.getAttribute("aria-disabled") !== "true" &&
			element.getAttribute("aria-busy") !== "true";

		if (visible && enabled) candidates.add(element);
	}

	return Array.from(candidates);
})()`

func (s *Session) WaitForCreateButton(
	ctx context.Context,
) (ButtonDetection, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return ButtonDetection{}, fmt.Errorf(
				"menunggu tombol Tambahkan: %w",
				err,
			)
		}

		result, err := s.DetectCreateButton(ctx)
		if err != nil {
			return ButtonDetection{}, err
		}

		if result.Count > 0 {
			return result, nil
		}

		select {
		case <-ctx.Done():
			return ButtonDetection{}, fmt.Errorf(
				"menunggu pilihan Buat akun WABA: %w",
				ctx.Err(),
			)
		case <-ticker.C:
		}
	}
}

func (s *Session) DetectCreateButton(
	ctx context.Context,
) (ButtonDetection, error) {
	script := `(() => {
		const candidates = ` + findCreateButtonsJS + `;

		return {
			count: candidates.length,
			labels: candidates.map(element => {
				const labelled = (
					element.getAttribute("aria-labelledby") || ""
				)
					.split(/\s+/)
					.filter(Boolean)
					.map(id => element.ownerDocument.getElementById(id))
					.filter(Boolean)
					.map(label => label.textContent)
					.join(" ");

				return labelled ||
					element.innerText ||
					element.value ||
					element.getAttribute("aria-label") ||
					element.getAttribute("title");
			})
		};
	})()`

	if err := ctx.Err(); err != nil {
		return ButtonDetection{}, err
	}

	evalCtx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	stop := context.AfterFunc(ctx, cancel)
	defer stop()

	result, err := chromedp.Run(
		evalCtx,
		chromedp.Evaluate[ButtonDetection](script),
	)
	if err != nil {
		return ButtonDetection{}, fmt.Errorf(
			"cari pilihan Buat akun WABA: %w",
			err,
		)
	}

	return result, nil
}

func (s *Session) ClickCreateButton(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("klik pilihan Buat akun WABA dibatalkan: %w", err)
	}

	script := `(() => {
		const candidates = ` + findCreateButtonsJS + `;

		if (candidates.length === 0) return "missing";
		if (candidates.length > 1) return "ambiguous";

		candidates[0].click();
		return "clicked";
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
		return fmt.Errorf("klik pilihan Buat akun WABA: %w", err)
	}

	switch result {
	case "clicked":
		return nil
	case "missing":
		return fmt.Errorf("pilihan Buat akun WABA tidak ditemukan")
	case "ambiguous":
		return fmt.Errorf("lebih dari satu pilihan Buat akun WABA ditemukan")
	default:
		return fmt.Errorf("hasil klik tidak dikenal: %q", result)
	}
}
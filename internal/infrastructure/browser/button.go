package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

type ButtonDetection struct {
	Count  int      `json:"count"`
	Labels []string `json:"labels"`
}

const findAddButtonsJS = `(() => {
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

	const matches = value =>
		["tambahkan", "add"].includes(normalize(value));

	const candidates = Array.from(
		document.querySelectorAll(
			'button, [role="button"], input[type="button"]'
		)
	).filter(element => {
		const style = getComputedStyle(element);

		const visible =
			element.getClientRects().length > 0 &&
			style.display !== "none" &&
			style.visibility !== "hidden";

		const enabled =
			!element.disabled &&
			element.getAttribute("aria-disabled") !== "true" &&
			element.getAttribute("aria-busy") !== "true";

		return visible && enabled && (
			matches(labelledText(element)) ||
			matches(element.innerText || element.value) ||
			matches(element.getAttribute("aria-label")) ||
			matches(element.getAttribute("title"))
		);
	});

	return candidates;
})()`

func (s *Session) WaitForAddButton(
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

		result, err := s.DetectAddButton(ctx)
		if err != nil {
			return ButtonDetection{}, err
		}

		if result.Count > 0 {
			return result, nil
		}

		select {
		case <-ctx.Done():
			return ButtonDetection{}, fmt.Errorf(
				"menunggu tombol Tambahkan: %w",
				ctx.Err(),
			)
		case <-ticker.C:
		}
	}
}

func (s *Session) DetectAddButton(
	ctx context.Context,
) (ButtonDetection, error) {
	script := `(() => {
		const candidates = ` + findAddButtonsJS + `;

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
			"cari tombol Tambahkan: %w",
			err,
		)
	}

	return result, nil
}

func (s *Session) ClickAddButton(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("klik Tambahkan dibatalkan: %w", err)
	}

	script := `(() => {
		const candidates = ` + findAddButtonsJS + `;

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
		return fmt.Errorf("klik Tambahkan: %w", err)
	}

	switch result {
	case "clicked":
		return nil
	case "missing":
		return fmt.Errorf("tombol Tambahkan tidak ditemukan")
	case "ambiguous":
		return fmt.Errorf("lebih dari satu tombol Tambahkan ditemukan")
	default:
		return fmt.Errorf("hasil klik tidak dikenal: %q", result)
	}
}
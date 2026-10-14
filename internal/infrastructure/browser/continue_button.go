package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

const findContinueButtonsJS = `(() => {
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
		["lanjutkan", "continue"].includes(normalize(value));

	return Array.from(
		document.querySelectorAll(
			'[role="dialog"] button, ' +
			'[role="dialog"] [role="button"], ' +
			'[role="dialog"] input[type="button"]'
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
			matches(element.getAttribute("aria-label"))
		);
	});
})()`

func (s *Session) DetectContinueButton(
	ctx context.Context,
) (ButtonDetection, error) {
	if err := ctx.Err(); err != nil {
		return ButtonDetection{}, err
	}

	script := `(() => {
		const candidates = ` + findContinueButtonsJS + `;

		return {
			count: candidates.length,
			labels: candidates.map(element =>
				element.innerText ||
				element.value ||
				element.getAttribute("aria-label") ||
				element.textContent
			)
		};
	})()`

	evalCtx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	stop := context.AfterFunc(ctx, cancel)
	defer stop()

	result, err := chromedp.Run(
		evalCtx,
		chromedp.Evaluate[ButtonDetection](script),
	)
	if err != nil {
		return ButtonDetection{}, fmt.Errorf("cari tombol Lanjutkan: %w", err)
	}

	return result, nil
}

func (s *Session) WaitForContinueButton(
	ctx context.Context,
) (ButtonDetection, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return ButtonDetection{}, fmt.Errorf("menunggu Lanjutkan: %w", err)
		}

		result, err := s.DetectContinueButton(ctx)
		if err != nil {
			return ButtonDetection{}, err
		}

		if result.Count > 0 {
			return result, nil
		}

		select {
		case <-ctx.Done():
			return ButtonDetection{}, fmt.Errorf(
				"menunggu Lanjutkan: %w",
				ctx.Err(),
			)
		case <-ticker.C:
		}
	}
}
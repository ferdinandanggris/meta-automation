package browser_test

import (
	"meta-automation/internal/application/onboarding"
	"meta-automation/internal/infrastructure/browser"
)

var _ onboarding.Browser = (*browser.Session)(nil)
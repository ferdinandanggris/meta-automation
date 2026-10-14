package browser

import (
	"errors"
	"net/url"
	"strings"
	"context"
)

func buildWABASettingsURL(businessID string) (string, error) {
	businessID = strings.TrimSpace(businessID)

	if businessID == "" {
		return "", errors.New("business ID wajib diisi")
	}

	for _, character := range businessID {
		if character < '0' || character > '9' {
			return "", errors.New("business ID harus berisi angka")
		}
	}

	target := url.URL{
		Scheme: "https",
		Host:   "business.facebook.com",
		Path:   "/latest/settings/whatsapp_business_presence",
	}

	query := url.Values{}
	query.Set("business_id", businessID)
	query.Set("selected_asset_type", "whatsapp-business-account")
	target.RawQuery = query.Encode()

	return target.String(), nil
}

func (s *Session) OpenWABASettings(businessID string) error {
	target, err := buildWABASettingsURL(businessID)
	if err != nil {
		return err
	}

	return s.Open(target)
}

func (s *Session) OpenWABASettingsContext(
	ctx context.Context,
	businessID string,
) error {
	target, err := buildWABASettingsURL(businessID)
	if err != nil {
		return err
	}

	return s.OpenContext(ctx, target)
}

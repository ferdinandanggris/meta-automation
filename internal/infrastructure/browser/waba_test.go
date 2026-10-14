package browser

import (
	"net/url"
	"testing"
)

func TestBuildWABASettingsURL(t *testing.T) {
	target, err := buildWABASettingsURL(" 123456 ")
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}

	if parsed.Scheme != "https" ||
		parsed.Host != "business.facebook.com" {
		t.Error("tujuan browser tidak sesuai")
	}

	if parsed.Path != "/latest/settings/whatsapp_business_presence" {
		t.Errorf("path tidak sesuai: %q", parsed.Path)
	}

	if parsed.Query().Get("business_id") != "123456" {
		t.Error("business ID tidak sesuai")
	}

	if parsed.Query().Get("selected_asset_type") != "whatsapp-business-account" {
		t.Error("jenis aset tidak sesuai")
	}
}

func TestBuildWABASettingsURLRejectsInvalidID(t *testing.T) {
	for _, input := range []string{"", "   ", "abc", "123&other=456"} {
		t.Run(input, func(t *testing.T) {
			if _, err := buildWABASettingsURL(input); err == nil {
				t.Error("business ID invalid harus ditolak")
			}
		})
	}
}

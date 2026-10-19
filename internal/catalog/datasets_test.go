package catalog

import "testing"

func TestKeyForTitleSupportsCurrentAndLegacyNames(t *testing.T) {
	cases := map[string]string{
		"Продажи игровых платформ · 2015–2024 (млн устройств)": "platform-sales",
		"Демо · продажи игровых платформ":                      "platform-sales",
		"Население стран · 2019–2023 (млн человек, OWID)":      "population-countries",
		"unknown dataset": "",
	}
	for title, want := range cases {
		if got := KeyForTitle(title); got != want {
			t.Errorf("KeyForTitle(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestLocalizedTitleUsesLocaleAndEnglishFallback(t *testing.T) {
	title := "Продажи игровых платформ · 2015–2024 (млн устройств)"
	if got := LocalizedTitle(title, "de-DE"); got != "Verkäufe von Spielekonsolen · 2015–2024 (Mio. Geräte)" {
		t.Fatalf("German title = %q", got)
	}
	if got := LocalizedTitle(title, "fr-FR"); got != "Game platform sales · 2015–2024 (million devices)" {
		t.Fatalf("French fallback title = %q", got)
	}
	if got := LocalizedTitle(title, "xx"); got != title {
		t.Fatalf("unknown locale should use canonical title, got %q", got)
	}
	if got := LocalizedTitle("My personal CSV", "en"); got != "My personal CSV" {
		t.Fatalf("personal title changed to %q", got)
	}
}

func TestUnknownTitleDoesNotReceiveAKey(t *testing.T) {
	if got := KeyForTitle("Мой CSV"); got != "" {
		t.Fatalf("custom title received catalog key %q", got)
	}
}

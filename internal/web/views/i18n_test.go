package views

import "testing"

func TestNormalizeLanguageAcceptsTelegramRegionsAndRejectsGarbage(t *testing.T) {
	tests := map[string]string{
		"ru":         "ru",
		"en-US":      "en",
		"uk_UA":      "uk",
		"be-BY":      "be",
		"pl":         "pl",
		"de":         "de",
		"english":    "ru",
		"en<script>": "ru",
		"":           "ru",
	}
	for raw, want := range tests {
		if got := NormalizeLanguage(raw); got != want {
			t.Errorf("NormalizeLanguage(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestLanguageOptionsExcludeCurrentLocale(t *testing.T) {
	data := PageData{Language: "pl"}
	options := data.LanguageOptions()
	if len(options) != len(languageOptions)-1 {
		t.Fatalf("got %d language options, want %d", len(options), len(languageOptions)-1)
	}
	for _, option := range options {
		if option.Code == "pl" {
			t.Fatal("current locale is still listed as a switch target")
		}
	}
	if got := data.CurrentLanguageLabel(); got != "Polski" {
		t.Fatalf("current locale label = %q", got)
	}
}

func TestTranslationsFallbackForPartialLocale(t *testing.T) {
	if got := tr("uk", "studio.title"); got != "Створи історію" {
		t.Fatalf("uk translation = %q", got)
	}
	if got := tr("uk", "welcome.retry"); got == "welcome.retry" || got == "" {
		t.Fatalf("missing key did not fall back to a readable translation: %q", got)
	}
	if got := tr("de", "status.title"); got != "Dein Video ist unterwegs." {
		t.Fatalf("German status translation = %q", got)
	}
	if got := tr("de", "cart.pay"); got != "Jetzt bezahlen" {
		t.Fatalf("German checkout translation = %q", got)
	}
}

func TestSystemDatasetTitlesAreLocalizedWithoutTouchingPersonalTitles(t *testing.T) {
	canonical := "Продажи игровых платформ · 2015–2024 (млн устройств)"
	if got := datasetDisplayTitle("en", canonical, true); got != "Game platform sales · 2015–2024 (million devices)" {
		t.Fatalf("system title = %q", got)
	}
	if got := datasetDisplayTitle("pl", canonical, true); got == canonical || got == "" {
		t.Fatalf("polish system title was not localized: %q", got)
	}
	if got := datasetDisplayTitle("en", canonical, false); got != canonical {
		t.Fatalf("personal title changed: %q", got)
	}
	if got := datasetDisplayTitle("en", "Unknown user title", true); got != "Unknown user title" {
		t.Fatalf("unknown system title changed: %q", got)
	}
}

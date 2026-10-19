// Package catalog contains stable presentation metadata for the built-in
// datasets. The database deliberately stores only the user-facing title: a
// separate slug column would require a migration and would make the seed
// catalog harder to evolve. Titles and their legacy aliases are enough to
// resolve the stable key while keeping user-uploaded datasets untouched.
package catalog

import "strings"

type datasetDefinition struct {
	key          string
	title        string
	legacyTitles []string
}

var datasetDefinitions = []datasetDefinition{
	{
		key:          "platform-sales",
		title:        "Продажи игровых платформ · 2015–2024 (млн устройств)",
		legacyTitles: []string{"Демо · продажи игровых платформ"},
	},
	{
		key:          "streaming-genres",
		title:        "Музыкальные жанры в стриминге · 2015–2024 (%)",
		legacyTitles: []string{"Демо · музыкальные жанры в стриминге"},
	},
	{
		key:          "renewables-growth",
		title:        "Рост возобновляемой энергии · 2015–2024 (%)",
		legacyTitles: []string{"Демо · рост возобновляемой энергии"},
	},
	{
		key:          "population-countries",
		title:        "Население стран · 2015–2024 (млн человек)",
		legacyTitles: []string{"Население стран · 2019–2023 (млн человек, OWID)"},
	},
	{
		key:          "co2-per-capita",
		title:        "Выбросы CO₂ на человека · 2015–2024 (тонн)",
		legacyTitles: []string{"Выбросы CO₂ на человека · 2019–2024 (тонн, OWID)"},
	},
	{
		key:          "renewable-share",
		title:        "Возобновляемая энергия · 2015–2024 (% финального потребления)",
		legacyTitles: []string{"Возобновляемая энергия · 2019–2023 (% финального потребления, OWID)"},
	},
}

// KeyForTitle returns the stable key for a seeded system dataset. It accepts
// legacy seed names so an older database can be localized before the next
// seed run renames it.
func KeyForTitle(title string) string {
	title = strings.TrimSpace(title)
	for _, definition := range datasetDefinitions {
		if title == definition.title {
			return definition.key
		}
		for _, legacy := range definition.legacyTitles {
			if title == legacy {
				return definition.key
			}
		}
	}
	return ""
}

// LocalizedTitle returns the presentation title for a built-in dataset. The
// database keeps one canonical row, so a language switch never duplicates or
// mutates the dataset. Personal CSV titles and unknown system titles pass
// through unchanged.
func LocalizedTitle(title, language string) string {
	key := KeyForTitle(title)
	if key == "" {
		return title
	}

	locale := localeBase(language)
	if values, ok := datasetTranslations[locale]; ok {
		if translated, ok := values[key]; ok {
			return translated
		}
	}
	return title
}

func localeBase(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	if index := strings.IndexAny(language, "-_"); index >= 0 {
		language = language[:index]
	}
	switch language {
	case "ru", "en", "uk", "be", "pl":
		return language
	case "de", "es", "fr", "it", "pt", "tr", "ar", "id", "vi", "ja", "ko", "zh", "nl", "cs", "sv":
		// The catalog has a complete English fallback for the additional UI
		// locales. Their chrome can grow independently of the seeded catalog.
		return "en"
	default:
		return "ru"
	}
}

var datasetTranslations = map[string]map[string]string{
	"ru": {
		"platform-sales":       "Продажи игровых платформ · 2015–2024 (млн устройств)",
		"streaming-genres":     "Музыкальные жанры в стриминге · 2015–2024 (%)",
		"renewables-growth":    "Рост возобновляемой энергии · 2015–2024 (%)",
		"population-countries": "Население стран · 2015–2024 (млн человек)",
		"co2-per-capita":       "Выбросы CO₂ на человека · 2015–2024 (тонн)",
		"renewable-share":      "Возобновляемая энергия · 2015–2024 (% финального потребления)",
	},
	"en": {
		"platform-sales":       "Game platform sales · 2015–2024 (million devices)",
		"streaming-genres":     "Music genres in streaming · 2015–2024 (%)",
		"renewables-growth":    "Renewable energy growth · 2015–2024 (%)",
		"population-countries": "Population by country · 2015–2024 (million people)",
		"co2-per-capita":       "CO₂ emissions per person · 2015–2024 (tonnes)",
		"renewable-share":      "Renewable energy · 2015–2024 (% of final consumption)",
	},
	"uk": {
		"platform-sales":       "Продажі ігрових платформ · 2015–2024 (млн пристроїв)",
		"streaming-genres":     "Музичні жанри у стримінгу · 2015–2024 (%)",
		"renewables-growth":    "Зростання відновлюваної енергії · 2015–2024 (%)",
		"population-countries": "Населення країн · 2015–2024 (млн людей)",
		"co2-per-capita":       "Викиди CO₂ на людину · 2015–2024 (тонн)",
		"renewable-share":      "Відновлювана енергія · 2015–2024 (% кінцевого споживання)",
	},
	"be": {
		"platform-sales":       "Продаж гульнявых платформаў · 2015–2024 (млн прылад)",
		"streaming-genres":     "Музычныя жанры ў стрымінгу · 2015–2024 (%)",
		"renewables-growth":    "Рост аднаўляльнай энергіі · 2015–2024 (%)",
		"population-countries": "Насельніцтва краін · 2015–2024 (млн чалавек)",
		"co2-per-capita":       "Выкіды CO₂ на чалавека · 2015–2024 (тон)",
		"renewable-share":      "Аднаўляльная энергія · 2015–2024 (% фінальнага спажывання)",
	},
	"pl": {
		"platform-sales":       "Sprzedaż platform do gier · 2015–2024 (mln urządzeń)",
		"streaming-genres":     "Gatunki muzyczne w streamingu · 2015–2024 (%)",
		"renewables-growth":    "Rozwój energii odnawialnej · 2015–2024 (%)",
		"population-countries": "Ludność krajów · 2015–2024 (mln osób)",
		"co2-per-capita":       "Emisje CO₂ na osobę · 2015–2024 (tony)",
		"renewable-share":      "Energia odnawialna · 2015–2024 (% zużycia końcowego)",
	},
}

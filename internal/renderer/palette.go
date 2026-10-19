package renderer

import "contgen/internal/storage/dbgen"

// Palette описывает набор цветов для конкретной темы оформления
type Palette struct {
	BgColor      string   `json:"background"` // Цвет фона видео
	CardColor    string   `json:"card"`       // Цвет подложки/карточки
	TextColor    string   `json:"text"`       // Основной цвет текста
	SubTextColor string   `json:"subtext"`    // Второстепенный цвет (даты, подписи)
	BarColors    []string `json:"bars"`       // Набор цветов для полосок
	AccentColor  string   `json:"accent"`
	Style        string   `json:"style"`
}

// GetPalette возвращает палитру по названию темы
func GetPalette(theme dbgen.RenderTheme) Palette {
	switch theme {
	case dbgen.RenderThemeDARKGLASS:
		return Palette{
			BgColor: "#0C1016", CardColor: "#202A36", TextColor: "#F6F4EF", SubTextColor: "#9BA7B5",
			AccentColor: "#F3BD78", Style: "glass",
			BarColors: []string{"#E8B778", "#8BCFBD", "#91B5EB", "#B6A0DD", "#DF9EA9", "#B2C583", "#81C3D4", "#D6AB90", "#9CA6D8", "#D8C27D"},
		}

	case dbgen.RenderThemeCYBERHUD:
		return Palette{
			BgColor: "#080E18", CardColor: "#203147", TextColor: "#F1F5FC", SubTextColor: "#94A5BD",
			AccentColor: "#41E7C5", Style: "hud",
			BarColors: []string{"#41E7C5", "#89B8FF", "#B9A0FF", "#F9A17F", "#F5D27C", "#82D4AC", "#8AD7E6", "#F3A5C6", "#A3AEEA", "#D2DDA0"},
		}

	case dbgen.RenderThemeCORPORATELIGHT:
		return Palette{
			BgColor: "#F6F8FC", CardColor: "#DCE4F0", TextColor: "#172238", SubTextColor: "#65758C",
			AccentColor: "#285AF0", Style: "light",
			BarColors: []string{"#3265E7", "#24A29A", "#9D72D4", "#DC7953", "#66A354", "#CC688C", "#DBAA43", "#429BBE", "#7184BA", "#AC8A65"},
		}

	case dbgen.RenderThemeVINTAGEPRINT:
		return Palette{
			BgColor: "#F2EBDE", CardColor: "#D5CBB8", TextColor: "#303229", SubTextColor: "#79796A",
			AccentColor: "#B85C3A", Style: "print",
			BarColors: []string{"#B85C3A", "#568575", "#C29A53", "#778BA5", "#A16E82", "#939652", "#4C7B89", "#AA7F61", "#8784A1", "#ACAC7A"},
		}

	default:
		// Фоновый дефолт
		return GetPalette(dbgen.RenderThemeDARKGLASS)
	}
}

package views

import (
	"contgen/internal/storage/dbgen"
	"fmt"
)

type Theme struct {
	Code        dbgen.RenderTheme
	Name        string
	Description string
	Swatches    []string
}

type Bar struct {
	Name  string
	Value string
	Width string
	Color string
}

type StudioData struct {
	PageData
	Music        []MusicOption
	MusicID      int64
	Price        string
	CanAdd       bool
	User         dbgen.User
	Datasets     []dbgen.Dataset
	Selected     *dbgen.Dataset
	Theme        dbgen.RenderTheme
	Themes       []Theme
	PreviewStyle string
	Bars         []Bar
	PointCount   int
	EntityCount  int
	FirstYear    int
	LastYear     int
	DataMessage  string
}

func (d StudioData) SelectedID() string {
	if d.Selected == nil {
		return ""
	}
	return fmt.Sprint(d.Selected.ID)
}

func (d StudioData) Period() string {
	if d.PointCount == 0 {
		return "—"
	}
	if d.FirstYear == d.LastYear {
		return fmt.Sprint(d.FirstYear)
	}
	return fmt.Sprintf("%d–%d", d.FirstYear, d.LastYear)
}

func previewMessage(d StudioData) string {
	if d.DataMessage != "" {
		return d.DataMessage
	}
	if d.Selected == nil {
		return tr(d.Language, "studio.preview.empty")
	}
	return tr(d.Language, "studio.preview.no.points")
}

func lastYear(d StudioData) string {
	if d.PointCount == 0 {
		return "—"
	}
	return fmt.Sprint(d.LastYear)
}

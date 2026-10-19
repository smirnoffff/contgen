package web

import (
	"contgen/internal/renderer"
	"contgen/internal/storage/dbgen"
	"contgen/internal/web/views"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func (s *server) studioData(r *http.Request, user dbgen.User) (views.StudioData, int, error) {
	d := views.StudioData{User: user, Theme: dbgen.RenderThemeCYBERHUD}
	d.Price = "50"
	page, pageErr := s.pageData(r, user)
	if pageErr != nil {
		return d, http.StatusServiceUnavailable, pageErr
	}
	d.PageData = page
	if s.db != nil {
		tracks, err := s.db.GetAvailableMusicTracks(r.Context(), &user.ID)
		if err != nil {
			return d, http.StatusServiceUnavailable, err
		}
		for _, track := range tracks {
			d.Music = append(d.Music, views.MusicOption{ID: track.ID, Title: track.Title, URL: fmt.Sprintf("/music/%d/audio", track.ID), Duration: durationLabel(track.DurationSec)})
		}
		if raw := r.URL.Query().Get("music_id"); raw != "" && raw != "0" {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || id <= 0 {
				return d, http.StatusBadRequest, errors.New("Некорректный трек")
			}
			if _, err := s.db.GetMusicForUser(r.Context(), dbgen.GetMusicForUserParams{ID: id, ViewerID: user.ID}); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return d, http.StatusNotFound, errors.New("Трек не найден")
				}
				return d, http.StatusServiceUnavailable, err
			}
			d.MusicID = id
		}
	}
	if theme := r.URL.Query().Get("theme"); theme != "" {
		d.Theme = dbgen.RenderTheme(theme)
		if !d.Theme.Valid() {
			return d, http.StatusBadRequest, errors.New("Неизвестная тема")
		}
	}
	var err error
	d.Datasets, err = s.store.GetAvailableDatasets(r.Context(), &user.ID)
	if err != nil {
		return d, http.StatusServiceUnavailable, err
	}
	if value := r.URL.Query().Get("dataset_id"); value != "" {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			return d, http.StatusBadRequest, errors.New("Некорректный датасет")
		}
		dataset, err := s.store.GetDatasetForUser(r.Context(), dbgen.GetDatasetForUserParams{ID: id, ViewerID: user.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			return d, http.StatusNotFound, errors.New("Датасет не найден")
		}
		if err != nil {
			return d, http.StatusServiceUnavailable, err
		}
		d.Selected = &dataset
	} else if len(d.Datasets) > 0 {
		d.Selected = &d.Datasets[0]
	}
	names := map[dbgen.RenderTheme][2]string{
		dbgen.RenderThemeDARKGLASS:      {"Dark Glass", "Тёплый свет, тёмное стекло"},
		dbgen.RenderThemeCORPORATELIGHT: {"Corporate Light", "Чистая редакционная подача"},
		dbgen.RenderThemeCYBERHUD:       {"Cyber HUD", "Неон и цифровая точность"},
		dbgen.RenderThemeVINTAGEPRINT:   {"Vintage Print", "Бумага, цвет и характер"},
	}
	for _, code := range dbgen.AllRenderThemeValues() {
		name := names[code]
		if name[0] == "" {
			name = [2]string{string(code), "Новая тема"}
		}
		palette := renderer.GetPalette(code)
		d.Themes = append(d.Themes, views.Theme{Code: code, Name: name[0], Description: name[1], Swatches: palette.BarColors[:4]})
	}
	palette := renderer.GetPalette(d.Theme)
	d.PreviewStyle = fmt.Sprintf("--preview-bg:%s;--preview-text:%s;--preview-sub:%s;--preview-card:%s;--preview-accent:%s", palette.BgColor, palette.TextColor, palette.SubTextColor, palette.CardColor, palette.AccentColor)
	if d.Selected == nil {
		return d, http.StatusOK, nil
	}
	points, err := s.store.GetDataPointsByDatasetID(r.Context(), d.Selected.ID)
	if err != nil {
		return d, http.StatusServiceUnavailable, err
	}
	d.PointCount = len(points)
	if len(points) == 0 {
		return d, http.StatusOK, nil
	}
	d.FirstYear, d.LastYear = points[0].RecordedAt, points[0].RecordedAt
	entities := make(map[string]bool)
	for _, p := range points {
		d.FirstYear = min(d.FirstYear, p.RecordedAt)
		d.LastYear = max(d.LastYear, p.RecordedAt)
		entities[p.EntityName] = true
		if strings.TrimSpace(p.EntityName) == "" {
			d.DataMessage = "В данных есть строки без названия сущности"
		}
		if p.Value < 0 || math.IsNaN(p.Value) || math.IsInf(p.Value, 0) {
			d.DataMessage = "Для графика нужны конечные значения от нуля и выше"
		}
	}
	d.EntityCount = len(entities)
	if d.DataMessage != "" {
		return d, http.StatusOK, nil
	}
	d.CanAdd = true
	latest := make(map[string]float64)
	for name := range entities {
		latest[name] = 0
	}
	for _, p := range points {
		if p.RecordedAt == d.LastYear {
			latest[p.EntityName] = p.Value
		}
	}
	rows := make([]dbgen.DataPoint, 0, len(latest))
	for name, value := range latest {
		rows = append(rows, dbgen.DataPoint{EntityName: name, Value: value})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Value == rows[j].Value {
			return rows[i].EntityName < rows[j].EntityName
		}
		return rows[i].Value > rows[j].Value
	})
	maxValue := rows[0].Value
	for i, row := range rows[:min(6, len(rows))] {
		width := 0.0
		if maxValue > 0 {
			width = row.Value / maxValue * 100
		}
		d.Bars = append(d.Bars, views.Bar{Name: row.EntityName, Value: strconv.FormatFloat(row.Value, 'f', -1, 64), Width: fmt.Sprintf("%.2f%%", width), Color: palette.BarColors[i%len(palette.BarColors)]})
	}
	return d, http.StatusOK, nil
}

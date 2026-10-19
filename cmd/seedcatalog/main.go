// Command seedcatalog adds a small, reusable system catalog for local demos.
// It is idempotent by title and does not change the database schema.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"contgen/internal/catalog"
	"contgen/internal/config"
	"contgen/internal/model"
	"contgen/internal/storage"
	"contgen/internal/storage/dbgen"
)

type catalogDataset struct {
	// key is stable across title polishing and is also used by the UI's
	// presentation mapping. It is intentionally not persisted as a column.
	key string
	title string
	// legacyTitles lets a seed upgrade polish the names of datasets created by
	// an earlier catalog version without creating a second copy in the UI.
	legacyTitles []string
	// startYear is the first year represented by the values below. The
	// original demo fixtures start in 2020; imported OWID snapshots start in
	// 2019 so the seed keeps the source time axis intact.
	startYear int
	series    map[string][]float64
}

var datasets = []catalogDataset{
	{
		key:         "platform-sales",
		title:        "Продажи игровых платформ · 2015–2024 (млн устройств)",
		legacyTitles: []string{"Демо · продажи игровых платформ"},
		startYear:    2015,
		series: map[string][]float64{
			"PlayStation": {26, 29, 31, 33, 34, 34, 41, 48, 56, 63},
			"Nintendo":    {13, 15, 17, 19, 21, 22, 29, 37, 45, 54},
			"Xbox":        {24, 25, 25, 26, 26, 26, 31, 35, 39, 43},
			"Steam Deck":  {0, 0, 0, 0, 0, 0, 4, 9, 15, 23},
		},
	},
	{
		key:         "streaming-genres",
		title:        "Музыкальные жанры в стриминге · 2015–2024 (%)",
		legacyTitles: []string{"Демо · музыкальные жанры в стриминге"},
		startYear:    2015,
		series: map[string][]float64{
			"Pop":        {49, 51, 53, 55, 57, 58, 64, 71, 79, 88},
			"Hip-hop":    {29, 33, 36, 39, 41, 42, 49, 57, 66, 75},
			"Rock":       {56, 55, 54, 52, 51, 51, 50, 49, 48, 47},
			"Electronic": {16, 18, 20, 22, 23, 24, 31, 39, 48, 58},
			"Jazz":       {9, 10, 11, 12, 12, 12, 14, 16, 19, 22},
		},
	},
	{
		key:         "renewables-growth",
		title:        "Рост возобновляемой энергии · 2015–2024 (%)",
		legacyTitles: []string{"Демо · рост возобновляемой энергии"},
		startYear:    2015,
		series: map[string][]float64{
			"Solar": {5, 7, 9, 10, 11, 12, 19, 28, 40, 55},
			"Wind":  {11, 13, 15, 17, 18, 18, 24, 31, 39, 48},
			"Hydro": {29, 30, 31, 31, 32, 32, 34, 35, 37, 39},
			"Bio":   {6, 7, 8, 8, 9, 9, 11, 14, 18, 23},
		},
	},
	{
		key:         "population-countries",
		title:        "Население стран · 2015–2024 (млн человек)",
		legacyTitles: []string{"Население стран · 2019–2023 (млн человек, OWID)"},
		startYear:    2015,
		// Values are a compact, rounded teaching snapshot in millions. The
		// long time axis makes changes legible in a 10+ second animation.
		series: map[string][]float64{
			"США":      {320.7, 323.1, 325.1, 327.2, 337.790069, 339.436156, 340.161438, 341.534041, 343.477330, 345.4},
			"Китай":    {1377.0, 1382.7, 1386.4, 1390.1, 1423.520357, 1426.106093, 1426.437269, 1425.179562, 1422.584930, 1419.3},
			"Индия":    {1311.1, 1326.8, 1340.0, 1353.0, 1389.030307, 1402.617694, 1414.203896, 1425.423209, 1438.069597, 1450.9},
			"Германия": {81.7, 82.3, 82.7, 82.9, 83.559184, 83.628711, 83.697080, 84.086227, 84.548233, 84.7},
			"Бразилия": {204.5, 206.8, 208.5, 209.5, 207.455459, 208.660845, 209.550291, 210.306411, 211.140731, 212.6},
		},
	},
	{
		key:         "co2-per-capita",
		title:        "Выбросы CO₂ на человека · 2015–2024 (тонн)",
		legacyTitles: []string{"Выбросы CO₂ на человека · 2019–2024 (тонн, OWID)"},
		startYear:    2015,
		series: map[string][]float64{
			"США":      {16.4, 15.7, 15.4, 15.2, 15.500492, 13.8168955, 14.758024, 14.802047, 14.319450, 14.197287},
			"Китай":    {6.7, 6.9, 7.0, 7.2, 7.5260706, 7.640750, 7.910899, 8.217776, 8.556261, 8.658390},
			"Индия":    {1.6, 1.7, 1.8, 1.8, 1.8798543, 1.7272933, 1.8920738, 1.9861692, 2.1297693, 2.2009783},
			"Германия": {9.4, 9.1, 8.8, 8.5, 8.480803, 7.738692, 8.100614, 7.942359, 7.022808, 6.768824},
			"Бразилия": {2.5, 2.4, 2.4, 2.3, 2.2979572, 2.1470218, 2.3696554, 2.2826612, 2.2922719, 2.278372},
		},
	},
	{
		key:         "renewable-share",
		title:        "Возобновляемая энергия · 2015–2024 (% финального потребления)",
		legacyTitles: []string{"Возобновляемая энергия · 2019–2023 (% финального потребления, OWID)"},
		startYear:    2015,
		series: map[string][]float64{
			"США":      {10.9, 11.2, 10.7, 10.4, 10.26, 11.03, 10.97, 11.36, 12.25, 13.4},
			"Китай":    {12.0, 12.5, 12.8, 13.1, 13.40, 13.66, 13.69, 14.21, 14.21, 15.1},
			"Индия":    {29.2, 31.0, 32.1, 33.0, 33.74, 36.21, 35.05, 34.21, 32.84, 35.4},
			"Германия": {15.3, 16.0, 16.7, 16.9, 17.07, 18.52, 17.93, 19.56, 21.23, 22.7},
			"Бразилия": {45.5, 46.8, 47.1, 47.0, 47.49, 49.98, 46.51, 48.39, 50.18, 52.0},
		},
	},
}

type catalogTrack struct {
	title        string
	legacyTitles []string
	fileName     string
	duration     int
	sourceURL    string
	// Frequencies are generated by ffmpeg's lavfi sine source and mixed into
	// a short MP3 when the remote CC0 file cannot be downloaded.
	frequencies []string
}

var tracks = []catalogTrack{
	{title: "Neon Pulse", legacyTitles: []string{"Neon Pulse · generated"}, fileName: "neon-pulse.wav", duration: 30, frequencies: []string{"220"}},
	{title: "Aurora Bloom", legacyTitles: []string{"Aurora Bloom · generated"}, fileName: "aurora-bloom.mp3", duration: 30, frequencies: []string{"261.63", "329.63", "392.00"}},
	{title: "Night Drive", legacyTitles: []string{"Night Drive · generated"}, fileName: "night-drive.mp3", duration: 30, frequencies: []string{"110.00", "146.83", "220.00"}},
	// These are public-domain/CC0 loops from OpenGameArt. The seed command
	// downloads them when online and keeps a generated fallback for offline
	// course/demo environments.
	{title: "City Loop · CC0", fileName: "city-loop.mp3", duration: 30, sourceURL: "https://opengameart.org/sites/default/files/city-loop_0.mp3", frequencies: []string{"196.00", "246.94", "293.66"}},
	{title: "JRPG Battle · CC0", fileName: "jrpg-battle.mp3", duration: 30, sourceURL: "https://opengameart.org/sites/default/files/jrpg_battle_loop.mp3", frequencies: []string{"164.81", "207.65", "246.94", "329.63"}},
	{title: "Music Loop · CC0", fileName: "music-loop.mp3", duration: 30, sourceURL: "https://opengameart.org/sites/default/files/loop_0.mp3", frequencies: []string{"130.81", "164.81", "196.00"}},
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cfg := config.Load()
	db, err := storage.New(ctx, cfg.DSN())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	existing, err := db.GetAvailableDatasets(ctx, nil)
	if err != nil {
		panic(err)
	}
	datasetsByTitle := make(map[string]dbgen.Dataset, len(existing))
	for _, d := range existing {
		datasetsByTitle[d.Title] = d
	}
	for _, item := range datasets {
		if item.key == "" || catalog.KeyForTitle(item.title) != item.key {
			panic(fmt.Sprintf("invalid catalog key for %q", item.title))
		}
		startYear := item.startYear
		if startYear == 0 {
			startYear = 2020
		}
		points := make([]model.Point, 0, len(item.series)*10)
		for entity, values := range item.series {
			for i, value := range values {
				points = append(points, model.Point{EntityName: entity, RecordedAt: startYear + i, Value: value})
			}
		}

		dataset, found := datasetsByTitle[item.title]
		if !found {
			for _, legacyTitle := range item.legacyTitles {
				legacy, ok := datasetsByTitle[legacyTitle]
				if !ok {
					continue
				}
				if _, err := db.RenameSystemDataset(ctx, dbgen.RenameSystemDatasetParams{ID: legacy.ID, Title: item.title}); err != nil {
					panic(err)
				}
				dataset = legacy
				dataset.Title = item.title
				delete(datasetsByTitle, legacyTitle)
				datasetsByTitle[item.title] = dataset
				found = true
				break
			}
		}
		if !found {
			created, err := db.CreateDatasetWithPoints(ctx, nil, item.title, points)
			if err != nil {
				panic(err)
			}
			datasetsByTitle[item.title] = *created
			fmt.Printf("dataset added: %s (%d points)\n", item.title, len(points))
			continue
		}

		rows, err := db.GetDataPointsByDatasetID(ctx, dataset.ID)
		if err != nil {
			panic(err)
		}
		known := make(map[string]struct{}, len(rows))
		for _, row := range rows {
			known[pointKey(row.EntityName, row.RecordedAt)] = struct{}{}
		}
		missing := make([]dbgen.CopyDataPointsParams, 0, len(points))
		for _, point := range points {
			key := pointKey(point.EntityName, point.RecordedAt)
			if _, ok := known[key]; ok {
				continue
			}
			missing = append(missing, dbgen.CopyDataPointsParams{
				DatasetID: dataset.ID, EntityName: point.EntityName,
				Value: point.Value, RecordedAt: point.RecordedAt,
			})
		}
		if len(missing) > 0 {
			if _, err := db.CopyDataPoints(ctx, missing); err != nil {
				panic(err)
			}
			fmt.Printf("dataset updated: %s (+%d points)\n", item.title, len(missing))
		} else {
			fmt.Printf("dataset exists: %s\n", item.title)
		}
	}

	availableTracks, err := db.GetAvailableMusicTracks(ctx, nil)
	if err != nil {
		panic(err)
	}
	tracksByPath := make(map[string]dbgen.MusicTrack, len(availableTracks))
	for _, track := range availableTracks {
		tracksByPath[track.FilePath] = track
	}
	root := cfg.DataDir
	if root == "" {
		root = "data"
	}
	for _, track := range tracks {
		relative := filepath.ToSlash(filepath.Join("music", track.fileName))
		absolute := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			panic(err)
		}
		if _, err := os.Stat(absolute); errors.Is(err, os.ErrNotExist) {
			if track.sourceURL != "" {
				if err := downloadTrack(ctx, absolute, track.sourceURL); err != nil {
					fmt.Printf("music download failed (%s): %v; generating fallback\n", track.title, err)
					if err := generateTrack(ctx, absolute, track); err != nil {
						panic(err)
					}
				} else {
					fmt.Printf("music downloaded: %s\n", track.title)
				}
			} else if err := generateTrack(ctx, absolute, track); err != nil {
				panic(err)
			}
		}
		if existing, ok := tracksByPath[relative]; ok {
			if existing.Title != track.title {
				if _, err := db.RenameSystemMusicTrack(ctx, dbgen.RenameSystemMusicTrackParams{ID: existing.ID, Title: track.title}); err != nil {
					panic(err)
				}
				fmt.Printf("music renamed: %s\n", track.title)
			} else {
				fmt.Printf("music exists: %s\n", track.title)
			}
			continue
		}
		if _, err := db.CreateMusicTrack(ctx, dbgen.CreateMusicTrackParams{UserID: nil, Title: track.title, FilePath: relative, DurationSec: ptr(track.duration)}); err != nil {
			panic(err)
		}
		fmt.Printf("music added: %s\n", track.title)
	}
}

func pointKey(entity string, year int) string {
	return fmt.Sprintf("%s\x00%d", entity, year)
}

func downloadTrack(ctx context.Context, path, sourceURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "CONTGEN catalog seed/1.0")
	client := &http.Client{Timeout: 45 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("HTTP %s", response.Status)
	}
	const maxAudioSize = 32 << 20
	tmp, err := os.CreateTemp(filepath.Dir(path), ".catalog-audio-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	written, copyErr := io.Copy(tmp, io.LimitReader(response.Body, maxAudioSize+1))
	if closeErr := tmp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return copyErr
	}
	if written == 0 {
		return errors.New("empty audio response")
	}
	if written > maxAudioSize {
		return fmt.Errorf("audio response exceeds %d MB", maxAudioSize>>20)
	}
	return os.Rename(tmpName, path)
}

func generateTrack(ctx context.Context, path string, track catalogTrack) error {
	args := []string{"-y"}
	for _, frequency := range track.frequencies {
		args = append(args, "-f", "lavfi", "-i", "sine=frequency="+frequency+":duration=30")
	}
	if len(track.frequencies) == 1 {
		args = append(args, "-filter:a", "volume=0.18", "-ar", "44100")
	} else {
		inputs := make([]string, len(track.frequencies))
		for i := range inputs {
			inputs[i] = fmt.Sprintf("[%d:a]", i)
		}
		args = append(args, "-filter_complex", strings.Join(inputs, "")+fmt.Sprintf("amix=inputs=%d:duration=longest,volume=0.12", len(inputs)), "-ar", "44100", "-c:a", "libmp3lame", "-b:a", "96k")
	}
	args = append(args, path)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg %s: %w", track.title, err)
	}
	return nil
}

func ptr(v int) *int { return &v }

// renderdemo exercises the real Go -> Python -> ffmpeg pipeline without a DB.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"contgen/internal/model"
	"contgen/internal/renderer"
	"contgen/internal/storage/dbgen"
)

func main() {
	output := flag.String("output", "preview.mp4", "output video path")
	theme := flag.String("theme", string(dbgen.RenderThemeCYBERHUD), "DARK_GLASS, CYBER_HUD, CORPORATE_LIGHT or VINTAGE_PRINT")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	series := []struct {
		name   string
		values []float64
	}{
		{"Python", []float64{18, 22, 31, 39, 46}},
		{"JavaScript", []float64{40, 39, 37, 34, 32}},
		{"Java", []float64{35, 34, 31, 28, 26}},
		{"C++", []float64{25, 26, 28, 30, 33}},
		{"Go", []float64{8, 12, 17, 25, 35}},
		{"Rust", []float64{4, 6, 11, 19, 29}},
		{"Котлин", []float64{9, 13, 16, 18, 21}},
		{"Очень длинное название языка", []float64{13.5, 14.2, 15.6, 17.1, 18.4}},
	}
	var points []model.Point
	for _, s := range series {
		for i, value := range s.values {
			points = append(points, model.Point{EntityName: s.name, RecordedAt: 2020 + i, Value: value})
		}
		}
		if err := renderer.RenderRacingBar(ctx, "Демо языков", points, dbgen.RenderTheme(*theme), *output); err != nil {
		log.Fatal(err)
	}
	log.Printf("Видео готово: %s", *output)
}

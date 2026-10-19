package main

import (
	"strings"
	"testing"

	"contgen/internal/catalog"
)

func TestCatalogDatasetsHaveAReadableTimeline(t *testing.T) {
	t.Helper()
	for _, dataset := range datasets {
		if dataset.key == "" || catalog.KeyForTitle(dataset.title) != dataset.key {
			t.Errorf("%q has an unstable or unknown catalog key %q", dataset.title, dataset.key)
		}
		if dataset.startYear == 0 {
			t.Errorf("%q has no start year", dataset.title)
		}
		if len(dataset.series) < 4 {
			t.Errorf("%q has only %d entities; the catalog should show a real comparison", dataset.title, len(dataset.series))
		}
		var periods int
		for entity, values := range dataset.series {
			if periods == 0 {
				periods = len(values)
			}
			if len(values) != periods {
				t.Errorf("%q has inconsistent period count for %q: got %d, want %d", dataset.title, entity, len(values), periods)
			}
		}
		if periods < 10 {
			t.Errorf("%q has only %d periods; starter videos need at least ten", dataset.title, periods)
		}
	}
}

func TestCatalogTrackTitlesAreUserFacing(t *testing.T) {
	for _, track := range tracks {
		if strings.Contains(strings.ToLower(track.title), "generated") {
			t.Errorf("technical generation marker leaked into track title %q", track.title)
		}
	}
}

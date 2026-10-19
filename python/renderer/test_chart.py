import math
import unittest
from unittest.mock import patch

import bar_chart_race as bcr

from chart import (MAX_VISIBLE_BARS, animation_timing, build_timeline,
                   format_value, prepare_data, render, visible_bar_count)


class ChartTest(unittest.TestCase):
    def test_sparse_data_and_duplicates(self):
        frame = prepare_data([
            {"name": "Бета", "year": 2022, "value": 20},
            {"name": "Альфа", "year": 2020, "value": 10},
            {"name": "Альфа", "year": 2020, "value": 12.5},
        ])
        self.assertEqual(frame.index.tolist(), [2020, 2022])
        self.assertEqual(frame.columns.tolist(), ["Альфа", "Бета"])
        self.assertEqual(frame.loc[2020, "Альфа"], 12.5)
        self.assertEqual(frame.loc[2020, "Бета"], 0)
        self.assertEqual(frame.loc[2022, "Альфа"], 0)

    def test_one_period_is_static(self):
        frame = prepare_data([{"name": "A", "year": 2020, "value": 4}])
        values, ranks = bcr.prepare_wide_data(frame, steps_per_period=3,
                                             interpolate_period=True)
        self.assertEqual(values["A"].tolist(), [4] * 4)
        self.assertEqual(ranks["A"].tolist(), [1] * 4)
        self.assertEqual(values.index.tolist(), [2020] * 4)

    def test_entity_names_do_not_collide_with_period_column(self):
        frame = prepare_data([
            {"name": name, "year": 2020, "value": 1}
            for name in ("index", "__contgen_year__")
        ])
        values, _ = bcr.prepare_wide_data(frame, steps_per_period=1,
                                         interpolate_period=True)
        self.assertEqual(values.columns.tolist(), ["__contgen_year__", "index"])

    def test_library_interpolates_positions_at_overtake(self):
        frame = prepare_data([
            {"name": "A", "year": 2020, "value": 10},
            {"name": "B", "year": 2020, "value": 20},
            {"name": "A", "year": 2021, "value": 30},
            {"name": "B", "year": 2021, "value": 10},
        ])
        values, ranks = bcr.prepare_wide_data(frame, steps_per_period=4,
                                             interpolate_period=True)
        self.assertEqual(ranks["A"].tolist(), [1, 1.25, 1.5, 1.75, 2])
        self.assertEqual(values["A"].tolist(), [10, 15, 20, 25, 30])

    def test_invalid_points(self):
        for value in (math.nan, math.inf, -1):
            with self.subTest(value=value), self.assertRaises(ValueError):
                prepare_data([{"name": "A", "year": 2020, "value": value}])
        with self.assertRaises(ValueError):
            prepare_data([])
        with self.assertRaises(ValueError):
            prepare_data([{"name": "", "year": 2020, "value": 1}])

    def test_visible_bar_count_keeps_all_entities_in_the_ranking(self):
        self.assertEqual(visible_bar_count({}, 3), 3)
        self.assertEqual(visible_bar_count({}, 100), MAX_VISIBLE_BARS)
        self.assertEqual(visible_bar_count({"max_bars": 6}, 100), 6)
        with self.assertRaises(ValueError):
            visible_bar_count({"max_bars": 0}, 10)
        with self.assertRaises(ValueError):
            visible_bar_count({"max_bars": 21}, 10)
        with self.assertRaises(ValueError):
            visible_bar_count({"max_bars": True}, 10)

    def test_timing_keeps_requested_duration(self):
        for periods in (2, 5, 8, 51, 400):
            steps, period_length = animation_timing(periods, 60, 5)
            source_frames = (periods - 1) * steps + 1
            source_fps = 1000 * steps / period_length
            self.assertAlmostEqual(source_frames / source_fps, 5)

    def test_fractional_labels(self):
        self.assertEqual(format_value(12.5), "12.5")
        self.assertEqual(format_value(0.005), "0.005")
        self.assertEqual(format_value(12000), "12 000")
        self.assertEqual(format_value(1200000000), "1.2B")

    def test_video_timeline_preserves_endpoints_and_holds(self):
        frame = prepare_data([
            {"name": "A", "year": 2020 + year, "value": 10 + year * 5}
            for year in range(8)
        ])
        values, ranks, years = build_timeline(frame, 60, 5, 1)
        self.assertEqual(values.shape, (300, 1))
        self.assertEqual(ranks.shape, (300, 1))
        self.assertEqual(values[0, 0], 10)
        self.assertEqual(values[-1, 0], 45)
        self.assertTrue(all(year == 2020 for year in years[:18]))
        self.assertTrue(all(year == 2027 for year in years[-30:]))
        self.assertTrue(all(a <= b for a, b in zip(years, years[1:])))

    def test_missing_ffmpeg(self):
        with patch("chart.shutil.which", return_value=None):
            with self.assertRaisesRegex(RuntimeError, "ffmpeg not found"):
                render({})


if __name__ == "__main__":
    unittest.main()

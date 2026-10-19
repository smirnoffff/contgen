"""Data preparation and video export for the bar race scene."""

import math
import shutil

import matplotlib

matplotlib.use("Agg")

import bar_chart_race as bcr
import matplotlib.pyplot as plt
import pandas as pd
import numpy as np
from matplotlib.animation import FuncAnimation, FFMpegWriter
from scene import RaceScene

# Visual limit only: all entities remain in the ranking, while at most this
# many bars can occupy the canvas at one time.
MAX_VISIBLE_BARS = 10


def visible_bar_count(request, entity_count):
    """Resolve the explicit top-N display setting for a render request."""
    requested = request.get("max_bars", MAX_VISIBLE_BARS)
    if isinstance(requested, bool) or not isinstance(requested, int):
        raise ValueError("max_bars must be a positive integer")
    if requested <= 0 or requested > 20:
        raise ValueError("max_bars must be between 1 and 20")
    return min(requested, entity_count)


def prepare_data(points):
    if not points:
        raise ValueError("no data points")
    records = {}
    for point in points:
        name, year, value = point["name"], point["year"], point["value"]
        if not isinstance(name, str) or not name.strip():
            raise ValueError("entity names must not be empty")
        if not isinstance(year, int):
            raise ValueError("years must be integers")
        if not isinstance(value, (int, float)) or not math.isfinite(value) or value < 0:
            raise ValueError("values must be finite and nonnegative")
        # Match the previous renderer: the last duplicate point wins.
        records[(year, name)] = float(value)
    frame = pd.Series(records).unstack().sort_index().fillna(0)
    # Stable column order also gives deterministic ordering when values tie.
    frame = frame.reindex(sorted(frame.columns), axis=1)
    # reset_index() inside the library needs a name distinct from any entity.
    frame.index.name = "__contgen_year__"
    while frame.index.name in frame.columns:
        frame.index.name += "_"
    if len(frame) == 1:
        frame = pd.concat([frame, frame])
    return frame


def animation_timing(periods, fps, seconds):
    # The library emits (periods - 1) * steps + 1 frames. Adjust its input
    # rate so duration stays exact even when the periods don't divide evenly.
    frames = round(fps * seconds)
    steps = max(1, round((frames - 1) / (periods - 1)))
    source_fps = ((periods - 1) * steps + 1) / seconds
    return steps, 1000 * steps / source_fps


def format_value(value):
    for divisor, suffix in ((1e12, "T"), (1e9, "B"), (1e6, "M")):
        if value >= divisor:
            return f"{value / divisor:.3g}{suffix}"
    if value >= 1000 or value == int(value):
        return f"{value:,.0f}".replace(",", " ")
    return f"{value:.3g}"


def build_timeline(frame, fps, seconds, n_bars):
    count = round(fps * seconds)
    steps, _ = animation_timing(len(frame), fps, seconds)
    values, ranks = bcr.prepare_wide_data(
        frame, n_bars=n_bars, steps_per_period=steps, interpolate_period=True,
    )
    # Brief holds let the viewer read both the opening and the final ranking.
    opening, closing = round(fps * min(0.3, seconds * 0.06)), round(fps * min(0.5, seconds * 0.1))
    progress = np.linspace(0, len(frame) - 1, count - opening - closing)
    segment = np.minimum(progress.astype(int), len(frame) - 2)
    fraction = progress - segment
    eased = fraction * fraction * (3 - 2 * fraction)
    positions = (segment + eased) * steps
    positions = np.concatenate((np.zeros(opening), positions,
                                np.full(closing, len(values) - 1)))
    source = np.arange(len(values))
    value_frames = np.column_stack([np.interp(positions, source, values[name])
                                    for name in values.columns])
    rank_frames = np.column_stack([np.interp(positions, source, ranks[name])
                                   for name in ranks.columns])
    years = np.interp(positions, source, values.index.to_numpy(dtype=float))
    return value_frames, rank_frames, years


def render(request):
    if shutil.which("ffmpeg") is None:
        raise RuntimeError("ffmpeg not found in PATH")
    width, height = request["width"], request["height"]
    fps, seconds = request["fps"], request["seconds"]
    if any(not isinstance(n, int) or n <= 0 for n in (width, height, fps)):
        raise ValueError("width, height and fps must be positive integers")
    if width % 2 or height % 2:
        raise ValueError("H.264 dimensions must be even")
    if not math.isfinite(seconds) or seconds <= 0 or round(fps * seconds) < 2:
        raise ValueError("duration must contain at least two frames")
    if not request["palette"]["bars"]:
        raise ValueError("bar palette must not be empty")
    frame = prepare_data(request["points"])
    n_bars = visible_bar_count(request, len(frame.columns))
    values, ranks, years = build_timeline(frame, fps, seconds, n_bars)
    scene = RaceScene(request, frame, n_bars, format_value)
    try:
        def update(index):
            scene.update(values[index], ranks[index], years[index])
            return ()

        animation = FuncAnimation(scene.fig, update, frames=len(years),
                                  interval=1000 / fps, repeat=False, cache_frame_data=False)
        writer = FFMpegWriter(fps=fps, codec="libx264", bitrate=-1,
                              extra_args=["-pix_fmt", "yuv420p", "-crf", "19",
                                          "-preset", "fast", "-movflags", "+faststart"])
        animation.save(request["output_path"], writer=writer, dpi=100)
    finally:
        plt.close(scene.fig)

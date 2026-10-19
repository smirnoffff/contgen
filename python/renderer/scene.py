"""A persistent motion-graphics scene, drawn in 1080 x 1920 design coordinates."""

from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np
from matplotlib.colors import to_rgb
from matplotlib.font_manager import FontProperties
from matplotlib.patches import FancyBboxPatch
from matplotlib.ticker import MaxNLocator


FONTS = Path(__file__).with_name("fonts")


class RaceScene:
    def __init__(self, request, frame, n_bars, formatter):
        self.scale = request["width"] / 1080
        self.palette = request["palette"]
        self.style = self.palette.get("style", "glass")
        self.accent = self.palette.get("accent", self.palette["bars"][0])
        self.names = list(frame.columns)
        self.n_bars = n_bars
        self.format_value = formatter
        self.first_year, self.last_year = frame.index[0], frame.index[-1]
        self.fig = plt.figure(figsize=(request["width"] / 100, request["height"] / 100),
                              dpi=100, facecolor=self.palette["background"])
        self.ax = self.fig.add_axes((0, 0, 1, 1), xlim=(0, 1080), ylim=(0, 1920))
        self.ax.set_axis_off()
        self.renderer = self.fig.canvas.get_renderer()
        self.draw_background()
        self.draw_header(request["title"])
        self.plot_x, self.plot_width = 142, 866
        self.chart_bottom = 438
        available = self.chart_top - self.chart_bottom
        self.stride = min(145, available / n_bars)
        self.top_row = self.chart_top - self.stride / 2 - max(0, available - n_bars * self.stride) / 2
        self.bar_height = 26 if n_bars < 9 else 22
        maximum = max(float(frame.max().max()), 1)
        ticks = MaxNLocator(nbins=5).tick_values(0, maximum)
        self.maximum = ticks[-1]
        self.draw_grid(ticks)
        self.draw_footer(frame.index.unique().tolist())
        # Create matplotlib artists only when an entity enters the visible
        # top-N. Datasets may contain hundreds of entities; hidden rows do not
        # need images, clip paths or shadows while the ranking still tracks
        # every column.
        self.rows = {}

    def font(self, px, weight="Medium", display=False):
        if display and self.style == "print":
            return FontProperties(family="DejaVu Serif", weight="bold", size=px * .72 * self.scale)
        return FontProperties(fname=FONTS / f"Manrope-{weight}.ttf", size=px * .72 * self.scale)

    def text(self, x, y, text, px=28, weight="Medium", color=None, **kwargs):
        display = kwargs.pop("display", False)
        return self.ax.text(x, y, text, fontproperties=self.font(px, weight, display),
                            color=color or self.palette["text"], zorder=50, **kwargs)

    def measured_width(self, text, font):
        return self.renderer.get_text_width_height_descent(text, font, False)[0] / self.scale

    def fit(self, text, px, width, weight="Medium", display=False):
        font = self.font(px, weight, display)
        if self.measured_width(text, font) <= width:
            return text
        while text and self.measured_width(text + "…", font) > width:
            text = text[:-1]
        return text.rstrip() + "…"

    def fit_with_size(self, text, px, width, weight="Medium", min_px=24,
                      display=False):
        """Shrink a label before truncating it with an ellipsis."""
        min_px = max(12, min(int(px), int(min_px)))
        for size in range(int(px), min_px - 1, -2):
            if self.measured_width(text, self.font(size, weight, display)) <= width:
                return text, size
        return self.fit(text, min_px, width, weight, display), min_px

    def box(self, x, y, width, height, color, radius=10, **kwargs):
        patch = FancyBboxPatch((x, y), width, height,
                              boxstyle=f"round,pad=0,rounding_size={radius}",
                              facecolor=color, edgecolor="none", **kwargs)
        self.ax.add_patch(patch)
        return patch

    def draw_background(self):
        yy, xx = np.mgrid[0:1:480j, 0:1:270j]
        background = np.array(to_rgb(self.palette["background"]))
        tint = np.array(to_rgb(self.accent))
        glow = np.exp(-((xx - .98) ** 2 / .3 + (yy - .9) ** 2 / .055))
        strength = .09 if self.style in ("hud", "glass") else .035
        image = background + (tint - background) * glow[..., None] * strength
        if self.style == "print":
            image += np.random.default_rng(7).normal(0, .002, (480, 270, 1))
        self.ax.imshow(np.clip(image, 0, 1), extent=(0, 1080, 0, 1920),
                       origin="lower", aspect="auto", zorder=0, interpolation="bilinear")
        if self.style == "hud":
            for x in range(72, 1010, 64):
                self.ax.plot([x, x], [400, 1820], color=self.accent, alpha=.026,
                             linewidth=.4, zorder=1)
        self.ax.plot([72, 1008], [378, 378], color=self.palette["subtext"],
                     alpha=.24, linewidth=.7, zorder=2)

    def wrap_title(self, text, px):
        font = self.font(px, "Extrabold", display=True)
        lines = []

        def split_long_word(word):
            """Break URLs, IDs and other unspaced tokens at glyph boundaries."""
            chunks, current = [], ""
            for char in word:
                candidate = current + char
                if current and self.measured_width(candidate, font) > 920:
                    chunks.append(current)
                    current = char
                else:
                    current = candidate
            if current:
                chunks.append(current)
            return chunks or [""]

        for paragraph in text.splitlines() or [""]:
            current = ""
            for word in paragraph.split():
                if self.measured_width(word, font) > 920:
                    if current:
                        lines.append(current)
                        current = ""
                    lines.extend(split_long_word(word))
                    continue
                candidate = f"{current} {word}".strip()
                if current and self.measured_width(candidate, font) > 920:
                    lines.append(current)
                    current = word
                else:
                    current = candidate
            if current:
                lines.append(current)
        lines = lines or [""]
        if len(lines) > 3:
            lines = lines[:3]
            lines[-1] = lines[-1].rstrip("…") + "…"
        return lines

    def draw_header(self, title):
        self.ax.plot([72, 112], [1818, 1818], color=self.accent, linewidth=3, zorder=3)
        self.text(132, 1818, "РЕЙТИНГ В ДВИЖЕНИИ", px=20, weight="Semibold",
                  color=self.accent, va="center")
        self.text(1008, 1818, f"ТОП {self.n_bars:02}", px=20,
                  color=self.palette["subtext"], ha="right", va="center")
        size = 82
        lines = self.wrap_title(title, size)
        while size > 44 and any(self.measured_width(line, self.font(size, "Extrabold", True)) > 920
                               for line in lines):
            size -= 2
            lines = self.wrap_title(title, size)
        for i, line in enumerate(lines):
            # fit handles a single unbroken word longer than the frame.
            if self.measured_width(line, self.font(size, "Extrabold", True)) > 920:
                line = self.fit(line, size, 920, "Extrabold", display=True)
            self.text(72, 1736 - i * size * 1.18, line, px=size,
                      weight="Extrabold", display=True, va="top")
        bottom = 1736 - len(lines) * size * 1.18
        self.text(74, bottom - 28, f"{self.first_year} — {self.last_year}", px=24,
                  color=self.palette["subtext"], va="center")
        self.chart_top = min(1430, bottom - 108)

    def draw_grid(self, ticks):
        for value in ticks:
            x = self.plot_x + value / self.maximum * self.plot_width
            self.ax.plot([x, x], [self.chart_bottom, self.chart_top - 46],
                         color=self.palette["subtext"], alpha=.12,
                         linewidth=.65, zorder=1)
            self.text(x, self.chart_top + 8, self.format_value(value), px=19,
                      color=self.palette["subtext"], ha="center", va="center")
        for index in range(self.n_bars):
            y = self.top_row - index * self.stride
            self.text(72, y + 26, f"{index + 1:02}", px=21,
                      color=self.accent if index == 0 else self.palette["subtext"], va="center")
            self.ax.plot([self.plot_x, 1008], [y - 21, y - 21],
                         color=self.palette["card"], alpha=.35, linewidth=1.3, zorder=2)
        self.box(60, self.top_row - 48, 966, 112, self.accent, radius=13,
                 alpha=.045 if self.style in ("hud", "glass") else .06, zorder=2)

    def draw_footer(self, years):
        self.text(74, 334, "ПЕРИОД", px=18, color=self.palette["subtext"])
        self.year_text = self.text(66, 190, str(self.first_year), px=148,
                                   weight="Extrabold", display=True)
        self.text(1008, 334, "ЛИДЕР", px=18, color=self.palette["subtext"], ha="right")
        self.leader_text = self.text(1008, 245, "", px=38, weight="Semibold", ha="right")
        self.leader_value = self.text(1008, 197, "", px=25,
                                      color=self.palette["subtext"], ha="right")
        self.box(72, 95, 936, 5, self.palette["card"], radius=2.5, zorder=3)
        self.progress = self.box(72, 95, .01, 5, self.accent, radius=2.5, zorder=4)
        selected = np.unique(np.linspace(0, len(years) - 1, min(6, len(years))).round().astype(int))
        for i in selected:
            fraction = 0 if self.last_year == self.first_year else (years[i] - self.first_year) / (self.last_year - self.first_year)
            x = 72 + fraction * 936
            self.ax.plot(x, 97.5, marker="o", markersize=3 * self.scale,
                         color=self.palette["subtext"], zorder=5)
            self.text(x, 47, str(years[i]), px=18, color=self.palette["subtext"],
                      ha="left" if i == 0 else "right" if i == len(years) - 1 else "center")
        self.cursor, = self.ax.plot([72], [97.5], marker="o", markersize=7 * self.scale,
                                    color=self.accent, markeredgecolor=self.palette["background"],
                                    markeredgewidth=1.5, zorder=6)

    def make_row(self, name, index):
        color = self.palette["bars"][index % len(self.palette["bars"])]
        rgb = np.array(to_rgb(color))
        ramp = np.linspace(.65 if self.style in ("hud", "glass") else .86, 1, 128)
        gradient = rgb[None, None, :] * ramp[None, :, None]
        if self.style in ("light", "print"):
            gradient = rgb[None, None, :] + (1 - rgb)[None, None, :] * (1 - ramp)[None, :, None]
        shadows = [self.box(0, 0, 1, 1, color, radius=20, alpha=0, zorder=7)
                   for _ in range(2)]
        clip = FancyBboxPatch((self.plot_x, 0), 1, self.bar_height,
                              boxstyle=f"round,pad=0,rounding_size={self.bar_height / 2}",
                              transform=self.ax.transData)
        image = self.ax.imshow(gradient, extent=(0, 1, 0, 1), origin="lower",
                               aspect="auto", interpolation="bilinear", zorder=10)
        image.set_clip_path(clip)
        label_text, label_px = self.fit_with_size(name, 38, 645, "Semibold", min_px=24)
        label = self.text(self.plot_x, 0, label_text,
                          px=label_px, weight="Semibold", va="center",
                          bbox={"facecolor": self.palette["background"], "edgecolor": "none",
                                "boxstyle": "round,pad=0.08"})
        number = self.text(1008, 0, "", px=38, weight="Semibold", ha="right", va="center")
        cap, = self.ax.plot([0], [0], color=color, marker="o", markersize=3.5 * self.scale, zorder=20)
        return {"color": color, "image": image, "clip": clip, "shadows": shadows,
                "label": label, "number": number, "cap": cap}

    def update(self, values, ranks, year):
        leader = int(np.argmax(values))
        # The larger value takes the foreground at a crossing. Its competitor's
        # text fades only while the labels would occupy the same space.
        order = np.argsort(values, kind="stable")
        priority = np.empty(len(values), dtype=int)
        priority[order] = np.arange(len(values))
        for i, name in enumerate(self.names):
            ordinal = self.n_bars + 1 - ranks[i]
            visibility = min(1, max(0, self.n_bars + .65 - ordinal) / .65)
            if ordinal < .3:
                visibility = 0
            y = self.top_row - (ordinal - 1) * self.stride
            length = self.plot_width * values[i] / self.maximum
            visible = visibility > 0 and values[i] > 0
            row = self.rows.get(i)
            # Keep the leader row available even when every value is zero;
            # otherwise the footer would have no colour source on an empty
            # first frame.
            if row is None and (visible or i == leader):
                row = self.make_row(name, i)
                self.rows[i] = row
            if row is None:
                continue
            row["image"].set_visible(visible)
            row["cap"].set_visible(visible)
            row["image"].set_alpha(visibility)
            row["image"].set_zorder(10 + int(priority[i]))
            if visible:
                length = max(length, .1)
                bottom = y - 21 - self.bar_height / 2
                row["image"].set_extent((self.plot_x, self.plot_x + length, bottom, bottom + self.bar_height))
                row["clip"].set_bounds(self.plot_x, bottom, length, self.bar_height)
                row["clip"].set_boxstyle(f"round,pad=0,rounding_size={min(length / 2, self.bar_height / 2)}")
                row["cap"].set_data([self.plot_x + length - min(8, length / 2)], [y - 21])
                row["cap"].set_alpha(visibility * .65)
            for j, shadow in enumerate(row["shadows"]):
                pad = (4, 9)[j]
                shadow.set_bounds(self.plot_x - pad, y - 21 - self.bar_height / 2 - pad,
                                  length + pad * 2, self.bar_height + pad * 2)
                shadow.set_alpha((.085, .025)[j] * visibility if i == leader and self.style in ("hud", "glass") else 0)
            stronger = ranks[priority > priority[i]]
            distance = np.min(np.abs(stronger - ranks[i])) if len(stronger) else 1
            fade = min(1, max(0, (distance - .62) / .32))
            label_alpha = visibility * fade * fade * (3 - 2 * fade)
            row["label"].set_position((self.plot_x, y + 26))
            row["number"].set_position((1008, y + 26))
            row["label"].set_alpha(label_alpha)
            row["label"].get_bbox_patch().set_alpha(label_alpha * .96)
            row["number"].set_alpha(label_alpha)
            row["number"].set_text(self.format_value(values[i]))
        self.year_text.set_text(f"{year:.0f}")
        leader_text, leader_px = self.fit_with_size(self.names[leader], 38, 470,
                                                    "Semibold", min_px=24)
        self.leader_text.set_text(leader_text)
        self.leader_text.set_fontproperties(self.font(leader_px, "Semibold"))
        self.leader_text.set_color(self.rows[leader]["color"])
        self.leader_value.set_text(self.format_value(values[leader]))
        progress = 0 if self.last_year == self.first_year else (year - self.first_year) / (self.last_year - self.first_year)
        self.progress.set_width(max(.01, 936 * progress))
        self.cursor.set_data([72 + 936 * progress], [97.5])

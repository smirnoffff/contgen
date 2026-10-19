"""Small actual videos verify the CLI, library and ffmpeg together."""

import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


@unittest.skipUnless(shutil.which("ffmpeg") and shutil.which("ffprobe"),
                     "ffmpeg and ffprobe are required")
class RenderIntegrationTest(unittest.TestCase):
    def test_video_parameters_and_edge_cases(self):
        palette = {"background": "#030712", "card": "#0F172A",
                   "text": "#38BDF8", "subtext": "#64748B",
                   "bars": ["#38BDF8", "#818CF8", "#34D399"]}
        for periods, entities in ((1, 2), (2, 2), (8, 3), (3, 12)):
            with self.subTest(periods=periods, entities=entities), tempfile.TemporaryDirectory() as directory:
                output = Path(directory) / "video with spaces.mp4"
                points = [{"year": 2020 + year,
                           "name": f"Кириллица и очень длинное название {entity}",
                           "value": (entity + 0.005) * (year + 1)}
                          for year in range(periods) for entity in range(entities)]
                request = {"title": "Очень длинный русский заголовок для проверки переноса текста",
                           "points": points, "palette": palette, "width": 360,
                           "height": 640, "fps": 12, "seconds": 1,
                           "output_path": str(output)}
                completed = subprocess.run(
                    [sys.executable, "-u", str(Path(__file__).with_name("main.py"))],
                    input=json.dumps(request), text=True, capture_output=True, timeout=30,
                )
                self.assertEqual(completed.returncode, 0, completed.stderr)
                self.assertEqual(completed.stdout, "")
                self.assertEqual(completed.stderr, "")
                probe = subprocess.run(
                    ["ffprobe", "-v", "error", "-select_streams", "v:0",
                     "-show_entries", "stream=width,height,r_frame_rate,nb_frames,codec_name,pix_fmt:format=duration",
                     "-of", "json", str(output)],
                    text=True, capture_output=True, check=True, timeout=5,
                )
                metadata = json.loads(probe.stdout)
                video = metadata["streams"][0]
                self.assertEqual((video["width"], video["height"]), (360, 640))
                self.assertEqual(video["codec_name"], "h264")
                self.assertEqual(video["pix_fmt"], "yuv420p")
                self.assertEqual(video["r_frame_rate"], "12/1")
                self.assertEqual(int(video["nb_frames"]), 12)
                self.assertAlmostEqual(float(metadata["format"]["duration"]), 1)


if __name__ == "__main__":
    unittest.main()

"""One stdin request, one video, then exit. All diagnostics go to stderr."""

import json
import os
import signal
import sys
import tempfile


def cancel(_signum, _frame):
    # Let Matplotlib's writer close its pipe and reap ffmpeg before exiting.
    raise KeyboardInterrupt


def main():
    signal.signal(signal.SIGTERM, cancel)
    try:
        request = json.load(sys.stdin)
        # A per-process cache works with read-only deployment directories, too.
        with tempfile.TemporaryDirectory(prefix="contgen-mpl-") as cache:
            os.environ.setdefault("MPLCONFIGDIR", cache)
            from chart import render

            render(request)
        return 0
    except KeyboardInterrupt:
        print("render cancelled", file=sys.stderr)
        return 130
    except Exception as error:
        print(f"render failed: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())

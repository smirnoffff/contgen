package renderer

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"contgen/internal/model"
	"contgen/internal/storage/dbgen"
)

func useScript(t *testing.T, script string) string {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is needed for subprocess tests")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "worker with spaces.py")
	if err := os.WriteFile(path, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RENDERER_PYTHON", python)
	t.Setenv("RENDERER_SCRIPT", path)
	return dir
}

func samplePoints() []model.Point {
	return []model.Point{{EntityName: "Кириллица", Value: 12.5, RecordedAt: 2024}}
}

func assertNoPartial(t *testing.T, dir string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, ".contgen-render-*.mp4"))
	if err != nil || len(files) != 0 {
		t.Fatalf("partial outputs remain: %v, %v", files, err)
	}
}

func TestRenderProtocolAndAtomicReplacement(t *testing.T) {
	t.Setenv("RENDER_SECONDS", "")
	dir := useScript(t, `import json, pathlib, sys
request = json.load(sys.stdin)
output = pathlib.Path(request["output_path"])
output.with_name("request.json").write_text(json.dumps(request), encoding="utf-8")
output.write_bytes(b"new video")
`)
	output := filepath.Join(dir, "result with spaces.mp4")
	if err := os.WriteFile(output, []byte("old video"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RenderRacingBar(context.Background(), "Русский заголовок", samplePoints(), dbgen.RenderThemeCYBERHUD, output); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request renderRequest
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	if request.Title != "Русский заголовок" || request.Points[0].Name != "Кириллица" || request.Points[0].Value != 12.5 || request.Points[0].Year != 2024 {
		t.Fatalf("lost data in request: %+v", request)
	}
	if request.Width != 1080 || request.Height != 1920 || request.FPS != 30 || request.Seconds != defaultRenderSeconds || request.Palette.BgColor != GetPalette(dbgen.RenderThemeCYBERHUD).BgColor {
		t.Fatalf("wrong video parameters: %+v", request)
	}
	if !filepath.IsAbs(request.OutputPath) || request.OutputPath == output {
		t.Fatalf("expected an absolute temporary output path: %s", request.OutputPath)
	}
	result, err := os.ReadFile(output)
	if err != nil || string(result) != "new video" {
		t.Fatalf("replacement failed: %s, %v", result, err)
	}
	assertNoPartial(t, dir)
}

func TestRenderDurationCanBeConfigured(t *testing.T) {
	dir := useScript(t, `import json, pathlib, sys
request = json.load(sys.stdin)
pathlib.Path(request["output_path"]).with_name("request.json").write_text(json.dumps(request), encoding="utf-8")
pathlib.Path(request["output_path"]).write_bytes(b"video")
`)
	t.Setenv("RENDER_SECONDS", "24")
	output := filepath.Join(dir, "configured.mp4")
	if err := RenderRacingBar(context.Background(), "title", samplePoints(), dbgen.RenderThemeDARKGLASS, output); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request renderRequest
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	if request.Seconds != 24 {
		t.Fatalf("configured duration = %v, want 24", request.Seconds)
	}
}

func TestRenderFailuresPreserveOutput(t *testing.T) {
	for _, scenario := range []struct {
		name, script, message string
	}{
		{"python error", "import sys\nsys.stderr.write('x' * 20000 + 'FINAL DIAGNOSTIC')\nsys.exit(7)\n", "FINAL DIAGNOSTIC"},
		{"empty output", "pass\n", "не создал видео"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			dir := useScript(t, scenario.script)
			output := filepath.Join(dir, "result.mp4")
			if err := os.WriteFile(output, []byte("old video"), 0600); err != nil {
				t.Fatal(err)
			}
			err := RenderRacingBar(context.Background(), "title", samplePoints(), dbgen.RenderThemeDARKGLASS, output)
			if err == nil || !strings.Contains(err.Error(), scenario.message) {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(err.Error()) > stderrLimit+1024 {
				t.Fatalf("unbounded diagnostic: %d bytes", len(err.Error()))
			}
			result, readErr := os.ReadFile(output)
			if readErr != nil || string(result) != "old video" {
				t.Fatalf("old output changed: %s, %v", result, readErr)
			}
			assertNoPartial(t, dir)
		})
	}
}

func TestRenderCancellationStopsChildren(t *testing.T) {
	dir := useScript(t, `import json, pathlib, signal, subprocess, sys, time
request = json.load(sys.stdin)
def cancel(signum, frame):
    raise KeyboardInterrupt
signal.signal(signal.SIGTERM, cancel)
child = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(60)"])
pathlib.Path(request["output_path"]).with_name("child.pid").write_text(str(child.pid))
try:
    time.sleep(60)
except KeyboardInterrupt:
    pass
finally:
    child.terminate()
    child.wait()
`)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- RenderRacingBar(ctx, "title", samplePoints(), dbgen.RenderThemeDARKGLASS, filepath.Join(dir, "result.mp4"))
	}()
	var childPID int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(filepath.Join(dir, "child.pid"))
		if err == nil && json.Unmarshal(data, &childPID) == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if childPID == 0 {
		t.Fatal("child did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("renderer did not stop")
	}
	if err := syscall.Kill(childPID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("child is still present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "result.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled render published a result: %v", err)
	}
	assertNoPartial(t, dir)
}

func TestRenderRejectsInvalidInputAndMissingPython(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "result.mp4")
	t.Setenv("RENDERER_PYTHON", filepath.Join(dir, "missing-python"))
	for _, points := range [][]model.Point{
		nil,
		{{EntityName: "A", Value: math.Inf(1), RecordedAt: 2020}},
		samplePoints(),
	} {
		if err := RenderRacingBar(context.Background(), "title", points, dbgen.RenderThemeDARKGLASS, output); err == nil {
			t.Fatal("expected an error")
		}
		assertNoPartial(t, dir)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RenderRacingBar(ctx, "title", samplePoints(), dbgen.RenderThemeDARKGLASS, output); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected early context cancellation: %v", err)
	}
}

func TestRealRendererCancellationReapsFFmpeg(t *testing.T) {
	python, err := filepath.Abs("../../.venv/bin/python")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(python); err != nil {
		t.Skip("run make renderer-setup for the real cancellation test")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required")
	}
	script, err := filepath.Abs("../../python/renderer/main.py")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("RENDERER_PYTHON", python)
	t.Setenv("RENDERER_SCRIPT", script)
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- RenderRacingBar(ctx, "Настоящий рендер", samplePoints(), dbgen.RenderThemeDARKGLASS, filepath.Join(dir, "result.mp4"))
	}()
	var ffmpegPID int
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) && ffmpegPID == 0 {
		select {
		case err := <-done:
			t.Fatalf("renderer exited before ffmpeg started: %v", err)
		default:
		}
		for _, pythonID := range childPIDs(os.Getpid()) {
			for _, childID := range childPIDs(pythonID) {
				comm, _ := os.ReadFile(filepath.Join("/proc", strconv.Itoa(childID), "comm"))
				if strings.TrimSpace(string(comm)) == "ffmpeg" {
					ffmpegPID = childID
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if ffmpegPID == 0 {
		t.Fatal("ffmpeg did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("real renderer did not stop")
	}
	if err := syscall.Kill(ffmpegPID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("ffmpeg was not reaped: %v", err)
	}
	assertNoPartial(t, dir)
	if _, err := os.Stat(filepath.Join(dir, "result.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled render published a result: %v", err)
	}
}

func childPIDs(pid int) []int {
	// Go can spawn subprocesses from any OS thread, so inspect every task.
	tasksPath := filepath.Join("/proc", strconv.Itoa(pid), "task")
	tasks, _ := os.ReadDir(tasksPath)
	var result []int
	for _, task := range tasks {
		data, _ := os.ReadFile(filepath.Join(tasksPath, task.Name(), "children"))
		for _, field := range strings.Fields(string(data)) {
			child, err := strconv.Atoi(field)
			if err == nil {
				result = append(result, child)
			}
		}
	}
	return result
}

package renderer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// MuxAudio adds a track to an already rendered video and atomically replaces
// the video. ffmpeg is kept here beside the Python subprocess boundary, so the
// storage layer never knows about codecs or file handles.
func MuxAudio(ctx context.Context, videoPath, audioPath string) error {
	if videoPath == "" || audioPath == "" {
		return fmt.Errorf("не указан видео или аудиофайл")
	}
	if _, err := os.Stat(audioPath); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(videoPath), ".contgen-audio-*.mp4")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	defer os.Remove(tmpPath)
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", videoPath, "-i", audioPath,
		"-map", "0:v:0", "-map", "1:a:0", "-c:v", "copy", "-c:a", "aac",
		"-shortest", "-movflags", "+faststart", tmpPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, string(output))
	}
	if err := os.Rename(tmpPath, videoPath); err != nil {
		return err
	}
	return nil
}

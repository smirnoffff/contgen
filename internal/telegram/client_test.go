package telegram

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteVideoOmitsEmptyCaption(t *testing.T) {
	parts := writeVideoParts(t, "")
	if _, ok := parts["caption"]; ok {
		t.Fatal("empty caption must not be sent to Telegram")
	}
	if got := parts["chat_id"]; got != "123" {
		t.Fatalf("chat_id = %q, want 123", got)
	}
	if got := parts["supports_streaming"]; got != "true" {
		t.Fatalf("supports_streaming = %q, want true", got)
	}
	if got := parts["video"]; got != "contgen.mp4" {
		t.Fatalf("video filename = %q, want contgen.mp4", got)
	}
}

func TestWriteVideoIncludesNonEmptyCaption(t *testing.T) {
	parts := writeVideoParts(t, "A title")
	if got := parts["caption"]; got != "A title" {
		t.Fatalf("caption = %q, want A title", got)
	}
}

func writeVideoParts(t *testing.T, caption string) map[string]string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.mp4")
	if err := os.WriteFile(path, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := writeVideo(w, f, 123, caption); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	result := make(map[string]string)
	r := multipart.NewReader(&body, w.Boundary())
	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		if part.FileName() != "" {
			result[part.FormName()] = part.FileName()
		} else {
			result[part.FormName()] = string(data)
		}
	}
	return result
}

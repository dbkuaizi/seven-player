package player

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindMPVWatchLaterState(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "state1.conf")
	content := "# http://127.0.0.1:12000/stream?pickcode=abc123&name=demo.mp4\nstart=321.5\n"
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	state, err := FindMPVWatchLaterState(dir, "pickcode=abc123", time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("FindMPVWatchLaterState() error = %v", err)
	}
	if state == nil {
		t.Fatal("expected watch-later state")
	}
	if state.StartMS != 321500 {
		t.Fatalf("unexpected start ms: %d", state.StartMS)
	}
	if state.ConfigPath != target {
		t.Fatalf("unexpected config path: %s", state.ConfigPath)
	}
}

func TestParseMPVWatchLaterStart(t *testing.T) {
	got, ok := parseMPVWatchLaterStart("volume=50\nstart=90.25\n")
	if !ok {
		t.Fatal("expected start to be parsed")
	}
	if got != 90250 {
		t.Fatalf("unexpected start ms: %d", got)
	}
}

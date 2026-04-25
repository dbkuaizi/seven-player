package player

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestStatusesRespectsSelectedPlayer(t *testing.T) {
	statuses := Statuses(Settings{
		PreferredPlayer: PlayerVLC,
		PlayerPaths:     map[string]string{},
	})

	found := false
	for _, status := range statuses {
		if status.ID == PlayerVLC {
			found = true
			if !status.Selected {
				t.Fatalf("expected VLC to be selected: %+v", status)
			}
			if !status.SupportsStartPosition || !status.SupportsSubtitle {
				t.Fatalf("expected VLC capabilities to be exposed: %+v", status)
			}
		}
	}
	if !found {
		t.Fatal("VLC status not found")
	}
}

func TestResolveCustomPath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "mpv.exe")
	if err := os.WriteFile(file, []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	spec := findAdapter(PlayerMPV)
	got, source, err := spec.resolve(file)
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}
	if got != file {
		t.Fatalf("resolve() got %q want %q", got, file)
	}
	if source != "custom" {
		t.Fatalf("resolve() source = %q", source)
	}
}

func TestBuildMPVArgs(t *testing.T) {
	args := buildMPVArgs(Request{
		URL:              "https://example.com/video.mp4",
		Title:            "Demo",
		Subtitle:         "E:/sub/demo.srt",
		StartMS:          90500,
		ManagedResumeDir: "E:/watchlater",
	}, "", "")

	want := []string{
		"--force-window=yes",
		"--resume-playback=no",
		"--save-position-on-quit",
		"--watch-later-options=start",
		"--watch-later-directory=E:/watchlater",
		"--write-filename-in-watch-later-config",
		"--title=Demo",
		"--sub-file=E:/sub/demo.srt",
		"--start=90.5",
		"https://example.com/video.mp4",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected args: got=%v want=%v", args, want)
	}
}

func TestBuildPotPlayerArgs(t *testing.T) {
	args := buildPotPlayerArgs(Request{
		URL:      "https://example.com/video.mp4",
		Subtitle: "E:/sub/demo.srt",
		StartMS:  90500,
	}, "", "")

	want := []string{
		"https://example.com/video.mp4",
		"/seek=00:01:30",
		"/sub=E:/sub/demo.srt",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected args: got=%v want=%v", args, want)
	}
}

func TestWindowsPaths(t *testing.T) {
	paths := windowsPaths(`VideoLAN\VLC\vlc.exe`)
	if runtime.GOOS == "windows" && len(paths) == 0 {
		t.Fatal("expected windows paths")
	}
}

func TestValidateRequestRejectsUnsupportedStartPosition(t *testing.T) {
	spec := findAdapter(PlayerMPCHC)
	err := spec.validateRequest(Request{
		URL:     "https://example.com/video.mp4",
		StartMS: 30000,
	})
	if err == nil {
		t.Fatal("expected unsupported start position error")
	}
}

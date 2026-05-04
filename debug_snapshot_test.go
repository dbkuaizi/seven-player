package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"panplayer/internal/config"
	"panplayer/internal/library"
)

func TestDebugSnapshot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	app, err := NewApp(logger)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	storePath := filepath.Join("bin", "panplayer.sqlite")
	if _, statErr := os.Stat(storePath); statErr == nil {
		_ = app.store.Close()
		realStore, openErr := config.NewStore(storePath)
		if openErr != nil {
			t.Fatalf("config.NewStore(%q) error = %v", storePath, openErr)
		}
		app.store = realStore
		app.library = library.NewRepository(realStore.DB())
		state, loadErr := realStore.Load()
		if loadErr != nil {
			t.Fatalf("realStore.Load() error = %v", loadErr)
		}
		app.state = state
	}

	app.startup(context.Background())
	defer app.shutdown(context.Background())

	snapshot, err := app.GetLibrarySnapshot()
	if err != nil {
		t.Fatalf("GetLibrarySnapshot() error = %v", err)
	}
	if len(snapshot.Items) == 0 {
		t.Fatalf("snapshot is empty")
	}

	for index, item := range snapshot.Items {
		if index >= 4 {
			break
		}
		t.Logf("item[%d] title=%q poster=%q backdrop=%q source=%q subject=%q cast=%d", index, item.Title, item.PosterURL, item.BackdropURL, item.SourceName, item.SourceSubjectID, len(item.CastMembers))
	}

	for _, item := range snapshot.Items {
		if item.Title != "权力的游戏 第六季" && item.Title != "风声" {
			continue
		}
		detail, err := app.GetLibraryDetail(item.ID)
		if err != nil {
			t.Fatalf("GetLibraryDetail(%q) error = %v", item.Title, err)
		}
		t.Logf("detail title=%q files=%d section=%q seasonCount=%d episodeCount=%d cast=%d backdrop=%q", detail.Title, len(detail.Files), detail.Section, detail.SeasonCount, detail.EpisodeCount, len(detail.CastMembers), detail.BackdropURL)
		for index, cast := range detail.CastMembers {
			if index >= 5 {
				break
			}
			t.Logf("%s cast[%d] name=%q avatar=%q role=%q character=%q", detail.Title, index, cast.Name, cast.AvatarURL, cast.Role, cast.Character)
		}
		for index, file := range detail.Files {
			if index >= 12 {
				break
			}
			t.Logf("%s episode[%d] s=%d e=%d dur=%d title=%q thumb=%q name=%q", detail.Title, index, file.SeasonNumber, file.EpisodeNumber, file.DurationSec, file.EpisodeTitle, file.ThumbnailURL, file.Name)
		}
	}
}

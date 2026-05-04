package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "panplayer.sqlite"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	want := State{
		Settings: Settings{
			PreferredPlayer:   "vlc",
			SmallFileFilterMB: 3,
			PlayerPaths: map[string]string{
				"mpv": "C:/tools/mpv.exe",
				"vlc": "C:/tools/vlc.exe",
			},
			OfflineRecentTargets: []DirectoryTarget{
				{
					ID: "100",
					Path: []Breadcrumb{
						{ID: "0", Name: "我的文件"},
						{ID: "100", Name: "影视"},
					},
				},
			},
			ScraperSources:      []string{"tmdb", "douban"},
			ScraperLanguage:     "zh-CN",
			TMDBReadAccessToken: "tmdb-demo-token",
		},
		Credential: &Credential{
			UID:  "u",
			CID:  "c",
			SEID: "s",
			KID:  "k",
		},
		Cookies: map[string]string{
			"UID":  "u",
			"CID":  "c",
			"SEID": "s",
			"KID":  "k",
		},
		LastDirectoryID: "123",
		PlaybackRecords: map[string]PlaybackRecord{
			"pc1": {
				PickCode:       "pc1",
				FileName:       "demo.mp4",
				LastPositionMS: 90500,
				SubtitlePath:   "D:/subs/demo.srt",
				LastPlayerID:   "mpv",
				LastPlayedAt:   "2026-04-25T10:11:12Z",
			},
		},
		Window: WindowState{
			Width:     1280,
			Height:    820,
			Maximised: true,
		},
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Settings.PreferredPlayer != want.Settings.PreferredPlayer {
		t.Fatalf("PreferredPlayer mismatch: got %q want %q", got.Settings.PreferredPlayer, want.Settings.PreferredPlayer)
	}
	if got.Settings.PlayerPaths["mpv"] != "C:/tools/mpv.exe" || got.Settings.PlayerPaths["vlc"] != "C:/tools/vlc.exe" {
		t.Fatalf("PlayerPaths mismatch: %+v", got.Settings.PlayerPaths)
	}
	if got.Settings.SmallFileFilterMB != want.Settings.SmallFileFilterMB {
		t.Fatalf("SmallFileFilterMB mismatch: got %v want %v", got.Settings.SmallFileFilterMB, want.Settings.SmallFileFilterMB)
	}
	if len(got.Settings.OfflineRecentTargets) != 1 || got.Settings.OfflineRecentTargets[0].ID != "100" {
		t.Fatalf("OfflineRecentTargets mismatch: %+v", got.Settings.OfflineRecentTargets)
	}
	if len(got.Settings.ScraperSources) != 2 || got.Settings.ScraperSources[0] != "tmdb" {
		t.Fatalf("ScraperSources mismatch: %+v", got.Settings.ScraperSources)
	}
	if got.Settings.TMDBReadAccessToken != "tmdb-demo-token" {
		t.Fatalf("TMDBReadAccessToken mismatch: got %q", got.Settings.TMDBReadAccessToken)
	}
	if got.LastDirectoryID != want.LastDirectoryID {
		t.Fatalf("LastDirectoryID mismatch: got %q want %q", got.LastDirectoryID, want.LastDirectoryID)
	}
	if got.Credential == nil || got.Credential.UID != "u" || got.Credential.KID != "k" {
		t.Fatalf("credential mismatch: %+v", got.Credential)
	}
	if got.Cookies["UID"] != "u" || got.Cookies["KID"] != "k" {
		t.Fatalf("cookies mismatch: %+v", got.Cookies)
	}
	record, ok := got.PlaybackRecords["pc1"]
	if !ok {
		t.Fatalf("playback record missing: %+v", got.PlaybackRecords)
	}
	if record.LastPositionMS != 90500 || record.SubtitlePath != "D:/subs/demo.srt" {
		t.Fatalf("playback record mismatch: %+v", record)
	}
	if got.Window.Width != 1280 || got.Window.Height != 820 || !got.Window.Maximised {
		t.Fatalf("window state mismatch: %+v", got.Window)
	}
}

func TestNormalizeLegacyMPVPath(t *testing.T) {
	settings := normalizeSettings(Settings{
		MPVPath: "D:/Program Files/mpv/mpv.exe",
	})

	if settings.PlayerPaths["mpv"] != "D:/Program Files/mpv/mpv.exe" {
		t.Fatalf("legacy mpv path not migrated: %+v", settings.PlayerPaths)
	}
	if settings.PreferredPlayer != "mpv" {
		t.Fatalf("preferred player mismatch: %q", settings.PreferredPlayer)
	}
}

func TestNormalizeLegacyHideSmallFiles(t *testing.T) {
	settings := normalizeSettings(Settings{
		HideSmallFiles: true,
	})

	if settings.SmallFileFilterMB != 1 {
		t.Fatalf("small file filter migration mismatch: %d", settings.SmallFileFilterMB)
	}
	if settings.HideSmallFiles {
		t.Fatalf("legacy hideSmallFiles flag should be cleared after migration")
	}
}

func TestNormalizeOfflineRecentTargets(t *testing.T) {
	settings := normalizeSettings(Settings{
		OfflineRecentTargets: []DirectoryTarget{
			{
				ID: "100",
				Path: []Breadcrumb{
					{ID: "100", Name: "影视"},
				},
			},
			{
				ID: "100",
				Path: []Breadcrumb{
					{ID: "0", Name: "我的文件"},
					{ID: "100", Name: "影视"},
				},
			},
			{
				ID: "200",
				Path: []Breadcrumb{
					{ID: "0", Name: "我的文件"},
					{ID: "200", Name: "动漫"},
				},
			},
			{
				ID: "300",
				Path: []Breadcrumb{
					{ID: "0", Name: "我的文件"},
					{ID: "300", Name: "纪录片"},
				},
			},
			{
				ID: "400",
				Path: []Breadcrumb{
					{ID: "0", Name: "我的文件"},
					{ID: "400", Name: "音乐"},
				},
			},
		},
	})

	if len(settings.OfflineRecentTargets) != 3 {
		t.Fatalf("OfflineRecentTargets length mismatch: %+v", settings.OfflineRecentTargets)
	}
	if settings.OfflineRecentTargets[0].Path[0].ID != "0" {
		t.Fatalf("root breadcrumb should be normalized: %+v", settings.OfflineRecentTargets[0].Path)
	}
	if settings.OfflineRecentTargets[0].ID != "100" || settings.OfflineRecentTargets[1].ID != "200" || settings.OfflineRecentTargets[2].ID != "300" {
		t.Fatalf("unexpected OfflineRecentTargets order: %+v", settings.OfflineRecentTargets)
	}
}

func TestNewStoreMigratesLegacyJSON(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(legacyPath, []byte(`{"lastDirectoryId":"456","cookies":{"UID":"demo"}}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := NewStore(filepath.Join(dir, "panplayer.sqlite"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.LastDirectoryID != "456" {
		t.Fatalf("LastDirectoryID mismatch: got %q want %q", got.LastDirectoryID, "456")
	}
	if got.Cookies["UID"] != "demo" {
		t.Fatalf("cookies mismatch: %+v", got.Cookies)
	}
}

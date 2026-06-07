package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	want := State{
		Settings: Settings{
			PreferredPlayer:   "vlc",
			SmallFileFilterMB: 3,
			ThemeColor:        "rose",
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
		HiddenModeEnabled:     true,
		HiddenModePasswordMD5: "abc123md5",
		LastDirectoryID:       "123",
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
	if got.Settings.ThemeColor != want.Settings.ThemeColor {
		t.Fatalf("ThemeColor mismatch: got %q want %q", got.Settings.ThemeColor, want.Settings.ThemeColor)
	}
	if len(got.Settings.OfflineRecentTargets) != 1 || got.Settings.OfflineRecentTargets[0].ID != "100" {
		t.Fatalf("OfflineRecentTargets mismatch: %+v", got.Settings.OfflineRecentTargets)
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
	if !got.HiddenModeEnabled || got.HiddenModePasswordMD5 != "abc123md5" {
		t.Fatalf("hidden mode state mismatch: enabled=%v md5=%q", got.HiddenModeEnabled, got.HiddenModePasswordMD5)
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

	assertNoSQLiteSidecars(t, dbPath)

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	assertNoSQLiteSidecars(t, dbPath)
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

func TestNormalizeThemeColor(t *testing.T) {
	settings := normalizeSettings(Settings{
		ThemeColor: "violet",
	})
	if settings.ThemeColor != "violet" {
		t.Fatalf("theme color mismatch: %q", settings.ThemeColor)
	}

	settings = normalizeSettings(Settings{
		ThemeColor: "unknown",
	})
	if settings.ThemeColor != "blue" {
		t.Fatalf("invalid theme color should normalize to blue: %q", settings.ThemeColor)
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

	store, err := NewStore(filepath.Join(dir, "data.db"))
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

func TestNewStoreMigratesSevenPlayerSQLite(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "seven-player.sqlite")
	legacyStore, err := NewStore(legacyPath)
	if err != nil {
		t.Fatalf("NewStore(legacy) error = %v", err)
	}
	if err := legacyStore.Save(State{
		LastDirectoryID: "789",
		Cookies: map[string]string{
			"UID": "legacy",
		},
	}); err != nil {
		t.Fatalf("Save(legacy) error = %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatalf("Close(legacy) error = %v", err)
	}

	store, err := NewStore(filepath.Join(dir, "data.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.LastDirectoryID != "789" {
		t.Fatalf("LastDirectoryID mismatch: got %q want %q", got.LastDirectoryID, "789")
	}
	if got.Cookies["UID"] != "legacy" {
		t.Fatalf("cookies mismatch: %+v", got.Cookies)
	}
	if _, err := os.Stat(legacyPath); err == nil {
		t.Fatalf("legacy sqlite should be removed after migration: %s", legacyPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(legacy) error = %v", err)
	}
}

func TestNewStoreMigratesPanPlayerSQLite(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "panplayer.sqlite")
	legacyStore, err := NewStore(legacyPath)
	if err != nil {
		t.Fatalf("NewStore(legacy) error = %v", err)
	}
	if err := legacyStore.Save(State{
		LastDirectoryID: "pan",
		Cookies: map[string]string{
			"UID": "pan-legacy",
		},
	}); err != nil {
		t.Fatalf("Save(legacy) error = %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatalf("Close(legacy) error = %v", err)
	}

	store, err := NewStore(filepath.Join(dir, "data.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.LastDirectoryID != "pan" {
		t.Fatalf("LastDirectoryID mismatch: got %q want %q", got.LastDirectoryID, "pan")
	}
	if got.Cookies["UID"] != "pan-legacy" {
		t.Fatalf("cookies mismatch: %+v", got.Cookies)
	}
}

func TestNewStoreRemovesLegacySQLiteWhenDataDBExists(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "data.db")
	store, err := NewStore(dataPath)
	if err != nil {
		t.Fatalf("NewStore(data) error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close(data) error = %v", err)
	}

	for _, legacyName := range []string{"seven-player.sqlite", "panplayer.sqlite"} {
		legacyPath := filepath.Join(dir, legacyName)
		if err := os.WriteFile(legacyPath, []byte("stale"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", legacyName, err)
		}
	}

	store, err = NewStore(dataPath)
	if err != nil {
		t.Fatalf("NewStore(existing data) error = %v", err)
	}
	defer store.Close()

	for _, legacyName := range []string{"seven-player.sqlite", "panplayer.sqlite"} {
		legacyPath := filepath.Join(dir, legacyName)
		if _, err := os.Stat(legacyPath); err == nil {
			t.Fatalf("legacy sqlite should be removed: %s", legacyPath)
		} else if !os.IsNotExist(err) {
			t.Fatalf("Stat(%s) error = %v", legacyPath, err)
		}
	}
}

func TestCleanupSQLiteSidecars(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data.db")
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if err := os.WriteFile(dbPath+suffix, []byte("stale"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", suffix, err)
		}
	}

	if err := cleanupSQLiteSidecars(dbPath); err != nil {
		t.Fatalf("cleanupSQLiteSidecars() error = %v", err)
	}

	assertNoSQLiteSidecars(t, dbPath)
}

func assertNoSQLiteSidecars(t *testing.T, dbPath string) {
	t.Helper()

	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		path := dbPath + suffix
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("unexpected sqlite sidecar exists: %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("Stat(%s) error = %v", path, err)
		}
	}
}

package config

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

const (
	appConfigDirName     = "seven-player"
	legacyConfigDirName  = "panplayer"
	sqliteFileName       = "seven-player.sqlite"
	legacySQLiteFileName = "panplayer.sqlite"
)

type Credential struct {
	UID  string `json:"uid"`
	CID  string `json:"cid"`
	SEID string `json:"seid"`
	KID  string `json:"kid"`
}

type Breadcrumb struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DirectoryTarget struct {
	ID   string       `json:"id"`
	Path []Breadcrumb `json:"path,omitempty"`
}

type Settings struct {
	PreferredPlayer          string            `json:"preferredPlayer,omitempty"`
	PlayerPaths              map[string]string `json:"playerPaths,omitempty"`
	DisabledPlayers          map[string]bool   `json:"disabledPlayers,omitempty"`
	MPVPath                  string            `json:"mpvPath,omitempty"`
	HideTitleBadges          bool              `json:"hideTitleBadges,omitempty"`
	DisableCleanTitleDisplay bool              `json:"disableCleanTitleDisplay,omitempty"`
	UIScalePercent           int               `json:"uiScalePercent,omitempty"`
	ThemeMode                string            `json:"themeMode,omitempty"`
	HideSmallFiles           bool              `json:"hideSmallFiles,omitempty"`
	SmallFileFilterMB        int               `json:"smallFileFilterMB,omitempty"`
	FileListDensity          string            `json:"fileListDensity,omitempty"`
	OfflineRecentTargets     []DirectoryTarget `json:"offlineRecentTargets,omitempty"`
}

type PlaybackRecord struct {
	PickCode       string `json:"pickCode,omitempty"`
	FileName       string `json:"fileName,omitempty"`
	LastPositionMS int64  `json:"lastPositionMs,omitempty"`
	SubtitlePath   string `json:"subtitlePath,omitempty"`
	LastPlayerID   string `json:"lastPlayerId,omitempty"`
	LastPlayedAt   string `json:"lastPlayedAt,omitempty"`
}

type WindowState struct {
	Width     int  `json:"width,omitempty"`
	Height    int  `json:"height,omitempty"`
	Maximised bool `json:"maximised,omitempty"`
}

type State struct {
	Settings              Settings                  `json:"settings"`
	Credential            *Credential               `json:"credential,omitempty"`
	Cookies               map[string]string         `json:"cookies,omitempty"`
	HiddenModeEnabled     bool                      `json:"hiddenModeEnabled,omitempty"`
	HiddenModePasswordMD5 string                    `json:"hiddenModePasswordMd5,omitempty"`
	LastDirectoryID       string                    `json:"lastDirectoryId"`
	PlaybackRecords       map[string]PlaybackRecord `json:"playbackRecords,omitempty"`
	Window                WindowState               `json:"window,omitempty"`
}

type Store struct {
	path string
	db   *sql.DB
	mu   sync.Mutex
}

func DefaultState() State {
	return State{
		Settings: Settings{
			PreferredPlayer:      "mpv",
			PlayerPaths:          map[string]string{},
			DisabledPlayers:      map[string]bool{},
			UIScalePercent:       100,
			ThemeMode:            "system",
			FileListDensity:      "default",
			OfflineRecentTargets: []DirectoryTarget{},
		},
		PlaybackRecords: map[string]PlaybackRecord{},
	}
}

func DefaultPath() (string, error) {
	executablePath, err := os.Executable()
	if err == nil && strings.TrimSpace(executablePath) != "" {
		return filepath.Join(filepath.Dir(executablePath), sqliteFileName), nil
	}

	workingDirectory, wdErr := os.Getwd()
	if wdErr == nil && strings.TrimSpace(workingDirectory) != "" {
		return filepath.Join(workingDirectory, sqliteFileName), nil
	}

	if err != nil {
		return "", err
	}
	return "", wdErr
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := migrateLegacySQLite(path); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	store := &Store{
		path: path,
		db:   db,
	}

	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := store.migrateLegacyJSON(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Load() (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := DefaultState()

	var data string
	err := s.db.QueryRow(`SELECT data FROM app_state WHERE id = 1`).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return State{}, err
	}

	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return State{}, err
	}

	return normalizeState(state), nil
}

func (s *Store) Save(state State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveLocked(state)
}

func (s *Store) initSchema() error {
	_, err := s.db.Exec(`
		PRAGMA journal_mode = WAL;
		CREATE TABLE IF NOT EXISTS app_state (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			data TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func (s *Store) saveLocked(state State) error {
	state = normalizeState(state)

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO app_state (id, data, updated_at)
		VALUES (1, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			data = excluded.data,
			updated_at = CURRENT_TIMESTAMP
	`, string(data))
	return err
}

func (s *Store) migrateLegacyJSON() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM app_state WHERE id = 1`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	for _, candidate := range legacyJSONPaths(s.path) {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		state := DefaultState()
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		return s.saveLocked(state)
	}

	return nil
}

func legacyJSONPaths(sqlitePath string) []string {
	candidates := []string{
		filepath.Join(filepath.Dir(sqlitePath), "config.json"),
	}

	if configDirectory, err := os.UserConfigDir(); err == nil && strings.TrimSpace(configDirectory) != "" {
		candidates = append(candidates,
			filepath.Join(configDirectory, appConfigDirName, "config.json"),
			filepath.Join(configDirectory, legacyConfigDirName, "config.json"),
		)
	}

	unique := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || slices.Contains(unique, candidate) {
			continue
		}
		unique = append(unique, candidate)
	}
	return unique
}

func migrateLegacySQLite(path string) error {
	if filepath.Base(path) != sqliteFileName {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	legacyPath := filepath.Join(filepath.Dir(path), legacySQLiteFileName)
	if _, err := os.Stat(legacyPath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	if err := copyFileIfExists(legacyPath, path, true); err != nil {
		return err
	}
	if err := copyFileIfExists(legacyPath+"-wal", path+"-wal", false); err != nil {
		return err
	}
	return copyFileIfExists(legacyPath+"-shm", path+"-shm", false)
}

func copyFileIfExists(sourcePath, targetPath string, exclusive bool) error {
	source, err := os.Open(sourcePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer source.Close()

	flags := os.O_CREATE | os.O_WRONLY
	if exclusive {
		flags |= os.O_EXCL
	} else {
		flags |= os.O_TRUNC
	}
	target, err := os.OpenFile(targetPath, flags, 0o644)
	if err != nil {
		return err
	}
	defer target.Close()

	_, err = io.Copy(target, source)
	return err
}

func normalizeSettings(settings Settings) Settings {
	if settings.PlayerPaths == nil {
		settings.PlayerPaths = map[string]string{}
	}
	if settings.DisabledPlayers == nil {
		settings.DisabledPlayers = map[string]bool{}
	}
	if settings.OfflineRecentTargets == nil {
		settings.OfflineRecentTargets = []DirectoryTarget{}
	}
	if settings.MPVPath != "" && settings.PlayerPaths["mpv"] == "" {
		settings.PlayerPaths["mpv"] = settings.MPVPath
	}
	settings.MPVPath = ""
	if settings.PreferredPlayer == "" {
		settings.PreferredPlayer = "mpv"
	}
	if settings.SmallFileFilterMB == 0 && settings.HideSmallFiles {
		settings.SmallFileFilterMB = 1
	}
	settings.SmallFileFilterMB = normalizeSmallFileFilterMB(settings.SmallFileFilterMB)
	settings.FileListDensity = normalizeFileListDensity(settings.FileListDensity)
	settings.UIScalePercent = normalizeUIScalePercent(settings.UIScalePercent)
	settings.ThemeMode = normalizeThemeMode(settings.ThemeMode)
	settings.OfflineRecentTargets = NormalizeOfflineRecentTargets(settings.OfflineRecentTargets)
	settings.HideSmallFiles = false
	return settings
}

func NormalizeOfflineRecentTargets(targets []DirectoryTarget) []DirectoryTarget {
	return NormalizeDirectoryTargets(targets, 3)
}

func NormalizeDirectoryTargets(targets []DirectoryTarget, limit int) []DirectoryTarget {
	if limit <= 0 {
		return []DirectoryTarget{}
	}

	normalized := make([]DirectoryTarget, 0, 3)
	seen := make(map[string]struct{}, limit)

	for _, target := range targets {
		current, ok := NormalizeDirectoryTarget(target)
		if !ok {
			continue
		}
		if _, exists := seen[current.ID]; exists {
			continue
		}
		seen[current.ID] = struct{}{}
		normalized = append(normalized, current)
		if len(normalized) == limit {
			break
		}
	}

	if normalized == nil {
		return []DirectoryTarget{}
	}
	return normalized
}

func NormalizeDirectoryTarget(target DirectoryTarget) (DirectoryTarget, bool) {
	target.ID = strings.TrimSpace(target.ID)
	target.Path = normalizeBreadcrumbs(target.Path)

	if target.ID == "" && len(target.Path) > 0 {
		target.ID = target.Path[len(target.Path)-1].ID
	}
	if target.ID == "" {
		target.ID = "0"
	}

	if target.ID == "0" {
		return DirectoryTarget{
			ID: "0",
			Path: []Breadcrumb{
				{ID: "0", Name: "我的文件"},
			},
		}, true
	}

	if len(target.Path) == 0 {
		return DirectoryTarget{}, false
	}

	lastID := target.Path[len(target.Path)-1].ID
	if lastID != target.ID {
		matchedIndex := -1
		for index, crumb := range target.Path {
			if crumb.ID == target.ID {
				matchedIndex = index
				break
			}
		}
		if matchedIndex == -1 {
			return DirectoryTarget{}, false
		}
		target.Path = append([]Breadcrumb(nil), target.Path[:matchedIndex+1]...)
	}

	return DirectoryTarget{
		ID:   target.ID,
		Path: append([]Breadcrumb(nil), target.Path...),
	}, true
}

func normalizeBreadcrumbs(path []Breadcrumb) []Breadcrumb {
	normalized := make([]Breadcrumb, 0, len(path)+1)

	for _, crumb := range path {
		id := strings.TrimSpace(crumb.ID)
		name := strings.TrimSpace(crumb.Name)
		if id == "" || name == "" {
			continue
		}
		normalized = append(normalized, Breadcrumb{
			ID:   id,
			Name: name,
		})
	}

	if len(normalized) == 0 {
		return nil
	}

	if normalized[0].ID != "0" {
		normalized = append([]Breadcrumb{{ID: "0", Name: "我的文件"}}, normalized...)
	}

	return normalized
}

func normalizeSmallFileFilterMB(value int) int {
	switch value {
	case 0, 1, 2, 3, 5, 10:
		return value
	default:
		return 0
	}
}

func normalizeFileListDensity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "compact":
		return "compact"
	case "comfortable":
		return "comfortable"
	default:
		return "default"
	}
}

func normalizeUIScalePercent(value int) int {
	if value < 100 {
		return 100
	}
	if value > 150 {
		return 150
	}
	return ((value + 2) / 5) * 5
}

func normalizeThemeMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "light", "dark":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "system"
	}
}

func normalizeState(state State) State {
	state.Settings = normalizeSettings(state.Settings)
	if state.Credential != nil {
		state.Credential.UID = strings.TrimSpace(state.Credential.UID)
		state.Credential.CID = strings.TrimSpace(state.Credential.CID)
		state.Credential.SEID = strings.TrimSpace(state.Credential.SEID)
		state.Credential.KID = strings.TrimSpace(state.Credential.KID)
		if state.Credential.UID == "" &&
			state.Credential.CID == "" &&
			state.Credential.SEID == "" &&
			state.Credential.KID == "" {
			state.Credential = nil
		}
	}
	if state.Cookies == nil {
		state.Cookies = map[string]string{}
	}
	state.HiddenModePasswordMD5 = strings.TrimSpace(state.HiddenModePasswordMD5)
	if state.PlaybackRecords == nil {
		state.PlaybackRecords = map[string]PlaybackRecord{}
	}
	if state.Window.Width < 0 {
		state.Window.Width = 0
	}
	if state.Window.Height < 0 {
		state.Window.Height = 0
	}
	return state
}

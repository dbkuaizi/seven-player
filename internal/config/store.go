package config

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
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
	PreferredPlayer      string            `json:"preferredPlayer,omitempty"`
	PlayerPaths          map[string]string `json:"playerPaths,omitempty"`
	DisabledPlayers      map[string]bool   `json:"disabledPlayers,omitempty"`
	MPVPath              string            `json:"mpvPath,omitempty"`
	HideTitleBadges      bool              `json:"hideTitleBadges,omitempty"`
	HideSmallFiles       bool              `json:"hideSmallFiles,omitempty"`
	SmallFileFilterMB    int               `json:"smallFileFilterMB,omitempty"`
	FileListDensity      string            `json:"fileListDensity,omitempty"`
	OfflineRecentTargets []DirectoryTarget `json:"offlineRecentTargets,omitempty"`
	ScraperDirectories   []DirectoryTarget `json:"scraperDirectories,omitempty"`
	ScraperSources       []string          `json:"scraperSources,omitempty"`
	ScraperLanguage      string            `json:"scraperLanguage,omitempty"`
	ScraperAutoScan      bool              `json:"scraperAutoScan,omitempty"`
	ScraperOverwrite     bool              `json:"scraperOverwrite,omitempty"`
	ScraperSkipImages    bool              `json:"scraperSkipImages,omitempty"`
	TMDBReadAccessToken  string            `json:"tmdbReadAccessToken,omitempty"`
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
	Settings        Settings                  `json:"settings"`
	Credential      *Credential               `json:"credential,omitempty"`
	Cookies         map[string]string         `json:"cookies,omitempty"`
	LastDirectoryID string                    `json:"lastDirectoryId"`
	PlaybackRecords map[string]PlaybackRecord `json:"playbackRecords,omitempty"`
	Window          WindowState               `json:"window,omitempty"`
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
			FileListDensity:      "default",
			OfflineRecentTargets: []DirectoryTarget{},
			ScraperDirectories:   []DirectoryTarget{},
			ScraperSources:       DefaultScraperSources(),
			ScraperLanguage:      "zh-CN",
		},
		PlaybackRecords: map[string]PlaybackRecord{},
	}
}

func DefaultPath() (string, error) {
	executablePath, err := os.Executable()
	if err == nil && strings.TrimSpace(executablePath) != "" {
		return filepath.Join(filepath.Dir(executablePath), "panplayer.sqlite"), nil
	}

	workingDirectory, wdErr := os.Getwd()
	if wdErr == nil && strings.TrimSpace(workingDirectory) != "" {
		return filepath.Join(workingDirectory, "panplayer.sqlite"), nil
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
		CREATE TABLE IF NOT EXISTS library_titles (
			id TEXT PRIMARY KEY,
			group_key TEXT NOT NULL UNIQUE,
			section TEXT NOT NULL,
			child TEXT NOT NULL,
			title TEXT NOT NULL,
			original_title TEXT NOT NULL DEFAULT '',
			search_title TEXT NOT NULL DEFAULT '',
			normalized_title TEXT NOT NULL DEFAULT '',
			year INTEGER NOT NULL DEFAULT 0,
			rating REAL NOT NULL DEFAULT 0,
			summary TEXT NOT NULL DEFAULT '',
			director TEXT NOT NULL DEFAULT '',
			cast_json TEXT NOT NULL DEFAULT '[]',
			tags_json TEXT NOT NULL DEFAULT '[]',
			poster_url TEXT NOT NULL DEFAULT '',
			poster_local_path TEXT NOT NULL DEFAULT '',
			backdrop_url TEXT NOT NULL DEFAULT '',
			backdrop_local_path TEXT NOT NULL DEFAULT '',
			quality TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			audio TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			source_name TEXT NOT NULL DEFAULT '',
			source_subject_id TEXT NOT NULL DEFAULT '',
			external_data_json TEXT NOT NULL DEFAULT '{}',
			default_file_id TEXT NOT NULL DEFAULT '',
			season_count INTEGER NOT NULL DEFAULT 0,
			episode_count INTEGER NOT NULL DEFAULT 0,
			total_duration_sec INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			scraped_at TEXT NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_library_titles_section ON library_titles(section, child, updated_at DESC);
		CREATE INDEX IF NOT EXISTS idx_library_titles_group_key ON library_titles(group_key);

		CREATE TABLE IF NOT EXISTS library_files (
			id TEXT PRIMARY KEY,
			title_id TEXT NOT NULL,
			file_id TEXT NOT NULL UNIQUE,
			pick_code TEXT NOT NULL DEFAULT '',
			parent_id TEXT NOT NULL DEFAULT '',
			path_json TEXT NOT NULL DEFAULT '[]',
			name TEXT NOT NULL,
			size INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL DEFAULT '',
			duration_sec INTEGER NOT NULL DEFAULT 0,
			season_number INTEGER NOT NULL DEFAULT 0,
			episode_number INTEGER NOT NULL DEFAULT 0,
			episode_title TEXT NOT NULL DEFAULT '',
			quality TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			audio TEXT NOT NULL DEFAULT '',
			search_title TEXT NOT NULL DEFAULT '',
			parsed_year INTEGER NOT NULL DEFAULT 0,
			section TEXT NOT NULL,
			child TEXT NOT NULL,
			display_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at_row TEXT NOT NULL,
			scraped_at TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(title_id) REFERENCES library_titles(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_library_files_title_id ON library_files(title_id, season_number, episode_number, display_order, name);
		CREATE INDEX IF NOT EXISTS idx_library_files_file_id ON library_files(file_id);

		CREATE TABLE IF NOT EXISTS scraper_jobs (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			message TEXT NOT NULL DEFAULT '',
			current_path TEXT NOT NULL DEFAULT '',
			scanned_directories INTEGER NOT NULL DEFAULT 0,
			total_directories INTEGER NOT NULL DEFAULT 0,
			discovered_files INTEGER NOT NULL DEFAULT 0,
			processed_files INTEGER NOT NULL DEFAULT 0,
			matched_items INTEGER NOT NULL DEFAULT 0,
			updated_titles INTEGER NOT NULL DEFAULT 0,
			error_count INTEGER NOT NULL DEFAULT 0,
			settings_json TEXT NOT NULL DEFAULT '{}',
			last_error TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			finished_at TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_scraper_jobs_updated_at ON scraper_jobs(updated_at DESC);
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
		candidates = append(candidates, filepath.Join(configDirectory, "panplayer", "config.json"))
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
	if settings.ScraperDirectories == nil {
		settings.ScraperDirectories = []DirectoryTarget{}
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
	settings.OfflineRecentTargets = NormalizeOfflineRecentTargets(settings.OfflineRecentTargets)
	settings.ScraperDirectories = NormalizeDirectoryTargets(settings.ScraperDirectories, 50)
	settings.ScraperSources = NormalizeScraperSources(settings.ScraperSources)
	settings.ScraperLanguage = NormalizeScraperLanguage(settings.ScraperLanguage)
	settings.TMDBReadAccessToken = strings.TrimSpace(settings.TMDBReadAccessToken)
	settings.HideSmallFiles = false
	return settings
}

func DefaultScraperSources() []string {
	return []string{"tmdb", "douban", "bangumi"}
}

func NormalizeScraperSources(sources []string) []string {
	allowed := map[string]struct{}{
		"tmdb":    {},
		"douban":  {},
		"bangumi": {},
	}
	normalized := make([]string, 0, len(allowed))
	seen := make(map[string]struct{}, len(allowed))

	for _, source := range sources {
		source = strings.ToLower(strings.TrimSpace(source))
		if _, ok := allowed[source]; !ok {
			continue
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}
		normalized = append(normalized, source)
	}

	if len(normalized) == 0 {
		return DefaultScraperSources()
	}
	return normalized
}

func NormalizeScraperLanguage(language string) string {
	switch strings.TrimSpace(language) {
	case "zh-CN", "zh-TW", "en-US", "ja-JP", "ko-KR":
		return strings.TrimSpace(language)
	default:
		return "zh-CN"
	}
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

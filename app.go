package main

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sevenplayer/internal/config"
	"sevenplayer/internal/pan"
	"sevenplayer/internal/player"
	"sevenplayer/internal/proxy"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type App struct {
	ctx    context.Context
	logger *slog.Logger
	window application.Window

	store   *config.Store
	pan     *pan.Service
	proxy   *proxy.Server
	started bool

	mu    sync.RWMutex
	state config.State
}

type BootstrapResult struct {
	LoggedIn                     bool                     `json:"loggedIn"`
	User                         *pan.UserView            `json:"user,omitempty"`
	HiddenMode                   pan.HiddenModeStatusView `json:"hiddenMode"`
	HiddenModePasswordRemembered bool                     `json:"hiddenModePasswordRemembered"`
	Settings                     SettingsView             `json:"settings"`
	CurrentID                    string                   `json:"currentId"`
	ProxyBase                    string                   `json:"proxyBase"`
	Version                      string                   `json:"version"`
}

type SettingsView struct {
	PreferredPlayer      string                `json:"preferredPlayer"`
	Players              []player.Status       `json:"players"`
	ConfigPath           string                `json:"configPath"`
	ShowTitleBadges      bool                  `json:"showTitleBadges"`
	CleanTitleDisplay    bool                  `json:"cleanTitleDisplay"`
	UIScalePercent       int                   `json:"uiScalePercent"`
	ThemeMode            string                `json:"themeMode"`
	SmallFileFilterMB    int                   `json:"smallFileFilterMB"`
	FileListDensity      string                `json:"fileListDensity"`
	OfflineRecentTargets []DirectoryTargetView `json:"offlineRecentTargets"`
}

type DirectoryTargetView struct {
	ID   string           `json:"id"`
	Path []pan.Breadcrumb `json:"path"`
}

const (
	defaultWindowWidth  = 820
	defaultWindowHeight = 660
	minWindowWidth      = 800
	minWindowHeight     = 660
)

func NewApp(logger *slog.Logger) (*App, error) {
	store, err := config.NewStore("")
	if err != nil {
		return nil, err
	}

	panService := pan.NewService()
	proxyServer := proxy.NewServer(panService, logger)

	return &App{
		logger: logger,
		store:  store,
		pan:    panService,
		proxy:  proxyServer,
		state:  config.DefaultState(),
	}, nil
}

func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.startup(ctx)
	return nil
}

func (a *App) ServiceShutdown() error {
	a.shutdown(context.Background())
	return nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if err := a.loadState(); err != nil {
		a.logger.Error("failed to load state", "error", err)
	}

	if err := a.proxy.Start(); err != nil {
		a.logger.Error("failed to start media proxy", "error", err)
	} else {
		a.logger.Info("media proxy started", "base", a.proxy.BaseURL())
	}

	if credential, cookies := a.currentCredential(), a.currentCookies(); credential != nil || len(cookies) > 0 {
		ok, user, err := a.pan.Restore(credential, cookies)
		if err != nil {
			a.logger.Warn("failed to restore login state", "error", err)
		}
		if !ok {
			a.clearCredential()
		} else {
			a.mu.RLock()
			restoreHiddenMode := a.state.HiddenModeEnabled
			a.mu.RUnlock()
			a.pan.RestoreHiddenMode(restoreHiddenMode)
			a.syncSessionState()
			a.logger.Info("login restored", "user", user.UserName)
		}
	}

	a.started = true
}

func (a *App) shutdown(ctx context.Context) {
	shutdownCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	a.persistWindowState()

	if err := a.proxy.Close(shutdownCtx); err != nil {
		a.logger.Warn("failed to stop proxy", "error", err)
	}
	if err := a.store.Close(); err != nil {
		a.logger.Warn("failed to close state store", "error", err)
	}
}

func (a *App) Bootstrap() (*BootstrapResult, error) {
	if !a.started {
		return nil, errors.New("app not ready")
	}

	user, loggedIn := a.pan.CurrentUser()
	return &BootstrapResult{
		LoggedIn:                     loggedIn,
		User:                         user,
		HiddenMode:                   a.pan.HiddenModeStatus(),
		HiddenModePasswordRemembered: a.hiddenModePasswordRemembered(),
		Settings:                     a.settingsView(),
		CurrentID:                    a.currentDirectoryID(),
		ProxyBase:                    a.proxy.BaseURL(),
		Version:                      "v1",
	}, nil
}

func (a *App) StartQRCodeLogin() (*pan.LoginSessionView, error) {
	return a.pan.StartQRCodeLogin()
}

func (a *App) CheckQRCodeLogin(sessionID string) (*pan.LoginStatusView, error) {
	result, _, err := a.pan.CheckQRCodeLogin(sessionID)
	if err != nil {
		return nil, err
	}

	if result.LoggedIn {
		a.persistLoggedInState()
	}

	return result, nil
}

func (a *App) LoginWithCookie(cookie string) (*pan.LoginStatusView, error) {
	result, err := a.pan.LoginWithCookie(cookie)
	if err != nil {
		return nil, err
	}

	if result.LoggedIn {
		a.persistLoggedInState()
	}

	return result, nil
}

func (a *App) Logout() error {
	a.pan.Logout()

	a.mu.Lock()
	a.state.Credential = nil
	a.state.Cookies = nil
	a.state.HiddenModeEnabled = false
	previousSettings := cloneSettings(a.state.Settings)
	a.state = config.DefaultState()
	a.state.Settings = previousSettings
	state := cloneState(a.state)
	a.mu.Unlock()

	return a.store.Save(state)
}

func (a *App) SetHiddenMode(enabled bool, password string, rememberPassword bool) (*pan.HiddenModeStatusView, error) {
	if !a.started {
		return nil, errors.New("app not ready")
	}

	password = strings.TrimSpace(password)
	passwordMD5 := ""
	if enabled && password == "" {
		passwordMD5 = a.currentHiddenModePasswordMD5()
	}

	status, err := a.pan.SetHiddenMode(enabled, password, passwordMD5)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	if enabled && rememberPassword {
		if password != "" {
			a.state.HiddenModePasswordMD5 = pan.MD5HexForClient(password)
		}
	} else if !enabled || !rememberPassword {
		a.state.HiddenModePasswordMD5 = ""
	}
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		return nil, err
	}

	a.syncSessionState()
	return status, nil
}

func (a *App) ListDirectory(dirID string, offset int, limit int) (*pan.DirectoryView, error) {
	result, err := a.pan.ListDirectory(dirID, offset, limit)
	if err != nil {
		return nil, err
	}
	a.enrichPlaybackItems(result.Items)

	a.mu.Lock()
	a.state.LastDirectoryID = result.DirID
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		a.logger.Warn("failed to persist last directory", "error", err)
	}

	return result, nil
}

func (a *App) PreviewDirectory(dirID string) (*pan.DirectoryView, error) {
	result, err := a.pan.ListDirectory(dirID, 0, pan.DefaultPreviewLimit())
	if err != nil {
		return nil, err
	}
	a.enrichPlaybackItems(result.Items)
	return result, nil
}

func (a *App) SearchFiles(keyword string, offset int, limit int) (*pan.SearchResultView, error) {
	result, err := a.pan.SearchFiles(keyword, offset, limit)
	if err != nil {
		return nil, err
	}
	a.enrichPlaybackItems(result.Items)
	return result, nil
}

func (a *App) PlayFile(req PlayRequest) (*PlayResult, error) {
	req, record := a.normalizePlayRequest(req)
	if req.PickCode == "" {
		return nil, errors.New("缺少 pickcode，无法播放")
	}

	startMS := a.resolveStartPosition(req, record)
	a.preSavePlaybackStart(req, startMS)

	subtitlePath := a.prepareSubtitlePath(req.PickCode, a.selectedSubtitlePath(req.PickCode, req.Subtitle))
	streamURL := a.proxy.StreamURL(req.PickCode, req.Name)
	settings := a.currentSettings()
	var err error
	settings, err = playOnceWithPlayer(settings, req.PlayerID)
	if err != nil {
		return nil, err
	}
	logDir := filepath.Dir(a.store.Path())

	if err := a.proxy.Probe(req.PickCode, req.Name); err != nil {
		a.logger.Error("stream probe failed", "pickcode", req.PickCode, "name", req.Name, "error", err)
		return nil, err
	}

	launcher := player.NewLauncher(player.Settings{
		PreferredPlayer: settings.PreferredPlayer,
		PlayerPaths:     settings.PlayerPaths,
		DisabledPlayers: settings.DisabledPlayers,
		LogDir:          logDir,
	})

	result, err := launcher.Launch(player.Request{
		URL:              streamURL,
		Title:            req.Name,
		StartMS:          startMS,
		Subtitle:         subtitlePath,
		ManagedResumeDir: filepath.Join(logDir, "mpv-watch-later"),
	})
	if err != nil {
		return nil, err
	}
	a.rememberLaunchPreference(req.PickCode, req.Name, result.PlayerID, subtitlePath, startMS, result.SupportsManagedResume())
	a.trackManagedResume(result, req.PickCode, req.Name, subtitlePath, time.Now())
	return &PlayResult{
		PlayerID:      result.PlayerID,
		PlayerName:    result.PlayerName,
		Path:          result.Path,
		StartMS:       startMS,
		ResumeUsed:    !req.FromStart && req.StartMS <= 0 && startMS > 0,
		Subtitle:      subtitlePath,
		ManagedResume: result.SupportsManagedResume(),
	}, nil
}

func (a *App) PrepareBuiltinPlayback(req PlayRequest) (*BuiltinPlaybackSource, error) {
	req, record := a.normalizePlayRequest(req)
	if req.PickCode == "" {
		return nil, errors.New("缺少 pickcode，无法播放")
	}

	startMS := a.resolveStartPosition(req, record)
	a.preSavePlaybackStart(req, startMS)

	subtitlePath := a.prepareSubtitlePath(req.PickCode, a.selectedSubtitlePath(req.PickCode, req.Subtitle))
	streamURL := a.proxy.StreamURL(req.PickCode, req.Name)

	if err := a.proxy.Probe(req.PickCode, req.Name); err != nil {
		a.logger.Error("stream probe failed", "pickcode", req.PickCode, "name", req.Name, "error", err)
		return nil, err
	}

	result := &BuiltinPlaybackSource{
		URL:        streamURL,
		Title:      req.Name,
		StartMS:    startMS,
		ResumeUsed: !req.FromStart && req.StartMS <= 0 && startMS > 0,
	}

	if subtitlePath != "" {
		result.SubtitlePath = subtitlePath
		result.SubtitleName = filepath.Base(subtitlePath)
		subtitleURL, subtitleType, ok := a.proxy.SubtitleURL(subtitlePath)
		result.SubtitleURL = subtitleURL
		result.SubtitleType = subtitleType
		result.SubtitleUsable = ok
	}

	return result, nil
}

func (a *App) SavePlaybackProgress(pickCode, name string, positionMS int64) (*PlaybackStateView, error) {
	pickCode = strings.TrimSpace(pickCode)
	if pickCode == "" {
		return nil, errors.New("缺少 pickcode")
	}
	if positionMS < 0 {
		positionMS = 0
	}

	if err := a.upsertPlaybackRecord(pickCode, func(record *config.PlaybackRecord) bool {
		if strings.TrimSpace(name) != "" {
			record.FileName = strings.TrimSpace(name)
		}
		record.LastPositionMS = positionMS
		record.LastPlayedAt = time.Now().Format(time.RFC3339)
		return record.LastPositionMS > 0 || strings.TrimSpace(record.SubtitlePath) != ""
	}); err != nil {
		return nil, err
	}

	return a.playbackStateView(pickCode), nil
}

func (a *App) SelectPlayerPath(playerID string) (*SettingsView, error) {
	playerID = strings.TrimSpace(playerID)
	title := "选择播放器可执行文件"
	if playerID != "" {
		title = "选择 " + player.NameOf(playerID)
	}

	dialog := application.Get().Dialog.OpenFile().
		SetTitle(title).
		AddFilter("播放器程序", "*.exe;*.app").
		AddFilter("所有文件", "*")
	if a.window != nil {
		dialog.AttachToWindow(a.window)
	}

	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(path) == "" {
		view := a.settingsView()
		return &view, nil
	}

	return a.SavePlayerPath(playerID, path)
}

func (a *App) SavePlayerPath(playerID, path string) (*SettingsView, error) {
	playerID = strings.ToLower(strings.TrimSpace(playerID))
	if playerID == "" {
		playerID = player.DefaultPreferredPlayer()
	}

	cleaned := strings.TrimSpace(path)

	a.mu.Lock()
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	if cleaned == "" {
		delete(a.state.Settings.PlayerPaths, playerID)
	} else {
		a.state.Settings.PlayerPaths[playerID] = cleaned
	}
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		return nil, err
	}

	view := a.settingsView()
	return &view, nil
}

func (a *App) DeletePlayerPath(playerID string) (*SettingsView, error) {
	playerID = strings.ToLower(strings.TrimSpace(playerID))
	if playerID == "" || !player.IsKnown(playerID) {
		return nil, errors.New("不支持的播放器")
	}

	return a.SavePlayerPath(playerID, "")
}

func (a *App) SavePlayerDisabled(playerID string, disabled bool) (*SettingsView, error) {
	playerID = strings.ToLower(strings.TrimSpace(playerID))
	if playerID == "" || !player.IsKnown(playerID) {
		return nil, errors.New("不支持的播放器")
	}

	a.mu.Lock()
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	if disabled {
		a.state.Settings.DisabledPlayers[playerID] = true
		if a.state.Settings.PreferredPlayer == playerID {
			a.state.Settings.PreferredPlayer = player.DefaultPreferredPlayer()
		}
	} else {
		delete(a.state.Settings.DisabledPlayers, playerID)
	}
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		return nil, err
	}

	view := a.settingsView()
	return &view, nil
}

func (a *App) SavePreferredPlayer(playerID string) (*SettingsView, error) {
	playerID = strings.ToLower(strings.TrimSpace(playerID))
	if playerID != "" && !player.IsKnown(playerID) {
		return nil, errors.New("不支持的播放器")
	}

	a.mu.Lock()
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	if playerID == "" {
		playerID = player.DefaultPreferredPlayer()
	}
	a.state.Settings.PreferredPlayer = playerID
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		return nil, err
	}

	view := a.settingsView()
	return &view, nil
}

func (a *App) SaveShowTitleBadgesEnabled(enabled bool) (*SettingsView, error) {
	a.mu.Lock()
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	a.state.Settings.HideTitleBadges = !enabled
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		return nil, err
	}

	view := a.settingsView()
	return &view, nil
}

func (a *App) SaveCleanTitleDisplayEnabled(enabled bool) (*SettingsView, error) {
	a.mu.Lock()
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	a.state.Settings.DisableCleanTitleDisplay = !enabled
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		return nil, err
	}

	view := a.settingsView()
	return &view, nil
}

func (a *App) SaveUIScalePercent(value int) (*SettingsView, error) {
	a.mu.Lock()
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	a.state.Settings.UIScalePercent = normalizeUIScalePercent(value)
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		return nil, err
	}

	view := a.settingsView()
	return &view, nil
}

func (a *App) SaveThemeMode(value string) (*SettingsView, error) {
	a.mu.Lock()
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	a.state.Settings.ThemeMode = normalizeThemeMode(value)
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		return nil, err
	}

	view := a.settingsView()
	return &view, nil
}

func (a *App) SaveSmallFileFilterMB(value int) (*SettingsView, error) {
	a.mu.Lock()
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	a.state.Settings.SmallFileFilterMB = normalizeSmallFileFilterMB(value)
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		return nil, err
	}

	view := a.settingsView()
	return &view, nil
}

func (a *App) SaveFileListDensity(value string) (*SettingsView, error) {
	a.mu.Lock()
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	a.state.Settings.FileListDensity = normalizeFileListDensity(value)
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		return nil, err
	}

	view := a.settingsView()
	return &view, nil
}

func (a *App) SaveHideSmallFilesEnabled(enabled bool) (*SettingsView, error) {
	if enabled {
		return a.SaveSmallFileFilterMB(1)
	}
	return a.SaveSmallFileFilterMB(0)
}

func (a *App) RevealConfigPath() string {
	return a.store.Path()
}

func (a *App) loadState() error {
	state, err := a.store.Load()
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.state = state
	a.mu.Unlock()

	return nil
}

func (a *App) clearCredential() {
	a.mu.Lock()
	a.state.Credential = nil
	a.state.Cookies = nil
	a.state.HiddenModeEnabled = false
	a.state.HiddenModePasswordMD5 = ""
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		a.logger.Warn("failed to clear invalid credential", "error", err)
	}
}

func (a *App) currentCredential() *config.Credential {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.state.Credential == nil {
		return nil
	}

	credential := *a.state.Credential
	return &credential
}

func (a *App) currentDirectoryID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.state.LastDirectoryID == "" {
		return "0"
	}
	return a.state.LastDirectoryID
}

func (a *App) currentCookies() map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]string, len(a.state.Cookies))
	for key, value := range a.state.Cookies {
		result[key] = value
	}
	return result
}

func (a *App) currentSettings() config.Settings {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return normalizeAppSettings(a.state.Settings)
}

func (a *App) settingsView() SettingsView {
	settings := a.currentSettings()
	return SettingsView{
		PreferredPlayer: settings.PreferredPlayer,
		Players: player.Statuses(player.Settings{
			PreferredPlayer: settings.PreferredPlayer,
			PlayerPaths:     settings.PlayerPaths,
			DisabledPlayers: settings.DisabledPlayers,
		}),
		ConfigPath:           a.store.Path(),
		ShowTitleBadges:      !settings.HideTitleBadges,
		CleanTitleDisplay:    !settings.DisableCleanTitleDisplay,
		UIScalePercent:       settings.UIScalePercent,
		ThemeMode:            settings.ThemeMode,
		SmallFileFilterMB:    settings.SmallFileFilterMB,
		FileListDensity:      settings.FileListDensity,
		OfflineRecentTargets: buildDirectoryTargetViews(settings.OfflineRecentTargets),
	}
}

func (a *App) notifyError(title string, err error) {
	if err == nil {
		return
	}
	a.logger.Error(title, "error", err)
	if a.window != nil {
		a.window.Error("%s: %v", title, err)
	}
}

func (a *App) syncSessionState() {
	credential, cookies := a.pan.SessionSnapshot()

	a.mu.Lock()
	a.state.Credential = credential
	a.state.Cookies = cookies
	a.state.HiddenModeEnabled = a.pan.HiddenModeStatus().Enabled
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		a.logger.Warn("failed to persist session state", "error", err)
	}
}

func (a *App) currentHiddenModePasswordMD5() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return strings.TrimSpace(a.state.HiddenModePasswordMD5)
}

func (a *App) hiddenModePasswordRemembered() bool {
	return a.currentHiddenModePasswordMD5() != ""
}

func (a *App) persistLoggedInState() {
	a.mu.Lock()
	if a.state.LastDirectoryID == "" {
		a.state.LastDirectoryID = "0"
	}
	a.mu.Unlock()

	a.syncSessionState()
}

func (a *App) applySavedWindowState() {
	if a.window == nil {
		return
	}

	a.mu.RLock()
	saved := a.state.Window
	a.mu.RUnlock()

	width := saved.Width
	height := saved.Height
	if width < minWindowWidth {
		width = defaultWindowWidth
	}
	if height < minWindowHeight {
		height = defaultWindowHeight
	}

	a.window.SetSize(width, height)
	a.window.SetMinSize(minWindowWidth, minWindowHeight)
	if saved.Maximised {
		a.window.Maximise()
	}
}

func (a *App) persistWindowState() {
	if a.window == nil {
		return
	}

	a.mu.Lock()
	windowState := a.state.Window
	windowState.Maximised = a.window.IsMaximised()
	if !windowState.Maximised {
		width, height := a.window.Size()
		if width >= minWindowWidth {
			windowState.Width = width
		}
		if height >= minWindowHeight {
			windowState.Height = height
		}
	}
	a.state.Window = windowState
	state := cloneState(a.state)
	a.mu.Unlock()

	if err := a.store.Save(state); err != nil {
		a.logger.Warn("failed to persist window state", "error", err)
	}
}

func normalizeAppSettings(settings config.Settings) config.Settings {
	if settings.PlayerPaths == nil {
		settings.PlayerPaths = map[string]string{}
	}
	if settings.DisabledPlayers == nil {
		settings.DisabledPlayers = map[string]bool{}
	}
	if settings.OfflineRecentTargets == nil {
		settings.OfflineRecentTargets = []config.DirectoryTarget{}
	}
	if settings.MPVPath != "" && settings.PlayerPaths[player.PlayerMPV] == "" {
		settings.PlayerPaths[player.PlayerMPV] = settings.MPVPath
	}
	settings.MPVPath = ""
	if settings.PreferredPlayer == "" {
		settings.PreferredPlayer = player.DefaultPreferredPlayer()
	}
	if settings.SmallFileFilterMB == 0 && settings.HideSmallFiles {
		settings.SmallFileFilterMB = 1
	}
	settings.SmallFileFilterMB = normalizeSmallFileFilterMB(settings.SmallFileFilterMB)
	settings.FileListDensity = normalizeFileListDensity(settings.FileListDensity)
	settings.UIScalePercent = normalizeUIScalePercent(settings.UIScalePercent)
	settings.ThemeMode = normalizeThemeMode(settings.ThemeMode)
	settings.OfflineRecentTargets = config.NormalizeOfflineRecentTargets(settings.OfflineRecentTargets)
	settings.HideSmallFiles = false
	if settings.DisabledPlayers[settings.PreferredPlayer] {
		for _, candidate := range []string{
			player.PlayerMPV,
			player.PlayerVLC,
			player.PlayerPotPlayer,
			player.PlayerMPCHC,
			player.PlayerMPCBE,
		} {
			if !settings.DisabledPlayers[candidate] {
				settings.PreferredPlayer = candidate
				break
			}
		}
	}
	return settings
}

func (a *App) resolveStartPosition(req PlayRequest, record config.PlaybackRecord) int64 {
	if req.FromStart {
		return 0
	}
	if req.StartMS > 0 {
		return req.StartMS
	}
	return record.LastPositionMS
}

func (a *App) preSavePlaybackStart(req PlayRequest, startMS int64) {
	if !req.FromStart && req.StartMS <= 0 {
		return
	}

	if err := a.upsertPlaybackRecord(req.PickCode, func(playback *config.PlaybackRecord) bool {
		if strings.TrimSpace(req.Name) != "" {
			playback.FileName = req.Name
		}
		if req.FromStart {
			playback.LastPositionMS = 0
			return strings.TrimSpace(playback.SubtitlePath) != ""
		}
		playback.LastPositionMS = startMS
		playback.LastPlayedAt = time.Now().Format(time.RFC3339)
		return playback.LastPositionMS > 0 || strings.TrimSpace(playback.SubtitlePath) != ""
	}); err != nil {
		a.logger.Warn("failed to pre-save playback state", "pickcode", req.PickCode, "error", err)
	}
}

func buildDirectoryTargetViews(targets []config.DirectoryTarget) []DirectoryTargetView {
	if len(targets) == 0 {
		return []DirectoryTargetView{}
	}

	result := make([]DirectoryTargetView, 0, len(targets))
	for _, target := range targets {
		path := make([]pan.Breadcrumb, 0, len(target.Path))
		for _, crumb := range target.Path {
			path = append(path, pan.Breadcrumb{
				ID:   crumb.ID,
				Name: crumb.Name,
			})
		}
		result = append(result, DirectoryTargetView{
			ID:   target.ID,
			Path: path,
		})
	}
	return result
}

func directoryTargetViewsToConfig(targets []DirectoryTargetView) []config.DirectoryTarget {
	if len(targets) == 0 {
		return []config.DirectoryTarget{}
	}

	result := make([]config.DirectoryTarget, 0, len(targets))
	for _, target := range targets {
		path := make([]config.Breadcrumb, 0, len(target.Path))
		for _, crumb := range target.Path {
			path = append(path, config.Breadcrumb{
				ID:   crumb.ID,
				Name: crumb.Name,
			})
		}
		result = append(result, config.DirectoryTarget{
			ID:   target.ID,
			Path: path,
		})
	}
	return result
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

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sevenplayer/internal/config"
	"sevenplayer/internal/pan"
	"sevenplayer/internal/player"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type PlayRequest struct {
	PickCode  string `json:"pickCode"`
	Name      string `json:"name"`
	StartMS   int64  `json:"startMs"`
	FromStart bool   `json:"fromStart"`
	Subtitle  string `json:"subtitle,omitempty"`
	PlayerID  string `json:"playerId,omitempty"`
}

type PlayResult struct {
	PlayerID      string `json:"playerId"`
	PlayerName    string `json:"playerName"`
	Path          string `json:"path"`
	StartMS       int64  `json:"startMs"`
	ResumeUsed    bool   `json:"resumeUsed"`
	Subtitle      string `json:"subtitle,omitempty"`
	ManagedResume bool   `json:"managedResume"`
}

type BuiltinPlaybackSource struct {
	URL            string `json:"url"`
	Title          string `json:"title"`
	StartMS        int64  `json:"startMs"`
	ResumeUsed     bool   `json:"resumeUsed"`
	SubtitleURL    string `json:"subtitleUrl,omitempty"`
	SubtitleName   string `json:"subtitleName,omitempty"`
	SubtitlePath   string `json:"subtitlePath,omitempty"`
	SubtitleType   string `json:"subtitleType,omitempty"`
	SubtitleUsable bool   `json:"subtitleUsable"`
}

type PlaybackStateView struct {
	PickCode     string `json:"pickCode"`
	ResumeMS     int64  `json:"resumeMs"`
	ResumeText   string `json:"resumeText,omitempty"`
	SubtitlePath string `json:"subtitlePath,omitempty"`
	SubtitleName string `json:"subtitleName,omitempty"`
	LastPlayedAt string `json:"lastPlayedAt,omitempty"`
}

func (a *App) SelectSubtitlePath(pickCode string) (*PlaybackStateView, error) {
	pickCode = strings.TrimSpace(pickCode)
	if pickCode == "" {
		return nil, errors.New("缺少 pickcode")
	}

	dialog := application.Get().Dialog.OpenFile().
		SetTitle("选择外挂字幕").
		AddFilter("字幕文件", "*.srt;*.ass;*.ssa;*.vtt;*.sub")
	if a.window != nil {
		dialog.AttachToWindow(a.window)
	}

	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(path) == "" {
		return a.playbackStateView(pickCode), nil
	}

	if err := a.upsertPlaybackRecord(pickCode, func(record *config.PlaybackRecord) bool {
		record.SubtitlePath = strings.TrimSpace(path)
		return record.LastPositionMS > 0 || strings.TrimSpace(record.SubtitlePath) != ""
	}); err != nil {
		return nil, err
	}

	return a.playbackStateView(pickCode), nil
}

func (a *App) ClearSubtitlePath(pickCode string) (*PlaybackStateView, error) {
	pickCode = strings.TrimSpace(pickCode)
	if pickCode == "" {
		return nil, errors.New("缺少 pickcode")
	}

	if err := a.upsertPlaybackRecord(pickCode, func(record *config.PlaybackRecord) bool {
		record.SubtitlePath = ""
		return record.LastPositionMS > 0
	}); err != nil {
		return nil, err
	}

	return a.playbackStateView(pickCode), nil
}

func (a *App) ClearPlaybackProgress(pickCode string) (*PlaybackStateView, error) {
	pickCode = strings.TrimSpace(pickCode)
	if pickCode == "" {
		return nil, errors.New("缺少 pickcode")
	}

	if err := a.upsertPlaybackRecord(pickCode, func(record *config.PlaybackRecord) bool {
		record.LastPositionMS = 0
		record.LastPlayedAt = time.Now().Format(time.RFC3339)
		return strings.TrimSpace(record.SubtitlePath) != ""
	}); err != nil {
		return nil, err
	}

	return a.playbackStateView(pickCode), nil
}

func (a *App) enrichPlaybackItems(items []pan.FileItem) {
	records := a.playbackRecordsSnapshot()
	for i := range items {
		record, ok := records[items[i].PickCode]
		if !ok {
			continue
		}
		items[i].ResumeMS = record.LastPositionMS
		items[i].SubtitlePath = record.SubtitlePath
		items[i].LastPlayedAt = record.LastPlayedAt
	}
}

func (a *App) playbackStateView(pickCode string) *PlaybackStateView {
	record, ok := a.playbackRecord(pickCode)
	if !ok {
		return &PlaybackStateView{PickCode: strings.TrimSpace(pickCode)}
	}
	return buildPlaybackStateView(record)
}

func (a *App) playbackRecord(pickCode string) (config.PlaybackRecord, bool) {
	pickCode = strings.TrimSpace(pickCode)
	if pickCode == "" {
		return config.PlaybackRecord{}, false
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	record, ok := a.state.PlaybackRecords[pickCode]
	return record, ok
}

func (a *App) playbackRecordsSnapshot() map[string]config.PlaybackRecord {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]config.PlaybackRecord, len(a.state.PlaybackRecords))
	for key, value := range a.state.PlaybackRecords {
		result[key] = value
	}
	return result
}

func (a *App) upsertPlaybackRecord(pickCode string, mutate func(record *config.PlaybackRecord) bool) error {
	pickCode = strings.TrimSpace(pickCode)
	if pickCode == "" {
		return errors.New("缺少 pickcode")
	}

	a.mu.Lock()
	a.state.PlaybackRecords = ensurePlaybackRecordMap(a.state.PlaybackRecords)
	record := a.state.PlaybackRecords[pickCode]
	if record.PickCode == "" {
		record.PickCode = pickCode
	}
	keep := mutate(&record)
	if keep {
		a.state.PlaybackRecords[pickCode] = record
	} else {
		delete(a.state.PlaybackRecords, pickCode)
	}
	state := cloneState(a.state)
	a.mu.Unlock()

	return a.store.Save(state)
}

func (a *App) normalizePlayRequest(req PlayRequest) (PlayRequest, config.PlaybackRecord) {
	req.PickCode = strings.TrimSpace(req.PickCode)
	req.Name = strings.TrimSpace(req.Name)
	req.Subtitle = strings.TrimSpace(req.Subtitle)
	req.PlayerID = strings.ToLower(strings.TrimSpace(req.PlayerID))

	record, _ := a.playbackRecord(req.PickCode)
	if req.FromStart {
		record.LastPositionMS = 0
	}
	return req, record
}

func (a *App) selectedSubtitlePath(pickCode string, explicit string) string {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		return explicit
	}

	record, ok := a.playbackRecord(pickCode)
	if !ok {
		return ""
	}
	return strings.TrimSpace(record.SubtitlePath)
}

func (a *App) prepareSubtitlePath(pickCode, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}

	a.logger.Warn("subtitle file missing, clearing saved path", "pickcode", pickCode, "path", path)
	if err := a.upsertPlaybackRecord(pickCode, func(record *config.PlaybackRecord) bool {
		record.SubtitlePath = ""
		return record.LastPositionMS > 0
	}); err != nil {
		a.logger.Warn("failed to clear missing subtitle path", "pickcode", pickCode, "error", err)
	}
	return ""
}

func (a *App) rememberLaunchPreference(pickCode, name, playerID, subtitlePath string, startMS int64, persistProgress bool) {
	if err := a.upsertPlaybackRecord(pickCode, func(record *config.PlaybackRecord) bool {
		if strings.TrimSpace(name) != "" {
			record.FileName = name
		}
		if strings.TrimSpace(playerID) != "" {
			record.LastPlayerID = playerID
		}
		if subtitlePath != "" {
			record.SubtitlePath = subtitlePath
		}
		if startMS > 0 && !persistProgress {
			record.LastPositionMS = startMS
			record.LastPlayedAt = time.Now().Format(time.RFC3339)
		}
		return record.LastPositionMS > 0 || strings.TrimSpace(record.SubtitlePath) != ""
	}); err != nil {
		a.logger.Warn("failed to persist playback preferences", "pickcode", pickCode, "error", err)
	}
}

func (a *App) trackManagedResume(result *player.LaunchResult, pickCode, name, subtitlePath string, launchedAt time.Time) {
	if result == nil || !result.SupportsManagedResume() || result.Done() == nil {
		return
	}

	watchLaterDir := filepath.Join(filepath.Dir(a.store.Path()), "mpv-watch-later")

	go func() {
		err, ok := <-result.Done()
		if ok && err != nil {
			a.logger.Warn("managed player exited with error", "player", result.PlayerID, "pickcode", pickCode, "error", err)
		}

		state, findErr := player.FindMPVWatchLaterState(watchLaterDir, "pickcode="+pickCode, launchedAt)
		if findErr != nil {
			a.logger.Warn("failed to read mpv watch-later state", "pickcode", pickCode, "error", findErr)
			return
		}
		if state == nil {
			return
		}

		if saveErr := a.upsertPlaybackRecord(pickCode, func(record *config.PlaybackRecord) bool {
			record.FileName = chooseName(record.FileName, name)
			record.LastPlayerID = player.PlayerMPV
			record.LastPositionMS = state.StartMS
			record.LastPlayedAt = time.Now().Format(time.RFC3339)
			if subtitlePath != "" {
				record.SubtitlePath = subtitlePath
			}
			return record.LastPositionMS > 0 || strings.TrimSpace(record.SubtitlePath) != ""
		}); saveErr != nil {
			a.logger.Warn("failed to persist managed resume state", "pickcode", pickCode, "error", saveErr)
		}
	}()
}

func buildPlaybackStateView(record config.PlaybackRecord) *PlaybackStateView {
	view := &PlaybackStateView{
		PickCode:     record.PickCode,
		ResumeMS:     record.LastPositionMS,
		SubtitlePath: record.SubtitlePath,
		LastPlayedAt: record.LastPlayedAt,
	}
	if record.LastPositionMS > 0 {
		view.ResumeText = formatDurationMS(record.LastPositionMS)
	}
	if strings.TrimSpace(record.SubtitlePath) != "" {
		view.SubtitleName = filepath.Base(record.SubtitlePath)
	}
	return view
}

func ensurePlaybackRecordMap(records map[string]config.PlaybackRecord) map[string]config.PlaybackRecord {
	if records == nil {
		return map[string]config.PlaybackRecord{}
	}
	return records
}

func formatDurationMS(ms int64) string {
	totalSeconds := ms / 1000
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return padTwo(hours) + ":" + padTwo(minutes) + ":" + padTwo(seconds)
	}
	return padTwo(minutes) + ":" + padTwo(seconds)
}

func padTwo(value int64) string {
	if value < 10 {
		return "0" + strconvI64(value)
	}
	return strconvI64(value)
}

func strconvI64(value int64) string {
	return strconv.FormatInt(value, 10)
}

func chooseName(existing, fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	return strings.TrimSpace(existing)
}

func playOnceWithPlayer(settings config.Settings, playerID string) (config.Settings, error) {
	playerID = strings.ToLower(strings.TrimSpace(playerID))
	if playerID == "" {
		return settings, nil
	}
	if !player.IsKnown(playerID) {
		return settings, errors.New("不支持的播放器")
	}
	if settings.DisabledPlayers[playerID] {
		return settings, errors.New(player.NameOf(playerID) + " 已禁用")
	}

	disabledPlayers := make(map[string]bool, len(settings.DisabledPlayers)+5)
	for id, disabled := range settings.DisabledPlayers {
		disabledPlayers[id] = disabled
	}
	for _, id := range []string{
		player.PlayerMPV,
		player.PlayerVLC,
		player.PlayerPotPlayer,
		player.PlayerMPCHC,
		player.PlayerMPCBE,
	} {
		disabledPlayers[id] = id != playerID
	}

	settings.PreferredPlayer = playerID
	settings.DisabledPlayers = disabledPlayers
	return settings, nil
}

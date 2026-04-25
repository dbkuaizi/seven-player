package main

import (
	"panplayer/internal/config"
	"panplayer/internal/pan"
)

type AddOfflineRequest struct {
	URLs        []string         `json:"urls"`
	SaveDirID   string           `json:"saveDirId"`
	SaveDirPath []pan.Breadcrumb `json:"saveDirPath,omitempty"`
}

type DeleteOfflineRequest struct {
	Hashes      []string `json:"hashes"`
	DeleteFiles bool     `json:"deleteFiles"`
}

func (a *App) ListOfflineTasks() (*pan.OfflineListView, error) {
	return a.pan.ListOfflineTasks()
}

func (a *App) AddOfflineTasks(req AddOfflineRequest) (*pan.OfflineListView, error) {
	result, err := a.pan.AddOfflineTasks(req.URLs, req.SaveDirID)
	if err != nil {
		return nil, err
	}

	if saveErr := a.rememberOfflineTarget(req.SaveDirID, req.SaveDirPath); saveErr != nil {
		a.logger.Warn("failed to persist offline target history", "dirId", req.SaveDirID, "error", saveErr)
	}

	return result, nil
}

func (a *App) DeleteOfflineTasks(req DeleteOfflineRequest) (*pan.OfflineListView, error) {
	return a.pan.DeleteOfflineTasks(req.Hashes, req.DeleteFiles)
}

func (a *App) rememberOfflineTarget(dirID string, path []pan.Breadcrumb) error {
	targetPath := make([]config.Breadcrumb, 0, len(path))
	for _, crumb := range path {
		targetPath = append(targetPath, config.Breadcrumb{
			ID:   crumb.ID,
			Name: crumb.Name,
		})
	}

	target, ok := config.NormalizeDirectoryTarget(config.DirectoryTarget{
		ID:   dirID,
		Path: targetPath,
	})
	if !ok {
		return nil
	}

	a.mu.Lock()
	a.state.Settings = normalizeAppSettings(a.state.Settings)
	a.state.Settings.OfflineRecentTargets = config.NormalizeOfflineRecentTargets(
		append([]config.DirectoryTarget{target}, a.state.Settings.OfflineRecentTargets...),
	)
	state := cloneState(a.state)
	a.mu.Unlock()

	return a.store.Save(state)
}

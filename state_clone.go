package main

import "panplayer/internal/config"

func cloneState(state config.State) config.State {
	cloned := state
	cloned.Settings = cloneSettings(state.Settings)
	cloned.Cookies = cloneStringMap(state.Cookies)
	cloned.PlaybackRecords = clonePlaybackRecords(state.PlaybackRecords)
	cloned.HiddenModePasswordMD5 = state.HiddenModePasswordMD5
	if state.Credential != nil {
		credential := *state.Credential
		cloned.Credential = &credential
	}
	return cloned
}

func cloneSettings(settings config.Settings) config.Settings {
	cloned := settings
	cloned.PlayerPaths = cloneStringMap(settings.PlayerPaths)
	cloned.DisabledPlayers = cloneBoolMap(settings.DisabledPlayers)
	cloned.OfflineRecentTargets = cloneDirectoryTargets(settings.OfflineRecentTargets)
	return cloned
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}

	target := make(map[string]string, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

func cloneStringSlice(source []string) []string {
	if len(source) == 0 {
		return []string{}
	}

	target := make([]string, len(source))
	copy(target, source)
	return target
}

func clonePlaybackRecords(source map[string]config.PlaybackRecord) map[string]config.PlaybackRecord {
	if len(source) == 0 {
		return map[string]config.PlaybackRecord{}
	}

	target := make(map[string]config.PlaybackRecord, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	if len(source) == 0 {
		return map[string]bool{}
	}

	target := make(map[string]bool, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

func cloneDirectoryTargets(source []config.DirectoryTarget) []config.DirectoryTarget {
	if len(source) == 0 {
		return []config.DirectoryTarget{}
	}

	target := make([]config.DirectoryTarget, 0, len(source))
	for _, item := range source {
		clonedPath := make([]config.Breadcrumb, len(item.Path))
		copy(clonedPath, item.Path)
		target = append(target, config.DirectoryTarget{
			ID:   item.ID,
			Path: clonedPath,
		})
	}
	return target
}

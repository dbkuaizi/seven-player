package player

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type WatchLaterState struct {
	StartMS    int64
	ConfigPath string
	UpdatedAt  time.Time
}

func FindMPVWatchLaterState(dir, needle string, updatedAfter time.Time) (*WatchLaterState, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	needle = strings.TrimSpace(needle)
	var best *WatchLaterState

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if !updatedAfter.IsZero() && info.ModTime().Before(updatedAfter.Add(-2*time.Second)) {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		text := string(data)
		if needle != "" && !strings.Contains(text, needle) {
			continue
		}

		startMS, found := parseMPVWatchLaterStart(text)
		if !found {
			continue
		}

		if best == nil || info.ModTime().After(best.UpdatedAt) {
			best = &WatchLaterState{
				StartMS:    startMS,
				ConfigPath: path,
				UpdatedAt:  info.ModTime(),
			}
		}
	}

	return best, nil
}

func parseMPVWatchLaterStart(text string) (int64, bool) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "start=") {
			continue
		}

		value := strings.TrimSpace(strings.TrimPrefix(line, "start="))
		seconds, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, false
		}
		return int64(seconds * 1000), true
	}
	return 0, false
}

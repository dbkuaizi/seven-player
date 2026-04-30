package player

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	PlayerMPV       = "mpv"
	PlayerVLC       = "vlc"
	PlayerPotPlayer = "potplayer"
	PlayerMPCHC     = "mpc-hc"
	PlayerMPCBE     = "mpc-be"
)

type Settings struct {
	PreferredPlayer string
	PlayerPaths     map[string]string
	DisabledPlayers map[string]bool
	LogDir          string
}

type Request struct {
	URL              string
	Title            string
	StartMS          int64
	Subtitle         string
	ManagedResumeDir string
}

type Status struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Supported             bool   `json:"supported"`
	Available             bool   `json:"available"`
	Disabled              bool   `json:"disabled"`
	Selected              bool   `json:"selected"`
	Path                  string `json:"path,omitempty"`
	Source                string `json:"source,omitempty"`
	SupportsStartPosition bool   `json:"supportsStartPosition"`
	SupportsSubtitle      bool   `json:"supportsSubtitle"`
	SupportsManagedResume bool   `json:"supportsManagedResume"`
}

type LaunchResult struct {
	PlayerID              string `json:"playerId"`
	PlayerName            string `json:"playerName"`
	Path                  string `json:"path"`
	done                  <-chan error
	supportsManagedResume bool
}

type Launcher struct {
	settings Settings
}

type adapter struct {
	id                    string
	name                  string
	oses                  []string
	binaries              []string
	commonPaths           func() []string
	buildArgs             func(req Request, executablePath string, logPath string) []string
	supportsStartPosition bool
	supportsSubtitle      bool
	supportsManagedResume bool
}

var registry = []*adapter{
	{
		id:       PlayerMPV,
		name:     "mpv",
		oses:     []string{"windows", "linux", "darwin"},
		binaries: []string{"mpv.exe", "mpv"},
		commonPaths: func() []string {
			return windowsPaths(
				`mpv\mpv.exe`,
				`mpv.net\mpvnet.exe`,
			)
		},
		buildArgs:             buildMPVArgs,
		supportsStartPosition: true,
		supportsSubtitle:      true,
		supportsManagedResume: true,
	},
	{
		id:       PlayerVLC,
		name:     "VLC",
		oses:     []string{"windows", "linux", "darwin"},
		binaries: []string{"vlc.exe", "vlc"},
		commonPaths: func() []string {
			return windowsPaths(
				`VideoLAN\VLC\vlc.exe`,
			)
		},
		buildArgs:             buildVLCArgs,
		supportsStartPosition: true,
		supportsSubtitle:      true,
	},
	{
		id:       PlayerPotPlayer,
		name:     "PotPlayer",
		oses:     []string{"windows"},
		binaries: []string{"PotPlayerMini64.exe", "PotPlayerMini.exe"},
		commonPaths: func() []string {
			return windowsPaths(
				`DAUM\PotPlayer\PotPlayerMini64.exe`,
				`DAUM\PotPlayer\PotPlayerMini.exe`,
			)
		},
		buildArgs:             buildPotPlayerArgs,
		supportsStartPosition: true,
		supportsSubtitle:      true,
	},
	{
		id:       PlayerMPCHC,
		name:     "MPC-HC",
		oses:     []string{"windows"},
		binaries: []string{"mpc-hc64.exe", "mpc-hc.exe"},
		commonPaths: func() []string {
			return windowsPaths(
				`MPC-HC\mpc-hc64.exe`,
				`MPC-HC\mpc-hc.exe`,
			)
		},
		buildArgs:        buildMPCArgs,
		supportsSubtitle: true,
	},
	{
		id:       PlayerMPCBE,
		name:     "MPC-BE",
		oses:     []string{"windows"},
		binaries: []string{"mpc-be64.exe", "mpc-be.exe"},
		commonPaths: func() []string {
			return windowsPaths(
				`MPC-BE x64\mpc-be64.exe`,
				`MPC-BE\mpc-be64.exe`,
				`MPC-BE\mpc-be.exe`,
			)
		},
		buildArgs:        buildMPCArgs,
		supportsSubtitle: true,
	},
}

func NewLauncher(settings Settings) *Launcher {
	return &Launcher{settings: normalizeSettings(settings)}
}

func DefaultPreferredPlayer() string {
	return PlayerMPV
}

func NameOf(id string) string {
	if spec := findAdapter(id); spec != nil {
		return spec.name
	}
	return normalizePlayerID(id)
}

func IsKnown(id string) bool {
	return findAdapter(id) != nil
}

func Statuses(settings Settings) []Status {
	settings = normalizeSettings(settings)
	result := make([]Status, 0, len(registry))

	for _, spec := range registry {
		status := Status{
			ID:                    spec.id,
			Name:                  spec.name,
			Supported:             spec.supports(runtime.GOOS),
			Selected:              spec.id == settings.PreferredPlayer,
			Disabled:              settings.DisabledPlayers[spec.id],
			SupportsStartPosition: spec.supportsStartPosition,
			SupportsSubtitle:      spec.supportsSubtitle,
			SupportsManagedResume: spec.supportsManagedResume,
		}

		if status.Supported {
			path, source, err := spec.resolve(settings.PlayerPaths[spec.id])
			if err == nil {
				status.Available = true
				status.Path = path
				status.Source = source
			}
		}

		result = append(result, status)
	}

	return result
}

func (l *Launcher) Launch(req Request) (*LaunchResult, error) {
	if strings.TrimSpace(req.URL) == "" {
		return nil, errors.New("播放地址为空")
	}

	settings := normalizeSettings(l.settings)
	candidates := orderedCandidatesWithSettings(settings)
	var lastResolveErr error

	for _, spec := range candidates {
		path, _, err := spec.resolve(settings.PlayerPaths[spec.id])
		if err != nil {
			lastResolveErr = err
			continue
		}

		if capabilityErr := spec.validateRequest(req); capabilityErr != nil {
			if spec.id == settings.PreferredPlayer {
				return nil, capabilityErr
			}
			lastResolveErr = capabilityErr
			continue
		}

		logPath := ""
		if strings.TrimSpace(settings.LogDir) != "" && spec.id == PlayerMPV {
			logPath = filepath.Join(settings.LogDir, spec.id+".log")
		}

		args := spec.buildArgs(req, path, logPath)
		cmd := exec.Command(path, args...)
		hideConsoleWindow(cmd)
		if err := cmd.Start(); err != nil {
			lastResolveErr = fmt.Errorf("%s 启动失败: %w", spec.name, err)
			continue
		}

		waitCh := make(chan error, 1)
		go func() {
			waitCh <- cmd.Wait()
		}()

		select {
		case err := <-waitCh:
			if err != nil {
				if logPath != "" {
					return nil, fmt.Errorf("%s 启动后立即退出，请查看日志: %s", spec.name, logPath)
				}
				return nil, fmt.Errorf("%s 启动后立即退出: %w", spec.name, err)
			}
			if logPath != "" {
				return nil, fmt.Errorf("%s 启动后立即退出，请查看日志: %s", spec.name, logPath)
			}
			return nil, fmt.Errorf("%s 启动后立即退出", spec.name)
		case <-time.After(1500 * time.Millisecond):
			return &LaunchResult{
				PlayerID:              spec.id,
				PlayerName:            spec.name,
				Path:                  path,
				done:                  waitCh,
				supportsManagedResume: spec.supportsManagedResume,
			}, nil
		}
	}

	if lastResolveErr != nil {
		return nil, lastResolveErr
	}

	return nil, errors.New("没有可用的播放器，请先配置播放器路径")
}

func orderedCandidates(preferred string) []*adapter {
	return orderedCandidatesWithSettings(Settings{PreferredPlayer: preferred})
}

func orderedCandidatesWithSettings(settings Settings) []*adapter {
	settings = normalizeSettings(settings)
	preferred := normalizePlayerID(settings.PreferredPlayer)
	items := make([]*adapter, 0, len(registry))

	if spec := findAdapter(preferred); spec != nil && spec.supports(runtime.GOOS) && !settings.DisabledPlayers[spec.id] {
		items = append(items, spec)
	}

	for _, spec := range registry {
		if spec.id == preferred || !spec.supports(runtime.GOOS) || settings.DisabledPlayers[spec.id] {
			continue
		}
		items = append(items, spec)
	}

	return items
}

func normalizeSettings(settings Settings) Settings {
	settings.PreferredPlayer = normalizePlayerID(settings.PreferredPlayer)
	if settings.PreferredPlayer == "" {
		settings.PreferredPlayer = DefaultPreferredPlayer()
	}
	if settings.PlayerPaths == nil {
		settings.PlayerPaths = map[string]string{}
	}
	if settings.DisabledPlayers == nil {
		settings.DisabledPlayers = map[string]bool{}
	}
	if settings.DisabledPlayers[settings.PreferredPlayer] {
		for _, spec := range registry {
			if spec.supports(runtime.GOOS) && !settings.DisabledPlayers[spec.id] {
				settings.PreferredPlayer = spec.id
				break
			}
		}
	}
	return settings
}

func IsDisabled(settings Settings, id string) bool {
	settings = normalizeSettings(settings)
	return settings.DisabledPlayers[normalizePlayerID(id)]
}

func normalizePlayerID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	switch id {
	case "", "default":
		return ""
	case "mpv":
		return PlayerMPV
	case "vlc":
		return PlayerVLC
	case "potplayer":
		return PlayerPotPlayer
	case "mpc-hc", "mpchc":
		return PlayerMPCHC
	case "mpc-be", "mpcbe":
		return PlayerMPCBE
	default:
		return id
	}
}

func findAdapter(id string) *adapter {
	id = normalizePlayerID(id)
	for _, spec := range registry {
		if spec.id == id {
			return spec
		}
	}
	return nil
}

func (a *adapter) supports(goos string) bool {
	return slices.Contains(a.oses, goos)
}

func (a *adapter) validateRequest(req Request) error {
	if req.StartMS > 0 && !a.supportsStartPosition {
		return fmt.Errorf("%s 不支持从指定时间启动，请改用 mpv、VLC 或 PotPlayer", a.name)
	}
	if strings.TrimSpace(req.Subtitle) != "" && !a.supportsSubtitle {
		return fmt.Errorf("%s 不支持外挂字幕参数，请切换到支持字幕文件的播放器", a.name)
	}
	return nil
}

func (a *adapter) resolve(customPath string) (string, string, error) {
	if strings.TrimSpace(customPath) != "" {
		resolved, err := filepath.Abs(customPath)
		if err != nil {
			return "", "", err
		}
		if _, err := os.Stat(resolved); err != nil {
			return "", "", fmt.Errorf("%s 路径不存在: %s", a.name, resolved)
		}
		return resolved, "custom", nil
	}

	for _, candidate := range a.commonPaths() {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, "well-known", nil
		}
	}

	for _, binary := range a.binaries {
		if path, err := exec.LookPath(binary); err == nil {
			return path, "path", nil
		}
	}

	return "", "", fmt.Errorf("未找到 %s，请安装它或手动指定可执行文件路径", a.name)
}

func windowsPaths(relatives ...string) []string {
	if runtime.GOOS != "windows" {
		return nil
	}

	baseDirs := []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
	}

	paths := make([]string, 0, len(baseDirs)*len(relatives))
	for _, base := range baseDirs {
		if strings.TrimSpace(base) == "" {
			continue
		}
		for _, relative := range relatives {
			paths = append(paths, filepath.Join(base, filepath.FromSlash(relative)))
		}
	}

	return paths
}

func buildMPVArgs(req Request, _ string, logPath string) []string {
	args := []string{"--force-window=yes"}
	if strings.TrimSpace(req.ManagedResumeDir) != "" {
		if err := os.MkdirAll(req.ManagedResumeDir, 0o755); err == nil {
			args = append(args,
				"--resume-playback=no",
				"--save-position-on-quit",
				"--watch-later-options=start",
				"--watch-later-directory="+req.ManagedResumeDir,
				"--write-filename-in-watch-later-config",
			)
		}
	}
	if strings.TrimSpace(logPath) != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err == nil {
			args = append(args, "--log-file="+logPath)
		}
	}
	if strings.TrimSpace(req.Title) != "" {
		args = append(args, "--title="+req.Title)
	}
	if strings.TrimSpace(req.Subtitle) != "" {
		args = append(args, "--sub-file="+req.Subtitle)
	}
	if req.StartMS > 0 {
		args = append(args, "--start="+formatSeconds(req.StartMS))
	}
	args = append(args, req.URL)
	return args
}

func buildVLCArgs(req Request, _ string, _ string) []string {
	args := []string{"--no-video-title-show"}
	if strings.TrimSpace(req.Subtitle) != "" {
		args = append(args, "--sub-file="+req.Subtitle)
	}
	if req.StartMS > 0 {
		args = append(args, "--start-time="+formatSeconds(req.StartMS))
	}
	args = append(args, req.URL)
	return args
}

func buildPotPlayerArgs(req Request, _ string, _ string) []string {
	args := []string{req.URL}
	if req.StartMS > 0 {
		args = append(args, "/seek="+formatClock(req.StartMS))
	}
	if strings.TrimSpace(req.Subtitle) != "" {
		args = append(args, "/sub="+req.Subtitle)
	}
	return args
}

func buildMPCArgs(req Request, _ string, _ string) []string {
	args := []string{req.URL}
	if strings.TrimSpace(req.Subtitle) != "" {
		args = append(args, "/sub", req.Subtitle)
	}
	return args
}

func (r *LaunchResult) Done() <-chan error {
	return r.done
}

func (r *LaunchResult) SupportsManagedResume() bool {
	return r != nil && r.supportsManagedResume
}

func formatSeconds(ms int64) string {
	return strconv.FormatFloat(float64(ms)/1000, 'f', -1, 64)
}

func formatClock(ms int64) string {
	totalSeconds := ms / 1000
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"

	"panplayer/internal/config"
	"panplayer/internal/pan"
)

const (
	scraperDirectoryRetryDelay = 450 * time.Millisecond
	scraperDirectoryRetryCount = 2
)

type PanClient interface {
	ListDirectory(dirID string, offset, limit int) (*pan.DirectoryView, error)
}

type Scraper struct {
	logger   *slog.Logger
	pan      PanClient
	repo     *Repository
	resolver *MetadataResolver

	mu                 sync.RWMutex
	currentJob         *ScrapeJobStatus
	currentJobSettings ScraperSettings
	cancelCurrent      context.CancelFunc
}

func NewScraper(logger *slog.Logger, panClient PanClient, repo *Repository) *Scraper {
	return &Scraper{
		logger:   logger,
		pan:      panClient,
		repo:     repo,
		resolver: NewMetadataResolver(),
	}
}

func (s *Scraper) LastJob(ctx context.Context) (*ScrapeJobStatus, error) {
	s.mu.RLock()
	if s.currentJob != nil {
		copyValue := *s.currentJob
		s.mu.RUnlock()
		return &copyValue, nil
	}
	s.mu.RUnlock()
	return s.repo.LastJob(ctx)
}

func (s *Scraper) Start(ctx context.Context, settings config.Settings, playback map[string]PlaybackInfo, assetDir string) (*ScrapeJobStatus, error) {
	if s == nil || s.repo == nil || s.pan == nil {
		return nil, errors.New("scraper unavailable")
	}

	normalizedSettings := normalizeScraperSettings(settings)
	if s.resolver != nil {
		s.resolver.SetTMDBReadAccessToken(normalizedSettings.TMDBReadAccessToken)
	}
	jobSettings := settingsToScraperSettings(normalizedSettings)

	s.mu.Lock()
	if s.currentJob != nil && (s.currentJob.Status == "running" || s.currentJob.Status == "queued") {
		copyValue := *s.currentJob
		s.mu.Unlock()
		return &copyValue, errors.New("已有刮削任务正在进行中")
	}
	job := &ScrapeJobStatus{
		ID:        uuid.NewString(),
		Status:    "running",
		Message:   "正在扫描目录",
		StartedAt: nowRFC3339(),
		UpdatedAt: nowRFC3339(),
	}
	s.currentJob = job
	s.currentJobSettings = jobSettings
	s.mu.Unlock()

	if err := s.repo.UpsertJob(ctx, *job, jobSettings); err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	if s.currentJob != nil && s.currentJob.ID == job.ID {
		s.cancelCurrent = cancel
	}
	s.mu.Unlock()

	go s.run(runCtx, job.ID, normalizedSettings, playback, assetDir)
	copyValue := *job
	return &copyValue, nil
}

func (s *Scraper) Pause(ctx context.Context) (*ScrapeJobStatus, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("scraper unavailable")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentJob == nil {
		return nil, errors.New("当前没有刮削任务")
	}
	if s.currentJob.Status != "running" && s.currentJob.Status != "queued" {
		copyValue := *s.currentJob
		return &copyValue, errors.New("当前没有正在运行的刮削任务")
	}

	s.currentJob.Status = "paused"
	s.currentJob.Message = "刮削已暂停"
	s.currentJob.FinishedAt = nowRFC3339()
	s.currentJob.UpdatedAt = nowRFC3339()

	copyValue := *s.currentJob
	if err := s.repo.UpsertJob(ctx, copyValue, s.currentJobSettings); err != nil {
		return nil, err
	}

	if s.cancelCurrent != nil {
		s.cancelCurrent()
		s.cancelCurrent = nil
	}
	return &copyValue, nil
}

func (s *Scraper) run(parent context.Context, jobID string, settings config.Settings, playback map[string]PlaybackInfo, assetDir string) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Minute)
	defer cancel()
	defer s.clearCurrentCancel(jobID)

	update := func(mutator func(job *ScrapeJobStatus)) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.currentJob == nil || s.currentJob.ID != jobID {
			return
		}
		if s.currentJob.Status == "paused" {
			return
		}
		mutator(s.currentJob)
		s.currentJob.UpdatedAt = nowRFC3339()
		_ = s.repo.UpsertJob(context.Background(), *s.currentJob, s.currentJobSettings)
	}

	failJob := func(err error) {
		if s.isJobPaused(jobID) {
			return
		}
		message := "刮削失败"
		if err != nil {
			message = pan.SanitizeTemporaryHTMLResponseMessage(err.Error(), "刮削失败，请稍后重试。")
		}
		update(func(job *ScrapeJobStatus) {
			job.Status = "failed"
			job.Message = "刮削失败"
			job.LastError = message
			job.ErrorCount++
			job.FinishedAt = nowRFC3339()
		})
	}

	scanTargets := settings.ScraperDirectories
	if len(scanTargets) == 0 {
		scanTargets = []config.DirectoryTarget{{ID: "0", Path: []config.Breadcrumb{{ID: "0", Name: "我的文件"}}}}
	}

	update(func(job *ScrapeJobStatus) {
		job.TotalDirectories = len(scanTargets)
		job.Message = "正在扫描目录"
	})

	parsedFiles := make([]ParsedFile, 0, 128)
	for _, target := range scanTargets {
		if err := ctx.Err(); err != nil {
			if s.isJobPaused(jobID) {
				return
			}
			failJob(err)
			return
		}
		path := make([]pan.Breadcrumb, 0, len(target.Path))
		for _, crumb := range target.Path {
			path = append(path, pan.Breadcrumb{ID: crumb.ID, Name: crumb.Name})
		}
		if err := s.scanDirectoryRecursive(ctx, target.ID, path, &parsedFiles, func(currentPath string, fileCount int) {
			update(func(job *ScrapeJobStatus) {
				job.CurrentPath = currentPath
				job.DiscoveredFiles = fileCount
			})
		}); err != nil {
			failJob(err)
			return
		}
		update(func(job *ScrapeJobStatus) {
			job.ScannedDirectories++
		})
	}

	if err := ctx.Err(); err != nil {
		if s.isJobPaused(jobID) {
			return
		}
		failJob(err)
		return
	}

	if len(parsedFiles) == 0 {
		update(func(job *ScrapeJobStatus) {
			job.Status = "completed"
			job.Message = "未发现可刮削的视频文件"
			job.FinishedAt = nowRFC3339()
		})
		return
	}

	update(func(job *ScrapeJobStatus) {
		job.Message = fmt.Sprintf("已发现 %d 个视频文件，开始匹配元数据", len(parsedFiles))
		job.DiscoveredFiles = len(parsedFiles)
	})

	groups := GroupParsedFiles(parsedFiles)
	posterDir := filepath.Join(assetDir, "library-posters")
	if settings.ScraperSkipImages {
		posterDir = ""
	} else if posterDir != "" {
		_ = os.MkdirAll(posterDir, 0o755)
	}

	titles := make([]Title, 0, len(groups))
	files := make([]File, 0, len(parsedFiles))

	for index, group := range groups {
		if err := ctx.Err(); err != nil {
			if s.isJobPaused(jobID) {
				return
			}
			failJob(err)
			return
		}
		update(func(job *ScrapeJobStatus) {
			job.ProcessedFiles = index
			job.Message = "正在匹配：" + group.BaseTitle
			job.CurrentPath = group.BaseTitle
		})

		metadata, err := s.resolver.Resolve(ctx, group, settings.ScraperSources, settings.ScraperLanguage)
		if err != nil {
			s.logger.Warn("scraper metadata lookup failed", "title", group.BaseTitle, "error", err)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				if s.isJobPaused(jobID) {
					return
				}
				failJob(err)
				return
			}
			update(func(job *ScrapeJobStatus) {
				job.ErrorCount++
				job.LastError = pan.SanitizeTemporaryHTMLResponseMessage(err.Error(), "部分元数据获取失败，已跳过异常条目。")
			})
		}

		titleID := uuid.NewString()
		title, titleErr := buildTitleFromGroup(ctx, group, metadata, titleID, settings, posterDir)
		if titleErr != nil {
			if errors.Is(titleErr, context.Canceled) || errors.Is(titleErr, context.DeadlineExceeded) {
				if s.isJobPaused(jobID) {
					return
				}
				failJob(titleErr)
				return
			}
			s.logger.Warn("scraper build title failed", "title", group.BaseTitle, "error", titleErr)
			update(func(job *ScrapeJobStatus) {
				job.ErrorCount++
				job.LastError = pan.SanitizeTemporaryHTMLResponseMessage(titleErr.Error(), "部分图片资源下载失败，已跳过异常条目。")
			})
		}
		titles = append(titles, title)
		titleFiles := buildFilesFromGroup(group, titleID)
		files = append(files, titleFiles...)

		update(func(job *ScrapeJobStatus) {
			job.ProcessedFiles = min(index+1, len(groups))
			if metadata != nil {
				job.MatchedItems++
			}
			job.UpdatedTitles++
		})
	}

	if err := ctx.Err(); err != nil {
		if s.isJobPaused(jobID) {
			return
		}
		failJob(err)
		return
	}

	job, _ := s.LastJob(ctx)
	finalJob := ScrapeJobStatus{
		ID:                 jobID,
		Status:             "completed",
		Message:            fmt.Sprintf("刮削完成：%d 个条目，%d 个文件", len(titles), len(files)),
		CurrentPath:        "",
		ScannedDirectories: len(scanTargets),
		TotalDirectories:   len(scanTargets),
		DiscoveredFiles:    len(parsedFiles),
		ProcessedFiles:     len(groups),
		MatchedItems:       valueOrJob(job, func(j *ScrapeJobStatus) int { return j.MatchedItems }),
		UpdatedTitles:      len(titles),
		ErrorCount:         valueOrJob(job, func(j *ScrapeJobStatus) int { return j.ErrorCount }),
		LastError:          valueOrJobString(job, func(j *ScrapeJobStatus) string { return j.LastError }),
		StartedAt:          valueOrJobString(job, func(j *ScrapeJobStatus) string { return j.StartedAt }),
		FinishedAt:         nowRFC3339(),
		UpdatedAt:          nowRFC3339(),
	}

	if err := s.repo.ReplaceLibrary(context.Background(), titles, files, finalJob); err != nil {
		failJob(err)
		return
	}

	s.mu.Lock()
	s.currentJob = &finalJob
	s.mu.Unlock()
}

func (s *Scraper) scanDirectoryRecursive(ctx context.Context, dirID string, dirPath []pan.Breadcrumb, parsedFiles *[]ParsedFile, onUpdate func(currentPath string, fileCount int)) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	offset := 0
	knownPath := dirPath
	subDirs := make([]pan.FileItem, 0, 32)
	for {
		view, err := s.listDirectoryPageWithRetry(ctx, dirID, offset, 200)
		if err != nil {
			return err
		}
		if len(view.Path) > 0 {
			knownPath = append([]pan.Breadcrumb(nil), view.Path...)
		}
		if onUpdate != nil {
			onUpdate(joinBreadcrumbPath(knownPath), len(*parsedFiles))
		}

		for _, item := range view.Items {
			if item.IsDirectory {
				subDirs = append(subDirs, item)
				continue
			}
			if !item.IsVideo || strings.TrimSpace(item.PickCode) == "" {
				continue
			}
			*parsedFiles = append(*parsedFiles, ParseMediaFile(item, knownPath))
		}

		if !view.HasMore || len(view.Items) == 0 {
			break
		}
		offset += view.Limit
		if view.Limit <= 0 {
			offset += len(view.Items)
		}
	}

	sort.SliceStable(subDirs, func(i, j int) bool {
		return strings.ToLower(subDirs[i].Name) < strings.ToLower(subDirs[j].Name)
	})

	for _, dir := range subDirs {
		nextPath := append(append([]pan.Breadcrumb(nil), knownPath...), pan.Breadcrumb{
			ID:   dir.FileID,
			Name: dir.Name,
		})
		if err := s.scanDirectoryRecursive(ctx, dir.FileID, nextPath, parsedFiles, onUpdate); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scraper) listDirectoryPageWithRetry(ctx context.Context, dirID string, offset, limit int) (*pan.DirectoryView, error) {
	var lastErr error

	for attempt := 0; attempt < scraperDirectoryRetryCount; attempt++ {
		view, err := s.pan.ListDirectory(dirID, offset, limit)
		if err == nil {
			return view, nil
		}

		lastErr = err
		if !pan.ShouldRetryTemporaryHTMLResponseError(err) || attempt == scraperDirectoryRetryCount-1 {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(scraperDirectoryRetryDelay):
		}
	}

	return nil, pan.NormalizeTemporaryHTMLResponseError(lastErr, "读取 115 目录时返回了异常页面，请稍后重试。")
}

func (s *Scraper) isJobPaused(jobID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentJob != nil && s.currentJob.ID == jobID && s.currentJob.Status == "paused"
}

func (s *Scraper) clearCurrentCancel(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentJob == nil || s.currentJob.ID != jobID {
		return
	}
	s.cancelCurrent = nil
}

func normalizeScraperSettings(settings config.Settings) config.Settings {
	if settings.ScraperDirectories == nil {
		settings.ScraperDirectories = []config.DirectoryTarget{}
	}
	settings.ScraperDirectories = config.NormalizeDirectoryTargets(settings.ScraperDirectories, 50)
	settings.ScraperSources = config.NormalizeScraperSources(settings.ScraperSources)
	settings.ScraperLanguage = config.NormalizeScraperLanguage(settings.ScraperLanguage)
	if len(settings.ScraperSources) == 0 {
		settings.ScraperSources = config.DefaultScraperSources()
	}
	settings.TMDBReadAccessToken = strings.TrimSpace(settings.TMDBReadAccessToken)
	return settings
}

func settingsToScraperSettings(settings config.Settings) ScraperSettings {
	targets := make([]DirectoryTarget, 0, len(settings.ScraperDirectories))
	for _, target := range settings.ScraperDirectories {
		path := make([]pan.Breadcrumb, 0, len(target.Path))
		for _, crumb := range target.Path {
			path = append(path, pan.Breadcrumb{
				ID:   crumb.ID,
				Name: crumb.Name,
			})
		}
		targets = append(targets, DirectoryTarget{
			ID:   target.ID,
			Path: path,
		})
	}

	return ScraperSettings{
		Directories:    targets,
		Sources:        append([]string(nil), settings.ScraperSources...),
		Language:       settings.ScraperLanguage,
		AutoScan:       settings.ScraperAutoScan,
		Overwrite:      settings.ScraperOverwrite,
		DownloadImages: !settings.ScraperSkipImages,
	}
}

func buildTitleFromGroup(ctx context.Context, group TitleGroup, metadata *Metadata, titleID string, settings config.Settings, posterDir string) (Title, error) {
	now := nowRFC3339()
	title := Title{
		ID:                titleID,
		GroupKey:          group.GroupKey,
		Section:           group.Section,
		Child:             group.Child,
		Title:             group.BaseTitle,
		OriginalTitle:     "",
		SearchTitle:       group.SearchTitle,
		NormalizedTitle:   group.Normalized,
		Year:              group.Year,
		Rating:            0,
		Summary:           "",
		Director:          "",
		Cast:              []string{},
		Tags:              []string{},
		PosterURL:         "",
		PosterLocalPath:   "",
		BackdropURL:       "",
		BackdropLocalPath: "",
		Quality:           group.Quality,
		Source:            group.Source,
		Audio:             group.Audio,
		Status:            defaultStatusForGroup(group),
		SourceName:        "",
		SourceSubjectID:   "",
		ExternalDataJSON:  "{}",
		DefaultFileID:     defaultFileIDForGroup(group),
		SeasonCount:       group.SeasonCount,
		EpisodeCount:      group.EpisodeCount,
		TotalDurationSec:  group.DurationTotal,
		CreatedAt:         now,
		UpdatedAt:         now,
		ScrapedAt:         now,
	}

	if metadata == nil {
		return title, nil
	}

	title.Title = preferNonEmpty(metadata.Title, title.Title)
	title.OriginalTitle = metadata.OriginalTitle
	if title.Year == 0 {
		title.Year = metadata.Year
	}
	title.Rating = metadata.Rating
	title.Summary = metadata.Summary
	title.Director = metadata.Director
	title.Cast = metadata.Cast
	title.Tags = metadata.Tags
	title.PosterURL = metadata.PosterURL
	title.BackdropURL = metadata.BackdropURL
	title.SourceName = metadata.SourceName
	title.SourceSubjectID = metadata.SourceSubjectID
	external := map[string]any{}
	external["search_title"] = group.SearchTitle
	external["group_title"] = group.BaseTitle
	external["section"] = group.Section
	backdropDir := ""
	avatarDir := ""
	episodeThumbDir := ""
	if posterDir != "" {
		backdropDir = filepath.Join(filepath.Dir(posterDir), "library-backdrops")
		avatarDir = filepath.Join(filepath.Dir(posterDir), "library-cast")
		episodeThumbDir = filepath.Join(filepath.Dir(posterDir), "library-episodes")
		_ = os.MkdirAll(backdropDir, 0o755)
		_ = os.MkdirAll(avatarDir, 0o755)
		_ = os.MkdirAll(episodeThumbDir, 0o755)
	}

	clonedExternal := cloneExternalData(metadata.ExternalData)
	localCastMembers := cloneCastMembers(metadata.CastMembers)
	if backdropDir != "" && strings.TrimSpace(metadata.BackdropURL) != "" {
		if localPath, err := downloadRemoteImage(ctx, metadata.BackdropURL, title.SourceName, title.SourceSubjectID, "backdrop", backdropDir); err == nil {
			title.BackdropLocalPath = localPath
		} else if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
			return title, err
		}
	}
	if avatarDir != "" && len(localCastMembers) > 0 {
		for index := range localCastMembers {
			if strings.TrimSpace(localCastMembers[index].AvatarURL) == "" {
				continue
			}
			if localPath, err := downloadRemoteImage(ctx, localCastMembers[index].AvatarURL, title.SourceName, title.SourceSubjectID, fmt.Sprintf("cast-%02d", index+1), avatarDir); err == nil {
				localCastMembers[index].AvatarLocalPath = localPath
			} else if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
				return title, err
			}
		}
	}
	if episodeThumbDir != "" {
		if assets, ok := clonedExternal["episode_assets"].([]EpisodeAsset); ok && len(assets) > 0 {
			for index := range assets {
				if strings.TrimSpace(assets[index].ThumbnailURL) != "" {
					if localPath, err := downloadRemoteImage(ctx, assets[index].ThumbnailURL, title.SourceName, title.SourceSubjectID, fmt.Sprintf("episode-s%02de%03d", max(assets[index].SeasonNumber, 1), max(assets[index].EpisodeNumber, index+1)), episodeThumbDir); err == nil {
						assets[index].ThumbnailLocalPath = localPath
					} else if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
						return title, err
					}
				}
				if strings.TrimSpace(assets[index].BackdropURL) != "" {
					if localPath, err := downloadRemoteImage(ctx, assets[index].BackdropURL, title.SourceName, title.SourceSubjectID, fmt.Sprintf("episode-backdrop-s%02de%03d", max(assets[index].SeasonNumber, 1), max(assets[index].EpisodeNumber, index+1)), episodeThumbDir); err == nil {
						assets[index].BackdropLocalPath = localPath
					} else if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
						return title, err
					}
				}
			}
			clonedExternal["episode_assets"] = assets
		}
	}

	for key, value := range clonedExternal {
		external[key] = value
	}
	if len(localCastMembers) > 0 {
		external["cast_members"] = localCastMembers
	}
	if len(external) > 0 {
		if payload, err := json.Marshal(external); err == nil {
			title.ExternalDataJSON = string(payload)
		}
	}
	if posterDir != "" && metadata.PosterURL != "" {
		if localPath, err := downloadPoster(ctx, metadata.PosterURL, title.SourceName, title.SourceSubjectID, posterDir); err == nil {
			title.PosterLocalPath = localPath
		} else if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			sanitized := pan.SanitizeTemporaryHTMLResponseMessage(err.Error(), "海报下载失败")
			_ = sanitized
		} else {
			return title, err
		}
	}
	return title, nil
}

func cloneExternalData(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	target := make(map[string]any, len(source))
	for key, value := range source {
		switch typed := value.(type) {
		case []EpisodeAsset:
			cloned := make([]EpisodeAsset, len(typed))
			copy(cloned, typed)
			target[key] = cloned
		case []CastMember:
			cloned := make([]CastMember, len(typed))
			copy(cloned, typed)
			target[key] = cloned
		default:
			target[key] = value
		}
	}
	return target
}

func cloneCastMembers(source []CastMember) []CastMember {
	if len(source) == 0 {
		return []CastMember{}
	}
	target := make([]CastMember, len(source))
	copy(target, source)
	return target
}

func downloadRemoteImage(ctx context.Context, rawURL, sourceName, subjectID, suffix, dir string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errors.New("empty image url")
	}
	if strings.TrimSpace(dir) == "" {
		return "", errors.New("empty image directory")
	}
	filenameSeed := strings.TrimSpace(subjectID)
	if filenameSeed == "" {
		filenameSeed = uuid.NewString()
	}
	if strings.TrimSpace(suffix) != "" {
		filenameSeed += "-" + strings.TrimSpace(suffix)
	}
	return downloadPoster(ctx, rawURL, sourceName, filenameSeed, dir)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func buildFilesFromGroup(group TitleGroup, titleID string) []File {
	files := make([]File, 0, len(group.Files))
	now := nowRFC3339()
	sorted := append([]ParsedFile(nil), group.Files...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]
		if left.SeasonNumber != right.SeasonNumber {
			return left.SeasonNumber < right.SeasonNumber
		}
		if left.EpisodeNumber != right.EpisodeNumber {
			return left.EpisodeNumber < right.EpisodeNumber
		}
		if compare := compareVariantRank(left.Name, right.Name); compare != 0 {
			return compare < 0
		}
		return strings.ToLower(left.Name) < strings.ToLower(right.Name)
	})

	for index, item := range sorted {
		files = append(files, File{
			ID:            uuid.NewString(),
			TitleID:       titleID,
			FileID:        item.FileID,
			PickCode:      item.PickCode,
			ParentID:      item.ParentID,
			Path:          append([]pan.Breadcrumb(nil), item.Path...),
			Name:          item.Name,
			Size:          item.Size,
			UpdatedAt:     item.UpdatedAt,
			DurationSec:   item.DurationSec,
			SeasonNumber:  item.SeasonNumber,
			EpisodeNumber: item.EpisodeNumber,
			EpisodeTitle:  item.EpisodeTitle,
			Quality:       item.Quality,
			Source:        item.Source,
			Audio:         item.Audio,
			SearchTitle:   item.SearchTitle,
			ParsedYear:    item.Year,
			Section:       item.Section,
			Child:         item.Child,
			DisplayOrder:  index,
			CreatedAt:     now,
			UpdatedAtRow:  now,
			ScrapedAt:     now,
		})
	}

	return files
}

func defaultStatusForGroup(group TitleGroup) string {
	if group.Section == "movies" {
		return "已入库"
	}
	if group.SeasonCount > 0 && group.EpisodeCount > 1 {
		return "更新完成"
	}
	return "已入库"
}

func defaultFileIDForGroup(group TitleGroup) string {
	if len(group.Files) == 0 {
		return ""
	}
	best := group.Files[0]
	bestRank := variantRank(best.Name)
	for _, file := range group.Files[1:] {
		rank := variantRank(file.Name)
		if file.EpisodeNumber > 0 && (best.EpisodeNumber == 0 || file.EpisodeNumber < best.EpisodeNumber) {
			best = file
			bestRank = rank
			continue
		}
		if file.EpisodeNumber == best.EpisodeNumber && rank < bestRank {
			best = file
			bestRank = rank
			continue
		}
		if best.EpisodeNumber == 0 && file.DurationSec > best.DurationSec {
			best = file
			bestRank = rank
		}
	}
	return best.FileID
}

func joinBreadcrumbPath(path []pan.Breadcrumb) string {
	if len(path) == 0 {
		return "/"
	}
	names := make([]string, 0, len(path))
	for _, crumb := range path {
		if strings.TrimSpace(crumb.Name) == "" {
			continue
		}
		names = append(names, crumb.Name)
	}
	if len(names) == 0 {
		return "/"
	}
	return strings.Join(names, " / ")
}

func downloadPoster(ctx context.Context, rawURL, sourceName, subjectID, dir string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.TrimSpace(dir) == "" {
		return "", errors.New("poster disabled")
	}
	filename := strings.TrimSpace(subjectID)
	if filename == "" {
		filename = strings.TrimSpace(maybePosterFilename(rawURL))
	}
	if filename == "" {
		filename = uuid.NewString()
	}
	filename = sanitizeFilename(strings.ToLower(sourceName) + "-" + filename)
	if filepath.Ext(filename) == "" {
		filename += filepath.Ext(rawURL)
		if filepath.Ext(filename) == "" {
			filename += ".jpg"
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 panplayer115/1.0")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("download poster failed: %s", resp.Status)
	}

	targetPath := filepath.Join(dir, filename)
	file, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}
	return targetPath, nil
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.NewReplacer(
		"<", "_", ">", "_", ":", "_", "\"", "_", "/", "_", "\\", "_", "|", "_", "?", "_", "*", "_",
	).Replace(name)
	name = strings.ReplaceAll(name, "..", ".")
	return name
}

func valueOrJob(job *ScrapeJobStatus, selector func(*ScrapeJobStatus) int) int {
	if job == nil {
		return 0
	}
	return selector(job)
}

func valueOrJobString(job *ScrapeJobStatus, selector func(*ScrapeJobStatus) string) string {
	if job == nil {
		return ""
	}
	return selector(job)
}

func compareVariantRank(leftName, rightName string) int {
	left := variantRank(leftName)
	right := variantRank(rightName)
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func variantRank(name string) int {
	normalized := strings.ToUpper(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, name))
	switch {
	case strings.Contains(normalized, "PART1"):
		return 1
	case strings.Contains(normalized, "PART2"):
		return 2
	case strings.Contains(normalized, "PART3"):
		return 3
	case strings.Contains(normalized, "EX1"):
		return 4
	case strings.Contains(normalized, "EX2"):
		return 5
	case strings.Contains(normalized, "SP1"):
		return 6
	case strings.Contains(normalized, "SP01"):
		return 6
	case strings.Contains(normalized, "SP2"):
		return 7
	case strings.Contains(normalized, "SP02"):
		return 7
	case strings.Contains(normalized, "PART"):
		return 8
	case strings.Contains(normalized, "EX"):
		return 9
	case strings.Contains(normalized, "SP"):
		return 10
	default:
		return 0
	}
}

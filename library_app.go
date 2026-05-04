package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"panplayer/internal/library"
	"panplayer/internal/pan"
)

type LibrarySnapshotView = library.LibrarySnapshot
type LibraryTitleSummaryView = library.TitleSummary
type LibraryTitleDetailView = library.TitleDetail
type ScrapeJobStatusView = library.ScrapeJobStatus

type LibraryPlayRequest struct {
	TitleID string `json:"titleId"`
	FileID  string `json:"fileId"`
	StartMS int64  `json:"startMs"`
}

func (a *App) StartScraper() (*ScrapeJobStatusView, error) {
	if !a.started {
		return nil, errors.New("app not ready")
	}
	settings := a.currentSettings()
	for _, source := range settings.ScraperSources {
		if strings.EqualFold(strings.TrimSpace(source), "tmdb") && strings.TrimSpace(settings.TMDBReadAccessToken) == "" {
			return nil, errors.New("已启用 TMDB 数据源，但未配置 TMDB Read Access Token")
		}
	}
	playback := a.playbackSnapshotForLibrary()
	assetDir := filepath.Dir(a.store.Path())
	return a.scraper.Start(context.Background(), settings, playback, assetDir)
}

func (a *App) PauseScraper() (*ScrapeJobStatusView, error) {
	if !a.started {
		return nil, errors.New("app not ready")
	}
	return a.scraper.Pause(context.Background())
}

func (a *App) GetScraperStatus() (*ScrapeJobStatusView, error) {
	if !a.started {
		return nil, errors.New("app not ready")
	}
	return a.scraper.LastJob(context.Background())
}

func (a *App) GetLibrarySnapshot() (*LibrarySnapshotView, error) {
	if !a.started {
		return nil, errors.New("app not ready")
	}
	playback := a.playbackSnapshotForLibrary()
	snapshot, err := a.library.LoadSnapshot(context.Background(), playback)
	if err != nil {
		return nil, err
	}
	a.enrichLibrarySnapshot(snapshot)
	return snapshot, nil
}

func (a *App) GetLibraryDetail(titleID string) (*LibraryTitleDetailView, error) {
	if !a.started {
		return nil, errors.New("app not ready")
	}
	playback := a.playbackSnapshotForLibrary()
	detail, err := a.library.GetTitleDetail(context.Background(), strings.TrimSpace(titleID), playback)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("未找到对应的影视条目")
	}
	if err != nil {
		return nil, err
	}
	a.enrichLibraryDetail(detail)
	return detail, nil
}

func (a *App) ResolveLibraryPlayRequest(req LibraryPlayRequest) (*PlayRequest, error) {
	titleID := strings.TrimSpace(req.TitleID)
	fileID := strings.TrimSpace(req.FileID)
	if titleID == "" {
		return nil, errors.New("缺少 titleId")
	}

	detail, err := a.GetLibraryDetail(titleID)
	if err != nil {
		return nil, err
	}

	var selected *library.EpisodeItem
	if fileID != "" {
		for i := range detail.Files {
			if detail.Files[i].FileID == fileID {
				selected = &detail.Files[i]
				break
			}
		}
	}
	if selected == nil {
		if len(detail.Files) == 0 {
			return nil, errors.New("当前影视条目没有可播放文件")
		}
		selected = &detail.Files[0]
	}

	name := selected.Name
	if strings.TrimSpace(name) == "" {
		name = detail.Title
	}

	return &PlayRequest{
		PickCode:  selected.PickCode,
		Name:      name,
		StartMS:   req.StartMS,
		FromStart: false,
	}, nil
}

func (a *App) playbackSnapshotForLibrary() map[string]library.PlaybackInfo {
	records := a.playbackRecordsSnapshot()
	result := make(map[string]library.PlaybackInfo, len(records))
	for key, value := range records {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		result[key] = library.PlaybackInfo{
			ResumeMS:     value.LastPositionMS,
			LastPlayedAt: value.LastPlayedAt,
		}
	}
	return result
}

func breadcrumbsToPan(path []pan.Breadcrumb) []pan.Breadcrumb {
	if len(path) == 0 {
		return []pan.Breadcrumb{}
	}
	result := make([]pan.Breadcrumb, len(path))
	copy(result, path)
	return result
}

func (a *App) enrichLibraryDetail(detail *library.TitleDetail) {
	if detail == nil {
		return
	}

	reclassifyLibraryDetail(detail)
	normalizeLibraryFileSeasonNumbers(detail)
	applyLibrarySummaryAssetURLs(a.proxy, &detail.TitleSummary)

	if len(detail.CastMembers) == 0 && len(detail.Cast) > 0 {
		detail.CastMembers = make([]library.CastMember, 0, len(detail.Cast))
		for _, name := range detail.Cast {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			detail.CastMembers = append(detail.CastMembers, library.CastMember{Name: name})
		}
	}

	if needsMetadataRefresh(detail) && a.resolver != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
		defer cancel()
		meta, err := a.resolver.ResolveBySourceID(ctx, detail.SourceName, detail.SourceSubjectID, detail.Title, detail.OriginalTitle)
		if err == nil && meta != nil {
			if shouldReplaceLibraryBackdrop(detail.BackdropURL, detail.PosterURL, meta.BackdropURL) {
				detail.BackdropURL = meta.BackdropURL
			}
			if detail.PosterURL == "" {
				detail.PosterURL = meta.PosterURL
			}
			if len(detail.CastMembers) == 0 && len(meta.CastMembers) > 0 {
				detail.CastMembers = meta.CastMembers
			}
			if detail.Director == "" {
				detail.Director = meta.Director
			}
			if detail.Summary == "" {
				detail.Summary = meta.Summary
			}
			if len(detail.Cast) == 0 && len(meta.Cast) > 0 {
				detail.Cast = meta.Cast
			}
			if len(meta.ExternalData) > 0 {
				detail.ExternalDataJSON = mergeLibraryExternalData(detail.ExternalDataJSON, meta.ExternalData)
			}
		}
	}

	if needsOnlineMetadataLookup(detail) && a.resolver != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		group := buildLookupGroupFromDetail(detail)
		meta, err := a.resolver.Resolve(ctx, group, []string{"tmdb", "douban", "bangumi"}, "zh-CN")
		if err == nil && meta != nil {
			applyResolvedMetadataToDetail(detail, meta)
		}
	}

	applyLibraryEpisodeAssets(detail)

	detail.PosterURL = a.resolveLibraryImageURL(detail.PosterURL, detail.PosterLocalPath)
	detail.BackdropURL = a.resolveLibraryImageURL(detail.BackdropURL, detail.BackdropLocalPath)

	episodeThumb := strings.TrimSpace(detail.BackdropURL)
	if episodeThumb == "" {
		episodeThumb = strings.TrimSpace(detail.PosterURL)
	}
	for index := range detail.Files {
		if detail.Files[index].ThumbnailURL == "" {
			detail.Files[index].ThumbnailURL = episodeThumb
		}
		detail.Files[index].ThumbnailURL = a.resolveLibraryImageURL(detail.Files[index].ThumbnailURL, "")
	}

	if detail.BackdropURL == "" && detail.PosterURL != "" {
		detail.BackdropURL = detail.PosterURL
	}
	for index := range detail.CastMembers {
		detail.CastMembers[index].AvatarURL = a.resolveLibraryImageURL(detail.CastMembers[index].AvatarURL, "")
	}
}

func (a *App) enrichLibrarySnapshot(snapshot *library.LibrarySnapshot) {
	if snapshot == nil {
		return
	}
	for index := range snapshot.Items {
		reclassifyLibrarySummary(&snapshot.Items[index])
		applyLibrarySummaryAssetURLs(a.proxy, &snapshot.Items[index])
	}
}

func needsMetadataRefresh(detail *library.TitleDetail) bool {
	if detail == nil {
		return false
	}
	if strings.TrimSpace(detail.SourceSubjectID) == "" || strings.TrimSpace(detail.SourceName) == "" {
		return false
	}
	if strings.TrimSpace(detail.BackdropURL) == "" || len(detail.CastMembers) == 0 {
		return true
	}
	return len(parseEpisodeAssets(detail.ExternalDataJSON)) == 0
}

func needsOnlineMetadataLookup(detail *library.TitleDetail) bool {
	if detail == nil {
		return false
	}
	if strings.TrimSpace(detail.PosterURL) == "" || strings.TrimSpace(detail.BackdropURL) == "" {
		return true
	}
	if len(detail.CastMembers) == 0 {
		return true
	}
	if detail.Section != "movies" && !hasUsefulEpisodeAssets(detail) {
		return true
	}
	return false
}

func buildLookupGroupFromDetail(detail *library.TitleDetail) library.TitleGroup {
	group := library.TitleGroup{
		BaseTitle:   strings.TrimSpace(detail.Title),
		SeriesTitle: strings.TrimSpace(detail.Title),
		SearchTitle: strings.TrimSpace(detail.Title),
		Section:     strings.TrimSpace(detail.Section),
		Child:       strings.TrimSpace(detail.Child),
		Year:        detail.Year,
	}

	if searchTitle := externalDataString(detail.ExternalDataJSON, "search_title"); searchTitle != "" {
		group.SearchTitle = searchTitle
	}
	if groupTitle := externalDataString(detail.ExternalDataJSON, "group_title"); groupTitle != "" {
		group.BaseTitle = groupTitle
		group.SeriesTitle = groupTitle
	}
	if group.SearchTitle == "" {
		group.SearchTitle = group.BaseTitle
	}
	return group
}

func applyResolvedMetadataToDetail(detail *library.TitleDetail, meta *library.Metadata) {
	if detail == nil || meta == nil {
		return
	}
	if detail.Title == "" && strings.TrimSpace(meta.Title) != "" {
		detail.Title = strings.TrimSpace(meta.Title)
	}
	if detail.OriginalTitle == "" {
		detail.OriginalTitle = strings.TrimSpace(meta.OriginalTitle)
	}
	if detail.Year == 0 {
		detail.Year = meta.Year
	}
	if detail.Rating <= 0 && meta.Rating > 0 {
		detail.Rating = meta.Rating
	}
	if detail.Summary == "" {
		detail.Summary = meta.Summary
	}
	if detail.Director == "" {
		detail.Director = meta.Director
	}
	if len(detail.Cast) == 0 && len(meta.Cast) > 0 {
		detail.Cast = append([]string(nil), meta.Cast...)
	}
	if len(detail.CastMembers) == 0 && len(meta.CastMembers) > 0 {
		detail.CastMembers = append([]library.CastMember(nil), meta.CastMembers...)
	}
	if detail.PosterURL == "" {
		detail.PosterURL = meta.PosterURL
	}
	if shouldReplaceLibraryBackdrop(detail.BackdropURL, detail.PosterURL, meta.BackdropURL) {
		detail.BackdropURL = meta.BackdropURL
	}
	if strings.TrimSpace(detail.SourceName) == "" {
		detail.SourceName = meta.SourceName
	}
	if strings.TrimSpace(detail.SourceSubjectID) == "" {
		detail.SourceSubjectID = meta.SourceSubjectID
	}
	if len(meta.ExternalData) > 0 {
		detail.ExternalDataJSON = mergeLibraryExternalData(detail.ExternalDataJSON, meta.ExternalData)
	}
}

func reclassifyLibraryDetail(detail *library.TitleDetail) {
	if detail == nil {
		return
	}
	reclassifyLibrarySummary(&detail.TitleSummary)
}

func reclassifyLibrarySummary(item *library.TitleSummary) {
	if item == nil {
		return
	}
	if item.Section == "movies" {
		return
	}

	fileName := strings.TrimSpace(item.DefaultName)
	if fileName == "" {
		return
	}
	parsed := library.ParseMediaFile(pan.FileItem{
		FileID:      item.DefaultFileID,
		Name:        fileName,
		IsVideo:     true,
		DurationSec: item.DurationSec,
	}, breadcrumbsToPan(item.DefaultPath))

	if parsed.Section != "movies" {
		return
	}

	item.Section = "movies"
	item.Child = parsed.Child
}

func applyLibraryEpisodeAssets(detail *library.TitleDetail) {
	if detail == nil || len(detail.Files) == 0 {
		return
	}

	assets := parseEpisodeAssets(detail.ExternalDataJSON)
	if len(assets) == 0 {
		return
	}

	if detail.BackdropURL == "" {
	for _, asset := range assets {
		if strings.TrimSpace(asset.BackdropURL) != "" {
			if strings.TrimSpace(asset.BackdropLocalPath) != "" {
				detail.BackdropURL = strings.TrimSpace(asset.BackdropLocalPath)
			} else {
				detail.BackdropURL = strings.TrimSpace(asset.BackdropURL)
			}
			break
		}
	}
	}

	fileIndexes := make([]int, 0, len(detail.Files))
	for index := range detail.Files {
		if detail.Files[index].DurationSec > 0 {
			fileIndexes = append(fileIndexes, index)
		}
	}
	if len(fileIndexes) == 0 {
		return
	}

	sort.SliceStable(fileIndexes, func(i, j int) bool {
		left := detail.Files[fileIndexes[i]]
		right := detail.Files[fileIndexes[j]]
		leftTime := parseSortableTime(left.UpdatedAt)
		rightTime := parseSortableTime(right.UpdatedAt)
		if !leftTime.Equal(rightTime) {
			return leftTime.Before(rightTime)
		}
		if left.SeasonNumber != right.SeasonNumber {
			return left.SeasonNumber < right.SeasonNumber
		}
		if left.EpisodeNumber != right.EpisodeNumber {
			return left.EpisodeNumber < right.EpisodeNumber
		}
		return strings.ToLower(strings.TrimSpace(left.Name)) < strings.ToLower(strings.TrimSpace(right.Name))
	})

	matched := map[int]bool{}
	for assetIndex, asset := range assets {
		fileIndex := matchEpisodeAssetToFile(detail.Files, fileIndexes, matched, assetIndex, asset)
		if fileIndex < 0 {
			continue
		}
		matched[fileIndex] = true
		if strings.TrimSpace(asset.ThumbnailLocalPath) != "" {
			detail.Files[fileIndex].ThumbnailURL = strings.TrimSpace(asset.ThumbnailLocalPath)
		} else if strings.TrimSpace(asset.ThumbnailURL) != "" {
			detail.Files[fileIndex].ThumbnailURL = strings.TrimSpace(asset.ThumbnailURL)
		}
		if strings.TrimSpace(asset.EpisodeTitle) != "" {
			detail.Files[fileIndex].EpisodeTitle = simplifyEpisodeTitle(asset.EpisodeTitle)
		}
	}
}

func parseEpisodeAssets(externalJSON string) []library.EpisodeAsset {
	if strings.TrimSpace(externalJSON) == "" {
		return []library.EpisodeAsset{}
	}
	var payload struct {
		EpisodeAssets []library.EpisodeAsset `json:"episode_assets"`
	}
	if err := json.Unmarshal([]byte(externalJSON), &payload); err != nil {
		return []library.EpisodeAsset{}
	}
	if payload.EpisodeAssets == nil {
		return []library.EpisodeAsset{}
	}
	return payload.EpisodeAssets
}

func externalDataString(externalJSON, key string) string {
	if strings.TrimSpace(externalJSON) == "" || strings.TrimSpace(key) == "" {
		return ""
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(externalJSON), &payload); err != nil {
		return ""
	}
	value, ok := payload[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func matchEpisodeAssetToFile(files []library.EpisodeItem, fileIndexes []int, matched map[int]bool, assetIndex int, asset library.EpisodeAsset) int {
	releaseDate := strings.TrimSpace(asset.ReleaseDate)
	bestIndex := -1
	bestScore := -1
	normalizedAssetTitle := normalizeEpisodeCompareText(asset.EpisodeTitle)
	assetEpisodeNumber := episodeNumberInTitle(asset.EpisodeTitle)
	for _, fileIndex := range fileIndexes {
		if matched[fileIndex] {
			continue
		}
		file := files[fileIndex]
		score := 0
		fileSeasonNumber := file.SeasonNumber
		if fileSeasonNumber <= 0 {
			fileSeasonNumber = 1
		}
		assetSeasonNumber := asset.SeasonNumber
		if assetSeasonNumber <= 0 && len(fileIndexes) > 0 {
			assetSeasonNumber = fileSeasonNumber
		}
		if assetSeasonNumber > 0 && fileSeasonNumber > 0 {
			if assetSeasonNumber == fileSeasonNumber {
				score += 14
			} else {
				diff := assetSeasonNumber - fileSeasonNumber
				if diff < 0 {
					diff = -diff
				}
				score -= diff * 10
			}
		}
		if releaseDate != "" && strings.HasPrefix(strings.TrimSpace(file.UpdatedAt), releaseDate) {
			score += 10
		}
		if asset.DurationSec > 0 && file.DurationSec > 0 {
			diff := asset.DurationSec - file.DurationSec
			if diff < 0 {
				diff = -diff
			}
			switch {
			case diff <= 45:
				score += 6
			case diff <= 180:
				score += 3
			}
		}
		if normalizedAssetTitle != "" {
			normalizedFileTitle := normalizeEpisodeCompareText(file.EpisodeTitle)
			switch {
			case normalizedAssetTitle == normalizedFileTitle:
				score += 5
			case normalizedFileTitle != "" && strings.Contains(normalizedAssetTitle, normalizedFileTitle):
				score += 3
			case normalizedAssetTitle != "" && normalizedFileTitle != "" && strings.Contains(normalizedFileTitle, normalizedAssetTitle):
				score += 2
			}
		}
		if file.EpisodeNumber > 0 && assetEpisodeNumber > 0 {
			if assetEpisodeNumber == file.EpisodeNumber {
				score += 18
			} else {
				diff := assetEpisodeNumber - file.EpisodeNumber
				if diff < 0 {
					diff = -diff
				}
				if diff == 1 {
					score += 1
				} else {
					score -= diff * 4
				}
			}
		}
		if asset.EpisodeNumber > 0 && file.EpisodeNumber > 0 {
			if asset.EpisodeNumber == file.EpisodeNumber {
				score += 10
			}
		}
		if assetIndex >= 0 && assetIndex < len(fileIndexes) && fileIndex == fileIndexes[assetIndex] {
			score += 4
		}
		preferredIndex := assetIndex
		if preferredIndex < 0 {
			preferredIndex = 0
		}
		if preferredIndex >= len(fileIndexes) {
			preferredIndex = len(fileIndexes) - 1
		}
		if bestIndex < 0 || score > bestScore || (score == bestScore && fileIndex == fileIndexes[preferredIndex]) {
			bestIndex = fileIndex
			bestScore = score
		}
	}
	if bestScore <= 0 {
		if assetIndex >= 0 && assetIndex < len(fileIndexes) {
			candidate := fileIndexes[assetIndex]
			if !matched[candidate] {
				return candidate
			}
		}
		return -1
	}
	return bestIndex
}

func simplifyTencentEpisodeTitle(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "第") && strings.Contains(value, "集") && strings.Contains(value, "：") {
		parts := strings.SplitN(value, "：", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			return strings.TrimSpace(parts[1])
		}
	}
	parts := strings.SplitN(value, "：", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[1])
	}
	return value
}

func simplifyEpisodeTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if simplified := simplifyTencentEpisodeTitle(value); simplified != "" && simplified != value {
		return simplified
	}
	return value
}

func normalizeEpisodeCompareText(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"：", "",
		":", "",
		" ", "",
		"上", "part1",
		"中", "part2",
		"下", "part3",
		"加更", "extra",
		"特别", "special",
		"期", "",
	)
	return strings.ToLower(replacer.Replace(value))
}

func episodeNumberInTitle(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	re := regexp.MustCompile(`第\s*([0-9]{1,3})`)
	match := re.FindStringSubmatch(value)
	if len(match) < 2 {
		return 0
	}
	number, _ := strconv.Atoi(match[1])
	return number
}

func parseSortableTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func mergeLibraryExternalData(current string, patch map[string]any) string {
	merged := map[string]any{}
	if strings.TrimSpace(current) != "" {
		_ = json.Unmarshal([]byte(current), &merged)
	}
	for key, value := range patch {
		merged[key] = value
	}
	if len(merged) == 0 {
		return ""
	}
	payload, err := json.Marshal(merged)
	if err != nil {
		return strings.TrimSpace(current)
	}
	return string(payload)
}

func shouldReplaceLibraryBackdrop(current, poster, candidate string) bool {
	current = strings.TrimSpace(current)
	poster = strings.TrimSpace(poster)
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return false
	}
	if current == "" {
		return true
	}
	if current == poster {
		return true
	}
	if strings.Contains(current, "doubanio.com") && !strings.Contains(candidate, "doubanio.com") {
		return true
	}
	return false
}

func applyLibrarySummaryAssetURLs(proxy interface {
	ImageURL(string) string
	ImagePathURL(string) string
}, item *library.TitleSummary) {
	if item == nil {
		return
	}
	item.PosterURL = resolveLibraryImageAssetURL(proxy, item.PosterURL, item.PosterLocalPath)
	item.BackdropURL = resolveLibraryImageAssetURL(proxy, item.BackdropURL, item.BackdropLocalPath)
	if strings.TrimSpace(item.BackdropURL) == "" {
		item.BackdropURL = item.PosterURL
	}
	for index := range item.CastMembers {
		item.CastMembers[index].AvatarURL = resolveLibraryImageAssetURL(proxy, item.CastMembers[index].AvatarURL, "")
	}
}

func resolveLibraryImageAssetURL(proxy interface {
	ImageURL(string) string
	ImagePathURL(string) string
}, rawURL, localPath string) string {
	localPath = strings.TrimSpace(localPath)
	if localPath != "" && proxy != nil {
		if resolved := strings.TrimSpace(proxy.ImagePathURL(localPath)); resolved != "" {
			return resolved
		}
	}

	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if looksLikeLocalFilesystemPath(rawURL) && proxy != nil {
		if resolved := strings.TrimSpace(proxy.ImagePathURL(rawURL)); resolved != "" {
			return resolved
		}
	}
	if isLocalLibraryProxyURL(rawURL) {
		return rawURL
	}
	if proxy == nil {
		return rawURL
	}
	if resolved := strings.TrimSpace(proxy.ImageURL(rawURL)); resolved != "" {
		return resolved
	}
	return rawURL
}

func (a *App) resolveLibraryImageURL(rawURL, localPath string) string {
	return resolveLibraryImageAssetURL(a.proxy, rawURL, localPath)
}

func hasUsefulEpisodeAssets(detail *library.TitleDetail) bool {
	if detail == nil || detail.Section == "movies" {
		return true
	}

	assets := parseEpisodeAssets(detail.ExternalDataJSON)
	if len(assets) == 0 {
		return false
	}

	playableCount := countDistinctPlayableEpisodes(detail.Files)
	if playableCount == 0 {
		playableCount = len(detail.Files)
	}
	if playableCount <= 0 {
		return len(assets) > 0
	}

	assetEpisodeCount := 0
	namedCount := 0
	thumbCount := 0
	seenEpisodes := map[string]struct{}{}
	for _, asset := range assets {
		if strings.TrimSpace(asset.ThumbnailURL) != "" {
			thumbCount++
		}
		if !isGenericEpisodeDisplayTitle(asset.EpisodeTitle) {
			namedCount++
		}
		if asset.EpisodeNumber > 0 {
			seasonNumber := asset.SeasonNumber
			if seasonNumber <= 0 {
				seasonNumber = 1
			}
			key := fmt.Sprintf("%d-%d", seasonNumber, asset.EpisodeNumber)
			if _, ok := seenEpisodes[key]; !ok {
				seenEpisodes[key] = struct{}{}
				assetEpisodeCount++
			}
		}
	}

	if assetEpisodeCount == 0 {
		assetEpisodeCount = len(assets)
	}

	minExpected := playableCount
	if minExpected > 3 {
		minExpected = maxInt(playableCount/2, 3)
	}

	if assetEpisodeCount < minExpected {
		return false
	}
	if namedCount == 0 {
		return false
	}
	if thumbCount < minExpected {
		return false
	}
	return true
}

func countDistinctPlayableEpisodes(files []library.EpisodeItem) int {
	if len(files) == 0 {
		return 0
	}
	seen := map[string]struct{}{}
	count := 0
	for _, file := range files {
		if file.DurationSec <= 0 {
			continue
		}
		episodeNumber := file.EpisodeNumber
		if episodeNumber <= 0 {
			count++
			continue
		}
		seasonNumber := file.SeasonNumber
		if seasonNumber <= 0 {
			seasonNumber = 1
		}
		key := fmt.Sprintf("%d-%d", seasonNumber, episodeNumber)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		count++
	}
	return count
}

func isGenericEpisodeDisplayTitle(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	re := regexp.MustCompile(`^(第\s*[0-9一二三四五六七八九十百零〇两]+\s*[集期话話]|EP?\s*0*\d+)$`)
	return re.MatchString(strings.ToUpper(value))
}

func isLocalLibraryProxyURL(rawURL string) bool {
	rawURL = strings.TrimSpace(strings.ToLower(rawURL))
	if rawURL == "" {
		return false
	}
	if !(strings.HasPrefix(rawURL, "http://127.0.0.1:") || strings.HasPrefix(rawURL, "http://localhost:")) {
		return false
	}
	return strings.Contains(rawURL, "/image?") || strings.Contains(rawURL, "/avatar?")
}

func looksLikeLocalFilesystemPath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, `\\`) {
		return true
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z]:\\`, value)
	return matched
}

func normalizeLibraryFileSeasonNumbers(detail *library.TitleDetail) {
	if detail == nil || len(detail.Files) == 0 {
		return
	}

	titleSeason := firstPositiveInt(
		parseSeasonHint(detail.Title),
		parseSeasonHint(detail.OriginalTitle),
		parseSeasonHint(externalDataString(detail.ExternalDataJSON, "search_title")),
	)
	if titleSeason <= 1 {
		return
	}

	distinct := map[int]struct{}{}
	for _, file := range detail.Files {
		season := file.SeasonNumber
		if season <= 0 {
			season = 1
		}
		distinct[season] = struct{}{}
	}
	if len(distinct) != 1 {
		return
	}
	if _, ok := distinct[1]; !ok {
		return
	}

	for index := range detail.Files {
		detail.Files[index].SeasonNumber = titleSeason
	}
}

func parseSeasonHint(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	reChinese := regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百零〇两]{1,4})\s*季`)
	if match := reChinese.FindStringSubmatch(value); len(match) >= 2 {
		return parseLooseSeasonNumber(match[1])
	}

	reEnglish := regexp.MustCompile(`(?i)\bseason\s*([0-9]{1,3})\b|\bs\s*([0-9]{1,3})\b`)
	if match := reEnglish.FindStringSubmatch(value); len(match) >= 3 {
		if strings.TrimSpace(match[1]) != "" {
			number, _ := strconv.Atoi(strings.TrimSpace(match[1]))
			return number
		}
		if strings.TrimSpace(match[2]) != "" {
			number, _ := strconv.Atoi(strings.TrimSpace(match[2]))
			return number
		}
	}

	return 0
}

func parseLooseSeasonNumber(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if number, err := strconv.Atoi(value); err == nil {
		return number
	}

	replacer := strings.NewReplacer(
		"零", "0",
		"〇", "0",
		"一", "1",
		"二", "2",
		"两", "2",
		"三", "3",
		"四", "4",
		"五", "5",
		"六", "6",
		"七", "7",
		"八", "8",
		"九", "9",
	)
	if strings.Contains(value, "十") {
		parts := strings.Split(value, "十")
		if len(parts) == 2 {
			left := 1
			right := 0
			if strings.TrimSpace(parts[0]) != "" {
				left = parseLooseSeasonNumber(parts[0])
			}
			if strings.TrimSpace(parts[1]) != "" {
				right = parseLooseSeasonNumber(parts[1])
			}
			return left*10 + right
		}
	}
	number, _ := strconv.Atoi(replacer.Replace(value))
	return number
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

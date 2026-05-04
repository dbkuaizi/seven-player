package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ReplaceLibrary(ctx context.Context, titles []Title, files []File, job ScrapeJobStatus) error {
	if r == nil || r.db == nil {
		return errors.New("library repository unavailable")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM library_files`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM library_titles`); err != nil {
		return err
	}

	titleStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO library_titles (
			id, group_key, section, child, title, original_title, search_title, normalized_title,
			year, rating, summary, director, cast_json, tags_json, poster_url, poster_local_path,
			backdrop_url, backdrop_local_path, quality, source, audio, status, source_name,
			source_subject_id, external_data_json, default_file_id, season_count, episode_count,
			total_duration_sec, created_at, updated_at, scraped_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer titleStmt.Close()

	fileStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO library_files (
			id, title_id, file_id, pick_code, parent_id, path_json, name, size, updated_at,
			duration_sec, season_number, episode_number, episode_title, quality, source, audio,
			search_title, parsed_year, section, child, display_order, created_at, updated_at_row, scraped_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer fileStmt.Close()

	for _, title := range titles {
		castJSON, _ := json.Marshal(nonNilStrings(title.Cast))
		tagsJSON, _ := json.Marshal(nonNilStrings(title.Tags))
		if _, err = titleStmt.ExecContext(ctx,
			emptyIfBlank(title.ID, uuid.NewString()),
			title.GroupKey,
			title.Section,
			title.Child,
			title.Title,
			title.OriginalTitle,
			title.SearchTitle,
			title.NormalizedTitle,
			title.Year,
			title.Rating,
			title.Summary,
			title.Director,
			string(castJSON),
			string(tagsJSON),
			title.PosterURL,
			title.PosterLocalPath,
			title.BackdropURL,
			title.BackdropLocalPath,
			title.Quality,
			title.Source,
			title.Audio,
			title.Status,
			title.SourceName,
			title.SourceSubjectID,
			emptyJSON(title.ExternalDataJSON, "{}"),
			title.DefaultFileID,
			title.SeasonCount,
			title.EpisodeCount,
			title.TotalDurationSec,
			ensureRFC3339(title.CreatedAt),
			ensureRFC3339(title.UpdatedAt),
			ensureRFC3339(title.ScrapedAt),
		); err != nil {
			return err
		}
	}

	for _, file := range files {
		pathJSON, _ := json.Marshal(file.Path)
		if _, err = fileStmt.ExecContext(ctx,
			emptyIfBlank(file.ID, uuid.NewString()),
			file.TitleID,
			file.FileID,
			file.PickCode,
			file.ParentID,
			string(pathJSON),
			file.Name,
			file.Size,
			file.UpdatedAt,
			file.DurationSec,
			file.SeasonNumber,
			file.EpisodeNumber,
			file.EpisodeTitle,
			file.Quality,
			file.Source,
			file.Audio,
			file.SearchTitle,
			file.ParsedYear,
			file.Section,
			file.Child,
			file.DisplayOrder,
			ensureRFC3339(file.CreatedAt),
			ensureRFC3339(file.UpdatedAtRow),
			ensureRFC3339(file.ScrapedAt),
		); err != nil {
			return err
		}
	}

	if err = upsertJobTx(ctx, tx, job); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) UpsertJob(ctx context.Context, job ScrapeJobStatus, settings ScraperSettings) error {
	if r == nil || r.db == nil {
		return errors.New("library repository unavailable")
	}

	settingsJSON, _ := json.Marshal(settings)
	now := nowRFC3339()
	if strings.TrimSpace(job.UpdatedAt) == "" {
		job.UpdatedAt = now
	}
	if strings.TrimSpace(job.StartedAt) == "" {
		job.StartedAt = now
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO scraper_jobs (
			id, status, message, current_path, scanned_directories, total_directories,
			discovered_files, processed_files, matched_items, updated_titles, error_count,
			settings_json, last_error, started_at, finished_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			message = excluded.message,
			current_path = excluded.current_path,
			scanned_directories = excluded.scanned_directories,
			total_directories = excluded.total_directories,
			discovered_files = excluded.discovered_files,
			processed_files = excluded.processed_files,
			matched_items = excluded.matched_items,
			updated_titles = excluded.updated_titles,
			error_count = excluded.error_count,
			settings_json = excluded.settings_json,
			last_error = excluded.last_error,
			started_at = excluded.started_at,
			finished_at = excluded.finished_at,
			updated_at = excluded.updated_at
	`, emptyIfBlank(job.ID, uuid.NewString()), job.Status, job.Message, job.CurrentPath,
		job.ScannedDirectories, job.TotalDirectories, job.DiscoveredFiles, job.ProcessedFiles,
		job.MatchedItems, job.UpdatedTitles, job.ErrorCount, string(settingsJSON), job.LastError,
		ensureRFC3339(job.StartedAt), ensureRFC3339(job.FinishedAt), ensureRFC3339(job.UpdatedAt))
	return err
}

func upsertJobTx(ctx context.Context, tx *sql.Tx, job ScrapeJobStatus) error {
	if strings.TrimSpace(job.ID) == "" {
		job.ID = uuid.NewString()
	}
	if strings.TrimSpace(job.UpdatedAt) == "" {
		job.UpdatedAt = nowRFC3339()
	}
	if strings.TrimSpace(job.StartedAt) == "" {
		job.StartedAt = job.UpdatedAt
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO scraper_jobs (
			id, status, message, current_path, scanned_directories, total_directories,
			discovered_files, processed_files, matched_items, updated_titles, error_count,
			settings_json, last_error, started_at, finished_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '{}', ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			message = excluded.message,
			current_path = excluded.current_path,
			scanned_directories = excluded.scanned_directories,
			total_directories = excluded.total_directories,
			discovered_files = excluded.discovered_files,
			processed_files = excluded.processed_files,
			matched_items = excluded.matched_items,
			updated_titles = excluded.updated_titles,
			error_count = excluded.error_count,
			last_error = excluded.last_error,
			started_at = excluded.started_at,
			finished_at = excluded.finished_at,
			updated_at = excluded.updated_at
	`, job.ID, job.Status, job.Message, job.CurrentPath, job.ScannedDirectories, job.TotalDirectories,
		job.DiscoveredFiles, job.ProcessedFiles, job.MatchedItems, job.UpdatedTitles, job.ErrorCount,
		job.LastError, ensureRFC3339(job.StartedAt), ensureRFC3339(job.FinishedAt), ensureRFC3339(job.UpdatedAt))
	return err
}

func (r *Repository) LoadSnapshot(ctx context.Context, playback map[string]PlaybackInfo) (*LibrarySnapshot, error) {
	items, err := r.ListTitles(ctx, playback)
	if err != nil {
		return nil, err
	}
	return &LibrarySnapshot{
		Sections:  DefaultSections(),
		Items:     items,
		UpdatedAt: nowRFC3339(),
	}, nil
}

type PlaybackInfo struct {
	ResumeMS     int64
	LastPlayedAt string
}

func (r *Repository) ListTitles(ctx context.Context, playback map[string]PlaybackInfo) ([]TitleSummary, error) {
	if r == nil || r.db == nil {
		return []TitleSummary{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			t.id, t.section, t.child, t.title, t.original_title, t.year, t.rating, t.quality,
			t.source, t.audio, t.status, t.poster_url, t.poster_local_path, t.backdrop_url, t.backdrop_local_path, t.summary, t.director, t.cast_json,
			t.external_data_json, t.source_name, t.source_subject_id, t.season_count, t.episode_count, t.default_file_id,
			t.total_duration_sec, f.pick_code, f.name, f.parent_id, f.path_json, f.duration_sec
		FROM library_titles t
		LEFT JOIN library_files f ON f.file_id = t.default_file_id
		ORDER BY
			CASE t.section
				WHEN 'movies' THEN 1
				WHEN 'series' THEN 2
				WHEN 'variety' THEN 3
				WHEN 'anime' THEN 4
				WHEN 'documentary' THEN 5
				ELSE 9
			END,
			t.updated_at DESC,
			t.title COLLATE NOCASE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TitleSummary, 0, 64)
	for rows.Next() {
		var (
			item         TitleSummary
			castJSON     string
			externalJSON string
			pathJSON     string
		)
		if err := rows.Scan(
			&item.ID, &item.Section, &item.Child, &item.Title, &item.OriginalTitle, &item.Year, &item.Rating,
			&item.Quality, &item.Source, &item.Audio, &item.Status, &item.PosterURL, &item.PosterLocalPath, &item.BackdropURL, &item.BackdropLocalPath, &item.Summary,
			&item.Director, &castJSON, &externalJSON, &item.SourceName, &item.SourceSubjectID, &item.SeasonCount,
			&item.EpisodeCount, &item.DefaultFileID, &item.DurationSec, &item.DefaultPickCode,
			&item.DefaultName, &item.DefaultParentID, &pathJSON, &item.DurationSec,
		); err != nil {
			return nil, err
		}
		item.ExternalDataJSON = externalJSON
		_ = json.Unmarshal([]byte(castJSON), &item.Cast)
		item.CastMembers = parseCastMembersJSON(externalJSON)
		_ = json.Unmarshal([]byte(pathJSON), &item.DefaultPath)
		item.PosterTone = inferPosterTone(item.Section, item.Title)
		item.Duration = formatLibraryDuration(item.Section, item.DurationSec, item.SeasonCount, item.EpisodeCount)
		applyPlaybackToSummary(&item, playback[item.DefaultPickCode])
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return filterVisibleTitleSummaries(items), nil
}

func (r *Repository) GetTitleDetail(ctx context.Context, titleID string, playback map[string]PlaybackInfo) (*TitleDetail, error) {
	titleID = strings.TrimSpace(titleID)
	if titleID == "" {
		return nil, errors.New("missing title id")
	}

	snapshot, err := r.ListTitles(ctx, playback)
	if err != nil {
		return nil, err
	}

	var summary *TitleSummary
	for i := range snapshot {
		if snapshot[i].ID == titleID {
			copyValue := snapshot[i]
			summary = &copyValue
			break
		}
	}
	if summary == nil {
		return nil, sql.ErrNoRows
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title_id, file_id, pick_code, parent_id, path_json, name, season_number,
		       episode_number, episode_title, duration_sec, quality, source, audio, size, updated_at
		FROM library_files
		WHERE title_id = ?
		ORDER BY season_number ASC, episode_number ASC, display_order ASC, name COLLATE NOCASE
	`, titleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]EpisodeItem, 0, 32)
	for rows.Next() {
		var (
			item     EpisodeItem
			pathJSON string
		)
		if err := rows.Scan(
			&item.ID, &item.TitleID, &item.FileID, &item.PickCode, &item.ParentID, &pathJSON,
			&item.Name, &item.SeasonNumber, &item.EpisodeNumber, &item.EpisodeTitle, &item.DurationSec,
			&item.Quality, &item.Source, &item.Audio, &item.Size, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(pathJSON), &item.Path)
		item.DurationText = formatSeconds(item.DurationSec)
		item.ThumbnailURL = episodeAssetThumbnailURL(summary.ExternalDataJSON, item.SeasonNumber, item.EpisodeNumber, summary.PosterURL)
		if info, ok := playback[item.PickCode]; ok {
			item.ResumeMS = info.ResumeMS
			item.LastPlayedAt = info.LastPlayedAt
			item.ResumeText = formatResumeText(info.ResumeMS, item.DurationSec)
		}
		files = append(files, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	detail := &TitleDetail{
		TitleSummary: *summary,
		Files:        files,
	}
	return detail, nil
}

func (r *Repository) LastJob(ctx context.Context) (*ScrapeJobStatus, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, status, message, current_path, scanned_directories, total_directories,
		       discovered_files, processed_files, matched_items, updated_titles, error_count,
		       last_error, started_at, finished_at, updated_at
		FROM scraper_jobs
		ORDER BY updated_at DESC
		LIMIT 1
	`)

	var job ScrapeJobStatus
	err := row.Scan(
		&job.ID, &job.Status, &job.Message, &job.CurrentPath, &job.ScannedDirectories,
		&job.TotalDirectories, &job.DiscoveredFiles, &job.ProcessedFiles, &job.MatchedItems,
		&job.UpdatedTitles, &job.ErrorCount, &job.LastError, &job.StartedAt, &job.FinishedAt, &job.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func inferPosterTone(section, title string) string {
	switch section {
	case "movies":
		return "amber"
	case "series":
		return "steel"
	case "variety":
		return "red"
	case "anime":
		return "violet"
	case "documentary":
		return "green"
	}

	runes := []rune(strings.TrimSpace(title))
	if len(runes) == 0 {
		return "steel"
	}
	tones := []string{"amber", "steel", "red", "rose", "green", "violet", "mint", "sky", "ink", "ocean", "cyan", "paper"}
	index := int(runes[0]) % len(tones)
	return tones[index]
}

func formatLibraryDuration(section string, durationSec int64, seasonCount, episodeCount int) string {
	if section == "movies" && durationSec > 0 {
		return formatSeconds(durationSec)
	}
	if seasonCount > 1 {
		return fmt.Sprintf("%d 季", seasonCount)
	}
	if episodeCount > 1 {
		if section == "variety" {
			return fmt.Sprintf("%d 期", episodeCount)
		}
		return fmt.Sprintf("%d 集", episodeCount)
	}
	if durationSec > 0 {
		return formatSeconds(durationSec)
	}
	return ""
}

func applyPlaybackToSummary(item *TitleSummary, playback PlaybackInfo) {
	item.ResumeMS = playback.ResumeMS
	item.LastPlayedAt = playback.LastPlayedAt
	item.ResumeText = formatResumeText(playback.ResumeMS, item.DurationSec)
	if playback.ResumeMS > 0 && item.DurationSec > 0 {
		progress := int((float64(playback.ResumeMS) / float64(item.DurationSec*1000)) * 100)
		if progress < 0 {
			progress = 0
		}
		if progress > 100 {
			progress = 100
		}
		item.Progress = progress
	}
}

func formatResumeText(resumeMS, durationSec int64) string {
	if resumeMS <= 0 {
		return ""
	}
	current := formatSeconds(resumeMS / 1000)
	if durationSec > 0 {
		return current + "/" + formatSeconds(durationSec)
	}
	return current
}

func formatSeconds(seconds int64) string {
	if seconds <= 0 {
		return ""
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

func emptyIfBlank(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func emptyJSON(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func ensureRFC3339(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nowRFC3339()
	}
	if _, err := time.Parse(time.RFC3339, value); err == nil {
		return value
	}
	return nowRFC3339()
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}

func nonNilStrings(source []string) []string {
	if len(source) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(source))
	for _, item := range source {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

func parseCastMembersJSON(externalJSON string) []CastMember {
	if strings.TrimSpace(externalJSON) == "" {
		return []CastMember{}
	}
	var payload struct {
		CastMembers []CastMember `json:"cast_members"`
	}
	if err := json.Unmarshal([]byte(externalJSON), &payload); err != nil {
		return []CastMember{}
	}
	if payload.CastMembers == nil {
		return []CastMember{}
	}
	return payload.CastMembers
}

func episodeAssetThumbnailURL(externalJSON string, seasonNumber, episodeNumber int, fallback string) string {
	assets := parseEpisodeAssetsFromExternalJSON(externalJSON)
	if len(assets) == 0 {
		return fallback
	}

	normalizedSeason := seasonNumber
	if normalizedSeason <= 0 {
		normalizedSeason = 1
	}

	for _, asset := range assets {
		assetSeason := asset.SeasonNumber
		if assetSeason <= 0 {
			assetSeason = 1
		}
		if asset.EpisodeNumber > 0 && asset.EpisodeNumber == episodeNumber && assetSeason == normalizedSeason {
			if strings.TrimSpace(asset.ThumbnailLocalPath) != "" {
				return strings.TrimSpace(asset.ThumbnailLocalPath)
			}
			if strings.TrimSpace(asset.ThumbnailURL) != "" {
				return strings.TrimSpace(asset.ThumbnailURL)
			}
		}
	}
	return fallback
}

func parseEpisodeAssetsFromExternalJSON(externalJSON string) []EpisodeAsset {
	if strings.TrimSpace(externalJSON) == "" {
		return []EpisodeAsset{}
	}
	var payload struct {
		EpisodeAssets []EpisodeAsset `json:"episode_assets"`
	}
	if err := json.Unmarshal([]byte(externalJSON), &payload); err != nil {
		return []EpisodeAsset{}
	}
	if payload.EpisodeAssets == nil {
		return []EpisodeAsset{}
	}
	return payload.EpisodeAssets
}

func filterVisibleTitleSummaries(items []TitleSummary) []TitleSummary {
	if len(items) == 0 {
		return items
	}

	existingTitles := make(map[string]struct{}, len(items))
	for _, item := range items {
		normalized := normalizeTitle(stripSeasonSuffix(strings.TrimSpace(item.Title)))
		if normalized == "" {
			continue
		}
		existingTitles[item.Section+"|"+normalized] = struct{}{}
	}

	filtered := make([]TitleSummary, 0, len(items))
	for _, item := range items {
		if shouldHideStandaloneLibrarySummary(item, existingTitles) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func shouldHideStandaloneLibrarySummary(item TitleSummary, existingTitles map[string]struct{}) bool {
	switch item.Section {
	case "series", "variety", "anime", "documentary":
	default:
		return false
	}

	if item.EpisodeCount > 1 || item.DurationSec <= 0 || item.DurationSec > 20*60 {
		return false
	}
	if len(item.DefaultPath) == 0 {
		return false
	}

	parentName := strings.TrimSpace(item.DefaultPath[len(item.DefaultPath)-1].Name)
	parentNormalized := normalizeTitle(stripSeasonSuffix(buildDirectorySearchTitle(parentName)))
	if parentNormalized == "" {
		return false
	}

	itemNormalized := normalizeTitle(stripSeasonSuffix(strings.TrimSpace(item.Title)))
	if itemNormalized == "" || itemNormalized == parentNormalized {
		return false
	}

	_, exists := existingTitles[item.Section+"|"+parentNormalized]
	return exists
}

package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const tmdbImageBaseURL = "https://image.tmdb.org/t/p/original"

type TMDBProvider struct {
	client          *http.Client
	mu              sync.RWMutex
	readAccessToken string
}

func NewTMDBProvider(client *http.Client) *TMDBProvider {
	if client == nil {
		client = &http.Client{Timeout: 18 * time.Second}
	}
	return &TMDBProvider{client: client}
}

func (p *TMDBProvider) Name() string {
	return "tmdb"
}

func (p *TMDBProvider) SetReadAccessToken(token string) {
	p.mu.Lock()
	p.readAccessToken = strings.TrimSpace(token)
	p.mu.Unlock()
}

func (p *TMDBProvider) Lookup(ctx context.Context, group TitleGroup, language string) (*Metadata, error) {
	query := strings.TrimSpace(group.SearchTitle)
	if query == "" {
		query = strings.TrimSpace(group.BaseTitle)
	}
	if query == "" {
		return nil, nil
	}

	mediaType := p.searchMediaType(group.Section)
	candidates, err := p.search(ctx, mediaType, query, language)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 && group.SeriesTitle != "" && group.SeriesTitle != query {
		candidates, err = p.search(ctx, mediaType, group.SeriesTitle, language)
		if err != nil {
			return nil, err
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	best := chooseBestTMDBResult(group, candidates)
	if best == nil {
		return nil, nil
	}
	return p.LookupByID(ctx, best.EncodedID(), group.BaseTitle, group.SeriesTitle)
}

func (p *TMDBProvider) LookupByID(ctx context.Context, encodedID, title, originalTitle string) (*Metadata, error) {
	mediaType, id, err := parseTMDBEncodedID(encodedID)
	if err != nil {
		return nil, err
	}

	switch mediaType {
	case "movie":
		return p.lookupMovieByID(ctx, id)
	case "tv":
		return p.lookupTVByID(ctx, id)
	default:
		return nil, errUnsupportedProvider
	}
}

type tmdbSearchResult struct {
	ID               int      `json:"id"`
	MediaType        string   `json:"media_type"`
	Title            string   `json:"title"`
	Name             string   `json:"name"`
	OriginalTitle    string   `json:"original_title"`
	OriginalName     string   `json:"original_name"`
	Overview         string   `json:"overview"`
	PosterPath       string   `json:"poster_path"`
	BackdropPath     string   `json:"backdrop_path"`
	ReleaseDate      string   `json:"release_date"`
	FirstAirDate     string   `json:"first_air_date"`
	VoteAverage      float64  `json:"vote_average"`
	GenreIDs         []int    `json:"genre_ids"`
	OriginCountry    []string `json:"origin_country"`
	OriginalLanguage string   `json:"original_language"`
}

func (r tmdbSearchResult) DisplayTitle() string {
	return preferNonEmpty(r.Title, r.Name)
}

func (r tmdbSearchResult) DisplayOriginalTitle() string {
	return preferNonEmpty(r.OriginalTitle, r.OriginalName)
}

func (r tmdbSearchResult) Date() string {
	return preferNonEmpty(r.ReleaseDate, r.FirstAirDate)
}

func (r tmdbSearchResult) EncodedID() string {
	mediaType := strings.TrimSpace(r.MediaType)
	if mediaType == "" {
		if strings.TrimSpace(r.Name) != "" || strings.TrimSpace(r.FirstAirDate) != "" {
			mediaType = "tv"
		} else {
			mediaType = "movie"
		}
	}
	return mediaType + ":" + strconv.Itoa(r.ID)
}

func (p *TMDBProvider) searchMediaType(section string) string {
	switch strings.TrimSpace(section) {
	case "movies":
		return "movie"
	case "series", "anime", "variety", "documentary":
		return "tv"
	default:
		return "multi"
	}
}

func (p *TMDBProvider) search(ctx context.Context, mediaType, query, language string) ([]tmdbSearchResult, error) {
	values := url.Values{}
	values.Set("query", query)
	values.Set("include_adult", "false")
	if strings.TrimSpace(language) != "" {
		values.Set("language", language)
	}

	endpoint := "https://api.themoviedb.org/3/search/" + mediaType + "?" + values.Encode()
	req, err := p.newRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Results []tmdbSearchResult `json:"results"`
	}
	if err := p.doJSON(req, &payload); err != nil {
		return nil, err
	}

	results := make([]tmdbSearchResult, 0, len(payload.Results))
	for _, item := range payload.Results {
		if item.ID <= 0 {
			continue
		}
		if item.MediaType == "person" {
			continue
		}
		if mediaType == "multi" {
			if item.MediaType != "movie" && item.MediaType != "tv" {
				continue
			}
		} else {
			item.MediaType = mediaType
		}
		results = append(results, item)
	}
	return results, nil
}

func chooseBestTMDBResult(group TitleGroup, items []tmdbSearchResult) *tmdbSearchResult {
	if len(items) == 0 {
		return nil
	}

	bestIndex := -1
	bestScore := -999.0
	for index, item := range items {
		score := subjectMatchScore(
			group,
			item.DisplayTitle(),
			item.DisplayOriginalTitle(),
			parseYearFromDate(item.Date()),
			true,
		)

		switch group.Section {
		case "movies":
			if item.MediaType == "movie" {
				score += 1.2
			} else {
				score -= 2.4
			}
		case "series", "anime", "variety", "documentary":
			if item.MediaType == "tv" {
				score += 1.1
			} else {
				score -= 2.2
			}
		}

		if group.Section == "documentary" {
			text := strings.ToLower(item.DisplayTitle() + " " + item.DisplayOriginalTitle() + " " + item.Overview)
			if containsAny(text, "documentary", "纪录", "docuseries", "history", "nature", "science") {
				score += 0.8
			}
		}

		if group.Section == "anime" {
			text := strings.ToLower(item.DisplayTitle() + " " + item.DisplayOriginalTitle())
			if containsAny(text, "anime", "动画", "アニメ") {
				score += 0.4
			}
		}

		if score > bestScore {
			bestIndex = index
			bestScore = score
		}
	}

	if bestIndex < 0 || bestScore < 2.8 {
		return nil
	}
	return &items[bestIndex]
}

type tmdbMovieDetails struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	ReleaseDate   string  `json:"release_date"`
	VoteAverage   float64 `json:"vote_average"`
	Genres        []struct {
		Name string `json:"name"`
	} `json:"genres"`
	Credits struct {
		Cast []struct {
			Name          string `json:"name"`
			ProfilePath   string `json:"profile_path"`
			Character     string `json:"character"`
			KnownForDept  string `json:"known_for_department"`
			Order         int    `json:"order"`
		} `json:"cast"`
		Crew []struct {
			Name        string `json:"name"`
			Job         string `json:"job"`
			Department  string `json:"department"`
			ProfilePath string `json:"profile_path"`
		} `json:"crew"`
	} `json:"credits"`
}

func (p *TMDBProvider) lookupMovieByID(ctx context.Context, id int) (*Metadata, error) {
	values := url.Values{}
	values.Set("append_to_response", "credits")
	endpoint := "https://api.themoviedb.org/3/movie/" + strconv.Itoa(id) + "?" + values.Encode()
	req, err := p.newRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var detail tmdbMovieDetails
	if err := p.doJSON(req, &detail); err != nil {
		return nil, err
	}

	return &Metadata{
		Title:           strings.TrimSpace(detail.Title),
		OriginalTitle:   strings.TrimSpace(detail.OriginalTitle),
		Year:            parseYearFromDate(detail.ReleaseDate),
		Rating:          detail.VoteAverage,
		Summary:         strings.TrimSpace(detail.Overview),
		Director:        tmdbDirectorName(detail.Credits.Crew),
		Cast:            tmdbCastNames(detail.Credits.Cast, 10),
		CastMembers:     tmdbCastMembers(detail.Credits.Cast, 10),
		PosterURL:       tmdbImageURL(detail.PosterPath),
		BackdropURL:     tmdbImageURL(detail.BackdropPath),
		Tags:            tmdbGenreNames(detail.Genres, 8),
		SourceName:      "tmdb",
		SourceSubjectID: "movie:" + strconv.Itoa(detail.ID),
		ExternalData: map[string]any{
			"type":      "tmdb",
			"mediaType": "movie",
			"id":        detail.ID,
		},
	}, nil
}

type tmdbTVDetails struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	OriginalName    string  `json:"original_name"`
	Overview        string  `json:"overview"`
	PosterPath      string  `json:"poster_path"`
	BackdropPath    string  `json:"backdrop_path"`
	FirstAirDate    string  `json:"first_air_date"`
	VoteAverage     float64 `json:"vote_average"`
	NumberOfSeasons int     `json:"number_of_seasons"`
	NumberOfEpisodes int    `json:"number_of_episodes"`
	Genres          []struct {
		Name string `json:"name"`
	} `json:"genres"`
	CreatedBy []struct {
		Name string `json:"name"`
	} `json:"created_by"`
	AggregateCredits struct {
		Cast []struct {
			Name          string `json:"name"`
			ProfilePath   string `json:"profile_path"`
			TotalEpisodeCount int `json:"total_episode_count"`
			Roles []struct {
				Character string `json:"character"`
			} `json:"roles"`
		} `json:"cast"`
		Crew []struct {
			Name        string `json:"name"`
			Department  string `json:"department"`
			Jobs []struct {
				Job string `json:"job"`
			} `json:"jobs"`
		} `json:"crew"`
	} `json:"aggregate_credits"`
	Seasons []struct {
		SeasonNumber int    `json:"season_number"`
		Name         string `json:"name"`
		PosterPath   string `json:"poster_path"`
		EpisodeCount int    `json:"episode_count"`
	} `json:"seasons"`
}

type tmdbSeasonDetails struct {
	ID           int    `json:"id"`
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
	PosterPath   string `json:"poster_path"`
	Episodes     []struct {
		ID            int     `json:"id"`
		EpisodeNumber int     `json:"episode_number"`
		Name          string  `json:"name"`
		Overview      string  `json:"overview"`
		StillPath     string  `json:"still_path"`
		AirDate       string  `json:"air_date"`
		Runtime       int     `json:"runtime"`
		VoteAverage   float64 `json:"vote_average"`
	} `json:"episodes"`
}

func (p *TMDBProvider) lookupTVByID(ctx context.Context, id int) (*Metadata, error) {
	values := url.Values{}
	values.Set("append_to_response", "aggregate_credits")
	endpoint := "https://api.themoviedb.org/3/tv/" + strconv.Itoa(id) + "?" + values.Encode()
	req, err := p.newRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var detail tmdbTVDetails
	if err := p.doJSON(req, &detail); err != nil {
		return nil, err
	}

	external := map[string]any{
		"type":         "tmdb",
		"mediaType":    "tv",
		"id":           detail.ID,
		"season_count": detail.NumberOfSeasons,
		"episode_count": detail.NumberOfEpisodes,
	}

	episodeAssets := make([]EpisodeAsset, 0)
	for _, season := range detail.Seasons {
		if season.SeasonNumber <= 0 {
			continue
		}
		assets, err := p.lookupSeasonAssets(ctx, detail.ID, season.SeasonNumber)
		if err != nil {
			continue
		}
		episodeAssets = append(episodeAssets, assets...)
	}
	if len(episodeAssets) > 0 {
		external["episode_assets"] = episodeAssets
	}

	return &Metadata{
		Title:           strings.TrimSpace(detail.Name),
		OriginalTitle:   strings.TrimSpace(detail.OriginalName),
		Year:            parseYearFromDate(detail.FirstAirDate),
		Rating:          detail.VoteAverage,
		Summary:         strings.TrimSpace(detail.Overview),
		Director:        tmdbTVCreatorName(detail),
		Cast:            tmdbAggregateCastNames(detail.AggregateCredits.Cast, 10),
		CastMembers:     tmdbAggregateCastMembers(detail.AggregateCredits.Cast, 10),
		PosterURL:       tmdbImageURL(detail.PosterPath),
		BackdropURL:     tmdbImageURL(detail.BackdropPath),
		Tags:            tmdbGenreNames(detail.Genres, 8),
		SourceName:      "tmdb",
		SourceSubjectID: "tv:" + strconv.Itoa(detail.ID),
		ExternalData:    external,
	}, nil
}

func (p *TMDBProvider) lookupSeasonAssets(ctx context.Context, tvID, seasonNumber int) ([]EpisodeAsset, error) {
	endpoint := "https://api.themoviedb.org/3/tv/" + strconv.Itoa(tvID) + "/season/" + strconv.Itoa(seasonNumber)
	req, err := p.newRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var detail tmdbSeasonDetails
	if err := p.doJSON(req, &detail); err != nil {
		return nil, err
	}

	assets := make([]EpisodeAsset, 0, len(detail.Episodes))
	for _, episode := range detail.Episodes {
		duration := int64(episode.Runtime) * 60
		title := strings.TrimSpace(episode.Name)
		if title == "" {
			title = fmt.Sprintf("第 %d 集", episode.EpisodeNumber)
		}
		assets = append(assets, EpisodeAsset{
			SeasonNumber:  detail.SeasonNumber,
			EpisodeNumber: episode.EpisodeNumber,
			EpisodeTitle: title,
			ThumbnailURL: tmdbImageURL(episode.StillPath),
			BackdropURL:  tmdbImageURL(episode.StillPath),
			ReleaseDate:  strings.TrimSpace(episode.AirDate),
			DurationSec:  duration,
		})
	}
	return assets, nil
}

func (p *TMDBProvider) newRequest(ctx context.Context, endpoint string) (*http.Request, error) {
	token := p.readToken()
	if token == "" {
		return nil, errors.New("未配置 TMDB Read Access Token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "panplayer115/1.0")
	return req, nil
}

func (p *TMDBProvider) doJSON(req *http.Request, target any) error {
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("tmdb request failed: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (p *TMDBProvider) readToken() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return strings.TrimSpace(p.readAccessToken)
}

func parseTMDBEncodedID(value string) (string, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, errors.New("missing tmdb id")
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", 0, errors.New("invalid tmdb id")
	}
	id, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || id <= 0 {
		return "", 0, errors.New("invalid tmdb numeric id")
	}
	mediaType := strings.ToLower(strings.TrimSpace(parts[0]))
	if mediaType != "movie" && mediaType != "tv" {
		return "", 0, errors.New("invalid tmdb media type")
	}
	return mediaType, id, nil
}

func tmdbImageURL(pathValue string) string {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return ""
	}
	if strings.HasPrefix(pathValue, "http://") || strings.HasPrefix(pathValue, "https://") {
		return pathValue
	}
	if !strings.HasPrefix(pathValue, "/") {
		pathValue = "/" + pathValue
	}
	return tmdbImageBaseURL + pathValue
}

func tmdbGenreNames(genres []struct{ Name string `json:"name"` }, limit int) []string {
	result := make([]string, 0, len(genres))
	for _, genre := range genres {
		name := strings.TrimSpace(genre.Name)
		if name == "" {
			continue
		}
		result = append(result, name)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func tmdbDirectorName(crew []struct {
	Name        string `json:"name"`
	Job         string `json:"job"`
	Department  string `json:"department"`
	ProfilePath string `json:"profile_path"`
}) string {
	for _, item := range crew {
		if strings.EqualFold(strings.TrimSpace(item.Job), "Director") {
			return strings.TrimSpace(item.Name)
		}
	}
	for _, item := range crew {
		if strings.EqualFold(strings.TrimSpace(item.Department), "Directing") {
			return strings.TrimSpace(item.Name)
		}
	}
	return ""
}

func tmdbCastNames(cast []struct {
	Name         string `json:"name"`
	ProfilePath  string `json:"profile_path"`
	Character    string `json:"character"`
	KnownForDept string `json:"known_for_department"`
	Order        int    `json:"order"`
}, limit int) []string {
	result := make([]string, 0, len(cast))
	for _, item := range cast {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		result = append(result, name)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func tmdbCastMembers(cast []struct {
	Name         string `json:"name"`
	ProfilePath  string `json:"profile_path"`
	Character    string `json:"character"`
	KnownForDept string `json:"known_for_department"`
	Order        int    `json:"order"`
}, limit int) []CastMember {
	result := make([]CastMember, 0, len(cast))
	for _, item := range cast {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		result = append(result, CastMember{
			Name:      name,
			AvatarURL: tmdbImageURL(item.ProfilePath),
			Character: normalizeCastCharacter(item.Character),
			Role:      "演员",
		})
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func tmdbAggregateCastNames(cast []struct {
	Name             string `json:"name"`
	ProfilePath      string `json:"profile_path"`
	TotalEpisodeCount int   `json:"total_episode_count"`
	Roles []struct {
		Character string `json:"character"`
	} `json:"roles"`
}, limit int) []string {
	result := make([]string, 0, len(cast))
	for _, item := range cast {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		result = append(result, name)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func tmdbAggregateCastMembers(cast []struct {
	Name             string `json:"name"`
	ProfilePath      string `json:"profile_path"`
	TotalEpisodeCount int   `json:"total_episode_count"`
	Roles []struct {
		Character string `json:"character"`
	} `json:"roles"`
}, limit int) []CastMember {
	result := make([]CastMember, 0, len(cast))
	for _, item := range cast {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		character := ""
		if len(item.Roles) > 0 {
			character = strings.TrimSpace(item.Roles[0].Character)
		}
		result = append(result, CastMember{
			Name:      name,
			AvatarURL: tmdbImageURL(item.ProfilePath),
			Character: normalizeCastCharacter(character),
			Role:      "演员",
		})
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func tmdbTVCreatorName(detail tmdbTVDetails) string {
	if len(detail.CreatedBy) > 0 {
		return strings.TrimSpace(detail.CreatedBy[0].Name)
	}
	for _, item := range detail.AggregateCredits.Crew {
		for _, job := range item.Jobs {
			if strings.EqualFold(strings.TrimSpace(job.Job), "Director") || strings.EqualFold(strings.TrimSpace(job.Job), "Series Director") {
				return strings.TrimSpace(item.Name)
			}
		}
	}
	return ""
}

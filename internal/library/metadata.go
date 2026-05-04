package library

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Metadata struct {
	Title           string
	OriginalTitle   string
	Year            int
	Rating          float64
	Summary         string
	Director        string
	Cast            []string
	CastMembers     []CastMember
	PosterURL       string
	BackdropURL     string
	Tags            []string
	SourceName      string
	SourceSubjectID string
	ExternalData    map[string]any
}

type EpisodeAsset struct {
	FileID        string `json:"fileId,omitempty"`
	FileName      string `json:"fileName,omitempty"`
	SeasonNumber  int    `json:"seasonNumber,omitempty"`
	EpisodeNumber int    `json:"episodeNumber,omitempty"`
	EpisodeTitle  string `json:"episodeTitle,omitempty"`
	ThumbnailURL  string `json:"thumbnailUrl,omitempty"`
	ThumbnailLocalPath string `json:"thumbnailLocalPath,omitempty"`
	BackdropURL   string `json:"backdropUrl,omitempty"`
	BackdropLocalPath string `json:"backdropLocalPath,omitempty"`
	SourceVID     string `json:"sourceVid,omitempty"`
	ReleaseDate   string `json:"releaseDate,omitempty"`
	DurationSec   int64  `json:"durationSec,omitempty"`
	CategoryValue string `json:"categoryValue,omitempty"`
}

type DoubanSuggestion struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SubTitle string `json:"sub_title"`
	Year     string `json:"year"`
	Type     string `json:"type"`
	Episode  string `json:"episode"`
	Img      string `json:"img"`
	URL      string `json:"url"`
	IsTV     bool   `json:"is_tv"`
}

type MetadataProvider interface {
	Name() string
	Lookup(ctx context.Context, group TitleGroup, language string) (*Metadata, error)
}

type MetadataResolver struct {
	client    *http.Client
	providers map[string]MetadataProvider
	tencent   *TencentVideoProvider
}

func NewMetadataResolver() *MetadataResolver {
	client := &http.Client{Timeout: 18 * time.Second}
	tencent := &TencentVideoProvider{client: client}
	tmdb := NewTMDBProvider(client)
	return &MetadataResolver{
		client:  client,
		tencent: tencent,
		providers: map[string]MetadataProvider{
			"tmdb":    tmdb,
			"bangumi": &BangumiProvider{client: client},
			"douban":  &DoubanProvider{client: client},
		},
	}
}

func (r *MetadataResolver) SetTMDBReadAccessToken(token string) {
	if r == nil {
		return
	}
	provider, _ := r.providers["tmdb"].(*TMDBProvider)
	if provider == nil {
		return
	}
	provider.SetReadAccessToken(token)
}

func (r *MetadataResolver) Resolve(ctx context.Context, group TitleGroup, orderedSources []string, language string) (*Metadata, error) {
	if r == nil {
		return nil, nil
	}
	group = normalizeLookupGroup(group)
	sources := normalizeMetadataSourceOrder(orderedSources)
	var resolved *Metadata
	var lastErr error
	for _, source := range sources {
		provider := r.providers[strings.ToLower(strings.TrimSpace(source))]
		if provider == nil {
			continue
		}
		metadata, err := provider.Lookup(ctx, group, language)
		if err == nil && metadata != nil {
			if resolved == nil {
				resolved = metadata
			} else {
				mergeMetadata(resolved, metadata)
			}
			continue
		}
		if err != nil {
			lastErr = err
		}
	}
	if resolved != nil {
		if r.tencent != nil {
			_ = r.tencent.EnrichMetadata(ctx, resolved)
		}
		return resolved, nil
	}
	return nil, lastErr
}

func normalizeLookupGroup(group TitleGroup) TitleGroup {
	group.SearchTitle = strings.TrimSpace(group.SearchTitle)
	group.BaseTitle = strings.TrimSpace(group.BaseTitle)
	group.SeriesTitle = strings.TrimSpace(group.SeriesTitle)
	if group.SearchTitle == "" {
		group.SearchTitle = group.BaseTitle
	}
	group.SearchTitle = cleanSearchTitle(group.SearchTitle, group.Section)
	if group.BaseTitle == "" {
		group.BaseTitle = group.SearchTitle
	} else {
		group.BaseTitle = cleanSearchTitle(group.BaseTitle, group.Section)
	}
	if group.SeriesTitle != "" {
		group.SeriesTitle = cleanSearchTitle(group.SeriesTitle, group.Section)
	}
	group.Normalized = normalizeTitle(group.SearchTitle)
	return group
}

func normalizeMetadataSourceOrder(sources []string) []string {
	seen := map[string]struct{}{}
	ordered := make([]string, 0, len(sources)+3)

	for _, source := range sources {
		value := strings.ToLower(strings.TrimSpace(source))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ordered = append(ordered, value)
	}

	if len(ordered) == 0 {
		ordered = append(ordered, "tmdb", "douban", "bangumi")
	}
	return ordered
}

func mergeMetadata(base *Metadata, patch *Metadata) {
	if base == nil || patch == nil {
		return
	}
	if strings.TrimSpace(base.Title) == "" {
		base.Title = strings.TrimSpace(patch.Title)
	}
	if strings.TrimSpace(base.OriginalTitle) == "" {
		base.OriginalTitle = strings.TrimSpace(patch.OriginalTitle)
	}
	if base.Year == 0 {
		base.Year = patch.Year
	}
	if base.Rating <= 0 && patch.Rating > 0 {
		base.Rating = patch.Rating
	}
	if strings.TrimSpace(base.Summary) == "" {
		base.Summary = strings.TrimSpace(patch.Summary)
	}
	if strings.TrimSpace(base.Director) == "" {
		base.Director = strings.TrimSpace(patch.Director)
	}
	if len(base.Cast) == 0 && len(patch.Cast) > 0 {
		base.Cast = append([]string(nil), patch.Cast...)
	}
	if len(base.CastMembers) == 0 && len(patch.CastMembers) > 0 {
		base.CastMembers = append([]CastMember(nil), patch.CastMembers...)
	}
	if strings.TrimSpace(base.PosterURL) == "" {
		base.PosterURL = strings.TrimSpace(patch.PosterURL)
	}
	if strings.TrimSpace(base.BackdropURL) == "" {
		base.BackdropURL = strings.TrimSpace(patch.BackdropURL)
	}
	if len(base.Tags) == 0 && len(patch.Tags) > 0 {
		base.Tags = append([]string(nil), patch.Tags...)
	}
	if strings.TrimSpace(base.SourceName) == "" {
		base.SourceName = strings.TrimSpace(patch.SourceName)
	}
	if strings.TrimSpace(base.SourceSubjectID) == "" {
		base.SourceSubjectID = strings.TrimSpace(patch.SourceSubjectID)
	}
	if len(base.ExternalData) == 0 && len(patch.ExternalData) > 0 {
		base.ExternalData = copyExternalData(patch.ExternalData)
		return
	}
	for key, value := range patch.ExternalData {
		if _, exists := base.ExternalData[key]; exists {
			continue
		}
		base.ExternalData[key] = value
	}
}

func copyExternalData(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	target := make(map[string]any, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

func (r *MetadataResolver) ResolveBySourceID(ctx context.Context, sourceName, sourceSubjectID, title, originalTitle string) (*Metadata, error) {
	if r == nil {
		return nil, nil
	}
	sourceName = strings.ToLower(strings.TrimSpace(sourceName))
	sourceSubjectID = strings.TrimSpace(sourceSubjectID)
	if sourceName == "" || sourceSubjectID == "" {
		return nil, nil
	}

	switch sourceName {
	case "tmdb":
		provider, _ := r.providers["tmdb"].(*TMDBProvider)
		if provider == nil {
			return nil, errUnsupportedProvider
		}
		meta, err := provider.LookupByID(ctx, sourceSubjectID, title, originalTitle)
		if err != nil || meta == nil {
			return meta, err
		}
		return meta, nil
	case "douban":
		provider, _ := r.providers["douban"].(*DoubanProvider)
		if provider == nil {
			return nil, errUnsupportedProvider
		}
		meta, err := provider.fetchRexxarSubject(ctx, sourceSubjectID, DoubanSuggestion{
			ID:       sourceSubjectID,
			Title:    strings.TrimSpace(title),
			SubTitle: strings.TrimSpace(originalTitle),
		})
		if err != nil || meta == nil {
			return meta, err
		}
		if r.tencent != nil {
			_ = r.tencent.EnrichMetadata(ctx, meta)
		}
		return meta, nil
	default:
		return nil, errUnsupportedProvider
	}
}

type BangumiProvider struct {
	client *http.Client
}

func (p *BangumiProvider) Name() string {
	return "bangumi"
}

func (p *BangumiProvider) Lookup(ctx context.Context, group TitleGroup, language string) (*Metadata, error) {
	query := strings.TrimSpace(group.SearchTitle)
	if query == "" {
		query = strings.TrimSpace(group.BaseTitle)
	}
	if query == "" {
		return nil, nil
	}

	typeFilter := []int{2, 6}
	if group.Section == "anime" {
		typeFilter = []int{2}
	}

	body, _ := json.Marshal(map[string]any{
		"keyword": query,
		"sort":    "match",
		"filter": map[string]any{
			"type": typeFilter,
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.bgm.tv/v0/search/subjects", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "panplayer115/1.0 (https://cnb.cool/dbkuaizi/115play)")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("bangumi search failed: %s", resp.Status)
	}

	var payload struct {
		Data []struct {
			ID     int               `json:"id"`
			Name   string            `json:"name"`
			NameCN string            `json:"name_cn"`
			Date   string            `json:"date"`
			Images map[string]string `json:"images"`
			Rating struct {
				Score float64 `json:"score"`
			} `json:"rating"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Data) == 0 {
		return nil, nil
	}

	best := payload.Data[0]
	bestScore := subjectMatchScore(group, best.NameCN, best.Name, parseYearFromDate(best.Date), true)
	for _, candidate := range payload.Data[1:] {
		score := subjectMatchScore(group, candidate.NameCN, candidate.Name, parseYearFromDate(candidate.Date), true)
		if score > bestScore {
			best = candidate
			bestScore = score
		}
	}
	if bestScore < 3.3 {
		return nil, nil
	}

	detailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.bgm.tv/v0/subjects/"+strconv.Itoa(best.ID), nil)
	if err != nil {
		return nil, err
	}
	detailReq.Header.Set("User-Agent", "panplayer115/1.0 (https://cnb.cool/dbkuaizi/115play)")
	detailReq.Header.Set("Accept", "application/json")
	detailResp, err := p.client.Do(detailReq)
	if err != nil {
		return nil, err
	}
	defer detailResp.Body.Close()
	if detailResp.StatusCode >= 400 {
		return nil, fmt.Errorf("bangumi detail failed: %s", detailResp.Status)
	}

	var detail struct {
		ID      int               `json:"id"`
		Name    string            `json:"name"`
		NameCN  string            `json:"name_cn"`
		Date    string            `json:"date"`
		Summary string            `json:"summary"`
		Images  map[string]string `json:"images"`
		Tags    []struct {
			Name string `json:"name"`
		} `json:"tags"`
		Rating struct {
			Score float64 `json:"score"`
		} `json:"rating"`
		Infobox []struct {
			Key   string `json:"key"`
			Value any    `json:"value"`
		} `json:"infobox"`
	}
	if err := json.NewDecoder(detailResp.Body).Decode(&detail); err != nil {
		return nil, err
	}

	director := ""
	cast := make([]string, 0, 4)
	for _, item := range detail.Infobox {
		key := strings.TrimSpace(item.Key)
		switch key {
		case "导演", "监督", "导演/脚本", "原作", "系列构成":
			if director == "" {
				director = infoboxValueString(item.Value)
			}
		case "中文名", "主演", "角色声优", "主要声优":
			if len(cast) == 0 {
				cast = append(cast, splitNames(infoboxValueString(item.Value))...)
			}
		}
	}

	tags := make([]string, 0, 6)
	for _, tag := range detail.Tags {
		if strings.TrimSpace(tag.Name) == "" {
			continue
		}
		tags = append(tags, strings.TrimSpace(tag.Name))
		if len(tags) == 6 {
			break
		}
	}

	external := map[string]any{
		"type": "bangumi",
		"id":   detail.ID,
	}
	return &Metadata{
		Title:           preferNonEmpty(detail.NameCN, detail.Name, group.BaseTitle),
		OriginalTitle:   preferNonEmpty(detail.Name, detail.NameCN),
		Year:            parseYearFromDate(detail.Date),
		Rating:          detail.Rating.Score,
		Summary:         strings.TrimSpace(detail.Summary),
		Director:        director,
		Cast:            uniqueFirst(cast, 8),
		CastMembers:     castMembersFromNames(uniqueFirst(cast, 8)),
		PosterURL:       preferNonEmpty(detail.Images["large"], detail.Images["common"], detail.Images["medium"]),
		BackdropURL:     "",
		Tags:            tags,
		SourceName:      "Bangumi",
		SourceSubjectID: strconv.Itoa(detail.ID),
		ExternalData:    external,
	}, nil
}

type DoubanProvider struct {
	client *http.Client
}

func (p *DoubanProvider) Name() string {
	return "douban"
}

func (p *DoubanProvider) Lookup(ctx context.Context, group TitleGroup, language string) (*Metadata, error) {
	query := strings.TrimSpace(group.SearchTitle)
	if query == "" {
		query = strings.TrimSpace(group.BaseTitle)
	}
	if query == "" {
		return nil, nil
	}

	endpoint := "https://movie.douban.com/j/subject_suggest?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 panplayer115/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("douban suggest failed: %s", resp.Status)
	}

	var suggestions []DoubanSuggestion
	if err := json.NewDecoder(resp.Body).Decode(&suggestions); err != nil {
		return nil, err
	}
	if len(suggestions) == 0 {
		suggestions, err = p.searchSubjects(ctx, query)
		if err != nil {
			return nil, err
		}
	}
	if len(suggestions) == 0 {
		return nil, nil
	}

	best := suggestions[0]
	bestScore := subjectMatchScore(group, best.Title, best.SubTitle, parseLooseNumber(best.Year), false) + doubanSuggestionTypeScore(group, best)
	for _, candidate := range suggestions[1:] {
		score := subjectMatchScore(group, candidate.Title, candidate.SubTitle, parseLooseNumber(candidate.Year), false) + doubanSuggestionTypeScore(group, candidate)
		if score > bestScore {
			best = candidate
			bestScore = score
		}
	}
	if bestScore < 3.1 {
		return nil, nil
	}

	meta, err := p.fetchSubjectPage(ctx, best.URL, best)
	if err == nil && metadataLooksUseful(meta) {
		return meta, nil
	}

	rexxarMeta, rexxarErr := p.fetchRexxarSubject(ctx, best.ID, best)
	if rexxarErr == nil && metadataLooksUseful(rexxarMeta) {
		return rexxarMeta, nil
	}

	if err == nil && meta != nil {
		return meta, nil
	}
	if rexxarErr == nil && rexxarMeta != nil {
		return rexxarMeta, nil
	}
	return &Metadata{
		Title:           preferNonEmpty(best.Title, group.BaseTitle),
		OriginalTitle:   best.SubTitle,
		Year:            parseLooseNumber(best.Year),
		PosterURL:       best.Img,
		SourceName:      "Douban",
		SourceSubjectID: best.ID,
		ExternalData: map[string]any{
			"type": "douban",
			"id":   best.ID,
		},
	}, nil
}

func (p *DoubanProvider) searchSubjects(ctx context.Context, query string) ([]DoubanSuggestion, error) {
	endpoint := "https://search.douban.com/movie/subject_search?search_text=" + url.QueryEscape(query) + "&cat=1002"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 panplayer115/1.0")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("douban subject search failed: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	payload := parseDoubanSearchItems(string(body))
	if len(payload) == 0 {
		return nil, nil
	}
	return payload, nil
}

func (p *DoubanProvider) fetchSubjectPage(ctx context.Context, rawURL string, item DoubanSuggestion) (*Metadata, error) {
	subjectURL := normalizeDoubanSubjectURL(rawURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, subjectURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 panplayer115/1.0")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("douban subject page failed: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, err
	}
	html := string(body)

	title, originalTitle := parseDoubanTitle(html, item.Title, item.SubTitle)
	year := parseDoubanYear(html)
	if year == 0 {
		year = parseLooseNumber(item.Year)
	}
	rating := parseDoubanRating(html)
	summary := parseDoubanSummary(html)
	director := parseDoubanField(html, "导演")
	cast := parseDoubanActors(html)
	tags := parseDoubanTags(html)

	return &Metadata{
		Title:           preferNonEmpty(title, item.Title),
		OriginalTitle:   preferNonEmpty(originalTitle, item.SubTitle),
		Year:            year,
		Rating:          rating,
		Summary:         summary,
		Director:        director,
		Cast:            uniqueFirst(cast, 8),
		CastMembers:     castMembersFromNames(uniqueFirst(cast, 8)),
		PosterURL:       item.Img,
		BackdropURL:     "",
		Tags:            uniqueFirst(tags, 8),
		SourceName:      "Douban",
		SourceSubjectID: item.ID,
		ExternalData: map[string]any{
			"type": "douban",
			"id":   item.ID,
		},
	}, nil
}

func (p *DoubanProvider) fetchRexxarSubject(ctx context.Context, subjectID string, item DoubanSuggestion) (*Metadata, error) {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return nil, errors.New("missing douban subject id")
	}

	endpoints := []string{
		"https://m.douban.com/rexxar/api/v2/tv/" + subjectID + "?ck=&for_mobile=1",
		"https://m.douban.com/rexxar/api/v2/movie/" + subjectID + "?ck=&for_mobile=1",
	}

	var lastErr error
	for _, endpoint := range endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 panplayer115/1.0")
		req.Header.Set("Referer", "https://m.douban.com/")
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("douban rexxar failed: %s", resp.Status)
			resp.Body.Close()
			continue
		}

		var payload struct {
			Title         string `json:"title"`
			OriginalTitle string `json:"original_title"`
			Year          string `json:"year"`
			Intro         string `json:"intro"`
			Pic           struct {
				Large  string `json:"large"`
				Normal string `json:"normal"`
			} `json:"pic"`
			Rating struct {
				Value float64 `json:"value"`
			} `json:"rating"`
			Genres []string `json:"genres"`
			Aka    []string `json:"aka"`
			Actors []struct {
				Name string `json:"name"`
			} `json:"actors"`
			Directors []struct {
				Name string `json:"name"`
			} `json:"directors"`
			EpisodesCount int    `json:"episodes_count"`
			CardSubtitle  string `json:"card_subtitle"`
			Cover         struct {
				Image struct {
					Large struct {
						URL string `json:"url"`
					} `json:"large"`
					Normal struct {
						URL string `json:"url"`
					} `json:"normal"`
					Small struct {
						URL string `json:"url"`
					} `json:"small"`
				} `json:"image"`
			} `json:"cover"`
			Vendors []struct {
				ID      string `json:"id"`
				Title   string `json:"title"`
				URI     string `json:"uri"`
				URL     string `json:"url"`
				BGImage string `json:"bg_image"`
			} `json:"vendors"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		if decodeErr != nil {
			lastErr = decodeErr
			continue
		}

		director := ""
		if len(payload.Directors) > 0 {
			director = strings.TrimSpace(payload.Directors[0].Name)
		}
		cast := make([]string, 0, len(payload.Actors))
		for _, actor := range payload.Actors {
			if strings.TrimSpace(actor.Name) == "" {
				continue
			}
			cast = append(cast, strings.TrimSpace(actor.Name))
		}
		castMembers, creditsErr := p.fetchRexxarCredits(ctx, subjectID, endpoints[0] == endpoint)
		if creditsErr == nil && len(castMembers) > 0 {
			cast = make([]string, 0, len(castMembers))
			for _, member := range castMembers {
				cast = append(cast, member.Name)
			}
		}
		tags := uniqueFirst(append([]string{}, payload.Genres...), 8)
		if len(tags) == 0 {
			tags = uniqueFirst(payload.Aka, 8)
		}
		backdropURL := preferNonEmpty(
			payload.Cover.Image.Large.URL,
			payload.Cover.Image.Normal.URL,
			firstVendorBackgroundFromRichVendor(payload.Vendors),
			item.Img,
		)
		posterURL := preferNonEmpty(
			payload.Pic.Large,
			payload.Pic.Normal,
			payload.Cover.Image.Normal.URL,
			payload.Cover.Image.Small.URL,
			item.Img,
		)

		external := map[string]any{
			"type":           "douban",
			"id":             subjectID,
			"source":         "rexxar",
			"episodes_count": payload.EpisodesCount,
			"card_subtitle":  payload.CardSubtitle,
		}
		if cid, vid := firstTencentVendorTarget(payload.Vendors); cid != "" {
			external["tencent_cid"] = cid
			if vid != "" {
				external["tencent_vid"] = vid
			}
		}

		return &Metadata{
			Title:           preferNonEmpty(payload.Title, item.Title),
			OriginalTitle:   preferNonEmpty(payload.OriginalTitle, item.SubTitle),
			Year:            parseLooseNumber(payload.Year),
			Rating:          payload.Rating.Value,
			Summary:         strings.TrimSpace(payload.Intro),
			Director:        director,
			Cast:            uniqueFirst(cast, 8),
			CastMembers:     firstCastMembers(castMembers, 8),
			PosterURL:       posterURL,
			BackdropURL:     backdropURL,
			Tags:            tags,
			SourceName:      "Douban",
			SourceSubjectID: subjectID,
			ExternalData:    external,
		}, nil
	}

	if lastErr == nil {
		lastErr = errors.New("douban rexxar unavailable")
	}
	return nil, lastErr
}

func subjectMatchScore(group TitleGroup, title, originalTitle string, year int, allowOriginalBonus bool) float64 {
	group = normalizeLookupGroup(group)
	base := normalizeTitle(group.SearchTitle)
	if base == "" {
		base = normalizeTitle(group.BaseTitle)
	}
	score := 0.0
	normalizedTitle := normalizeTitle(title)
	normalizedOriginal := normalizeTitle(originalTitle)

	switch {
	case normalizedTitle == base:
		score += 4.5
	case strings.Contains(normalizedTitle, base) || strings.Contains(base, normalizedTitle):
		score += 3.6
	}
	if allowOriginalBonus {
		switch {
		case normalizedOriginal == base:
			score += 2.2
		case normalizedOriginal != "" && (strings.Contains(normalizedOriginal, base) || strings.Contains(base, normalizedOriginal)):
			score += 1.4
		}
	}
	if group.Year > 0 && year > 0 {
		diff := group.Year - year
		if diff < 0 {
			diff = -diff
		}
		switch diff {
		case 0:
			score += 1.5
		case 1:
			score += 0.8
		case 2:
			score += 0.3
		}
	}
	if group.Section == "anime" && containsAny(strings.ToLower(title+" "+originalTitle), "剧场", "movie", "ova") {
		if group.Child == "剧场版" {
			score += 0.8
		} else {
			score -= 0.4
		}
	}
	if group.Section == "variety" {
		text := title + " " + originalTitle
		if containsAny(strings.ToLower(text), "第", "季", "五哈", "哈哈哈哈哈") {
			score += 0.6
		}
	}
	if group.Section == "movies" {
		if containsAny(strings.ToLower(title+" "+originalTitle), "第", "季", "season", "tv", "电视剧", "剧集") {
			score -= 1.5
		}
	}
	if group.Section == "series" || group.Section == "documentary" {
		if containsAny(strings.ToLower(title+" "+originalTitle), "movie", "电影", "剧场版") {
			score -= 1.1
		}
	}
	return score
}

func doubanSuggestionTypeScore(group TitleGroup, suggestion DoubanSuggestion) float64 {
	kind := strings.ToLower(strings.TrimSpace(suggestion.Type))
	isTV := suggestion.IsTV || kind == "tv" || kind == "电视剧" || strings.Contains(kind, "tv")
	switch group.Section {
	case "movies":
		if isTV {
			return -1.8
		}
		return 0.4
	case "series", "variety", "anime", "documentary":
		if isTV {
			return 0.7
		}
		if kind == "movie" || kind == "电影" {
			return -1.4
		}
	}
	return 0
}

func infoboxValueString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			switch child := item.(type) {
			case string:
				if strings.TrimSpace(child) != "" {
					parts = append(parts, strings.TrimSpace(child))
				}
			case map[string]any:
				if raw, ok := child["v"]; ok {
					if text := strings.TrimSpace(fmt.Sprint(raw)); text != "" {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, " / ")
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func splitNames(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '/' || r == '／' || r == ',' || r == '，' || r == '|' || r == '、'
	})
	result := make([]string, 0, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func uniqueFirst(items []string, limit int) []string {
	if len(items) == 0 {
		return []string{}
	}
	if limit <= 0 {
		limit = len(items)
	}
	result := make([]string, 0, min(limit, len(items)))
	seen := map[string]struct{}{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
		if len(result) == limit {
			break
		}
	}
	return result
}

func normalizeDoubanSubjectURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parsed.RawQuery = ""
	return parsed.String()
}

var (
	reDoubanTitle       = regexp.MustCompile(`(?is)<title>\s*([^<]+?)\s*</title>`)
	reDoubanYear        = regexp.MustCompile(`(?is)<span[^>]*class="year"[^>]*>\s*\((\d{4})\)\s*</span>`)
	reDoubanRating      = regexp.MustCompile(`(?is)<strong[^>]*class="ll rating_num"[^>]*property="v:average"[^>]*>\s*([0-9.]+)\s*</strong>`)
	reDoubanSummary     = regexp.MustCompile(`(?is)<span[^>]*property="v:summary"[^>]*>(.*?)</span>`)
	reDoubanField       = regexp.MustCompile(`(?is)<span[^>]*class="pl"[^>]*>\s*%s\s*</span>\s*(?:</span>)?\s*(.*?)<br`)
	reDoubanCelebrityA  = regexp.MustCompile(`(?is)<a[^>]+rel="v:directedBy"[^>]*>(.*?)</a>`)
	reDoubanActorA      = regexp.MustCompile(`(?is)<a[^>]+rel="v:starring"[^>]*>(.*?)</a>`)
	reDoubanTagLinks    = regexp.MustCompile(`(?is)<a[^>]+href="https://movie\.douban\.com/tag/[^"]+"[^>]*>(.*?)</a>`)
	reDoubanSearchJSON  = regexp.MustCompile(`window\.__DATA__\s*=\s*(\{.*?\})\s*;`)
	reHTMLTags          = regexp.MustCompile(`(?is)<[^>]+>`)
	reHTMLSpaceCollapse = regexp.MustCompile(`\s+`)
)

func parseDoubanSearchItems(html string) []DoubanSuggestion {
	match := reDoubanSearchJSON.FindStringSubmatch(html)
	if len(match) < 2 {
		return nil
	}

	var payload struct {
		Items []struct {
			ID       int    `json:"id"`
			Title    string `json:"title"`
			Abstract string `json:"abstract"`
			CoverURL string `json:"cover_url"`
			URL      string `json:"url"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(match[1]), &payload); err != nil {
		return nil
	}

	result := make([]DoubanSuggestion, 0, len(payload.Items))
	for _, item := range payload.Items {
		title, year := splitDoubanSearchTitle(item.Title)
		if strings.TrimSpace(title) == "" {
			continue
		}
		result = append(result, DoubanSuggestion{
			ID:       strconv.Itoa(item.ID),
			Title:    title,
			SubTitle: "",
			Year:     year,
			Img:      item.CoverURL,
			URL:      item.URL,
		})
	}
	return result
}

func splitDoubanSearchTitle(value string) (string, string) {
	value = strings.TrimSpace(strings.TrimSuffix(value, "‎"))
	if value == "" {
		return "", ""
	}
	index := strings.LastIndex(value, "(")
	if index <= 0 || !strings.HasSuffix(value, ")") {
		return value, ""
	}
	title := strings.TrimSpace(value[:index])
	year := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value[index:], "("), ")"))
	if len(year) == 4 && parseLooseNumber(year) > 1900 {
		return title, year
	}
	return value, ""
}

func metadataLooksUseful(meta *Metadata) bool {
	if meta == nil {
		return false
	}
	return strings.TrimSpace(meta.Summary) != "" ||
		strings.TrimSpace(meta.Director) != "" ||
		len(meta.Cast) > 0 ||
		meta.Rating > 0
}

func (p *DoubanProvider) fetchRexxarCredits(ctx context.Context, subjectID string, isTV bool) ([]CastMember, error) {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return nil, errors.New("missing douban subject id")
	}

	subjectType := "movie"
	if isTV {
		subjectType = "tv"
	}
	endpoint := fmt.Sprintf("https://m.douban.com/rexxar/api/v2/%s/%s/celebrities?start=0&count=30&ck=&for_mobile=1", subjectType, subjectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 panplayer115/1.0")
	req.Header.Set("Referer", "https://m.douban.com/")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("douban celebrities failed: %s", resp.Status)
	}

	var payload struct {
		Actors []struct {
			Name      string `json:"name"`
			Character string `json:"character"`
			Avatar    struct {
				Large  string `json:"large"`
				Normal string `json:"normal"`
			} `json:"avatar"`
			Roles []string `json:"roles"`
		} `json:"actors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	result := make([]CastMember, 0, len(payload.Actors))
	for _, actor := range payload.Actors {
		name := strings.TrimSpace(actor.Name)
		if name == "" {
			continue
		}
		result = append(result, CastMember{
			Name:      name,
			AvatarURL: preferNonEmpty(actor.Avatar.Large, actor.Avatar.Normal),
			Character: normalizeCastCharacter(actor.Character),
			Role:      firstNonEmpty(actor.Roles...),
		})
	}
	return result, nil
}

func castMembersFromNames(names []string) []CastMember {
	if len(names) == 0 {
		return []CastMember{}
	}
	result := make([]CastMember, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		result = append(result, CastMember{Name: name})
	}
	return result
}

func firstCastMembers(items []CastMember, limit int) []CastMember {
	if len(items) == 0 {
		return []CastMember{}
	}
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	result := make([]CastMember, 0, limit)
	seen := map[string]struct{}{}
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, CastMember{
			Name:      name,
			AvatarURL: strings.TrimSpace(item.AvatarURL),
			Character: strings.TrimSpace(item.Character),
			Role:      strings.TrimSpace(item.Role),
		})
		if len(result) == limit {
			break
		}
	}
	return result
}

func normalizeCastCharacter(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "自己")
	value = strings.TrimPrefix(value, "Self")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return value
}

func firstVendorBackground(vendors []struct {
	BGImage string `json:"bg_image"`
}) string {
	for _, vendor := range vendors {
		if strings.TrimSpace(vendor.BGImage) != "" {
			return strings.TrimSpace(vendor.BGImage)
		}
	}
	return ""
}

func firstVendorBackgroundFromRichVendor(vendors []struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	URI     string `json:"uri"`
	URL     string `json:"url"`
	BGImage string `json:"bg_image"`
}) string {
	for _, vendor := range vendors {
		if strings.TrimSpace(vendor.BGImage) != "" {
			return strings.TrimSpace(vendor.BGImage)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseDoubanTitle(html, fallbackTitle, fallbackOriginal string) (string, string) {
	match := reDoubanTitle.FindStringSubmatch(html)
	if len(match) < 2 {
		return fallbackTitle, fallbackOriginal
	}
	titleText := cleanHTMLText(match[1])
	titleText = strings.TrimSuffix(titleText, "豆瓣")
	titleText = strings.TrimSpace(strings.TrimSuffix(titleText, "-"))
	if titleText == "" {
		return fallbackTitle, fallbackOriginal
	}

	if index := strings.LastIndex(titleText, " "); index > 0 {
		main := strings.TrimSpace(titleText[:index])
		yearText := strings.TrimSpace(titleText[index+1:])
		if len(yearText) == 4 && parseLooseNumber(yearText) > 1900 {
			titleText = main
		}
	}
	if strings.Contains(titleText, "/") {
		parts := strings.Split(titleText, "/")
		title := strings.TrimSpace(parts[0])
		original := ""
		for _, part := range parts[1:] {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if part != title {
				original = part
				break
			}
		}
		return preferNonEmpty(title, fallbackTitle), preferNonEmpty(original, fallbackOriginal)
	}
	return preferNonEmpty(titleText, fallbackTitle), fallbackOriginal
}

func parseDoubanYear(html string) int {
	match := reDoubanYear.FindStringSubmatch(html)
	if len(match) < 2 {
		return 0
	}
	value, _ := strconv.Atoi(match[1])
	return value
}

func parseDoubanRating(html string) float64 {
	match := reDoubanRating.FindStringSubmatch(html)
	if len(match) < 2 {
		return 0
	}
	value, _ := strconv.ParseFloat(strings.TrimSpace(match[1]), 64)
	return value
}

func parseDoubanSummary(html string) string {
	match := reDoubanSummary.FindStringSubmatch(html)
	if len(match) < 2 {
		return ""
	}
	return cleanHTMLText(match[1])
}

func parseDoubanField(html, label string) string {
	pattern := regexp.MustCompile(fmt.Sprintf(reDoubanField.String(), regexp.QuoteMeta(label)))
	match := pattern.FindStringSubmatch(html)
	if len(match) < 2 {
		return ""
	}
	return cleanHTMLText(match[1])
}

func parseDoubanActors(html string) []string {
	matches := reDoubanActorA.FindAllStringSubmatch(html, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		text := cleanHTMLText(match[1])
		if text == "" {
			continue
		}
		result = append(result, text)
	}
	return result
}

func parseDoubanTags(html string) []string {
	matches := reDoubanTagLinks.FindAllStringSubmatch(html, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		text := cleanHTMLText(match[1])
		if text == "" {
			continue
		}
		result = append(result, text)
	}
	sort.Strings(result)
	return result
}

func cleanHTMLText(value string) string {
	value = htmlEntityReplacer.Replace(value)
	value = reHTMLTags.ReplaceAllString(value, " ")
	value = reHTMLSpaceCollapse.ReplaceAllString(strings.TrimSpace(value), " ")
	return strings.TrimSpace(value)
}

var htmlEntityReplacer = strings.NewReplacer(
	"&nbsp;", " ",
	"&amp;", "&",
	"&quot;", "\"",
	"&#39;", "'",
	"&lt;", "<",
	"&gt;", ">",
	"&#x2F;", "/",
)

func parseYearFromDate(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if len(value) >= 4 {
		year, _ := strconv.Atoi(value[:4])
		return year
	}
	return 0
}

func preferNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maybePosterFilename(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return path.Base(parsed.Path)
}

var errUnsupportedProvider = errors.New("unsupported provider")

type TencentVideoProvider struct {
	client *http.Client
}

type tencentUnionResponse struct {
	Data struct {
		CoverInfos map[string]tencentCoverInfo `json:"cover_infos"`
		VideoInfos map[string]tencentVideoInfo `json:"video_infos"`
	} `json:"data"`
	Ret int    `json:"ret"`
	Msg string `json:"msg"`
}

type tencentCoverInfo struct {
	Title            string   `json:"title"`
	SecondTitle      string   `json:"second_title"`
	NewPicHZ         string   `json:"new_pic_hz"`
	NewPicVT         string   `json:"new_pic_vt"`
	BigHorizontalPic string   `json:"big_horizontal_pic_url"`
	Description      string   `json:"description"`
	LeadingActor     []string `json:"leading_actor"`
	Guests           []string `json:"guests"`
	VideoIDs         []string `json:"video_ids"`
	Alias            []string `json:"alias"`
	AreaName         string   `json:"area_name"`
	EpisodeAll       string   `json:"episode_all"`
}

type tencentVideoInfo struct {
	Title         string   `json:"title"`
	CTitleDetail  string   `json:"c_title_detail"`
	SecondTitle   string   `json:"second_title"`
	Desc          string   `json:"desc"`
	Pic496x280    string   `json:"pic496x280"`
	CategoryValue string   `json:"c_category_value"`
	CategoryMap   []string `json:"category_map"`
	ExtInfo       struct {
		ReleaseTime   string `json:"release_time"`
		VideoDuration string `json:"video_duration"`
	} `json:"ext_info"`
}

func (p *TencentVideoProvider) EnrichMetadata(ctx context.Context, meta *Metadata) error {
	if p == nil || p.client == nil || meta == nil {
		return nil
	}

	if strings.TrimSpace(meta.SourceSubjectID) == "" {
		return nil
	}

	var cover *tencentCoverInfo
	if cid := strings.TrimSpace(externalString(meta.ExternalData, "tencent_cid")); cid != "" {
		coverMap, err := p.fetchCoverInfos(ctx, []string{cid})
		if err == nil {
			if value, ok := coverMap[cid]; ok && len(value.VideoIDs) > 0 {
				copyValue := value
				cover = &copyValue
			}
		}
	}
	if cover == nil {
		searchTitles := []string{
			strings.TrimSpace(meta.Title),
			strings.TrimSpace(meta.OriginalTitle),
			strings.TrimSpace(externalString(meta.ExternalData, "search_title")),
			strings.TrimSpace(externalString(meta.ExternalData, "group_title")),
		}
		for _, alias := range metadataAliasStrings(meta.ExternalData) {
			searchTitles = append(searchTitles, alias)
		}
		fallbackCover, err := p.findBestCover(ctx, searchTitles)
		if err == nil && fallbackCover != nil {
			cover = fallbackCover
			if meta.ExternalData == nil {
				meta.ExternalData = map[string]any{}
			}
			if cid := findTencentCoverCID(searchTitles, fallbackCover, meta.ExternalData); cid != "" {
				meta.ExternalData["tencent_cid"] = cid
			}
		}
	}
	if cover == nil {
		return nil
	}

	videoMap, err := p.fetchVideoInfos(ctx, cover.VideoIDs)
	if err != nil {
		return err
	}

	bestBackdrop := normalizeTencentImageURL(
		firstNonEmpty(
			cover.NewPicHZ,
			cover.BigHorizontalPic,
			cover.NewPicVT,
		),
	)
	if bestBackdrop != "" {
		meta.BackdropURL = bestBackdrop
	}

	if strings.TrimSpace(meta.Summary) == "" {
		meta.Summary = strings.TrimSpace(cover.Description)
	}

	if len(meta.Cast) == 0 {
		meta.Cast = uniqueFirst(append(append([]string{}, cover.LeadingActor...), cover.Guests...), 8)
	}
	if len(meta.CastMembers) == 0 && len(meta.Cast) > 0 {
		meta.CastMembers = castMembersFromNames(meta.Cast)
	}
	if strings.TrimSpace(meta.PosterURL) == "" {
		meta.PosterURL = normalizeTencentImageURL(firstNonEmpty(cover.NewPicVT, cover.NewPicHZ, cover.BigHorizontalPic))
	}

	assets := buildTencentEpisodeAssets(videoMap)
	if len(assets) > 0 {
		if meta.ExternalData == nil {
			meta.ExternalData = map[string]any{}
		}
		meta.ExternalData["episode_assets"] = assets
		meta.ExternalData["tencent_cover"] = map[string]any{
			"title":        strings.TrimSpace(cover.Title),
			"second_title": strings.TrimSpace(cover.SecondTitle),
			"backdrop_url": bestBackdrop,
		}
	}
	return nil
}

func metadataAliasStrings(external map[string]any) []string {
	if len(external) == 0 {
		return []string{}
	}
	raw, ok := external["aka"]
	if !ok {
		return []string{}
	}
	switch value := raw.(type) {
	case []string:
		return filterNonEmptyStrings(value)
	case []any:
		result := make([]string, 0, len(value))
		for _, item := range value {
			result = append(result, strings.TrimSpace(fmt.Sprint(item)))
		}
		return filterNonEmptyStrings(result)
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" {
			return []string{}
		}
		return []string{text}
	}
}

func findTencentCoverCID(searchTitles []string, cover *tencentCoverInfo, external map[string]any) string {
	if cover == nil {
		return ""
	}
	if current := strings.TrimSpace(externalString(external, "tencent_cid")); current != "" {
		return current
	}
	_ = searchTitles
	return ""
}

func (p *TencentVideoProvider) findBestCover(ctx context.Context, searchTitles []string) (*tencentCoverInfo, error) {
	searchTitles = filterNonEmptyStrings(searchTitles)
	if len(searchTitles) == 0 {
		return nil, nil
	}

	for _, query := range searchTitles {
		suggestions, err := p.searchSubjects(ctx, query)
		if err != nil {
			continue
		}
		best := chooseTencentCoverSuggestion(query, searchTitles, suggestions)
		if best == nil {
			continue
		}
		coverMap, err := p.fetchCoverInfos(ctx, []string{best.CID})
		if err != nil {
			return nil, err
		}
		cover := coverMap[best.CID]
		if len(cover.VideoIDs) > 0 {
			copyValue := cover
			return &copyValue, nil
		}
	}
	return nil, nil
}

type tencentSearchSuggestion struct {
	CID         string `json:"cid"`
	Title       string `json:"title"`
	SecondTitle string `json:"second_title"`
	Type        int    `json:"type"`
}

func (p *TencentVideoProvider) searchSubjects(ctx context.Context, keyword string) ([]tencentSearchSuggestion, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, nil
	}

	endpoint := "https://pbaccess.video.qq.com/trpc.vector_layout.page_view.PageService/getPage?video_appid=3000010&vversion_platform=2&vversion_name=8.5.96"
	payload := map[string]any{
		"page_params": map[string]any{
			"page_id":   "search_result_page",
			"page_type": "search_result_page",
			"query":     keyword,
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 panplayer115/1.0")
	req.Header.Set("Referer", "https://v.qq.com/")
	req.Header.Set("Origin", "https://v.qq.com")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tencent search failed: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}

	pattern := regexp.MustCompile(`"cid":"([^"]+)".{0,240}?"title":"([^"]+)".{0,240}?"second_title":"([^"]*)".{0,120}?"type":"?(\d+)"?`)
	matches := pattern.FindAllStringSubmatch(string(bodyBytes), -1)
	if len(matches) == 0 {
		return nil, nil
	}

	results := make([]tencentSearchSuggestion, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		if len(match) < 5 {
			continue
		}
		cid := strings.TrimSpace(match[1])
		if cid == "" {
			continue
		}
		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}
		typed, _ := strconv.Atoi(strings.TrimSpace(match[4]))
		results = append(results, tencentSearchSuggestion{
			CID:         cid,
			Title:       strings.TrimSpace(match[2]),
			SecondTitle: strings.TrimSpace(match[3]),
			Type:        typed,
		})
	}
	return results, nil
}

func chooseTencentCoverSuggestion(primary string, aliases []string, items []tencentSearchSuggestion) *tencentSearchSuggestion {
	if len(items) == 0 {
		return nil
	}
	group := TitleGroup{
		BaseTitle:   strings.TrimSpace(primary),
		SearchTitle: strings.TrimSpace(primary),
		Section:     "variety",
	}
	bestIndex := -1
	bestScore := -1.0
	for index, item := range items {
		if item.Type != 10 && item.Type != 2 {
			continue
		}
		score := subjectMatchScore(group, item.Title, item.SecondTitle, 0, true)
		for _, alias := range aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			aliasGroup := group
			aliasGroup.BaseTitle = alias
			aliasGroup.SearchTitle = alias
			aliasScore := subjectMatchScore(aliasGroup, item.Title, item.SecondTitle, 0, true)
			if aliasScore > score {
				score = aliasScore
			}
		}
		if score > bestScore {
			bestScore = score
			bestIndex = index
		}
	}
	if bestIndex < 0 {
		return nil
	}
	return &items[bestIndex]
}

func (p *TencentVideoProvider) fetchCoverInfos(ctx context.Context, cids []string) (map[string]tencentCoverInfo, error) {
	result := map[string]tencentCoverInfo{}
	if len(cids) == 0 {
		return result, nil
	}
	payload := map[string]any{
		"cids":  cids,
		"appid": "10001",
	}
	response, err := p.fillUnionInfo(ctx, payload)
	if err != nil {
		return nil, err
	}
	for key, value := range response.Data.CoverInfos {
		result[key] = value
	}
	return result, nil
}

func (p *TencentVideoProvider) fetchVideoInfos(ctx context.Context, vids []string) (map[string]tencentVideoInfo, error) {
	result := map[string]tencentVideoInfo{}
	if len(vids) == 0 {
		return result, nil
	}
	payload := map[string]any{
		"vids":  vids,
		"appid": "10001",
	}
	response, err := p.fillUnionInfo(ctx, payload)
	if err != nil {
		return nil, err
	}
	for key, value := range response.Data.VideoInfos {
		result[key] = value
	}
	return result, nil
}

func (p *TencentVideoProvider) fillUnionInfo(ctx context.Context, payload map[string]any) (*tencentUnionResponse, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://pbaccess.video.qq.com/trpc.universal_backend_service.union_extra_data.UnionExtraData/FillUnionInfo?video_appid=3000010&vversion_platform=2&vversion_name=8.5.96", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 panplayer115/1.0")
	req.Header.Set("Referer", "https://v.qq.com/")
	req.Header.Set("Origin", "https://v.qq.com")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tencent union info failed: %s", resp.Status)
	}

	var response tencentUnionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	if response.Ret != 0 {
		return nil, fmt.Errorf("tencent union info failed: %s (%d)", strings.TrimSpace(response.Msg), response.Ret)
	}
	return &response, nil
}

func buildTencentEpisodeAssets(videoMap map[string]tencentVideoInfo) []EpisodeAsset {
	if len(videoMap) == 0 {
		return []EpisodeAsset{}
	}

	type item struct {
		vid   string
		value tencentVideoInfo
	}
	items := make([]item, 0, len(videoMap))
	for vid, value := range videoMap {
		items = append(items, item{vid: vid, value: value})
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := parseTencentReleaseTime(items[i].value.ExtInfo.ReleaseTime)
		right := parseTencentReleaseTime(items[j].value.ExtInfo.ReleaseTime)
		if !left.Equal(right) {
			return left.Before(right)
		}
		return strings.TrimSpace(items[i].vid) < strings.TrimSpace(items[j].vid)
	})

	result := make([]EpisodeAsset, 0, len(items))
	for _, current := range items {
		duration, _ := strconv.ParseInt(strings.TrimSpace(current.value.ExtInfo.VideoDuration), 10, 64)
		result = append(result, EpisodeAsset{
			EpisodeTitle:  strings.TrimSpace(preferNonEmpty(current.value.CTitleDetail, current.value.Title)),
			ThumbnailURL:  normalizeTencentImageURL(current.value.Pic496x280),
			BackdropURL:   normalizeTencentImageURL(current.value.Pic496x280),
			SourceVID:     strings.TrimSpace(current.vid),
			ReleaseDate:   parseTencentReleaseTime(current.value.ExtInfo.ReleaseTime).Format("2006-01-02"),
			DurationSec:   duration,
			CategoryValue: strings.TrimSpace(current.value.CategoryValue),
		})
	}
	return result
}

func parseTencentReleaseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds <= 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0).UTC()
}

func normalizeTencentImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "http://") {
		return "https://" + strings.TrimPrefix(raw, "http://")
	}
	return raw
}

func firstTencentVendorTarget(vendors []struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	URI     string `json:"uri"`
	URL     string `json:"url"`
	BGImage string `json:"bg_image"`
}) (string, string) {
	for _, vendor := range vendors {
		text := strings.ToLower(strings.TrimSpace(vendor.ID + " " + vendor.Title + " " + vendor.URI + " " + vendor.URL))
		if !strings.Contains(text, "qq") && !strings.Contains(text, "tencent") && !strings.Contains(text, "txvideo") && !strings.Contains(text, "v.qq.com") {
			continue
		}
		cid, vid := parseTencentCIDVID(vendor.URI)
		if cid == "" {
			cid, vid = parseTencentCIDVID(vendor.URL)
		}
		if cid != "" {
			return cid, vid
		}
	}
	return "", ""
}

func parseTencentCIDVID(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}

	tryURLs := []string{raw}
	if strings.HasPrefix(raw, "txvideo://") {
		tryURLs = append(tryURLs, strings.TrimPrefix(raw, "txvideo://"))
	}
	for _, current := range tryURLs {
		if parsed, err := url.Parse(current); err == nil {
			query := parsed.Query()
			cid := strings.TrimSpace(firstNonEmpty(query.Get("cid"), query.Get("coverid")))
			vid := strings.TrimSpace(firstNonEmpty(query.Get("vid"), query.Get("vids")))
			if cid != "" || vid != "" {
				return cid, vid
			}
		}
	}

	reCID := regexp.MustCompile(`cid=([a-zA-Z0-9]+)`)
	reVID := regexp.MustCompile(`vid=([a-zA-Z0-9]+)`)
	cid := ""
	vid := ""
	if match := reCID.FindStringSubmatch(raw); len(match) >= 2 {
		cid = strings.TrimSpace(match[1])
	}
	if match := reVID.FindStringSubmatch(raw); len(match) >= 2 {
		vid = strings.TrimSpace(match[1])
	}
	return cid, vid
}

func filterNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func externalString(data map[string]any, key string) string {
	if len(data) == 0 {
		return ""
	}
	value, ok := data[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

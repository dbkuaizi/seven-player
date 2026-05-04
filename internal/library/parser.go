package library

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"panplayer/internal/pan"
)

var (
	reYear           = regexp.MustCompile(`(?i)\b(19\d{2}|20\d{2})\b`)
	reSeasonEpisode  = regexp.MustCompile(`(?i)\bS(\d{1,2})\s*E(\d{1,3})\b`)
	reEpisodeOnly    = regexp.MustCompile(`(?i)\bE[P]?\s?(\d{1,3})\b`)
	reEpisodePrefix  = regexp.MustCompile(`(?i)^\s*(?:ep?)?\s*0*([1-9]\d{0,2})(?:\s*[集话話期])?(?:[ ._\-]+|$)`)
	reSeasonOnly     = regexp.MustCompile(`(?i)\bS(?:eason)?\s?(\d{1,2})\b`)
	reSpecialOnly    = regexp.MustCompile(`(?i)\bS(\d{1,2})\s*SP(\d{1,3})\b|\bSP\s?(\d{1,3})\b`)
	reExtraOnly      = regexp.MustCompile(`(?i)\bEX\s?(\d{1,3})\b`)
	rePartOnly       = regexp.MustCompile(`(?i)\bPART\s?(\d{1,3})\b`)
	reChineseSeason  = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十]{1,3})\s*季`)
	reChineseEpisode = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百零〇两]{1,4})\s*[集话話期]`)
	reTotalEpisode   = regexp.MustCompile(`全\s*\d+\s*[集期话話]`)
	reBracketContent = regexp.MustCompile(`[\[\(\{（【][^\]\)\}）】]{1,80}[\]\)\}）】]`)
	reQualityToken   = regexp.MustCompile(`(?i)\b(2160p|1080p|720p|4k|8k|hdr10\+?|hdr|dv|dolby[ .-]?vision)\b`)
	reSourceToken    = regexp.MustCompile(`(?i)\b(remux|bluray|bdrip|web[- .]?dl|web[- .]?rip|hdtv|dvdrip)\b`)
	reAudioToken     = regexp.MustCompile(`(?i)\b(atmos|truehd|dts(?:-hd)?|aac|ac3|eac3|flac|ddp)\b`)
	reCodecToken     = regexp.MustCompile(`(?i)\b(hevc|h\.?265|x265|h\.?264|x264|av1|avc)\b`)
	reNoiseToken     = regexp.MustCompile(`(?i)\b(x264|x265|h264|h265|hevc|av1|aac|ac3|eac3|dts(?:-hd)?|truehd|flac|atmos|remux|bluray|bdrip|web[- .]?dl|web[- .]?rip|hdtv|dvdrip|2160p|1080p|720p|4k|8k|hdr10\+?|hdr|dv|dolby[ .-]?vision|hd|chs|cht|eng|jpn|kor|gb|big5|外挂|内封|内嵌|内挂|中字|双语|简体|繁体|国语|粤语|中配|国配|无水印)\b`)
	reReleaseInlineNoise = regexp.MustCompile(`(?i)(国语|粤语|中字|双语|内封|内嵌|内挂|简体|繁体|中文字幕|中英字幕|中配|国配|无水印|无删减|未删减|完整版|最新电影|最新剧集|最新资源|hd)`)
	reBracketNoise   = regexp.MustCompile(`(?i)(www\.|http|字幕|配音|国语|粤语|英语|日语|韩语|中英|简中|繁中|高清|蓝光|网盘|下载|发布|ddhdtv|blacktv|sample|预告|合集|收藏|片源)`)
	reEditionNoise   = regexp.MustCompile(`(?i)(杜比|视界|dolby|vision|imax|版本|剪辑|加长|未删减|完整版|重制版|修复版|纪念版|特别版|国语版|粤语版|双语版|台配版|港版|美版|韩版|日版)`)
	reTitleEditionSuffix = regexp.MustCompile(`(?i)(?:\s+|[._\-/])*(杜比视界(?:版|版本)?|杜比(?:版|版本)?|dolby[ ._-]?vision|imax(?:版)?|导演剪辑版|加长版|未删减版|完整版|重制版|修复版|纪念版|特别版|国语版|粤语版|英语版|日语版|韩语版|国粤双语版|中英双语版|双语版|台配版|港版|美版|韩版|日版)\s*$`)
	reSplitTokens    = regexp.MustCompile(`[._\-]+`)
	reMultiSpace     = regexp.MustCompile(`\s+`)
)

type ParsedFile struct {
	Name            string
	FileID          string
	PickCode        string
	ParentID        string
	Path            []pan.Breadcrumb
	Size            int64
	UpdatedAt       string
	DurationSec     int64
	Section         string
	Child           string
	Title           string
	SeriesTitle     string
	SearchTitle     string
	GroupTitle      string
	Normalized      string
	GroupNormalized string
	Year            int
	SeasonNumber    int
	EpisodeNumber   int
	EpisodeTitle    string
	Quality         string
	Source          string
	Audio           string
	Codec           string
	IsSeriesLike    bool
}

type TitleGroup struct {
	GroupKey      string
	BaseTitle     string
	SeriesTitle   string
	SearchTitle   string
	Normalized    string
	Section       string
	Child         string
	Year          int
	Quality       string
	Source        string
	Audio         string
	Files         []ParsedFile
	SeasonCount   int
	EpisodeCount  int
	DurationTotal int64
}

func ParseMediaFile(item pan.FileItem, dirPath []pan.Breadcrumb) ParsedFile {
	name := strings.TrimSpace(item.Name)
	ext := strings.ToLower(path.Ext(name))
	baseName := strings.TrimSpace(strings.TrimSuffix(name, ext))
	quality := normalizeMatch(reQualityToken.FindString(baseName))
	source := normalizeSourceToken(reSourceToken.FindString(baseName))
	audio := normalizeAudioToken(reAudioToken.FindString(baseName))
	codec := normalizeCodecToken(reCodecToken.FindString(baseName))
	year := parseYear(baseName)
	seasonNumber, episodeNumber := parseSeasonEpisode(baseName)
	if seasonNumber <= 0 {
		seasonNumber = parseSeasonFromPath(dirPath)
	}
	if year <= 0 {
		year = parseYearFromPath(dirPath)
	}
	isSeriesLike := seasonNumber > 0 || episodeNumber > 0
	section := classifySection(baseName, dirPath, isSeriesLike)
	if shouldTreatAsStandaloneMovie(section, baseName, dirPath, item.DurationSec, seasonNumber, episodeNumber) {
		section = "movies"
	}
	child := classifyChild(section, baseName, dirPath, isSeriesLike)
	searchTitle, inlineEpisodeTitle := buildSearchTitle(baseName)
	dirTitle := extractBestPathTitle(dirPath, section)
	searchTitle = choosePreferredSearchTitle(searchTitle, dirTitle, baseName, section, seasonNumber)
	searchTitle = cleanSearchTitle(searchTitle, section)
	groupTitle := seriesGroupTitle(searchTitle, dirPath, section)
	if strings.TrimSpace(groupTitle) == "" {
		groupTitle = cleanSearchTitle(stripSeasonSuffix(searchTitle), section)
	}
	if strings.TrimSpace(groupTitle) == "" {
		groupTitle = cleanSearchTitle(stripSeasonSuffix(baseName), section)
	}
	groupTitle = cleanSearchTitle(groupTitle, section)
	episodeTitle := buildEpisodeDisplayTitle(section, episodeNumber, inlineEpisodeTitle, parseEpisodeVariant(baseName))
	normalized := normalizeTitle(searchTitle)
	groupNormalized := normalizeTitle(groupTitle)

	if seasonNumber <= 0 && section != "movies" && isSeriesLike {
		seasonNumber = 1
	}
	if seasonNumber <= 0 && (section == "anime" || section == "variety") && episodeNumber > 0 {
		seasonNumber = 1
	}

	return ParsedFile{
		Name:            name,
		FileID:          item.FileID,
		PickCode:        item.PickCode,
		ParentID:        item.ParentID,
		Path:            append([]pan.Breadcrumb(nil), dirPath...),
		Size:            item.Size,
		UpdatedAt:       item.UpdatedAt,
		DurationSec:     item.DurationSec,
		Section:         section,
		Child:           child,
		Title:           displayTitleFromSearch(searchTitle, baseName),
		SeriesTitle:     displayTitleFromSearch(groupTitle, baseName),
		SearchTitle:     searchTitle,
		GroupTitle:      groupTitle,
		Normalized:      normalized,
		GroupNormalized: groupNormalized,
		Year:            year,
		SeasonNumber:    seasonNumber,
		EpisodeNumber:   episodeNumber,
		EpisodeTitle:    episodeTitle,
		Quality:         quality,
		Source:          source,
		Audio:           audio,
		Codec:           codec,
		IsSeriesLike:    isSeriesLike || section == "series" || section == "anime" || section == "variety" || section == "documentary",
	}
}

func GroupParsedFiles(files []ParsedFile) []TitleGroup {
	groups := map[string]*TitleGroup{}
	order := make([]string, 0, len(files))

	for _, file := range files {
		if shouldSkipStandaloneExtra(file, files) {
			continue
		}
		groupKey := buildGroupKey(file)
		group := groups[groupKey]
		if group == nil {
			group = &TitleGroup{
				GroupKey:    groupKey,
				BaseTitle:   preferGroupBaseTitle(file),
				SeriesTitle: file.SeriesTitle,
				SearchTitle: preferGroupSearchTitle(file),
				Normalized:  preferGroupNormalized(file),
				Section:     file.Section,
				Child:       file.Child,
				Year:        file.Year,
				Quality:     file.Quality,
				Source:      file.Source,
				Audio:       file.Audio,
			}
			groups[groupKey] = group
			order = append(order, groupKey)
		}

		if group.Quality == "" {
			group.Quality = file.Quality
		}
		if group.Source == "" {
			group.Source = file.Source
		}
		if group.Audio == "" {
			group.Audio = file.Audio
		}
		if group.Year == 0 {
			group.Year = file.Year
		}
		if shouldReplaceGroupBaseTitle(group.BaseTitle, file) {
			group.BaseTitle = preferGroupBaseTitle(file)
		}
		if strings.TrimSpace(group.SeriesTitle) == "" && strings.TrimSpace(file.SeriesTitle) != "" {
			group.SeriesTitle = file.SeriesTitle
		}
		if strings.TrimSpace(group.SearchTitle) == "" {
			group.SearchTitle = preferGroupSearchTitle(file)
		}
		if strings.TrimSpace(group.Normalized) == "" {
			group.Normalized = preferGroupNormalized(file)
		}
		if preferredSection(group.Section, file.Section) {
			group.Section = file.Section
			group.Child = file.Child
		}

		group.Files = append(group.Files, file)
		group.DurationTotal += file.DurationSec
	}

	result := make([]TitleGroup, 0, len(order))
	for _, key := range order {
		group := groups[key]
		group.SeasonCount = countSeasons(group.Files)
		group.EpisodeCount = countPrimaryEpisodes(group.Files)
		result = append(result, *group)
	}
	return result
}

func shouldSkipStandaloneExtra(file ParsedFile, allFiles []ParsedFile) bool {
	if !(file.Section == "series" || file.Section == "variety" || file.Section == "anime" || file.Section == "documentary") {
		return false
	}
	if file.EpisodeNumber > 0 || file.IsSeriesLike {
		return false
	}
	if file.DurationSec <= 0 || file.DurationSec > 20*60 {
		return false
	}
	if len(file.Path) == 0 {
		return false
	}

	parentFolder := strings.TrimSpace(file.Path[len(file.Path)-1].Name)
	parentNormalized := normalizeTitle(stripSeasonSuffix(buildDirectorySearchTitle(parentFolder)))
	if parentNormalized == "" {
		return false
	}

	hasSiblingEpisodes := false
	for _, sibling := range allFiles {
		if sibling.FileID == file.FileID {
			continue
		}
		if sibling.Section != file.Section {
			continue
		}
		if sibling.EpisodeNumber <= 0 || sibling.DurationSec <= 0 {
			continue
		}
		if !sameBreadcrumbPath(file.Path, sibling.Path) {
			continue
		}
		siblingGroup := sibling.GroupNormalized
		if siblingGroup == "" {
			siblingGroup = normalizeTitle(stripSeasonSuffix(strings.TrimSpace(sibling.GroupTitle)))
		}
		if siblingGroup == "" {
			siblingGroup = normalizeTitle(stripSeasonSuffix(strings.TrimSpace(sibling.SearchTitle)))
		}
		if siblingGroup == "" {
			siblingGroup = normalizeTitle(stripSeasonSuffix(strings.TrimSpace(sibling.Title)))
		}
		if siblingGroup == parentNormalized {
			hasSiblingEpisodes = true
			break
		}
	}

	if !hasSiblingEpisodes {
		return false
	}

	if containsHan(strings.TrimSpace(file.Title)) && strings.TrimSpace(file.SearchTitle) != "" {
		if normalizeTitle(stripSeasonSuffix(file.SearchTitle)) == parentNormalized {
			return false
		}
	}

	return true
}

func buildGroupKey(file ParsedFile) string {
	base := file.GroupNormalized
	if base == "" {
		base = normalizeTitle(file.GroupTitle)
	}
	if base == "" {
		base = stripSeasonMarkersNormalized(file.Normalized)
	}
	if base == "" {
		base = normalizeTitle(file.Name)
	}
	if file.Section == "movies" && file.Year > 0 {
		return file.Section + "|" + base + "|" + strconv.Itoa(file.Year)
	}
	return file.Section + "|" + base
}

func preferredSection(current, next string) bool {
	return sectionPriority(next) > sectionPriority(current)
}

func sectionPriority(section string) int {
	switch section {
	case "anime":
		return 5
	case "variety":
		return 4
	case "series":
		return 3
	case "documentary":
		return 2
	case "movies":
		return 1
	default:
		return 0
	}
}

func countSeasons(files []ParsedFile) int {
	seen := map[int]struct{}{}
	for _, file := range files {
		season := file.SeasonNumber
		if season <= 0 {
			season = 1
		}
		seen[season] = struct{}{}
	}
	return len(seen)
}

func countPrimaryEpisodes(files []ParsedFile) int {
	seen := map[string]struct{}{}
	for _, file := range files {
		if file.EpisodeNumber <= 0 {
			continue
		}
		season := file.SeasonNumber
		if season <= 0 {
			season = 1
		}
		key := fmt.Sprintf("%d-%d", season, file.EpisodeNumber)
		seen[key] = struct{}{}
	}
	if len(seen) > 0 {
		return len(seen)
	}
	return len(files)
}

func buildSearchTitle(name string) (string, string) {
	episodeTitle := ""
	cleaned := cleanTitleText(name, false)

	if parts := strings.Split(cleaned, " - "); len(parts) >= 2 {
		cleaned = strings.TrimSpace(parts[0])
		episodeTitle = strings.TrimSpace(parts[1])
	}

	return cleaned, episodeTitle
}

func preferGroupBaseTitle(file ParsedFile) string {
	if strings.TrimSpace(file.SeriesTitle) != "" {
		return strings.TrimSpace(file.SeriesTitle)
	}
	if strings.TrimSpace(file.GroupTitle) != "" {
		return strings.TrimSpace(file.GroupTitle)
	}
	return strings.TrimSpace(file.Title)
}

func preferGroupSearchTitle(file ParsedFile) string {
	if strings.TrimSpace(file.GroupTitle) != "" {
		return strings.TrimSpace(file.GroupTitle)
	}
	if strings.TrimSpace(file.SeriesTitle) != "" {
		return strings.TrimSpace(file.SeriesTitle)
	}
	return strings.TrimSpace(file.SearchTitle)
}

func preferGroupNormalized(file ParsedFile) string {
	if strings.TrimSpace(file.GroupNormalized) != "" {
		return strings.TrimSpace(file.GroupNormalized)
	}
	return strings.TrimSpace(file.Normalized)
}

func shouldReplaceGroupBaseTitle(current string, file ParsedFile) bool {
	next := preferGroupBaseTitle(file)
	if strings.TrimSpace(next) == "" {
		return false
	}
	if strings.TrimSpace(current) == "" {
		return true
	}
	if hasSeasonMarker(current) && !hasSeasonMarker(next) {
		return true
	}
	if looksLikeContainerName(current) && !looksLikeContainerName(next) {
		return true
	}
	return false
}

func buildDirectorySearchTitle(name string) string {
	cleaned := cleanTitleText(name, true)
	if looksLikeReleaseName(name) {
		cleaned = trimASCIIAliasTail(cleaned)
	}
	return strings.TrimSpace(cleaned)
}

func cleanTitleText(name string, preserveSeason bool) string {
	cleaned := htmlNoiseToSpace(name)
	cleaned = reBracketContent.ReplaceAllStringFunc(cleaned, func(value string) string {
		inner := strings.TrimSpace(strings.Trim(value, "[](){}（）【】"))
		if shouldKeepBracketText(inner) {
			return " " + inner + " "
		}
		return " "
	})
	cleaned = reSeasonEpisode.ReplaceAllString(cleaned, " ")
	cleaned = reEpisodeOnly.ReplaceAllString(cleaned, " ")
	cleaned = reSpecialOnly.ReplaceAllString(cleaned, " ")
	cleaned = reExtraOnly.ReplaceAllString(cleaned, " ")
	cleaned = rePartOnly.ReplaceAllString(cleaned, " ")
	if !preserveSeason {
		cleaned = reSeasonOnly.ReplaceAllString(cleaned, " ")
		cleaned = reChineseSeason.ReplaceAllString(cleaned, " ")
	}
	cleaned = reChineseEpisode.ReplaceAllString(cleaned, " ")
	cleaned = reNoiseToken.ReplaceAllString(cleaned, " ")
	cleaned = reQualityToken.ReplaceAllString(cleaned, " ")
	cleaned = reSourceToken.ReplaceAllString(cleaned, " ")
	cleaned = reAudioToken.ReplaceAllString(cleaned, " ")
	cleaned = reCodecToken.ReplaceAllString(cleaned, " ")
	cleaned = reYear.ReplaceAllString(cleaned, " ")
	cleaned = reSplitTokens.ReplaceAllString(cleaned, " ")
	cleaned = reMultiSpace.ReplaceAllString(strings.TrimSpace(cleaned), " ")
	return strings.TrimSpace(cleaned)
}

func shouldKeepBracketText(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	if reQualityToken.MatchString(lower) || reSourceToken.MatchString(lower) || reAudioToken.MatchString(lower) || reCodecToken.MatchString(lower) || reYear.MatchString(lower) {
		return false
	}
	if reTotalEpisode.MatchString(value) || reBracketNoise.MatchString(lower) || reEditionNoise.MatchString(lower) {
		return false
	}
	if strings.ContainsAny(lower, "0123456789") && (strings.Contains(lower, "1080") || strings.Contains(lower, "2160") || strings.Contains(lower, "720")) {
		return false
	}
	return len([]rune(value)) <= 16 && !strings.Contains(lower, "www.")
}

func displayTitleFromSearch(searchTitle, fallback string) string {
	if strings.TrimSpace(searchTitle) != "" {
		return searchTitle
	}
	return strings.TrimSpace(fallback)
}

func parseYear(name string) int {
	match := reYear.FindString(name)
	if match == "" {
		return 0
	}
	value, _ := strconv.Atoi(match)
	return value
}

func parseSeasonEpisode(name string) (int, int) {
	if match := reSeasonEpisode.FindStringSubmatch(name); len(match) == 3 {
		season, _ := strconv.Atoi(match[1])
		episode, _ := strconv.Atoi(match[2])
		return season, episode
	}

	season := 0
	episode := 0

	if match := reChineseSeason.FindStringSubmatch(name); len(match) == 2 {
		season = parseLooseNumber(match[1])
	} else if match := reSeasonOnly.FindStringSubmatch(name); len(match) == 2 {
		season, _ = strconv.Atoi(match[1])
	}

	if match := reChineseEpisode.FindStringSubmatch(name); len(match) == 2 {
		episode = parseLooseNumber(match[1])
	} else if match := reEpisodeOnly.FindStringSubmatch(name); len(match) == 2 {
		episode, _ = strconv.Atoi(match[1])
	} else if match := reEpisodePrefix.FindStringSubmatch(name); len(match) == 2 {
		episode, _ = strconv.Atoi(match[1])
	}

	return season, episode
}

type episodeVariant struct {
	Kind   string
	Number int
}

func parseEpisodeVariant(name string) episodeVariant {
	if match := reSpecialOnly.FindStringSubmatch(name); len(match) >= 4 {
		number := 0
		switch {
		case strings.TrimSpace(match[3]) != "":
			number, _ = strconv.Atoi(match[3])
		case strings.TrimSpace(match[2]) != "":
			number, _ = strconv.Atoi(match[2])
		}
		return episodeVariant{Kind: "special", Number: number}
	}
	if match := reExtraOnly.FindStringSubmatch(name); len(match) == 2 {
		number, _ := strconv.Atoi(match[1])
		return episodeVariant{Kind: "extra", Number: number}
	}
	if match := rePartOnly.FindStringSubmatch(name); len(match) == 2 {
		number, _ := strconv.Atoi(match[1])
		return episodeVariant{Kind: "part", Number: number}
	}
	return episodeVariant{Kind: "main"}
}

func buildEpisodeDisplayTitle(section string, episodeNumber int, inline string, variant episodeVariant) string {
	inline = strings.TrimSpace(inline)
	if inline != "" {
		return inline
	}

	unit := "集"
	if section == "variety" {
		unit = "期"
	}

	switch variant.Kind {
	case "special":
		if variant.Number > 0 {
			return fmt.Sprintf("特别篇 %d", variant.Number)
		}
		return "特别篇"
	case "extra":
		if episodeNumber > 0 {
			if section == "variety" {
				return fmt.Sprintf("第 %d 期 加更 %d", episodeNumber, maxInt(variant.Number, 1))
			}
			return fmt.Sprintf("第 %d %s 番外 %d", episodeNumber, unit, maxInt(variant.Number, 1))
		}
		return fmt.Sprintf("番外 %d", maxInt(variant.Number, 1))
	case "part":
		if episodeNumber > 0 {
			if section == "variety" {
				switch variant.Number {
				case 1:
					return fmt.Sprintf("第 %d 期 上", episodeNumber)
				case 2:
					return fmt.Sprintf("第 %d 期 下", episodeNumber)
				default:
					return fmt.Sprintf("第 %d 期 Part %d", episodeNumber, maxInt(variant.Number, 1))
				}
			}
			return fmt.Sprintf("第 %d %s Part %d", episodeNumber, unit, maxInt(variant.Number, 1))
		}
		return fmt.Sprintf("Part %d", maxInt(variant.Number, 1))
	default:
		if episodeNumber > 0 {
			return fmt.Sprintf("第 %d %s", episodeNumber, unit)
		}
		return ""
	}
}

func parseLooseNumber(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if n, err := strconv.Atoi(value); err == nil {
		return n
	}

	replacer := strings.NewReplacer("零", "0", "〇", "0", "一", "1", "二", "2", "两", "2", "三", "3", "四", "4", "五", "5", "六", "6", "七", "7", "八", "8", "九", "9")
	if strings.Contains(value, "十") {
		return parseChineseTenNumber(value)
	}
	value = replacer.Replace(value)
	n, _ := strconv.Atoi(value)
	return n
}

func parseChineseTenNumber(value string) int {
	if value == "十" {
		return 10
	}
	parts := strings.Split(value, "十")
	if len(parts) != 2 {
		return 0
	}
	left := 1
	right := 0
	if strings.TrimSpace(parts[0]) != "" {
		left = parseLooseNumber(parts[0])
	}
	if strings.TrimSpace(parts[1]) != "" {
		right = parseLooseNumber(parts[1])
	}
	return left*10 + right
}

func classifySection(name string, dirPath []pan.Breadcrumb, isSeriesLike bool) string {
	text := strings.ToLower(strings.Join(pathNames(dirPath), " ")) + " " + strings.ToLower(name)
	switch {
	case containsAny(text, "动漫", "动画", "番剧", "anime", "国漫", "日漫", "ova", "剧场版"):
		return "anime"
	case containsAny(text, "综艺", "真人秀", "脱口秀", "访谈", "晚会", "选秀", "音乐综艺", "团综", "五哈", "variety", "reality"):
		return "variety"
	case containsAny(text, "纪录片", "纪录", "documentary", "docu", "nature", "history", "science"):
		return "documentary"
	case containsAny(text, "电视剧", "剧集", "美剧", "英剧", "日剧", "韩剧", "国产剧", "series", "season") || isSeriesLike:
		return "series"
	default:
		return "movies"
	}
}

func shouldTreatAsStandaloneMovie(section, name string, dirPath []pan.Breadcrumb, durationSec int64, seasonNumber, episodeNumber int) bool {
	if section != "series" {
		return false
	}
	if seasonNumber > 0 || episodeNumber > 0 {
		return false
	}
	if durationSec <= 0 || durationSec < 70*60 {
		return false
	}

	text := strings.ToLower(strings.Join(pathNames(dirPath), " ")) + " " + strings.ToLower(name)
	if containsAny(text, "综艺", "真人秀", "脱口秀", "晚会", "选秀", "纪录片", "纪录", "documentary", "动漫", "动画", "番剧", "anime", "ova") {
		return false
	}

	if looksLikeReleaseName(name) || parseYear(name) > 0 {
		return true
	}

	lastName := strings.TrimSpace(name)
	if len(dirPath) > 0 {
		lastName = strings.TrimSpace(dirPath[len(dirPath)-1].Name)
	}
	if containsHan(lastName) && !hasSeasonMarker(lastName) && !looksLikeSeasonFolder(lastName) {
		return true
	}

	return false
}

func classifyChild(section, name string, dirPath []pan.Breadcrumb, isSeriesLike bool) string {
	text := strings.ToLower(strings.Join(pathNames(dirPath), " ")) + " " + strings.ToLower(name)
	switch section {
	case "anime":
		switch {
		case containsAny(text, "国漫"):
			return "国漫"
		case containsAny(text, "剧场", "movie", "ova", "oad"):
			return "剧场版"
		default:
			return "番剧"
		}
	case "variety":
		switch {
		case containsAny(text, "脱口秀", "talk"):
			return "脱口秀"
		case containsAny(text, "音乐", "演唱会", "live", "concert"):
			return "音乐"
		case containsAny(text, "访谈", "talk show", "interview"):
			return "访谈"
		case containsAny(text, "竞技", "竞演", "competition"):
			return "竞技"
		case containsAny(text, "真人秀", "团综", "reality", "variety"):
			return "真人秀"
		default:
			return "全部综艺"
		}
	case "documentary":
		switch {
		case containsAny(text, "自然", "nature"):
			return "自然"
		case containsAny(text, "历史", "history"):
			return "历史"
		case containsAny(text, "科技", "science", "tech"):
			return "科技"
		case containsAny(text, "人文", "culture", "humanity"):
			return "人文"
		default:
			return "全部纪录片"
		}
	case "series":
		switch {
		case containsAny(text, "韩", "日剧", "日韩", "kdrama", "jdrama"):
			return "日韩剧"
		case containsAny(text, "美剧", "英剧", "欧美", "us", "uk", "hbo", "netflix"):
			return "英美剧"
		case containsAny(text, "短剧", "mini series"):
			return "短剧"
		default:
			if isSeriesLike {
				return "国产剧"
			}
			return "全部剧集"
		}
	default:
		switch {
		case containsAny(text, "动画", "anime"):
			return "动画电影"
		case containsAny(text, "日", "韩", "日韩"):
			return "日韩电影"
		case containsAny(text, "欧美", "美国", "英国", "france", "germany", "usa", "uk"):
			return "欧美电影"
		case containsAny(text, "纪录"):
			return "纪录电影"
		default:
			return "华语电影"
		}
	}
}

func pathNames(path []pan.Breadcrumb) []string {
	if len(path) == 0 {
		return nil
	}
	result := make([]string, 0, len(path))
	for _, crumb := range path {
		if strings.TrimSpace(crumb.Name) == "" {
			continue
		}
		result = append(result, crumb.Name)
	}
	return result
}

func sameBreadcrumbPath(left, right []pan.Breadcrumb) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if strings.TrimSpace(left[index].ID) != strings.TrimSpace(right[index].ID) {
			return false
		}
	}
	return true
}

func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if keyword != "" && strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func normalizeTitle(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.In(r, unicode.Cf) {
			return -1
		}
		return unicode.ToLower(r)
	}, strings.TrimSpace(value))
	value = strings.NewReplacer(
		"：", "",
		":", "",
		"·", "",
		"•", "",
		".", "",
		",", "",
		"，", "",
		"!", "",
		"！", "",
		"?", "",
		"？", "",
		"'", "",
		"’", "",
		"\"", "",
		"“", "",
		"”", "",
		"/", "",
		"\\", "",
		"(", "",
		")", "",
		"[", "",
		"]", "",
		"{", "",
		"}", "",
		"-", "",
		"_", "",
		" ", "",
	).Replace(value)
	return value
}

func normalizeMatch(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToUpper(strings.ReplaceAll(value, " ", ""))
}

func normalizeSourceToken(value string) string {
	value = normalizeMatch(value)
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, "-", "")
	switch value {
	case "WEBDL":
		return "WEB-DL"
	case "WEBRIP":
		return "WEB-RIP"
	case "BLURAY":
		return "BluRay"
	case "BDRIP":
		return "BDRip"
	case "DVDRIP":
		return "DVDRip"
	case "REMUX":
		return "REMUX"
	default:
		return value
	}
}

func normalizeAudioToken(value string) string {
	value = normalizeMatch(value)
	switch value {
	case "DDP":
		return "DDP"
	case "TRUEHD":
		return "TrueHD"
	default:
		return value
	}
}

func normalizeCodecToken(value string) string {
	value = normalizeMatch(value)
	value = strings.ReplaceAll(value, ".", "")
	switch value {
	case "H265":
		return "HEVC"
	case "H264":
		return "AVC"
	default:
		return value
	}
}

func htmlNoiseToSpace(value string) string {
	value = strings.NewReplacer("【", "[", "】", "]", "（", "(", "）", ")", "　", " ", "／", "/", "：", ":", "—", "-", "–", "-").Replace(value)
	return value
}

func seriesGroupTitle(searchTitle string, dirPath []pan.Breadcrumb, section string) string {
	searchTitle = stripSeasonSuffix(searchTitle)
	if strings.TrimSpace(searchTitle) != "" && !looksLikeContainerName(searchTitle) {
		return strings.TrimSpace(searchTitle)
	}

	for index := len(dirPath) - 1; index >= 0; index-- {
		name := strings.TrimSpace(dirPath[index].Name)
		if name == "" || isContainerFolderName(name) {
			continue
		}
		title := buildDirectorySearchTitle(name)
		title = stripSeasonSuffix(title)
		if title == "" || isContainerFolderName(title) || looksLikeSeasonFolder(name) {
			continue
		}
		return title
	}

	return strings.TrimSpace(searchTitle)
}

func choosePreferredSearchTitle(fileTitle, dirTitle, rawBaseName, section string, seasonNumber int) string {
	fileTitle = strings.TrimSpace(fileTitle)
	dirTitle = strings.TrimSpace(dirTitle)
	if dirTitle == "" {
		return cleanSearchTitle(appendSeasonSuffix(fileTitle, section, seasonNumber), section)
	}
	if fileTitle == "" {
		return cleanSearchTitle(dirTitle, section)
	}
	if shouldPreferDirectoryTitle(fileTitle, rawBaseName, section) {
		return cleanSearchTitle(dirTitle, section)
	}
	if hasSeasonMarker(dirTitle) && !hasSeasonMarker(fileTitle) {
		return cleanSearchTitle(dirTitle, section)
	}
	if looksLikeReleaseTitle(fileTitle, rawBaseName) {
		return cleanSearchTitle(dirTitle, section)
	}
	if containsHan(dirTitle) && !containsHan(fileTitle) {
		return cleanSearchTitle(dirTitle, section)
	}
	return cleanSearchTitle(appendSeasonSuffix(fileTitle, section, seasonNumber), section)
}

func shouldPreferDirectoryTitle(fileTitle, rawBaseName, section string) bool {
	if section == "movies" {
		return false
	}

	fileTitle = strings.TrimSpace(fileTitle)
	rawBaseName = strings.TrimSpace(rawBaseName)
	if fileTitle == "" || rawBaseName == "" {
		return false
	}

	match := reEpisodePrefix.FindStringSubmatch(rawBaseName)
	if len(match) != 2 {
		return false
	}

	prefix := strings.TrimSpace(match[0])
	remainder := strings.TrimSpace(strings.TrimPrefix(rawBaseName, prefix))
	remainder = cleanupEpisodeReleaseRemainder(remainder)

	if remainder == "" {
		return true
	}

	if normalizeTitle(remainder) == normalizeTitle(fileTitle) {
		return true
	}

	return looksLikeReleaseTitle(fileTitle, rawBaseName)
}

func cleanupEpisodeReleaseRemainder(value string) string {
	value = htmlNoiseToSpace(value)
	value = reBracketContent.ReplaceAllStringFunc(value, func(token string) string {
		inner := strings.TrimSpace(strings.Trim(token, "[](){}（）【】"))
		if shouldKeepBracketText(inner) {
			return " " + inner + " "
		}
		return " "
	})
	value = reQualityToken.ReplaceAllString(value, " ")
	value = reSourceToken.ReplaceAllString(value, " ")
	value = reAudioToken.ReplaceAllString(value, " ")
	value = reCodecToken.ReplaceAllString(value, " ")
	value = reNoiseToken.ReplaceAllString(value, " ")
	value = reReleaseInlineNoise.ReplaceAllString(value, " ")
	value = reYear.ReplaceAllString(value, " ")
	value = reSplitTokens.ReplaceAllString(value, " ")
	value = reMultiSpace.ReplaceAllString(strings.TrimSpace(value), " ")
	return strings.TrimSpace(value)
}

func cleanSearchTitle(value, section string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if section == "movies" || section == "series" || section == "anime" || section == "documentary" {
		for {
			next := strings.TrimSpace(reTitleEditionSuffix.ReplaceAllString(value, " "))
			next = strings.Trim(next, " ._-")
			next = reMultiSpace.ReplaceAllString(strings.TrimSpace(next), " ")
			if next == "" || next == value {
				break
			}
			value = next
		}
	}

	value = strings.Trim(value, " ._-")
	value = reMultiSpace.ReplaceAllString(strings.TrimSpace(value), " ")
	return strings.TrimSpace(value)
}

func appendSeasonSuffix(title, section string, seasonNumber int) string {
	title = strings.TrimSpace(title)
	if title == "" || seasonNumber <= 0 || section == "movies" || hasSeasonMarker(title) {
		return title
	}
	if section == "series" || section == "anime" || section == "variety" || section == "documentary" {
		return fmt.Sprintf("%s 第%d季", title, seasonNumber)
	}
	return title
}

func stripSeasonSuffix(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	title = reChineseSeason.ReplaceAllString(title, " ")
	title = reSeasonOnly.ReplaceAllString(title, " ")
	title = reMultiSpace.ReplaceAllString(strings.TrimSpace(title), " ")
	return strings.TrimSpace(title)
}

func stripSeasonMarkersNormalized(value string) string {
	return normalizeTitle(stripSeasonSuffix(value))
}

func extractBestPathTitle(dirPath []pan.Breadcrumb, section string) string {
	if len(dirPath) == 0 {
		return ""
	}
	for index := len(dirPath) - 1; index >= 0; index-- {
		name := strings.TrimSpace(dirPath[index].Name)
		if name == "" || isContainerFolderName(name) {
			continue
		}
		title := buildDirectorySearchTitle(name)
		if title == "" || isContainerFolderName(title) {
			continue
		}
		if section != "movies" && !hasSeasonMarker(title) {
			if season, _ := parseSeasonEpisode(name); season > 0 {
				title = appendSeasonSuffix(title, section, season)
			}
		}
		return title
	}
	return ""
}

func looksLikeSeasonFolder(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if reChineseSeason.MatchString(value) || reSeasonOnly.MatchString(value) {
		return true
	}
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "season ") || strings.HasPrefix(lower, "s")
}

func looksLikeContainerName(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	lower := strings.ToLower(value)
	switch lower {
	case "全部", "电影", "剧集", "电视剧", "综艺", "动漫", "动画", "纪录片":
		return true
	default:
		return false
	}
}

func parseSeasonFromPath(dirPath []pan.Breadcrumb) int {
	for index := len(dirPath) - 1; index >= 0; index-- {
		season, _ := parseSeasonEpisode(dirPath[index].Name)
		if season > 0 {
			return season
		}
	}
	return 0
}

func parseYearFromPath(dirPath []pan.Breadcrumb) int {
	for index := len(dirPath) - 1; index >= 0; index-- {
		if year := parseYear(dirPath[index].Name); year > 0 {
			return year
		}
	}
	return 0
}

func looksLikeReleaseTitle(title, rawBaseName string) bool {
	if strings.TrimSpace(title) == "" {
		return true
	}
	if containsHan(title) {
		return false
	}
	lowerRaw := strings.ToLower(rawBaseName)
	return strings.Count(rawBaseName, ".") >= 2 ||
		reQualityToken.MatchString(lowerRaw) ||
		reSourceToken.MatchString(lowerRaw) ||
		reAudioToken.MatchString(lowerRaw) ||
		reCodecToken.MatchString(lowerRaw) ||
		reSeasonEpisode.MatchString(rawBaseName)
}

func looksLikeReleaseName(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if lower == "" {
		return false
	}
	return strings.Count(raw, ".") >= 2 ||
		reQualityToken.MatchString(lower) ||
		reSourceToken.MatchString(lower) ||
		reAudioToken.MatchString(lower) ||
		reCodecToken.MatchString(lower) ||
		reSeasonEpisode.MatchString(raw)
}

func trimASCIIAliasTail(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !containsHan(value) {
		return value
	}
	parts := strings.Fields(value)
	if len(parts) <= 1 {
		return value
	}
	kept := make([]string, 0, len(parts))
	for index, part := range parts {
		if containsHan(part) || hasSeasonMarker(part) {
			kept = append(kept, part)
			continue
		}
		remainingHasHan := false
		for _, next := range parts[index+1:] {
			if containsHan(next) {
				remainingHasHan = true
				break
			}
		}
		if !remainingHasHan {
			break
		}
		kept = append(kept, part)
	}
	result := strings.TrimSpace(strings.Join(kept, " "))
	if result == "" {
		return value
	}
	return result
}

func hasSeasonMarker(value string) bool {
	value = strings.TrimSpace(value)
	return reChineseSeason.MatchString(value) || reSeasonOnly.MatchString(value)
}

func containsHan(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func isContainerFolderName(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "我的文件", "爆米花tv", "电影", "剧集", "电视剧", "动漫", "动画", "综艺", "纪录片":
		return true
	default:
		return false
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

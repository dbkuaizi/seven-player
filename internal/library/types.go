package library

import "panplayer/internal/pan"

type SectionInfo struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Icon     string   `json:"icon"`
	Color    string   `json:"color"`
	Children []string `json:"children"`
}

type CastMember struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl"`
	AvatarLocalPath string `json:"avatarLocalPath,omitempty"`
	Character string `json:"character"`
	Role      string `json:"role"`
}

type Title struct {
	ID                string   `json:"id"`
	GroupKey          string   `json:"groupKey"`
	Section           string   `json:"section"`
	Child             string   `json:"child"`
	Title             string   `json:"title"`
	OriginalTitle     string   `json:"originalTitle"`
	SearchTitle       string   `json:"searchTitle"`
	NormalizedTitle   string   `json:"normalizedTitle"`
	Year              int      `json:"year"`
	Rating            float64  `json:"rating"`
	Summary           string   `json:"summary"`
	Director          string   `json:"director"`
	Cast              []string `json:"cast"`
	Tags              []string `json:"tags"`
	PosterURL         string   `json:"posterUrl"`
	PosterLocalPath   string   `json:"posterLocalPath"`
	BackdropURL       string   `json:"backdropUrl"`
	BackdropLocalPath string   `json:"backdropLocalPath"`
	Quality           string   `json:"quality"`
	Source            string   `json:"source"`
	Audio             string   `json:"audio"`
	Status            string   `json:"status"`
	SourceName        string   `json:"sourceName"`
	SourceSubjectID   string   `json:"sourceSubjectId"`
	ExternalDataJSON  string   `json:"externalDataJson"`
	DefaultFileID     string   `json:"defaultFileId"`
	SeasonCount       int      `json:"seasonCount"`
	EpisodeCount      int      `json:"episodeCount"`
	TotalDurationSec  int64    `json:"totalDurationSec"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
	ScrapedAt         string   `json:"scrapedAt"`
}

type File struct {
	ID            string           `json:"id"`
	TitleID       string           `json:"titleId"`
	FileID        string           `json:"fileId"`
	PickCode      string           `json:"pickCode"`
	ParentID      string           `json:"parentId"`
	Path          []pan.Breadcrumb `json:"path"`
	Name          string           `json:"name"`
	Size          int64            `json:"size"`
	UpdatedAt     string           `json:"updatedAt"`
	DurationSec   int64            `json:"durationSec"`
	SeasonNumber  int              `json:"seasonNumber"`
	EpisodeNumber int              `json:"episodeNumber"`
	EpisodeTitle  string           `json:"episodeTitle"`
	Quality       string           `json:"quality"`
	Source        string           `json:"source"`
	Audio         string           `json:"audio"`
	SearchTitle   string           `json:"searchTitle"`
	ParsedYear    int              `json:"parsedYear"`
	Section       string           `json:"section"`
	Child         string           `json:"child"`
	DisplayOrder  int              `json:"displayOrder"`
	CreatedAt     string           `json:"createdAt"`
	UpdatedAtRow  string           `json:"updatedAtRow"`
	ScrapedAt     string           `json:"scrapedAt"`
}

type TitleSummary struct {
	ID                string           `json:"id"`
	Section           string           `json:"section"`
	Child             string           `json:"child"`
	Title             string           `json:"title"`
	OriginalTitle     string           `json:"originalTitle"`
	Year              int              `json:"year"`
	Rating            float64          `json:"rating"`
	Duration          string           `json:"duration"`
	Quality           string           `json:"quality"`
	Source            string           `json:"source"`
	Audio             string           `json:"audio"`
	Status            string           `json:"status"`
	PosterTone        string           `json:"posterTone"`
	PosterURL         string           `json:"posterUrl"`
	PosterLocalPath   string           `json:"posterLocalPath"`
	BackdropURL       string           `json:"backdropUrl"`
	BackdropLocalPath string           `json:"backdropLocalPath"`
	Summary           string           `json:"summary"`
	Director          string           `json:"director"`
	Cast              []string         `json:"cast"`
	CastMembers       []CastMember     `json:"castMembers"`
	Progress          int              `json:"progress"`
	ResumeMS          int64            `json:"resumeMs"`
	ResumeText        string           `json:"resumeText"`
	DurationSec       int64            `json:"durationSec"`
	LastPlayedAt      string           `json:"lastPlayedAt"`
	SourceName        string           `json:"sourceName"`
	SourceSubjectID   string           `json:"sourceSubjectId"`
	SeasonCount       int              `json:"seasonCount"`
	EpisodeCount      int              `json:"episodeCount"`
	DefaultFileID     string           `json:"defaultFileId"`
	DefaultPickCode   string           `json:"defaultPickCode"`
	DefaultName       string           `json:"defaultName"`
	DefaultParentID   string           `json:"defaultParentId"`
	DefaultPath       []pan.Breadcrumb `json:"defaultPath"`
	ExternalDataJSON  string           `json:"-"`
}

type TitleDetail struct {
	TitleSummary
	Files []EpisodeItem `json:"files"`
}

type EpisodeItem struct {
	ID            string           `json:"id"`
	TitleID       string           `json:"titleId"`
	FileID        string           `json:"fileId"`
	PickCode      string           `json:"pickCode"`
	ParentID      string           `json:"parentId"`
	Path          []pan.Breadcrumb `json:"path"`
	Name          string           `json:"name"`
	SeasonNumber  int              `json:"seasonNumber"`
	EpisodeNumber int              `json:"episodeNumber"`
	EpisodeTitle  string           `json:"episodeTitle"`
	DurationSec   int64            `json:"durationSec"`
	DurationText  string           `json:"durationText"`
	Quality       string           `json:"quality"`
	Source        string           `json:"source"`
	Audio         string           `json:"audio"`
	ThumbnailURL  string           `json:"thumbnailUrl"`
	Size          int64            `json:"size"`
	UpdatedAt     string           `json:"updatedAt"`
	ResumeMS      int64            `json:"resumeMs"`
	ResumeText    string           `json:"resumeText"`
	LastPlayedAt  string           `json:"lastPlayedAt"`
}

type LibrarySnapshot struct {
	Sections  []SectionInfo  `json:"sections"`
	Items     []TitleSummary `json:"items"`
	UpdatedAt string         `json:"updatedAt"`
}

type ScraperSettings struct {
	Directories    []DirectoryTarget `json:"directories"`
	Sources        []string          `json:"sources"`
	Language       string            `json:"language"`
	AutoScan       bool              `json:"autoScan"`
	Overwrite      bool              `json:"overwrite"`
	DownloadImages bool              `json:"downloadImages"`
}

type DirectoryTarget struct {
	ID   string           `json:"id"`
	Path []pan.Breadcrumb `json:"path"`
}

type ScrapeJobStatus struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	Message            string `json:"message"`
	CurrentPath        string `json:"currentPath"`
	ScannedDirectories int    `json:"scannedDirectories"`
	TotalDirectories   int    `json:"totalDirectories"`
	DiscoveredFiles    int    `json:"discoveredFiles"`
	ProcessedFiles     int    `json:"processedFiles"`
	MatchedItems       int    `json:"matchedItems"`
	UpdatedTitles      int    `json:"updatedTitles"`
	ErrorCount         int    `json:"errorCount"`
	LastError          string `json:"lastError"`
	StartedAt          string `json:"startedAt"`
	FinishedAt         string `json:"finishedAt"`
	UpdatedAt          string `json:"updatedAt"`
}

package library

import (
	"testing"

	"panplayer/internal/pan"
)

func TestGroupParsedFilesMergesSeasonsIntoSingleTitle(t *testing.T) {
	pathSeason1 := []pan.Breadcrumb{
		{ID: "0", Name: "我的文件"},
		{ID: "10", Name: "爆米花TV"},
		{ID: "11", Name: "剧集"},
		{ID: "12", Name: "风骚律师"},
		{ID: "13", Name: "第1季"},
	}
	pathSeason2 := []pan.Breadcrumb{
		{ID: "0", Name: "我的文件"},
		{ID: "10", Name: "爆米花TV"},
		{ID: "11", Name: "剧集"},
		{ID: "12", Name: "风骚律师"},
		{ID: "14", Name: "第2季"},
	}

	file1 := ParseMediaFile(pan.FileItem{
		FileID:      "f1",
		PickCode:    "p1",
		ParentID:    "13",
		Name:        "风骚律师.S01E01.1080p.WEB-DL.mkv",
		IsVideo:     true,
		DurationSec: 3200,
	}, pathSeason1)
	file2 := ParseMediaFile(pan.FileItem{
		FileID:      "f2",
		PickCode:    "p2",
		ParentID:    "14",
		Name:        "风骚律师.S02E01.1080p.WEB-DL.mkv",
		IsVideo:     true,
		DurationSec: 3210,
	}, pathSeason2)

	groups := GroupParsedFiles([]ParsedFile{file1, file2})
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].SeasonCount != 2 {
		t.Fatalf("expected merged season count 2, got %d", groups[0].SeasonCount)
	}
	if groups[0].BaseTitle != "风骚律师" {
		t.Fatalf("expected merged base title 风骚律师, got %q", groups[0].BaseTitle)
	}
}

func TestSeriesGroupTitleSkipsSeasonFolderName(t *testing.T) {
	dirPath := []pan.Breadcrumb{
		{ID: "0", Name: "我的文件"},
		{ID: "10", Name: "爆米花TV"},
		{ID: "11", Name: "综艺"},
		{ID: "12", Name: "哈哈哈哈哈 第四季"},
		{ID: "13", Name: "Season 1"},
	}

	title := seriesGroupTitle("哈哈哈哈哈 第四季", dirPath, "variety")
	if title != "哈哈哈哈哈" {
		t.Fatalf("expected stripped title 哈哈哈哈哈, got %q", title)
	}
}

func TestParseMediaFileUsesDirectoryTitleForEpisodePrefixedReleaseNames(t *testing.T) {
	dirPath := []pan.Breadcrumb{
		{ID: "0", Name: "我的文件"},
		{ID: "10", Name: "爆米花TV"},
		{ID: "11", Name: "电视剧"},
		{ID: "12", Name: "唐朝诡事录之长安.2160p"},
	}

	file := ParseMediaFile(pan.FileItem{
		FileID:      "f1",
		PickCode:    "p1",
		ParentID:    "12",
		Name:        "01.2160p.HD国语内封中字无水印[最新地址www.5266ys.com].mkv",
		IsVideo:     true,
		DurationSec: 2589,
	}, dirPath)

	if file.EpisodeNumber != 1 {
		t.Fatalf("expected episode number 1, got %d", file.EpisodeNumber)
	}
	if file.SearchTitle != "唐朝诡事录之长安" {
		t.Fatalf("expected search title 唐朝诡事录之长安, got %q", file.SearchTitle)
	}
	if file.GroupTitle != "唐朝诡事录之长安" {
		t.Fatalf("expected group title 唐朝诡事录之长安, got %q", file.GroupTitle)
	}
	if file.SeriesTitle != "唐朝诡事录之长安" {
		t.Fatalf("expected series title 唐朝诡事录之长安, got %q", file.SeriesTitle)
	}
}

func TestGroupParsedFilesMergesEpisodePrefixedReleaseNamesFromSingleDirectory(t *testing.T) {
	dirPath := []pan.Breadcrumb{
		{ID: "0", Name: "我的文件"},
		{ID: "10", Name: "爆米花TV"},
		{ID: "11", Name: "电视剧"},
		{ID: "12", Name: "唐朝诡事录之长安.2160p"},
	}

	file1 := ParseMediaFile(pan.FileItem{
		FileID:      "f1",
		PickCode:    "p1",
		ParentID:    "12",
		Name:        "01.2160p.HD国语内封中字无水印[最新地址www.5266ys.com].mkv",
		IsVideo:     true,
		DurationSec: 2589,
	}, dirPath)
	file2 := ParseMediaFile(pan.FileItem{
		FileID:      "f2",
		PickCode:    "p2",
		ParentID:    "12",
		Name:        "02.2160p.HD国语内封中字无水印[最新地址www.5266ys.com].mkv",
		IsVideo:     true,
		DurationSec: 2648,
	}, dirPath)

	groups := GroupParsedFiles([]ParsedFile{file1, file2})
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].BaseTitle != "唐朝诡事录之长安" {
		t.Fatalf("expected merged base title 唐朝诡事录之长安, got %q", groups[0].BaseTitle)
	}
	if groups[0].EpisodeCount != 2 {
		t.Fatalf("expected episode count 2, got %d", groups[0].EpisodeCount)
	}
}

func TestParseMediaFileTreatsSingleLongVideoAsMovieEvenUnderSeriesDirectory(t *testing.T) {
	dirPath := []pan.Breadcrumb{
		{ID: "0", Name: "我的文件"},
		{ID: "10", Name: "爆米花TV"},
		{ID: "11", Name: "电视剧"},
		{ID: "12", Name: "【高清影视之家发布 www.ZXVAC.com】疯狂动物城[杜比视界版本][粤英多音轨+粤语配音+中文字幕].2016.2160p.DSNP.WEB-DL.H265.DV.DDP5.1.Atmos-QuickIO"},
	}

	file := ParseMediaFile(pan.FileItem{
		FileID:      "f1",
		PickCode:    "p1",
		ParentID:    "12",
		Name:        "Zootopia.2016.2160p.DSNP.WEB-DL.H265.DV.DDP5.1.Atmos-QuickIO.mkv",
		IsVideo:     true,
		DurationSec: 6550,
	}, dirPath)

	if file.Section != "movies" {
		t.Fatalf("expected section movies, got %q", file.Section)
	}
	if file.SearchTitle != "疯狂动物城" {
		t.Fatalf("expected search title 疯狂动物城, got %q", file.SearchTitle)
	}
	if file.GroupTitle != "疯狂动物城" {
		t.Fatalf("expected group title 疯狂动物城, got %q", file.GroupTitle)
	}
}

func TestCleanSearchTitleRemovesEditionSuffixForMovie(t *testing.T) {
	got := cleanSearchTitle("疯狂动物城 杜比视界版本", "movies")
	if got != "疯狂动物城" {
		t.Fatalf("expected cleaned movie title 疯狂动物城, got %q", got)
	}
}

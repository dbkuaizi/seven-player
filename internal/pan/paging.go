package pan

import (
	"strconv"
	"strings"

	driver "github.com/jianxcao/115driver/pkg/driver"
)

const (
	defaultListPageSize = 20
	defaultPreviewLimit = 200
	maxListPageSize     = 200
)

func DefaultPreviewLimit() int {
	return defaultPreviewLimit
}

type rawFileListResp struct {
	driver.BasicResp
	Count    int           `json:"count"`
	Offset   int           `json:"offset"`
	Limit    int           `json:"limit"`
	PageSize int           `json:"page_size"`
	Path     []rawPathItem `json:"path"`
	Items    []rawFileInfo `json:"data"`
}

type rawPathItem struct {
	AreaID     driver.IntString `json:"aid"`
	CategoryID driver.IntString `json:"cid"`
	ParentID   driver.IntString `json:"pid"`
	Name       string           `json:"name"`
}

type rawFileInfo struct {
	AreaID     driver.IntString    `json:"aid"`
	CategoryID driver.IntString    `json:"cid"`
	FileID     string              `json:"fid"`
	ParentID   string              `json:"pid"`
	Name       string              `json:"n"`
	Type       string              `json:"ico"`
	Size       driver.StringInt64  `json:"s"`
	Sha1       string              `json:"sha"`
	PickCode   string              `json:"pc"`
	IsStar     driver.StringInt    `json:"m"`
	Labels     []*driver.LabelInfo `json:"fl"`
	CreateTime driver.StringInt64  `json:"tp"`
	UpdateTime string              `json:"t"`
	PlayLong   driver.StringInt64  `json:"play_long"`
}

func normalizePageRequest(offset, limit, defaultLimit, maxLimit int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}
	return offset, limit
}

func resolvePageLimit(value, fallback int) int {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return defaultListPageSize
}

func (info rawFileInfo) toDriverFileInfo() driver.FileInfo {
	return driver.FileInfo{
		AreaID:     info.AreaID,
		CategoryID: info.CategoryID,
		FileID:     info.FileID,
		ParentID:   info.ParentID,
		Name:       info.Name,
		Type:       info.Type,
		Size:       info.Size,
		Sha1:       info.Sha1,
		PickCode:   info.PickCode,
		IsStar:     info.IsStar,
		Labels:     info.Labels,
		CreateTime: info.CreateTime,
		UpdateTime: info.UpdateTime,
	}
}

func fileItemFromRaw(info rawFileInfo) FileItem {
	driverInfo := info.toDriverFileInfo()
	file := (&driver.File{}).From(&driverInfo)
	return FileItem{
		FileID:      file.FileID,
		ParentID:    file.ParentID,
		Name:        file.Name,
		Size:        file.Size,
		PickCode:    file.PickCode,
		IsDirectory: file.IsDirectory,
		IsVideo:     isVideo(file.Name, file.IsDirectory),
		UpdatedAt:   file.UpdateTime.Format(timeLayoutRFC3339),
		DurationSec: int64(info.PlayLong),
	}
}

func breadcrumbsFromRawPath(pathItems []rawPathItem) []Breadcrumb {
	if len(pathItems) == 0 {
		return nil
	}

	crumbs := make([]Breadcrumb, 0, len(pathItems))
	seen := map[string]bool{}

	for _, item := range pathItems {
		id := strings.TrimSpace(string(item.CategoryID))
		name := strings.TrimSpace(item.Name)
		if id == "" {
			continue
		}
		if id == "0" {
			name = "我的文件"
		}
		if name == "" || seen[id] {
			continue
		}
		crumbs = append(crumbs, Breadcrumb{
			ID:   id,
			Name: name,
		})
		seen[id] = true
	}

	if len(crumbs) == 0 {
		return []Breadcrumb{{ID: "0", Name: "我的文件"}}
	}
	if crumbs[0].ID != "0" {
		return append([]Breadcrumb{{ID: "0", Name: "我的文件"}}, crumbs...)
	}
	return crumbs
}

func (s *Service) listDirectoryPage(client *driver.Pan115Client, dirID string, offset, limit int) (*rawFileListResp, error) {
	result := rawFileListResp{}
	req := client.NewRequest().
		ForceContentType("application/json;charset=UTF-8").
		SetQueryParams(map[string]string{
			"aid":              "1",
			"cid":              dirID,
			"offset":           strconv.Itoa(offset),
			"limit":            strconv.Itoa(limit),
			"show_dir":         "1",
			"fc_mix":           "1",
			"natsort":          "1",
			"count_folders":    "1",
			"record_open_time": "1",
			"custom_order":     "0",
			"snap":             "0",
			"type":             "0",
			"format":           "json",
		}).
		SetResult(&result)

	resp, err := req.Get(driver.ApiFileList)
	if err = driver.CheckErr(err, &result, resp); err != nil {
		return nil, err
	}

	return &result, nil
}

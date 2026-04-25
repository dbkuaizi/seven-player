package pan

import (
	"strings"

	driver "github.com/jianxcao/115driver/pkg/driver"
)

const (
	apiFileSearch = "https://webapi.115.com/files/search"
	maxSearchSize = 200
)

type SearchResultView struct {
	Query   string     `json:"query"`
	Count   int        `json:"count"`
	Offset  int        `json:"offset"`
	Limit   int        `json:"limit"`
	HasMore bool       `json:"hasMore"`
	Items   []FileItem `json:"items"`
}

type fileSearchResp struct {
	driver.BasicResp
	Count    int           `json:"count"`
	Offset   int           `json:"offset"`
	PageSize int           `json:"page_size"`
	Items    []rawFileInfo `json:"data"`
}

func (s *Service) SearchFiles(keyword string, offset, limit int) (*SearchResultView, error) {
	client, err := s.authenticatedClient()
	if err != nil {
		return nil, err
	}

	keyword = strings.TrimSpace(keyword)
	offset, limit = normalizePageRequest(offset, limit, defaultListPageSize, maxSearchSize)
	if keyword == "" {
		return &SearchResultView{
			Query:  "",
			Offset: offset,
			Limit:  limit,
			Items:  []FileItem{},
		}, nil
	}

	result := fileSearchResp{}
	req := client.NewRequest().
		ForceContentType("application/json;charset=UTF-8").
		SetQueryParams(map[string]string{
			"search_value": keyword,
			"format":       "json",
			"aid":          "1",
			"limit":        toDecimalString(limit),
			"offset":       toDecimalString(offset),
			"show_dir":     "1",
			"fc_mix":       "1",
		}).
		SetResult(&result)

	resp, err := req.Get(apiFileSearch)
	if err = driver.CheckErr(err, &result, resp); err != nil {
		return nil, err
	}

	items := make([]FileItem, 0, len(result.Items))
	for _, fileInfo := range result.Items {
		items = append(items, fileItemFromRaw(fileInfo))
	}

	return &SearchResultView{
		Query:   keyword,
		Count:   result.Count,
		Offset:  result.Offset,
		Limit:   resolvePageLimit(result.PageSize, limit),
		HasMore: result.Offset+len(items) < result.Count,
		Items:   items,
	}, nil
}

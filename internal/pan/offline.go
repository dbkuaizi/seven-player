package pan

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	driver "github.com/jianxcao/115driver/pkg/driver"
)

type OfflineTaskView struct {
	InfoHash     string  `json:"infoHash"`
	Name         string  `json:"name"`
	Size         int64   `json:"size"`
	URL          string  `json:"url"`
	AddTime      string  `json:"addTime"`
	UpdateTime   string  `json:"updateTime"`
	Status       string  `json:"status"`
	StatusCode   int     `json:"statusCode"`
	StatusGroup  string  `json:"statusGroup"`
	Percent      float64 `json:"percent"`
	PercentText  string  `json:"percentText"`
	SpeedText    string  `json:"speedText"`
	LeftTimeText string  `json:"leftTimeText"`
	Peers        int64   `json:"peers"`
	FileID       string  `json:"fileId"`
	DeleteFileID string  `json:"deleteFileId"`
	DirID        string  `json:"dirId"`
}

type OfflineListView struct {
	Quota          int64             `json:"quota"`
	Total          int64             `json:"total"`
	ActiveCount    int               `json:"activeCount"`
	FailedCount    int               `json:"failedCount"`
	CompletedCount int               `json:"completedCount"`
	Tasks          []OfflineTaskView `json:"tasks"`
}

func (s *Service) ListOfflineTasks() (*OfflineListView, error) {
	client, err := s.authenticatedClient()
	if err != nil {
		return nil, err
	}

	return s.listOfflineTasksWithRetry(client)
}

func (s *Service) listOfflineTasksWithRetry(client *driver.Pan115Client) (*OfflineListView, error) {
	var lastErr error

	for attempt := 0; attempt < 2; attempt++ {
		view, err := s.listOfflineTasksOnce(client)
		if err == nil {
			return view, nil
		}

		lastErr = err
		if !shouldRetryOfflineError(err) || attempt == 1 {
			break
		}

		time.Sleep(350 * time.Millisecond)
	}

	return nil, normalizeOfflineError(lastErr, "读取云下载失败")
}

func (s *Service) listOfflineTasksOnce(client *driver.Pan115Client) (*OfflineListView, error) {
	firstPage, err := client.ListOfflineTask(1)
	if err != nil {
		return nil, err
	}

	rawTasks := make([]*driver.OfflineTask, 0, len(firstPage.Tasks))
	for _, task := range firstPage.Tasks {
		if task != nil {
			rawTasks = append(rawTasks, task)
		}
	}

	for page := int64(2); page <= firstPage.PageCount; page++ {
		resp, pageErr := client.ListOfflineTask(page)
		if pageErr != nil {
			return nil, pageErr
		}
		for _, task := range resp.Tasks {
			if task != nil {
				rawTasks = append(rawTasks, task)
			}
		}
	}

	view := &OfflineListView{
		Tasks: make([]OfflineTaskView, 0, len(rawTasks)),
	}

	view.Quota, view.Total = normalizeOfflineQuota(firstPage.Quota, firstPage.Total)

	sort.Slice(rawTasks, func(i, j int) bool {
		if rawTasks[i].AddTime == rawTasks[j].AddTime {
			return rawTasks[i].Name < rawTasks[j].Name
		}
		return rawTasks[i].AddTime > rawTasks[j].AddTime
	})

	for _, task := range rawTasks {
		taskView := buildOfflineTaskView(task)
		view.Tasks = append(view.Tasks, taskView)

		switch taskView.StatusGroup {
		case "completed":
			view.CompletedCount++
		case "failed":
			view.FailedCount++
		default:
			view.ActiveCount++
		}
	}

	return view, nil
}

func normalizeOfflineQuota(quota, total int64) (int64, int64) {
	if quota < 0 {
		quota = 0
	}
	if total < 0 {
		total = 0
	}
	if total == 0 {
		total = quota
	}
	if quota > total {
		total = quota
	}
	return quota, total
}

func runOfflineMutation(action func() error, fallback string) error {
	var lastErr error

	for attempt := 0; attempt < 2; attempt++ {
		if err := action(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if !shouldRetryOfflineError(lastErr) || attempt == 1 {
			break
		}

		time.Sleep(350 * time.Millisecond)
	}

	return normalizeOfflineError(lastErr, fallback)
}

func (s *Service) AddOfflineTasks(urls []string, saveDirID string) (*OfflineListView, error) {
	client, err := s.authenticatedClient()
	if err != nil {
		return nil, err
	}

	normalized := normalizeOfflineURLs(urls)
	if len(normalized) == 0 {
		return nil, errors.New("请至少输入一个下载链接")
	}

	saveDirID = strings.TrimSpace(saveDirID)
	if saveDirID == "" {
		saveDirID = "0"
	}

	if err := runOfflineMutation(func() error {
		_, err := client.AddOfflineTaskURIs(normalized, saveDirID)
		return err
	}, "添加云下载失败"); err != nil {
		return nil, err
	}

	view, err := s.listOfflineTasksWithRetry(client)
	if err != nil {
		return nil, normalizeOfflineError(err, "云下载任务已提交，但刷新任务列表失败，请稍后手动刷新")
	}

	return view, nil
}

func (s *Service) DeleteOfflineTasks(hashes []string, deleteFiles bool) (*OfflineListView, error) {
	client, err := s.authenticatedClient()
	if err != nil {
		return nil, err
	}

	normalized := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		hash = strings.TrimSpace(hash)
		if hash == "" {
			continue
		}
		normalized = append(normalized, hash)
	}

	if len(normalized) == 0 {
		return nil, errors.New("没有可删除的离线任务")
	}

	if err := runOfflineMutation(func() error {
		return client.DeleteOfflineTasks(normalized, deleteFiles)
	}, "删除云下载任务失败"); err != nil {
		return nil, err
	}

	view, err := s.listOfflineTasksWithRetry(client)
	if err != nil {
		return nil, normalizeOfflineError(err, "离线任务已删除，但刷新任务列表失败，请稍后手动刷新")
	}

	return view, nil
}

func buildOfflineTaskView(task *driver.OfflineTask) OfflineTaskView {
	return OfflineTaskView{
		InfoHash:     task.InfoHash,
		Name:         chooseOfflineName(task.Name, task.Url),
		Size:         task.Size,
		URL:          task.Url,
		AddTime:      unixToDateTime(task.AddTime),
		UpdateTime:   unixToDateTime(task.UpdateTime),
		Status:       offlineStatusText(task.Status),
		StatusCode:   task.Status,
		StatusGroup:  offlineStatusGroup(task.Status),
		Percent:      task.Percent,
		PercentText:  formatPercent(task.Percent, task.Status),
		SpeedText:    formatSpeed(task.RateDownload, task.Status),
		LeftTimeText: formatLeftTime(task.LeftTime, task.Status),
		Peers:        task.Peers,
		FileID:       task.FileId,
		DeleteFileID: task.DelFileId,
		DirID:        task.DirId,
	}
}

func normalizeOfflineURLs(urls []string) []string {
	result := make([]string, 0, len(urls))
	seen := map[string]struct{}{}

	for _, raw := range urls {
		for _, item := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == '\r' || r == '\n'
		}) {
			value := strings.TrimSpace(item)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}

	return result
}

func offlineStatusGroup(status int) string {
	switch status {
	case 2:
		return "completed"
	case -1:
		return "failed"
	default:
		return "active"
	}
}

func offlineStatusText(status int) string {
	switch status {
	case 0:
		return "准备中"
	case 1:
		return "下载中"
	case 2:
		return "已完成"
	case -1:
		return "失败"
	default:
		return fmt.Sprintf("未知状态(%d)", status)
	}
}

func formatPercent(percent float64, status int) string {
	if status == 2 {
		return "100%"
	}
	if percent <= 0 {
		return "0%"
	}
	if percent >= 100 {
		return "100%"
	}
	if percent >= 10 {
		return fmt.Sprintf("%.0f%%", percent)
	}
	return fmt.Sprintf("%.1f%%", percent)
}

func formatSpeed(rate float64, status int) string {
	if status != 1 || rate <= 0 {
		return "--"
	}

	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	value := rate
	index := 0
	for value >= 1024 && index < len(units)-1 {
		value /= 1024
		index++
	}

	if value >= 10 || index == 0 {
		return fmt.Sprintf("%.0f %s", value, units[index])
	}
	return fmt.Sprintf("%.1f %s", value, units[index])
}

func formatLeftTime(seconds int64, status int) string {
	if status != 1 || seconds <= 0 {
		return "--"
	}

	duration := time.Duration(seconds) * time.Second
	if duration >= 24*time.Hour {
		days := int64(duration / (24 * time.Hour))
		return fmt.Sprintf("%d 天", days)
	}
	if duration >= time.Hour {
		hours := int64(duration / time.Hour)
		minutes := int64((duration % time.Hour) / time.Minute)
		return fmt.Sprintf("%d 小时 %d 分", hours, minutes)
	}
	if duration >= time.Minute {
		minutes := int64(duration / time.Minute)
		return fmt.Sprintf("%d 分", minutes)
	}
	return fmt.Sprintf("%d 秒", seconds)
}

func unixToDateTime(unixSec int64) string {
	if unixSec <= 0 {
		return "--"
	}
	return time.Unix(unixSec, 0).Format("2006-01-02 15:04")
}

func chooseOfflineName(name, rawURL string) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	if strings.TrimSpace(rawURL) != "" {
		return strings.TrimSpace(rawURL)
	}
	return "未命名任务"
}

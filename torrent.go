package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) SelectTorrentFileAsMagnet() (string, error) {
	if a.ctx == nil {
		return "", errors.New("runtime unavailable")
	}

	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择 BT 种子文件",
		Filters: []wruntime.FileFilter{
			{
				DisplayName: "BT 种子文件",
				Pattern:     "*.torrent",
			},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取 BT 种子失败: %w", err)
	}

	magnet, err := buildMagnetFromTorrentData(data)
	if err != nil {
		return "", fmt.Errorf("解析 BT 种子失败: %w", err)
	}

	return magnet, nil
}

type torrentMeta struct {
	InfoBytes []byte
	Name      string
	Trackers  []string
}

func buildMagnetFromTorrentData(data []byte) (string, error) {
	meta, err := parseTorrentMeta(data)
	if err != nil {
		return "", err
	}

	sum := sha1.Sum(meta.InfoBytes)
	params := url.Values{}
	params.Set("xt", "urn:btih:"+strings.ToUpper(hex.EncodeToString(sum[:])))

	if strings.TrimSpace(meta.Name) != "" {
		params.Set("dn", meta.Name)
	}

	for _, tracker := range meta.Trackers {
		if strings.TrimSpace(tracker) == "" {
			continue
		}
		params.Add("tr", tracker)
	}

	return "magnet:?" + params.Encode(), nil
}

func parseTorrentMeta(data []byte) (torrentMeta, error) {
	if len(data) == 0 || data[0] != 'd' {
		return torrentMeta{}, errors.New("不是有效的 torrent 文件")
	}

	meta := torrentMeta{}
	trackers := make([]string, 0, 4)
	seenTrackers := map[string]struct{}{}
	index := 1

	for index < len(data) && data[index] != 'e' {
		key, next, err := parseBencodeString(data, index)
		if err != nil {
			return torrentMeta{}, err
		}
		index = next

		valueStart := index
		value, next, err := parseBencodeValue(data, index)
		if err != nil {
			return torrentMeta{}, err
		}
		index = next

		switch key {
		case "announce":
			appendTrackerValue(extractBencodeString(value), &trackers, seenTrackers)
		case "announce-list":
			appendTrackerValues(value, &trackers, seenTrackers)
		case "info":
			meta.InfoBytes = append([]byte(nil), data[valueStart:index]...)
			meta.Name = extractTorrentName(value)
		}
	}

	if len(meta.InfoBytes) == 0 {
		return torrentMeta{}, errors.New("BT 种子缺少 info 字段")
	}

	meta.Trackers = trackers
	return meta, nil
}

func parseBencodeValue(data []byte, index int) (any, int, error) {
	if index >= len(data) {
		return nil, index, errors.New("torrent 数据不完整")
	}

	switch data[index] {
	case 'i':
		return parseBencodeInteger(data, index)
	case 'l':
		return parseBencodeList(data, index)
	case 'd':
		return parseBencodeDictionary(data, index)
	default:
		if data[index] >= '0' && data[index] <= '9' {
			return parseBencodeString(data, index)
		}
	}

	return nil, index, fmt.Errorf("无法识别的 bencode 标记: %q", data[index])
}

func parseBencodeInteger(data []byte, index int) (int64, int, error) {
	if index >= len(data) || data[index] != 'i' {
		return 0, index, errors.New("无效的整数标记")
	}

	end := index + 1
	for end < len(data) && data[end] != 'e' {
		end++
	}
	if end >= len(data) {
		return 0, index, errors.New("整数未正确结束")
	}

	var value int64
	if _, err := fmt.Sscanf(string(data[index+1:end]), "%d", &value); err != nil {
		return 0, index, fmt.Errorf("整数解析失败: %w", err)
	}

	return value, end + 1, nil
}

func parseBencodeString(data []byte, index int) (string, int, error) {
	if index >= len(data) {
		return "", index, errors.New("字符串起始位置无效")
	}

	colon := index
	for colon < len(data) && data[colon] != ':' {
		if data[colon] < '0' || data[colon] > '9' {
			return "", index, errors.New("字符串长度格式无效")
		}
		colon++
	}
	if colon >= len(data) {
		return "", index, errors.New("字符串长度缺少分隔符")
	}

	var length int
	if _, err := fmt.Sscanf(string(data[index:colon]), "%d", &length); err != nil {
		return "", index, fmt.Errorf("字符串长度解析失败: %w", err)
	}

	start := colon + 1
	end := start + length
	if length < 0 || end > len(data) {
		return "", index, errors.New("字符串长度超出范围")
	}

	return string(data[start:end]), end, nil
}

func parseBencodeList(data []byte, index int) ([]any, int, error) {
	if index >= len(data) || data[index] != 'l' {
		return nil, index, errors.New("无效的列表标记")
	}

	items := make([]any, 0)
	index++

	for index < len(data) && data[index] != 'e' {
		value, next, err := parseBencodeValue(data, index)
		if err != nil {
			return nil, index, err
		}
		items = append(items, value)
		index = next
	}

	if index >= len(data) {
		return nil, index, errors.New("列表未正确结束")
	}

	return items, index + 1, nil
}

func parseBencodeDictionary(data []byte, index int) (map[string]any, int, error) {
	if index >= len(data) || data[index] != 'd' {
		return nil, index, errors.New("无效的字典标记")
	}

	values := make(map[string]any)
	index++

	for index < len(data) && data[index] != 'e' {
		key, next, err := parseBencodeString(data, index)
		if err != nil {
			return nil, index, err
		}
		index = next

		value, next, err := parseBencodeValue(data, index)
		if err != nil {
			return nil, index, err
		}
		values[key] = value
		index = next
	}

	if index >= len(data) {
		return nil, index, errors.New("字典未正确结束")
	}

	return values, index + 1, nil
}

func extractTorrentName(value any) string {
	dict, ok := value.(map[string]any)
	if !ok {
		return ""
	}

	if name := extractBencodeString(dict["name.utf-8"]); strings.TrimSpace(name) != "" {
		return name
	}

	return extractBencodeString(dict["name"])
}

func extractBencodeString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func appendTrackerValue(value string, trackers *[]string, seen map[string]struct{}) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if _, exists := seen[value]; exists {
		return
	}
	seen[value] = struct{}{}
	*trackers = append(*trackers, value)
}

func appendTrackerValues(value any, trackers *[]string, seen map[string]struct{}) {
	switch current := value.(type) {
	case string:
		appendTrackerValue(current, trackers, seen)
	case []any:
		for _, item := range current {
			appendTrackerValues(item, trackers, seen)
		}
	}
}

package pan

import (
	"errors"
	"strings"
)

func normalizeOfflineError(err error, fallback string) error {
	if err == nil {
		if strings.TrimSpace(fallback) == "" {
			return nil
		}
		return errors.New(strings.TrimSpace(fallback))
	}

	if looksLikeOfflineHTMLResponse(err.Error()) {
		return errors.New("115 云下载接口暂时返回了异常页面，请稍后手动刷新重试。登录状态仍然有效。")
	}

	return err
}

func shouldRetryOfflineError(err error) bool {
	return err != nil && looksLikeOfflineHTMLResponse(err.Error())
}

func looksLikeOfflineHTMLResponse(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}

	return strings.HasPrefix(normalized, "<!doctype") ||
		strings.HasPrefix(normalized, "<html") ||
		strings.Contains(normalized, "<meta charset=") ||
		strings.Contains(normalized, "<body") ||
		strings.Contains(normalized, "invalid character '<'") ||
		strings.Contains(normalized, "block_url_tips") ||
		strings.Contains(normalized, "block_message") ||
		strings.Contains(normalized, "traceid") ||
		strings.Contains(normalized, "tb1") ||
		strings.Contains(normalized, "unexpected error")
}

package pan

import (
	"encoding/json"
	"errors"
	"strings"
)

func SanitizeTemporaryHTMLResponseMessage(message, fallback string) string {
	normalized := strings.TrimSpace(message)
	if normalized == "" {
		return strings.TrimSpace(fallback)
	}

	if looksLikeTemporaryHTMLResponse(normalized) {
		if strings.TrimSpace(fallback) != "" {
			return strings.TrimSpace(fallback)
		}
		return "远程接口暂时返回了异常页面，请稍后重试。"
	}

	if nested := extractNestedHTMLResponseMessage(normalized); nested != "" {
		if strings.TrimSpace(fallback) != "" {
			return strings.TrimSpace(fallback)
		}
		return nested
	}

	return normalized
}

func NormalizeTemporaryHTMLResponseError(err error, fallback string) error {
	if err == nil {
		if strings.TrimSpace(fallback) == "" {
			return nil
		}
		return errors.New(strings.TrimSpace(fallback))
	}

	message := SanitizeTemporaryHTMLResponseMessage(err.Error(), fallback)
	if strings.TrimSpace(message) == "" {
		return nil
	}
	return errors.New(message)
}

func ShouldRetryTemporaryHTMLResponseError(err error) bool {
	return err != nil && looksLikeTemporaryHTMLResponse(err.Error())
}

func normalizeOfflineError(err error, fallback string) error {
	if err == nil {
		if strings.TrimSpace(fallback) == "" {
			return nil
		}
		return errors.New(strings.TrimSpace(fallback))
	}

	if ShouldRetryTemporaryHTMLResponseError(err) {
		return errors.New("115 云下载接口暂时返回了异常页面，请稍后手动刷新重试。登录状态仍然有效。")
	}

	return err
}

func shouldRetryOfflineError(err error) bool {
	return ShouldRetryTemporaryHTMLResponseError(err)
}

func looksLikeTemporaryHTMLResponse(message string) bool {
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

func extractNestedHTMLResponseMessage(message string) string {
	normalized := strings.TrimSpace(message)
	if normalized == "" || !strings.HasPrefix(normalized, "{") {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(normalized), &payload); err != nil {
		return ""
	}

	for _, key := range []string{"message", "error", "msg"} {
		value := strings.TrimSpace(asString(payload[key]))
		if value == "" || !looksLikeTemporaryHTMLResponse(value) {
			continue
		}
		return "115 接口暂时返回了异常页面，请稍后重试。"
	}

	return ""
}

func normalizeRemoteJSONHTMLError(err error, fallback string) error {
	if err == nil {
		return nil
	}

	if nested := extractNestedHTMLResponseMessage(err.Error()); nested != "" {
		return errors.New(nested)
	}

	return NormalizeTemporaryHTMLResponseError(err, fallback)
}

package pan

import (
	"errors"
	"strings"

	"panplayer/internal/config"
)

func (s *Service) LoginWithCookie(raw string) (*LoginStatusView, error) {
	credential, cookies, err := parseCookieInput(raw)
	if err != nil {
		return nil, err
	}

	ok, user, err := s.Restore(credential, cookies)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("cookie 登录失败")
	}

	return &LoginStatusView{
		State:    "authenticated",
		Message:  "Cookie 登录成功",
		LoggedIn: true,
		User:     user,
	}, nil
}

func parseCookieInput(raw string) (*config.Credential, map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil, errors.New("请输入 115 Cookie")
	}

	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	if strings.HasPrefix(strings.ToLower(raw), "cookie:") {
		raw = strings.TrimSpace(raw[len("cookie:"):])
	}

	items := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ';' || r == '\n'
	})

	cookies := make(map[string]string, len(items))
	for _, item := range items {
		key, value, ok := strings.Cut(strings.TrimSpace(item), "=")
		if !ok {
			continue
		}

		key = strings.ToUpper(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}

		switch key {
		case "PATH", "DOMAIN", "EXPIRES", "MAX-AGE", "SAMESITE", "HTTPONLY", "SECURE":
			continue
		}

		cookies[key] = value
	}

	if len(cookies) == 0 {
		return nil, nil, errors.New("没有从输入内容里解析到 Cookie")
	}

	if strings.TrimSpace(cookies["UID"]) == "" ||
		strings.TrimSpace(cookies["CID"]) == "" ||
		strings.TrimSpace(cookies["SEID"]) == "" {
		return nil, nil, errors.New("Cookie 缺少 UID、CID 或 SEID")
	}

	return normalizeCredential(nil, cookies), cookies, nil
}

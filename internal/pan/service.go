package pan

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"panplayer/internal/config"

	"github.com/google/uuid"
driver "github.com/jianxcao/115driver/pkg/driver"
)

const (
	fileAPIRetryCount = 2
	fileAPIRetryDelay = 350 * time.Millisecond
)
type UserView struct {
	UserID      int64  `json:"userId"`
	UserName    string `json:"userName"`
	FaceURL     string `json:"faceUrl"`
	IsVIP       bool   `json:"isVip"`
	VIPLabel    string `json:"vipLabel,omitempty"`
	VIPForever  bool   `json:"vipForever,omitempty"`
	VIPExpireAt string `json:"vipExpireAt,omitempty"`
	SpaceTotal  int64  `json:"spaceTotal,omitempty"`
	SpaceUsed   int64  `json:"spaceUsed,omitempty"`
	SpaceRemain int64  `json:"spaceRemain,omitempty"`
}

type Breadcrumb struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FileItem struct {
	FileID       string `json:"fileId"`
	ParentID     string `json:"parentId"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	PickCode     string `json:"pickCode"`
	IsDirectory  bool   `json:"isDirectory"`
	IsVideo      bool   `json:"isVideo"`
	UpdatedAt    string `json:"updatedAt"`
	DurationSec  int64  `json:"durationSec,omitempty"`
	ResumeMS     int64  `json:"resumeMs,omitempty"`
	SubtitlePath string `json:"subtitlePath,omitempty"`
	LastPlayedAt string `json:"lastPlayedAt,omitempty"`
}

type DirectoryView struct {
	DirID    string       `json:"dirId"`
	ParentID string       `json:"parentId"`
	Name     string       `json:"name"`
	Path     []Breadcrumb `json:"path"`
	Count    int          `json:"count"`
	Offset   int          `json:"offset"`
	Limit    int          `json:"limit"`
	HasMore  bool         `json:"hasMore"`
	Items    []FileItem   `json:"items"`
}

type LoginSessionView struct {
	SessionID      string `json:"sessionId"`
	QRCodeDataURL  string `json:"qrCodeDataUrl"`
	QRCodeContent  string `json:"qrCodeContent"`
	ExpiresIn      int64  `json:"expiresIn"`
	CreatedUnixSec int64  `json:"createdUnixSec"`
}

type LoginStatusView struct {
	State    string    `json:"state"`
	Message  string    `json:"message"`
	LoggedIn bool      `json:"loggedIn"`
	User     *UserView `json:"user,omitempty"`
}

type loginSession struct {
	id        string
	client    *driver.Pan115Client
	qr        *driver.QRCodeSession
	createdAt time.Time
}

type Service struct {
	mu sync.RWMutex

	client     *driver.Pan115Client
	credential *config.Credential
	cookies    map[string]string
	user       *UserView
	sessions   map[string]*loginSession
}

func NewService() *Service {
	return &Service{
		sessions: make(map[string]*loginSession),
	}
}

func (s *Service) StartQRCodeLogin() (*LoginSessionView, error) {
	client := newClient()
	qrSession, err := client.QRCodeStart()
	if err != nil {
		return nil, err
	}

	image, err := qrSession.QRCode()
	if err != nil {
		return nil, err
	}

	sessionID := uuid.NewString()

	s.mu.Lock()
	s.sessions[sessionID] = &loginSession{
		id:        sessionID,
		client:    client,
		qr:        qrSession,
		createdAt: time.Now(),
	}
	s.mu.Unlock()

	return &LoginSessionView{
		SessionID:      sessionID,
		QRCodeDataURL:  "data:image/png;base64," + base64.StdEncoding.EncodeToString(image),
		QRCodeContent:  qrSession.QrcodeContent,
		ExpiresIn:      180,
		CreatedUnixSec: time.Now().Unix(),
	}, nil
}

func (s *Service) CheckQRCodeLogin(sessionID string) (*LoginStatusView, *config.Credential, error) {
	s.mu.RLock()
	entry := s.sessions[sessionID]
	s.mu.RUnlock()

	if entry == nil {
		return nil, nil, errors.New("二维码会话不存在，请重新生成")
	}

	status, err := entry.client.QRCodeStatus(entry.qr)
	if err != nil {
		return nil, nil, err
	}

	switch {
	case status.IsWaiting():
		return &LoginStatusView{State: "waiting", Message: "等待扫码"}, nil, nil
	case status.IsScanned():
		return &LoginStatusView{State: "scanned", Message: "已扫码，等待在手机上确认"}, nil, nil
	case status.IsCanceled():
		s.deleteSession(sessionID)
		return &LoginStatusView{State: "canceled", Message: "本次扫码已取消"}, nil, nil
	case status.IsExpired():
		s.deleteSession(sessionID)
		return &LoginStatusView{State: "expired", Message: "二维码已过期，请刷新"}, nil, nil
	case !status.IsAllowed():
		return &LoginStatusView{State: "unknown", Message: "登录状态未知，请重试"}, nil, nil
	}

	credential, err := entry.client.QRCodeLogin(entry.qr)
	if err != nil || credential == nil {
		appCredential, loginErr := loginWithQRCode(entry.client, entry.qr)
		if loginErr != nil {
			if err != nil {
				return nil, nil, err
			}
			return nil, nil, loginErr
		}
		credential = &driver.Credential{
			UID:  appCredential.UID,
			CID:  appCredential.CID,
			SEID: appCredential.SEID,
			KID:  appCredential.KID,
		}
	}

	appCredential := &config.Credential{
		UID:  credential.UID,
		CID:  credential.CID,
		SEID: credential.SEID,
		KID:  credential.KID,
	}

	ok, user, err := s.Restore(appCredential, nil)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, errors.New("扫码成功，但恢复登录失败")
	}

	s.deleteSession(sessionID)
	return &LoginStatusView{
		State:    "authenticated",
		Message:  "登录成功",
		LoggedIn: true,
		User:     user,
	}, appCredential, nil
}

func (s *Service) Restore(credential *config.Credential, cookies map[string]string) (bool, *UserView, error) {
	if credential == nil && len(cookies) == 0 {
		return false, nil, nil
	}

	client := newClient()
	importCookies(client, cookies)
	if credential != nil {
		client.ImportCredential(&driver.Credential{
			UID:  credential.UID,
			CID:  credential.CID,
			SEID: credential.SEID,
			KID:  credential.KID,
		})
	}

	view, err := fetchUserView(client)
	if err != nil {
		if _, listErr := client.ListPage("0", 0, 1); listErr != nil {
			return false, nil, nil
		}
		view = &UserView{
			UserName: "115 用户",
		}
	}

	snapshot := exportCookies(client)
	normalized := normalizeCredential(credential, snapshot)

	s.mu.Lock()
	s.client = client
	s.credential = normalized
	s.cookies = snapshot
	s.user = view
	s.mu.Unlock()

	return true, view, nil
}

func (s *Service) CurrentUser() (*UserView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.user == nil {
		return nil, false
	}
	user := *s.user
	return &user, true
}

func (s *Service) Logout() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.client = nil
	s.credential = nil
	s.cookies = nil
	s.user = nil
	s.sessions = make(map[string]*loginSession)
}

func (s *Service) ListDirectory(dirID string, offset, limit int) (*DirectoryView, error) {
	client, err := s.authenticatedClient()
	if err != nil {
		return nil, err
	}

	dirID = strings.TrimSpace(dirID)
	if dirID == "" {
		dirID = "0"
	}
	offset, limit = normalizePageRequest(offset, limit, defaultListPageSize, maxListPageSize)

	result, err := s.listDirectoryPageWithRetry(client, dirID, offset, limit)
	if err != nil {
		return nil, err
	}

	items := make([]FileItem, 0, len(result.Items))
	for _, fileInfo := range result.Items {
		items = append(items, fileItemFromRaw(fileInfo))
	}

	view := &DirectoryView{
		DirID:    dirID,
		ParentID: "",
		Name:     "我的文件",
		Path: []Breadcrumb{
			{ID: "0", Name: "我的文件"},
		},
		Count:   result.Count,
		Offset:  result.Offset,
		Limit:   resolvePageLimit(result.Limit, limit),
		HasMore: result.Offset+len(items) < result.Count,
		Items:   items,
	}

	if rawPath := breadcrumbsFromRawPath(result.Path); len(rawPath) > 0 {
		view.Path = rawPath
		view.Name = rawPath[len(rawPath)-1].Name
		if len(rawPath) >= 2 {
			view.ParentID = rawPath[len(rawPath)-2].ID
		}
	}

	if dirID == "0" {
		return view, nil
	}

	if len(view.Path) > 1 && strings.TrimSpace(view.Name) != "" {
		return view, nil
	}

	if file, fileErr := client.GetFile(dirID); fileErr == nil {
		view.ParentID = file.ParentID
		if name := strings.TrimSpace(file.Name); name != "" {
			view.Name = name
		}
	}

	stat, err := client.Stat(dirID)
	if err == nil {
		view.Name = stat.Name
		view.Path = buildBreadcrumbs(dirID, stat)
		if view.ParentID == "" {
			view.ParentID = guessParentID(dirID, stat)
		}
	}

	return view, nil
}

func (s *Service) listDirectoryPageWithRetry(client *driver.Pan115Client, dirID string, offset, limit int) (*rawFileListResp, error) {
	var lastErr error

	for attempt := 0; attempt < fileAPIRetryCount; attempt++ {
		result, err := s.listDirectoryPage(client, dirID, offset, limit)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !ShouldRetryTemporaryHTMLResponseError(err) || attempt == fileAPIRetryCount-1 {
			break
		}
		time.Sleep(fileAPIRetryDelay)
	}

	return nil, normalizeRemoteJSONHTMLError(lastErr, "115 目录接口暂时返回了异常页面，请稍后重试。")
}
func (s *Service) DownloadInfo(pickCode string) (*driver.DownloadInfo, error) {
	client, err := s.authenticatedClient()
	if err != nil {
		return nil, err
	}

	info, err := client.Download(strings.TrimSpace(pickCode))
	if err != nil {
		return nil, err
	}

	if info.Header == nil {
		info.Header = make(map[string][]string)
	}

	info.Header.Set("Referer", driver.CookieUrl)

	mergedCookies := exportCookies(client)
	for key, value := range exportCookiesForURL(client, info.Url.Url) {
		mergedCookies[key] = value
	}
	if cookieHeader := cookiesToHeader(mergedCookies); cookieHeader != "" {
		info.Header.Set("Cookie", cookieHeader)
	}

	return info, nil
}

func (s *Service) StreamClient() (*http.Client, error) {
	client, err := s.authenticatedClient()
	if err != nil {
		return nil, err
	}

	httpClient := client.Client.GetClient()
	if httpClient == nil {
		return nil, errors.New("115 http client unavailable")
	}

	jar := httpClient.Jar
	baseTransport := httpClient.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	return &http.Client{
		Transport: &jarRoundTripper{
			base: baseTransport,
			jar:  jar,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 {
				req.Header = via[len(via)-1].Header.Clone()
			}
			return nil
		},
		Timeout: 0,
	}, nil
}

func (s *Service) authenticatedClient() (*driver.Pan115Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.client == nil {
		return nil, errors.New("你还没有登录 115")
	}
	return s.client, nil
}

func (s *Service) deleteSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func (s *Service) SessionSnapshot() (*config.Credential, map[string]string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var credential *config.Credential
	if s.credential != nil {
		copyValue := *s.credential
		credential = &copyValue
	}

	cookies := make(map[string]string, len(s.cookies))
	for key, value := range s.cookies {
		cookies[key] = value
	}

	return credential, cookies
}

func buildBreadcrumbs(dirID string, stat *driver.FileStatInfo) []Breadcrumb {
	crumbs := []Breadcrumb{{ID: "0", Name: "我的文件"}}
	seen := map[string]bool{"0": true}

	for _, parent := range stat.Parents {
		if parent == nil || parent.ID == "" || seen[parent.ID] {
			continue
		}
		crumbs = append(crumbs, Breadcrumb{
			ID:   parent.ID,
			Name: parent.Name,
		})
		seen[parent.ID] = true
	}

	if !seen[dirID] {
		crumbs = append(crumbs, Breadcrumb{
			ID:   dirID,
			Name: stat.Name,
		})
	}

	return crumbs
}

func guessParentID(dirID string, stat *driver.FileStatInfo) string {
	if stat == nil || len(stat.Parents) == 0 {
		return "0"
	}

	last := stat.Parents[len(stat.Parents)-1]
	if last != nil && last.ID != "" && last.ID != dirID {
		return last.ID
	}

	if len(stat.Parents) >= 2 {
		prev := stat.Parents[len(stat.Parents)-2]
		if prev != nil && prev.ID != "" {
			return prev.ID
		}
	}

	return "0"
}

func isVideo(name string, isDirectory bool) bool {
	if isDirectory {
		return false
	}

	switch strings.ToLower(path.Ext(name)) {
	case ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".m4v", ".rmvb", ".ts", ".webm":
		return true
	default:
		return false
	}
}

func newClient() *driver.Pan115Client {
	return driver.New(driver.UA(driver.UA115Browser))
}

type jarRoundTripper struct {
	base http.RoundTripper
	jar  http.CookieJar
}

func (t *jarRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.jar != nil {
		for _, cookie := range t.jar.Cookies(req.URL) {
			req.AddCookie(cookie)
		}
	}
	return t.base.RoundTrip(req)
}

func importCookies(client *driver.Pan115Client, cookies map[string]string) {
	if client == nil || len(cookies) == 0 {
		return
	}

	client.ImportCookies(cookies,
		driver.CookieDomain115,
		"115.com",
		"passportapi.115.com",
		"webapi.115.com",
		"proapi.115.com",
		"my.115.com",
	)
}

func exportCookies(client *driver.Pan115Client) map[string]string {
	result := map[string]string{}
	if client == nil || client.Client == nil || client.Client.GetClient() == nil || client.Client.GetClient().Jar == nil {
		return result
	}

	u, err := url.Parse(driver.CookieUrl)
	if err != nil {
		return result
	}

	for _, cookie := range client.Client.GetClient().Jar.Cookies(u) {
		if cookie == nil || strings.TrimSpace(cookie.Name) == "" {
			continue
		}
		result[strings.ToUpper(cookie.Name)] = cookie.Value
	}

	return result
}

func exportCookiesForURL(client *driver.Pan115Client, rawURL string) map[string]string {
	result := map[string]string{}
	if client == nil || client.Client == nil || client.Client.GetClient() == nil || client.Client.GetClient().Jar == nil {
		return result
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return result
	}

	for _, cookie := range client.Client.GetClient().Jar.Cookies(u) {
		if cookie == nil || strings.TrimSpace(cookie.Name) == "" {
			continue
		}
		result[strings.ToUpper(cookie.Name)] = cookie.Value
	}

	return result
}

func normalizeCredential(credential *config.Credential, cookies map[string]string) *config.Credential {
	if credential == nil {
		credential = &config.Credential{}
	} else {
		copyValue := *credential
		credential = &copyValue
	}

	if credential.UID == "" {
		credential.UID = cookies["UID"]
	}
	if credential.CID == "" {
		credential.CID = cookies["CID"]
	}
	if credential.SEID == "" {
		credential.SEID = cookies["SEID"]
	}
	if credential.KID == "" {
		credential.KID = cookies["KID"]
	}

	if !credentialValid(credential) {
		return nil
	}
	return credential
}

func cookiesToHeader(cookies map[string]string) string {
	if len(cookies) == 0 {
		return ""
	}

	keys := make([]string, 0, len(cookies))
	for key := range cookies {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(cookies[key]) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+cookies[key])
	}

	return strings.Join(parts, "; ")
}

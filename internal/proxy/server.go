package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	driver "github.com/jianxcao/115driver/pkg/driver"
)

type downloadProvider interface {
	DownloadInfo(pickCode string) (*driver.DownloadInfo, error)
	StreamClient() (*http.Client, error)
}

type Server struct {
	provider   downloadProvider
	logger     *slog.Logger
	server     *http.Server
	listener   net.Listener
	baseURL    string
	client     *http.Client
	subtitleMu sync.RWMutex
	subtitles  map[string]subtitleEntry
}

type subtitleEntry struct {
	path string
	typ  string
}

func NewServer(provider downloadProvider, logger *slog.Logger) *Server {
	return &Server{
		provider:  provider,
		logger:    logger,
		subtitles: map[string]subtitleEntry{},
		client: &http.Client{
			Timeout: 0,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) == 0 {
					return nil
				}
				prev := via[len(via)-1]
				req.Header = prev.Header.Clone()
				return nil
			},
		},
	}
}

func (s *Server) Start() error {
	if s.listener != nil {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/stream", s.handleStream)
	mux.HandleFunc("/avatar", s.handleAvatar)
	mux.HandleFunc("/image", s.handleImage)
	mux.HandleFunc("/subtitle", s.handleSubtitle)
	mux.HandleFunc("/healthz", s.handleHealth)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	s.listener = listener
	s.baseURL = "http://" + listener.Addr().String()
	s.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("proxy serve failed", "error", err)
		}
	}()

	return nil
}

func (s *Server) Close(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) BaseURL() string {
	return s.baseURL
}

func (s *Server) StreamURL(pickCode, name string) string {
	query := url.Values{}
	query.Set("pickcode", pickCode)
	query.Set("name", name)
	return fmt.Sprintf("%s/stream?%s", s.baseURL, query.Encode())
}

func (s *Server) ImageURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || s.baseURL == "" {
		return ""
	}

	query := url.Values{}
	query.Set("url", rawURL)
	return fmt.Sprintf("%s/image?%s", s.baseURL, query.Encode())
}

func (s *Server) ImagePathURL(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || s.baseURL == "" {
		return ""
	}

	query := url.Values{}
	query.Set("path", path)
	return fmt.Sprintf("%s/image?%s", s.baseURL, query.Encode())
}

func (s *Server) SubtitleURL(path string) (string, string, bool) {
	path = strings.TrimSpace(path)
	if path == "" || s.baseURL == "" {
		return "", "", false
	}

	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", "", false
	}

	typ, ok := subtitleType(resolved)
	if !ok {
		return "", "", false
	}

	token := uuid.NewString()
	s.subtitleMu.Lock()
	s.subtitles[token] = subtitleEntry{
		path: resolved,
		typ:  typ,
	}
	s.subtitleMu.Unlock()

	query := url.Values{}
	query.Set("token", token)
	return fmt.Sprintf("%s/subtitle?%s", s.baseURL, query.Encode()), typ, true
}

func (s *Server) Probe(pickCode, name string) error {
	if strings.TrimSpace(pickCode) == "" {
		return errors.New("缺少 pickcode")
	}

	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, s.StreamURL(pickCode, name), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", "bytes=0-1")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("流预检失败: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2))
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	allowProxyCORS(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = io.WriteString(w, `{"ok":true}`)
}

func (s *Server) handleAvatar(w http.ResponseWriter, r *http.Request) {
	allowProxyCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}

	targetURL, err := url.Parse(rawURL)
	if err != nil || targetURL == nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	if !strings.EqualFold(targetURL.Scheme, "http") && !strings.EqualFold(targetURL.Scheme, "https") {
		http.Error(w, "unsupported scheme", http.StatusBadRequest)
		return
	}

	host := strings.ToLower(strings.TrimSpace(targetURL.Hostname()))
	if host == "" || !strings.Contains(host, "115") {
		http.Error(w, "invalid avatar host", http.StatusBadRequest)
		return
	}

	client, err := s.provider.StreamClient()
	if err != nil {
		s.logger.Error("avatar proxy failed to create stream client", "url", rawURL, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL.String(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	req.Header.Set("Referer", driver.CookieUrl)
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")
	req.Host = req.URL.Host

	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("avatar proxy upstream request failed", "url", rawURL, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for _, key := range []string{
		"Content-Length",
		"Content-Type",
		"ETag",
		"Last-Modified",
		"Cache-Control",
	} {
		if value := resp.Header.Get(key); value != "" {
			w.Header().Set(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		s.logger.Error("avatar proxy upstream returned error", "url", rawURL, "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		_, _ = w.Write(body)
		return
	}
	if r.Method == http.MethodHead {
		return
	}

	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleImage(w http.ResponseWriter, r *http.Request) {
	allowProxyCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	localPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if localPath != "" {
		s.serveLocalImage(w, r, localPath)
		return
	}

	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" {
		http.Error(w, "missing image url", http.StatusBadRequest)
		return
	}

	targetURL, err := url.Parse(rawURL)
	if err != nil || targetURL == nil {
		http.Error(w, "invalid image url", http.StatusBadRequest)
		return
	}

	if !strings.EqualFold(targetURL.Scheme, "http") && !strings.EqualFold(targetURL.Scheme, "https") {
		http.Error(w, "unsupported image scheme", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 seven-player/1.0")
	if referer := imageReferer(targetURL.Hostname()); referer != "" {
		req.Header.Set("Referer", referer)
	}
	req.Host = req.URL.Host

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error("image proxy upstream request failed", "url", rawURL, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for _, key := range []string{
		"Accept-Ranges",
		"Content-Length",
		"Content-Type",
		"ETag",
		"Last-Modified",
		"Cache-Control",
	} {
		if value := resp.Header.Get(key); value != "" {
			w.Header().Set(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		s.logger.Error("image proxy upstream returned error", "url", rawURL, "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		_, _ = w.Write(body)
		return
	}
	if r.Method == http.MethodHead {
		return
	}

	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	allowProxyCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	pickCode := strings.TrimSpace(r.URL.Query().Get("pickcode"))
	if pickCode == "" {
		http.Error(w, "missing pickcode", http.StatusBadRequest)
		return
	}

	info, err := s.provider.DownloadInfo(pickCode)
	if err != nil {
		s.logger.Error("proxy failed to resolve download info", "pickcode", pickCode, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	client, err := s.provider.StreamClient()
	if err != nil {
		s.logger.Error("proxy failed to create stream client", "pickcode", pickCode, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, info.Url.Url, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	for key, values := range info.Header {
		if strings.EqualFold(key, "Accept-Encoding") {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	req.Host = req.URL.Host

	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("proxy upstream request failed", "pickcode", pickCode, "url", info.Url.Url, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for _, key := range []string{
		"Accept-Ranges",
		"Content-Disposition",
		"Content-Length",
		"Content-Range",
		"Content-Type",
		"ETag",
		"Last-Modified",
		"Cache-Control",
	} {
		if value := resp.Header.Get(key); value != "" {
			w.Header().Set(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		s.logger.Error("proxy upstream returned error", "pickcode", pickCode, "status", resp.StatusCode, "body", strings.TrimSpace(string(body)))
		_, _ = w.Write(body)
		return
	}
	if r.Method == http.MethodHead {
		return
	}

	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleSubtitle(w http.ResponseWriter, r *http.Request) {
	allowProxyCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	s.subtitleMu.RLock()
	entry, ok := s.subtitles[token]
	s.subtitleMu.RUnlock()
	if !ok {
		http.Error(w, "subtitle not found", http.StatusNotFound)
		return
	}

	data, err := os.ReadFile(entry.path)
	if err != nil {
		s.logger.Error("subtitle proxy failed to read file", "path", entry.path, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", subtitleContentType(entry.typ))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(data)
}

func subtitleType(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".vtt":
		return "vtt", true
	case ".srt":
		return "srt", true
	case ".ass":
		return "ass", true
	case ".ssa":
		return "ssa", true
	default:
		return "", false
	}
}

func subtitleContentType(typ string) string {
	if typ == "vtt" {
		return "text/vtt; charset=utf-8"
	}
	return "text/plain; charset=utf-8"
}

func allowProxyCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")
	w.Header().Set("Access-Control-Expose-Headers", "Accept-Ranges, Content-Length, Content-Range, Content-Type")
}

func (s *Server) serveLocalImage(w http.ResponseWriter, r *http.Request, localPath string) {
	resolved, err := filepath.Abs(strings.TrimSpace(localPath))
	if err != nil {
		http.Error(w, "invalid image path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if info.IsDir() {
		http.Error(w, "invalid image path", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, resolved)
}

func imageReferer(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	switch {
	case strings.Contains(host, "doubanio.com"), strings.Contains(host, "douban.com"):
		return "https://movie.douban.com/"
	case strings.Contains(host, "qpic.cn"), strings.Contains(host, "gtimg.com"), strings.Contains(host, "qq.com"):
		return "https://v.qq.com/"
	case strings.Contains(host, "bangumi.tv"), strings.Contains(host, "bgm.tv"):
		return "https://bangumi.tv/"
	default:
		return ""
	}
}

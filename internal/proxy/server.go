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
	"strings"
	"time"

	driver "github.com/jianxcao/115driver/pkg/driver"
)

type downloadProvider interface {
	DownloadInfo(pickCode string) (*driver.DownloadInfo, error)
	StreamClient() (*http.Client, error)
}

type Server struct {
	provider downloadProvider
	logger   *slog.Logger
	server   *http.Server
	listener net.Listener
	baseURL  string
	client   *http.Client
}

func NewServer(provider downloadProvider, logger *slog.Logger) *Server {
	return &Server{
		provider: provider,
		logger:   logger,
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = io.WriteString(w, `{"ok":true}`)
}

func (s *Server) handleAvatar(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
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

package proxy

import (
	"strings"
	"testing"
)

func TestStreamURLIncludesFileNameInPath(t *testing.T) {
	server := &Server{baseURL: "http://127.0.0.1:1234"}
	got := server.StreamURL("abc123", "刺杀小说家2.2160p高清版.mkv")

	if !strings.Contains(got, "/stream/%E5%88%BA%E6%9D%80%E5%B0%8F%E8%AF%B4%E5%AE%B62.2160p%E9%AB%98%E6%B8%85%E7%89%88.mkv?") {
		t.Fatalf("stream URL should include escaped file name in path: %s", got)
	}
	if !strings.Contains(got, "pickcode=abc123") || !strings.Contains(got, "name=") {
		t.Fatalf("stream URL missing query values: %s", got)
	}
}

func TestContentDispositionUsesFileName(t *testing.T) {
	got := contentDisposition(`folder\Demo "Clip".mkv`)

	if !strings.Contains(got, `filename="Demo \"Clip\".mkv"`) {
		t.Fatalf("content disposition missing quoted filename: %s", got)
	}
	if !strings.Contains(got, "filename*=UTF-8''Demo%20%22Clip%22.mkv") {
		t.Fatalf("content disposition missing encoded filename: %s", got)
	}
}

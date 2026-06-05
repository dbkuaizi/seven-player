package pan

import (
	"crypto/md5"
	"encoding/hex"
	"testing"
)

func TestIsVideo(t *testing.T) {
	cases := []struct {
		name        string
		isDirectory bool
		want        bool
	}{
		{name: "movie.mp4", want: true},
		{name: "movie.MKV", want: true},
		{name: "notes.txt", want: false},
		{name: "folder", isDirectory: true, want: false},
	}

	for _, tc := range cases {
		if got := isVideo(tc.name, tc.isDirectory); got != tc.want {
			t.Fatalf("isVideo(%q, %v) = %v, want %v", tc.name, tc.isDirectory, got, tc.want)
		}
	}
}

func TestBreadcrumbsFromRawPath(t *testing.T) {
	got := breadcrumbsFromRawPath([]rawPathItem{
		{CategoryID: "0", Name: "根目录"},
		{CategoryID: "3320172536691425004", Name: "爆米花TV"},
		{CategoryID: "3320556682207035274", Name: "电视剧"},
	})

	if len(got) != 3 {
		t.Fatalf("breadcrumbsFromRawPath() length = %d, want 3", len(got))
	}
	if got[0].ID != "0" || got[0].Name != "我的文件" {
		t.Fatalf("unexpected root breadcrumb: %+v", got[0])
	}
	if got[1].Name != "爆米花TV" || got[2].Name != "电视剧" {
		t.Fatalf("unexpected breadcrumbs: %+v", got)
	}
}

func TestMD5Hex(t *testing.T) {
	want := md5.Sum([]byte("123456"))
	got := md5Hex("123456")
	if got != hex.EncodeToString(want[:]) {
		t.Fatalf("md5Hex mismatch: got %q want %q", got, hex.EncodeToString(want[:]))
	}
}

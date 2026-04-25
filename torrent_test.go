package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

func TestBuildMagnetFromTorrentData(t *testing.T) {
	bstr := func(value string) string {
		return fmt.Sprintf("%d:%s", len(value), value)
	}

	info := "d" +
		bstr("length") + "i12345e" +
		bstr("name") + bstr("test.iso") +
		bstr("piece length") + "i16384e" +
		bstr("pieces") + bstr("01234567890123456789") +
		"e"

	trackerA := "http://tracker.example/announce"
	trackerB := "udp://tracker.example:80/announce"
	data := "d" +
		bstr("announce") + bstr(trackerA) +
		bstr("announce-list") + "ll" + bstr(trackerA) + "el" + bstr(trackerB) + "ee" +
		bstr("info") + info +
		"e"

	magnet, err := buildMagnetFromTorrentData([]byte(data))
	if err != nil {
		t.Fatalf("buildMagnetFromTorrentData() error = %v", err)
	}

	parsed, err := url.Parse(magnet)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Scheme != "magnet" {
		t.Fatalf("unexpected scheme: %q", parsed.Scheme)
	}

	query := parsed.Query()
	sum := sha1.Sum([]byte(info))
	wantHash := strings.ToUpper(hex.EncodeToString(sum[:]))
	if got := query.Get("xt"); got != "urn:btih:"+wantHash {
		t.Fatalf("xt mismatch: got %q want %q", got, "urn:btih:"+wantHash)
	}
	if got := query.Get("dn"); got != "test.iso" {
		t.Fatalf("dn mismatch: got %q", got)
	}

	trackers := query["tr"]
	if len(trackers) != 2 {
		t.Fatalf("trackers mismatch: %+v", trackers)
	}
	if trackers[0] != trackerA || trackers[1] != trackerB {
		t.Fatalf("trackers order mismatch: %+v", trackers)
	}
}

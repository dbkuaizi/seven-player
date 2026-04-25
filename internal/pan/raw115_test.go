package pan

import (
	"net/http"
	"testing"
)

func TestCredentialFromMap(t *testing.T) {
	credential := credentialFromMap(map[string]any{
		"UID":  "u1",
		"CID":  "c1",
		"SEID": "s1",
		"KID":  "k1",
	})
	if credential == nil {
		t.Fatal("credentialFromMap returned nil")
	}
	if credential.UID != "u1" || credential.CID != "c1" || credential.SEID != "s1" || credential.KID != "k1" {
		t.Fatalf("unexpected credential: %+v", credential)
	}
}

func TestCredentialFromAnyStringCookie(t *testing.T) {
	credential := credentialFromAny("UID=u2; CID=c2; SEID=s2; KID=k2")
	if credential == nil {
		t.Fatal("credentialFromAny returned nil")
	}
	if credential.UID != "u2" || credential.CID != "c2" || credential.SEID != "s2" || credential.KID != "k2" {
		t.Fatalf("unexpected credential: %+v", credential)
	}
}

func TestCredentialFromResponseCookies(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Add("Set-Cookie", "UID=u3")
	resp.Header.Add("Set-Cookie", "CID=c3")
	resp.Header.Add("Set-Cookie", "SEID=s3")
	resp.Header.Add("Set-Cookie", "KID=k3")

	credential := credentialFromResponseCookies(resp)
	if credential == nil {
		t.Fatal("credentialFromResponseCookies returned nil")
	}
	if credential.UID != "u3" || credential.CID != "c3" || credential.SEID != "s3" || credential.KID != "k3" {
		t.Fatalf("unexpected credential: %+v", credential)
	}
}

func TestExtractFaceURL(t *testing.T) {
	got := extractFaceURL(map[string]any{
		"face_m": "https://example.com/m.png",
	})
	if got != "https://example.com/m.png" {
		t.Fatalf("extractFaceURL got %q", got)
	}
}

func TestMergeUserViewFromOpenProfile(t *testing.T) {
	view := &UserView{}

	mergeUserViewFromOpenProfile(view, map[string]any{
		"user_info": map[string]any{
			"user_id":   float64(7),
			"user_name": "双双筷子",
			"face":      "https://example.com/avatar.png",
		},
		"vip_info": map[string]any{
			"level_name": "超级 VIP",
			"expire":     float64(1_777_777_777),
		},
		"rt_space_info": map[string]any{
			"all_total":  map[string]any{"size": float64(10 * 1024 * 1024 * 1024)},
			"all_use":    map[string]any{"size": float64(4 * 1024 * 1024 * 1024)},
			"all_remain": map[string]any{"size": float64(6 * 1024 * 1024 * 1024)},
		},
	})

	if view.UserID != 7 || view.UserName != "双双筷子" {
		t.Fatalf("unexpected user basics: %+v", view)
	}
	if view.FaceURL != "https://example.com/avatar.png" {
		t.Fatalf("unexpected face url: %q", view.FaceURL)
	}
	if !view.IsVIP || view.VIPLabel != "超级 VIP" || view.VIPExpireAt == "" {
		t.Fatalf("unexpected vip info: %+v", view)
	}
	if view.SpaceTotal != 10*1024*1024*1024 || view.SpaceUsed != 4*1024*1024*1024 || view.SpaceRemain != 6*1024*1024*1024 {
		t.Fatalf("unexpected space info: %+v", view)
	}
}

func TestMergeUserViewFromIndexInfo(t *testing.T) {
	view := &UserView{}

	mergeUserViewFromIndexInfo(view, map[string]any{
		"space_info": map[string]any{
			"all_total": map[string]any{
				"size":        45498297723793.23,
				"size_format": "41.38TB",
			},
			"all_remain": map[string]any{
				"size":        40312287513386.23,
				"size_format": "36.66TB",
			},
			"all_use": map[string]any{
				"size":        5186010210407.0,
				"size_format": "4.72TB",
			},
		},
	})

	if view.SpaceTotal != 45498297723793 || view.SpaceRemain != 40312287513386 || view.SpaceUsed != 5186010210407 {
		t.Fatalf("unexpected index space info: %+v", view)
	}
}

func TestParseCookieInput(t *testing.T) {
	credential, cookies, err := parseCookieInput("Cookie: UID=u1; CID=c1; SEID=s1; KID=k1; foo=bar")
	if err != nil {
		t.Fatalf("parseCookieInput returned error: %v", err)
	}
	if credential == nil {
		t.Fatal("parseCookieInput returned nil credential")
	}
	if credential.UID != "u1" || credential.CID != "c1" || credential.SEID != "s1" || credential.KID != "k1" {
		t.Fatalf("unexpected credential: %+v", credential)
	}
	if cookies["FOO"] != "bar" {
		t.Fatalf("unexpected cookies: %+v", cookies)
	}
}

func TestParseCookieInputRejectsMissingFields(t *testing.T) {
	if _, _, err := parseCookieInput("UID=u1; foo=bar"); err == nil {
		t.Fatal("expected parseCookieInput to reject incomplete cookie")
	}
}

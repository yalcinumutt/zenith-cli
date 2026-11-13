package appapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.2.0", "0.1.0-beta", true},
		{"0.1.0", "0.1.0-beta", true},
		{"0.1.0-beta", "0.1.0", false},
		{"0.1.0-beta", "0.1.0-beta", false},
		{"0.10.0", "0.9.0", true},
		{"1.0.0", "1.0.0", false},
	}
	for _, c := range cases {
		if got := versionNewer(c.a, c.b); got != c.want {
			t.Errorf("versionNewer(%q,%q)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestNewestDesktopIgnoresCLIAndDrafts(t *testing.T) {
	list := []ghRelease{
		{Tag: "v9.9.9", URL: "cli"},
		{Tag: "desktop-v0.3.0", URL: "draft", Draft: true},
		{Tag: "desktop-v0.2.0", URL: "ok", Body: "notes"},
		{Tag: "desktop-v0.1.0-beta", URL: "old"},
	}
	got := newestDesktop(list, "0.1.0-beta")
	if !got.Available || got.Version != "0.2.0" || got.URL != "ok" {
		t.Fatalf("got %+v", got)
	}
	if newestDesktop(list, "0.2.0").Available {
		t.Fatal("up to date must not report an update")
	}
}

func TestCheckForUpdateOverHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"tag_name":"desktop-v0.2.0","html_url":"https://x/r","body":"hi"}]`))
	}))
	defer srv.Close()
	s := &Service{version: "0.1.0-beta", releasesURL: srv.URL}
	got, err := s.CheckForUpdate()
	if err != nil || !got.Available || got.URL != "https://x/r" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

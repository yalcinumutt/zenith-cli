package appapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Desktop releases live beside the CLI's in one repository but are tagged
// `desktop-v*` (ADR 0009), so "latest" on GitHub is not ours: the check walks
// the release list and picks the newest desktop tag itself.
const (
	releasesURL   = "https://api.github.com/repos/yalcinumutt/zenith-cli/releases?per_page=30"
	desktopPrefix = "desktop-v"
)

// UpdateInfo says whether a newer desktop release exists. The app only informs:
// the installer assets (.dmg, .exe) cannot be swapped in place, so the front end
// sends the user to URL to download.
type UpdateInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	URL       string `json:"url"`
	Notes     string `json:"notes"`
}

type ghRelease struct {
	Tag     string `json:"tag_name"`
	URL     string `json:"html_url"`
	Body    string `json:"body"`
	Draft   bool   `json:"draft"`
	Created string `json:"created_at"`
}

// CheckForUpdate asks GitHub for the newest desktop release. Network trouble is
// an error the front end ignores quietly; being offline is normal.
func (s *Service) CheckForUpdate() (UpdateInfo, error) {
	url := s.releasesURL
	if url == "" {
		url = releasesURL
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return UpdateInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UpdateInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{}, fmt.Errorf("update check: GitHub answered %d", resp.StatusCode)
	}
	var list []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return UpdateInfo{}, err
	}
	return newestDesktop(list, s.version), nil
}

// newestDesktop picks the highest desktop-v* release and reports it only when it
// is newer than current.
func newestDesktop(list []ghRelease, current string) UpdateInfo {
	var best ghRelease
	bestV := ""
	for _, r := range list {
		if r.Draft || !strings.HasPrefix(r.Tag, desktopPrefix) {
			continue
		}
		v := strings.TrimPrefix(r.Tag, desktopPrefix)
		if bestV == "" || versionNewer(v, bestV) {
			best, bestV = r, v
		}
	}
	if bestV == "" || !versionNewer(bestV, current) {
		return UpdateInfo{}
	}
	return UpdateInfo{Available: true, Version: bestV, URL: best.URL, Notes: best.Body}
}

// versionNewer reports whether a is newer than b for versions like "0.2.0" or
// "0.1.0-beta". A pre-release suffix sorts below the same number without one.
func versionNewer(a, b string) bool {
	an, apre := splitVersion(a)
	bn, bpre := splitVersion(b)
	for i := 0; i < 3; i++ {
		if an[i] != bn[i] {
			return an[i] > bn[i]
		}
	}
	if apre == bpre {
		return false
	}
	if apre == "" {
		return true
	}
	if bpre == "" {
		return false
	}
	return apre > bpre
}

func splitVersion(v string) ([3]int, string) {
	v = strings.TrimPrefix(v, "v")
	pre := ""
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v, pre = v[:i], v[i+1:]
	}
	var n [3]int
	for i, part := range strings.SplitN(v, ".", 3) {
		n[i], _ = strconv.Atoi(part)
	}
	return n, pre
}

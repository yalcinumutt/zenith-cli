package cmd

import (
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
)

func TestUpdateSlugMatchesModuleOwner(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("no build info")
	}
	want := strings.TrimPrefix(info.Main.Path, "github.com/")
	if updateRepoSlug != want {
		t.Errorf("update slug %q does not match module %q", updateRepoSlug, want)
	}
	if !strings.HasPrefix(updateRepoSlug, "yalcinumutt/") {
		t.Errorf("update slug %q must use the canonical owner yalcinumutt", updateRepoSlug)
	}
}

func TestUpdateFilterIgnoresNonCLIAssets(t *testing.T) {
	re := regexp.MustCompile(strings.Join(updateAssetFilters, "|"))
	for _, name := range []string{
		"zenith-cli_darwin_arm64.tar.gz",
		"zenith-cli_linux_amd64.tar.gz",
		"zenith-cli_windows_amd64.zip",
	} {
		if !re.MatchString(name) {
			t.Errorf("CLI asset %q should be selectable", name)
		}
	}
	for _, name := range []string{
		"Zenith-0.1.0-darwin-arm64.dmg",
		"Zenith-0.1.0-windows-amd64-installer.exe",
		"checksums.txt",
	} {
		if re.MatchString(name) {
			t.Errorf("non-CLI asset %q must be ignored", name)
		}
	}
}

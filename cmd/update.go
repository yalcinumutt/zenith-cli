package cmd

import (
	"fmt"
	"os"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

const updateRepoSlug = "yalcinumutt/zenith-cli"

// updateAssetFilters keeps the updater on CLI archives so a desktop installer
// in the same release can never be selected (ADR 0009).
var updateAssetFilters = []string{`^zenith-cli_`}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update zenith to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		v, err := semver.ParseTolerant(Version)
		if err != nil {
			return fmt.Errorf("invalid version format: %w", err)
		}

		updater, err := selfupdate.NewUpdater(selfupdate.Config{Filters: updateAssetFilters})
		if err != nil {
			return fmt.Errorf("could not configure updater: %w", err)
		}
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not locate executable: %w", err)
		}
		latest, err := updater.UpdateCommand(exe, v, updateRepoSlug)
		if err != nil {
			return fmt.Errorf("binary update failed: %w", err)
		}
		if latest.Version.Equals(v) {
			fmt.Printf("Current version %s is the latest\n", Version)
		} else {
			fmt.Printf("Successfully updated to version %s\n", latest.Version)
			fmt.Println("Release notes:\n", latest.ReleaseNotes)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

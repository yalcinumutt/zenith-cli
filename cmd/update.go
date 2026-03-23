package cmd

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update zenith to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		v, err := semver.ParseTolerant(Version)
		if err != nil {
			return fmt.Errorf("invalid version format: %w", err)
		}

		latest, err := selfupdate.UpdateSelf(v, "yalcinumut/zenith-cli")
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

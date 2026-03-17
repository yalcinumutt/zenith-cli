package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive TUI dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Start(store)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

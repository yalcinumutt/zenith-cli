package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/storage"
)

var (
	store storage.Store
)

var rootCmd = &cobra.Command{
	Use:   "zenith",
	Short: "Zenith is a CLI Productivity Tool",
	Long: `Zenith is a powerful CLI for habit tracking, task management, and project management.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		s, err := storage.NewSQLiteStore()
		if err != nil {
			return err
		}
		store = s
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Root flags if any
}

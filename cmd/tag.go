package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "List tags (attach and detach them with 'task tag' / 'task untag')",
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		tags, err := app.Tags.List()
		if err != nil {
			return err
		}
		if len(tags) == 0 {
			fmt.Println("No tags found.")
			return nil
		}
		fmt.Println("ID  | Color   | Name")
		fmt.Println("----|---------|------")
		for _, t := range tags {
			fmt.Printf("%-3d | %-7s | %s\n", t.ID, t.Color, t.Name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.AddCommand(tagListCmd)
}

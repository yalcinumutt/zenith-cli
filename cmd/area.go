package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var areaCmd = &cobra.Command{
	Use:   "area",
	Short: "Manage areas (long-lived buckets that hold projects and tasks)",
}

var areaAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new area",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		area, err := app.Areas.Add(strings.Join(args, " "))
		if err != nil {
			return err
		}
		fmt.Printf("Area added with ID: %d\n", area.ID)
		return nil
	},
}

var areaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all areas",
	RunE: func(cmd *cobra.Command, args []string) error {
		areas, err := app.Areas.List()
		if err != nil {
			return err
		}
		if len(areas) == 0 {
			fmt.Println("No areas found.")
			return nil
		}
		fmt.Println("ID  | Name")
		fmt.Println("----|------")
		for _, a := range areas {
			fmt.Printf("%-3d | %s\n", a.ID, a.Name)
		}
		return nil
	},
}

var areaRenameCmd = &cobra.Command{
	Use:   "rename [id] [name]",
	Short: "Rename an area",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("area ID", args[0])
		if err != nil {
			return err
		}
		if err := app.Areas.Rename(id, strings.Join(args[1:], " ")); err != nil {
			return err
		}
		fmt.Printf("Area %d renamed.\n", id)
		return nil
	},
}

var areaDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete an area; its projects become standalone and its tasks unfiled",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("area ID", args[0])
		if err != nil {
			return err
		}
		if err := app.Areas.Delete(id); err != nil {
			return err
		}
		fmt.Printf("Area %d deleted.\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(areaCmd)
	areaCmd.AddCommand(areaAddCmd, areaListCmd, areaRenameCmd, areaDeleteCmd)
}

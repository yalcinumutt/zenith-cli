package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var headingCmd = &cobra.Command{
	Use:   "heading",
	Short: "Manage headings (sections inside a project that hold only tasks)",
}

var headingAddCmd = &cobra.Command{
	Use:   "add [projectID] [title]",
	Short: "Add a heading to a project",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := parseID("project ID", args[0])
		if err != nil {
			return err
		}
		h, err := app.Headings.Add(projectID, strings.Join(args[1:], " "))
		if err != nil {
			return err
		}
		fmt.Printf("Heading added with ID: %d\n", h.ID)
		return nil
	},
}

var headingListCmd = &cobra.Command{
	Use:   "list [projectID]",
	Short: "List the headings of a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := parseID("project ID", args[0])
		if err != nil {
			return err
		}
		headings, err := app.Headings.List(projectID)
		if err != nil {
			return err
		}
		if len(headings) == 0 {
			fmt.Println("No headings found.")
			return nil
		}
		fmt.Println("ID  | Title")
		fmt.Println("----|------")
		for _, h := range headings {
			fmt.Printf("%-3d | %s\n", h.ID, h.Title)
		}
		return nil
	},
}

var headingRenameCmd = &cobra.Command{
	Use:   "rename [id] [title]",
	Short: "Rename a heading",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("heading ID", args[0])
		if err != nil {
			return err
		}
		if err := app.Headings.Rename(id, strings.Join(args[1:], " ")); err != nil {
			return err
		}
		fmt.Printf("Heading %d renamed.\n", id)
		return nil
	},
}

var headingMoveCmd = &cobra.Command{
	Use:   "move [id] [projectID]",
	Short: "Move a heading, with its tasks, to another project",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("heading ID", args[0])
		if err != nil {
			return err
		}
		projectID, err := parseID("project ID", args[1])
		if err != nil {
			return err
		}
		if err := app.Headings.Move(id, projectID); err != nil {
			return err
		}
		fmt.Printf("Heading %d moved to project %d.\n", id, projectID)
		return nil
	},
}

var headingDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a heading and the tasks under it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("heading ID", args[0])
		if err != nil {
			return err
		}
		if err := app.Headings.Delete(id); err != nil {
			return err
		}
		fmt.Printf("Heading %d deleted.\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(headingCmd)
	headingCmd.AddCommand(headingAddCmd, headingListCmd, headingRenameCmd, headingMoveCmd, headingDeleteCmd)
}

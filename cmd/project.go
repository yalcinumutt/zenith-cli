package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
}

var projectAreaFlag int64

var projectAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new project",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var areaID *int64
		if projectAreaFlag != 0 {
			areaID = &projectAreaFlag
		}
		project, err := app.Projects.Add(strings.Join(args, " "), areaID)
		if err != nil {
			return err
		}
		fmt.Printf("Project added with ID: %d\n", project.ID)
		return nil
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := app.Projects.List()
		if err != nil {
			return err
		}

		if len(projects) == 0 {
			fmt.Println("No projects found.")
			return nil
		}

		areas, err := app.Areas.List()
		if err != nil {
			return err
		}
		areaName := map[int64]string{}
		for _, a := range areas {
			areaName[a.ID] = a.Name
		}

		fmt.Println("ID  | Status    | Area            | Name")
		fmt.Println("----|-----------|-----------------|------")
		for _, p := range projects {
			area := "-"
			if p.AreaID != nil {
				area = areaName[*p.AreaID]
			}
			fmt.Printf("%-3d | %-9s | %-15s | %s\n", p.ID, p.Status, area, p.Name)
		}
		return nil
	},
}

var projectRenameCmd = &cobra.Command{
	Use:   "rename [id] [name]",
	Short: "Rename a project",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("project ID", args[0])
		if err != nil {
			return err
		}
		if err := app.Projects.Rename(id, strings.Join(args[1:], " ")); err != nil {
			return err
		}
		fmt.Printf("Project %d renamed.\n", id)
		return nil
	},
}

var projectMoveCmd = &cobra.Command{
	Use:   "move [id] [areaID|none]",
	Short: "Move a project into an area, or make it standalone with 'none'",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("project ID", args[0])
		if err != nil {
			return err
		}
		areaID, err := parseOptionalID("area ID", args[1])
		if err != nil {
			return err
		}
		if err := app.Projects.Move(id, areaID); err != nil {
			return err
		}
		fmt.Printf("Project %d moved.\n", id)
		return nil
	},
}

func projectAction(use, short, done string, fn func(id int64) error) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [id]",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID("project ID", args[0])
			if err != nil {
				return err
			}
			if err := fn(id); err != nil {
				return err
			}
			fmt.Printf("Project %d %s.\n", id, done)
			return nil
		},
	}
}

func init() {
	projectAddCmd.Flags().Int64Var(&projectAreaFlag, "area", 0, "Area ID to file the project under")

	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectAddCmd, projectListCmd, projectRenameCmd, projectMoveCmd)
	projectCmd.AddCommand(
		projectAction("done", "Mark a project completed (refused while it has open tasks)", "completed", func(id int64) error { return app.Projects.Complete(id) }),
		projectAction("cancel", "Cancel a project and its open tasks", "cancelled", func(id int64) error { return app.Projects.Cancel(id) }),
		projectAction("reopen", "Reopen a completed or cancelled project", "reopened", func(id int64) error { return app.Projects.Reopen(id) }),
		projectAction("delete", "Delete a project with its headings and tasks", "deleted", func(id int64) error { return app.Projects.Delete(id) }),
	)
}

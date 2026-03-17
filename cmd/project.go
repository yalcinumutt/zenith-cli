package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/models"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
}

var projectAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new project",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.Join(args, " ")
		project := &models.Project{
			Name: name,
		}
		if err := store.AddProject(project); err != nil {
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
		projects, err := store.GetProjects()
		if err != nil {
			return err
		}

		if len(projects) == 0 {
			fmt.Println("No projects found.")
			return nil
		}

		fmt.Println("ID  | Name")
		fmt.Println("----|------")
		for _, p := range projects {
			fmt.Printf("%-3d | %s\n", p.ID, p.Name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectListCmd)
}

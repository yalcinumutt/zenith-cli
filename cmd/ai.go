package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/ai"
	"github.com/yalcinumut/zenith-cli/internal/ui"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI powered productivity helper (Ollama / LM Studio)",
	Long:  `Zenith AI connects to local LLM engines (Ollama or LM Studio) to analyze your tasks and habits, suggest next actions, and summarize your daily achievements.`,
}

var aiSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Get AI task recommendations and schedule analysis",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.HeaderStyle.Render(" Zenith AI Suggestion "))
		fmt.Println("Analyzing tasks and habits... Please wait.")

		tasks, err := store.GetTasks()
		if err != nil {
			return err
		}

		habits, err := store.GetHabits()
		if err != nil {
			return err
		}

		suggestion, err := ai.SuggestTasks(tasks, habits)
		if err != nil {
			return err
		}

		fmt.Println("\n" + suggestion)
		return nil
	},
}

var aiSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Get AI productivity summary and accomplishments review",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.HeaderStyle.Render(" Zenith AI Daily Summary "))
		fmt.Println("Analyzing accomplishments... Please wait.")

		tasks, err := store.GetTasks()
		if err != nil {
			return err
		}

		habits, err := store.GetHabits()
		if err != nil {
			return err
		}

		summaryText, err := ai.GenerateSummary(tasks, habits)
		if err != nil {
			return err
		}

		fmt.Println("\n" + summaryText)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(aiSuggestCmd)
	aiCmd.AddCommand(aiSummaryCmd)
}

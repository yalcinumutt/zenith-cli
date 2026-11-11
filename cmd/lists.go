package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
)

func describeWhen(w models.When, evening bool) string {
	switch w.Kind {
	case models.WhenScheduled:
		s := "scheduled for " + w.Date
		if evening {
			s += " (This Evening)"
		}
		return s
	case models.WhenSomeday:
		return "someday"
	}
	return "anytime"
}

var listCmd = &cobra.Command{
	Use:   "list [inbox|today|upcoming|anytime|someday|logbook]",
	Short: "Show one of the six lists (all counts when no list is given)",
	Args:  cobra.MaximumNArgs(1),
	ValidArgs: []string{
		string(core.ListInbox), string(core.ListToday), string(core.ListUpcoming),
		string(core.ListAnytime), string(core.ListSomeday), string(core.ListLogbook),
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			counts, err := app.Lists.Counts()
			if err != nil {
				return err
			}
			fmt.Printf("Inbox %d | Today %d | Upcoming %d\n", counts.Inbox, counts.Today, counts.Upcoming)
			fmt.Println("Show a list with: zenith list inbox|today|upcoming|anytime|someday|logbook")
			return nil
		}
		kind := core.ListKind(strings.ToLower(args[0]))
		valid := false
		for _, k := range core.ListKinds {
			valid = valid || k == kind
		}
		if !valid {
			return fmt.Errorf("unknown list %q (use inbox, today, upcoming, anytime, someday or logbook)", args[0])
		}

		if kind == core.ListToday {
			today, err := app.Lists.Today()
			if err != nil {
				return err
			}
			if len(today.Day)+len(today.Evening) == 0 {
				fmt.Println("Nothing in Today.")
				return nil
			}
			printListHeader()
			for _, t := range today.Day {
				printListRow(t)
			}
			if len(today.Evening) > 0 {
				fmt.Println("--- This Evening ---")
				for _, t := range today.Evening {
					printListRow(t)
				}
			}
			return nil
		}

		tasks, err := app.Lists.Get(kind)
		if err != nil {
			return err
		}
		if len(tasks) == 0 {
			fmt.Printf("Nothing in %s.\n", strings.Title(string(kind))) //nolint:staticcheck
			return nil
		}
		printListHeader()
		for _, t := range tasks {
			printListRow(t)
		}
		return nil
	},
}

func printListHeader() {
	fmt.Printf("%-3s | %-24s | %-12s | %-10s\n", "ID", "Title", "When", "Deadline")
	fmt.Println("----|--------------------------|--------------|-----------")
}

func printListRow(t models.Task) {
	when := string(t.When.Kind)
	if t.When.Kind == models.WhenScheduled {
		when = t.When.Date
	}
	mark := ""
	if t.Deadline != "" && t.Deadline < time.Now().Format(models.DateLayout) && t.Status == models.StatusOpen {
		mark = " !"
	}
	fmt.Printf("%-3d | %-24s | %-12s | %-10s%s\n", t.ID, t.Title, when, t.Deadline, mark)
}

func init() {
	rootCmd.AddCommand(listCmd)
}

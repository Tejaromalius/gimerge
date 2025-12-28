package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gimerge/git"
	"gimerge/tui"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cleanup := flag.Bool("delete", false, "Cleanup mode: list and delete merged tags")
	flag.BoolVar(cleanup, "d", false, "Cleanup mode (shorthand)")
	flag.Parse()

	if _, err := git.GetCurrentBranch(); err != nil {
		fmt.Printf("Error (not in a git repo?): %v\n", err)
		os.Exit(1)
	}

	var items []list.Item
	var title string
	var multi bool
	var err error

	if *cleanup {
		tags, err := git.GetMergedTags()
		if err == nil {
			for _, tg := range tags {
				t := time.Unix(tg.TS, 0)
				dateStr := t.Format("2006-01-02 15:04:05")
				desc := fmt.Sprintf("%s: %s -> %s", dateStr, tg.Source, tg.Target)
				items = append(items, &tui.Item{
					Title:       tg.Name,
					Description: desc,
					IsTag:       true,
				})
			}
		}
		title = "Select tags to DELETE (Space to toggle, Enter to confirm)"
		multi = true
	} else {
		branches, err := git.GetBranches()
		if err == nil {
			for _, b := range branches {
				items = append(items, &tui.Item{Title: b})
			}
		}
		title = "Select branch to tag merge"
		multi = false
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(items) == 0 {
		fmt.Println("No items found.")
		os.Exit(0)
	}

	const defaultWidth = 20
	const listHeight = 14

	l := list.New(items, tui.ItemDelegate{}, defaultWidth, listHeight)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = tui.TitleStyle
	l.Styles.PaginationStyle = tui.PaginationStyle
	l.Styles.HelpStyle = tui.HelpStyle

	m := tui.Model{List: l, MultiSelect: multi}

	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
	m, _ = finalModel.(tui.Model)

	if len(m.Selections) > 0 {
		if *cleanup {
			if err := git.CleanupTagsAndBranches(m.Selections); err != nil {
				fmt.Printf("\nError: %v\n", err)
				os.Exit(1)
			}
		} else {
			tagName, err := git.TagBranch(m.Selections[0])
			if err != nil {
				fmt.Printf("\nError creating tag: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("\nSuccessfully created tag: %s\n", tagName)
		}
	}
}

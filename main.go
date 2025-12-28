package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item struct {
	title       string
	description string
	selected    bool
	isTag       bool
}

func (i item) FilterValue() string { return i.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(*item)
	if !ok {
		return
	}

	str := i.title
	if i.isTag && i.description != "" {
		str = i.description
	}

	if i.selected {
		str = "[x] " + str
	} else {
		str = "[ ] " + str
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list        list.Model
	selections  []string
	quitting    bool
	multiSelect bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case " ":
			if m.multiSelect {
				if i, ok := m.list.SelectedItem().(*item); ok {
					i.selected = !i.selected
				}
				return m, nil
			}
		case "enter":
			if m.multiSelect {
				for _, itm := range m.list.Items() {
					if i, ok := itm.(*item); ok && i.selected {
						m.selections = append(m.selections, i.title)
					}
				}
				// If nothing was checked, treat current selection as intent
				if len(m.selections) == 0 {
					if i, ok := m.list.SelectedItem().(*item); ok {
						m.selections = append(m.selections, i.title)
					}
				}
			} else {
				if i, ok := m.list.SelectedItem().(*item); ok {
					m.selections = []string{i.title}
				}
			}
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

func getBranches() ([]list.Item, error) {
	cmd := exec.Command("git", "for-each-ref", "--sort=-committerdate", "--format=%(refname:short)", "refs/heads", "refs/remotes")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	items := make([]list.Item, 0, len(lines))
	for _, l := range lines {
		if l != "" {
			items = append(items, &item{title: l})
		}
	}
	return items, nil
}

func getMergedTags() ([]list.Item, error) {
	cmd := exec.Command("git", "tag", "-l", "merged*")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	// Match pattern: merged.{source}->{target}@timestamp
	re := regexp.MustCompile(`^merged\.?\{(.+)\}->\{(.+)\}@(\d+)$`)

	type tagged struct {
		name   string
		source string
		target string
		ts     int64
	}
	var detected []tagged

	for _, l := range lines {
		if l == "" {
			continue
		}
		matches := re.FindStringSubmatch(l)
		if len(matches) < 4 {
			continue
		}

		ts, _ := strconv.ParseInt(matches[3], 10, 64)
		detected = append(detected, tagged{
			name:   l,
			source: matches[1],
			target: matches[2],
			ts:     ts,
		})
	}

	sort.Slice(detected, func(i, j int) bool {
		return detected[i].ts > detected[j].ts // Descending
	})

	items := make([]list.Item, 0, len(detected))
	for _, d := range detected {
		t := time.Unix(d.ts, 0)
		dateStr := t.Format("2006-01-02 15:04:05")
		desc := fmt.Sprintf("%s: %s -> %s", dateStr, d.source, d.target)
		items = append(items, &item{
			title:       d.name,
			description: desc,
			isTag:       true,
		})
	}
	return items, nil
}

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func tagBranch(selectedBranch string) (string, error) {
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	timestamp := time.Now().Unix()
	tagName := fmt.Sprintf("merged.{%s}->{%s}@%d", currentBranch, selectedBranch, timestamp)

	cmd := exec.Command("git", "tag", tagName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git tag failed: %s: %s", err, string(output))
	}

	return tagName, nil
}

func cleanupTagsAndBranches(tags []string) error {
	re := regexp.MustCompile(`^merged\.?\{(.+)\}->\{.*\}@\d+$`)
	currentBranch, _ := getCurrentBranch()

	for _, tag := range tags {
		matches := re.FindStringSubmatch(tag)
		var source string
		if len(matches) >= 2 {
			source = matches[1]
		}

		// Check if source branch is current branch
		if source != "" && source == currentBranch {
			fmt.Printf("Warning: Skipped cleanup for tag '%s' because branch '%s' is currently checked out.\n", tag, source)
			continue
		}

		// 1. Delete the Tag
		cmdTag := exec.Command("git", "tag", "-d", tag)
		if output, err := cmdTag.CombinedOutput(); err != nil {
			fmt.Printf("Warning: failed to delete tag %s: %s\n", tag, string(output))
		} else {
			fmt.Printf("Deleted tag: %s\n", tag)
		}

		// 2. Try to Delete the Source Branch
		if source != "" {
			// Try safe delete (-d)
			cmdBranch := exec.Command("git", "branch", "-d", source)
			if _, err := cmdBranch.CombinedOutput(); err != nil {
				fmt.Printf("Note: branch '%s' safe-delete failed (maybe not merged into current HEAD). Skipping.\n", source)
			} else {
				fmt.Printf("Deleted branch: %s\n", source)
			}
		}
	}
	return nil
}

func main() {
	cleanup := flag.Bool("delete", false, "Cleanup mode: list and delete merged tags")
	flag.BoolVar(cleanup, "d", false, "Cleanup mode (shorthand)")
	flag.Parse()

	if _, err := getCurrentBranch(); err != nil {
		fmt.Printf("Error (not in a git repo?): %v\n", err)
		os.Exit(1)
	}

	var items []list.Item
	var title string
	var multi bool
	var err error

	if *cleanup {
		items, err = getMergedTags()
		title = "Select tags to DELETE (Space to toggle, Enter to confirm)"
		multi = true
	} else {
		items, err = getBranches()
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

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l, multiSelect: multi}

	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
	m, _ = finalModel.(model)

	if len(m.selections) > 0 {
		if *cleanup {
			if err := cleanupTagsAndBranches(m.selections); err != nil {
				fmt.Printf("\nError: %v\n", err)
				os.Exit(1)
			}
		} else {
			tagName, err := tagBranch(m.selections[0])
			if err != nil {
				fmt.Printf("\nError creating tag: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("\nSuccessfully created tag: %s\n", tagName)
		}
	}
}

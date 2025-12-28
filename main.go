package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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

type item string

func (i item) FilterValue() string { return string(i) }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := string(i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list     list.Model
	choice   string
	quitting bool
	err      error
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
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
	if m.choice != "" {
		return ""
	}
	if m.quitting {
		return quitTextStyle.Render("Cancelled.")
	}
	return "\n" + m.list.View()
}

func getBranches() ([]list.Item, error) {
	// refs/heads for local, refs/remotes for remote
	// sorting by committerdate (most recent last, so we use -committerdate)
	cmd := exec.Command("git", "for-each-ref", "--sort=-committerdate", "--format=%(refname:short)", "refs/heads", "refs/remotes")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	items := make([]list.Item, 0, len(lines))
	for _, l := range lines {
		if l != "" {
			items = append(items, item(l))
		}
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

	// Validate tag format? Git will complain if it's invalid.
	// But let's just try running it.

	// Tag the CURRENT HEAD with this name?
	// The prompt says "tag the current branch ... using this format ... where selected_branch_name is selected".
	// Usually tagging implies tagging the current commit (HEAD).
	// So we are creating a tag ON HEAD, but using a name derived from another branch.

	cmd := exec.Command("git", "tag", tagName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git tag failed: %s: %s", err, string(output))
	}

	return tagName, nil
}

func main() {
	branches, err := getBranches()
	if err != nil {
		fmt.Printf("Error getting branches: %v\nIs this a git repository?\n", err)
		os.Exit(1)
	}

	if len(branches) == 0 {
		fmt.Println("No branches found.")
		os.Exit(0)
	}

	const defaultWidth = 20
	const listHeight = 14

	l := list.New(branches, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Select branch to tag merge"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
	m, _ = finalModel.(model)

	if m.choice != "" {
		tagName, err := tagBranch(m.choice)
		if err != nil {
			fmt.Printf("\nError creating tag: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nSuccessfully created tag: %s\n", tagName)
	}
}

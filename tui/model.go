package tui

import (


	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	TitleStyle        = lipgloss.NewStyle().MarginLeft(2)
	ItemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	SelectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	PaginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	HelpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type Item struct {
	Title       string
	Description string
	Selected    bool
	IsTag       bool
}

func (i Item) FilterValue() string { return i.Title }

type Model struct {
	List        list.Model
	Selections  []string
	Quitting    bool
	MultiSelect bool
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		case " ":
			if m.MultiSelect {
				if i, ok := m.List.SelectedItem().(*Item); ok {
					i.Selected = !i.Selected
				}
				return m, nil
			}
		case "enter":
			if m.MultiSelect {
				for _, itm := range m.List.Items() {
					if i, ok := itm.(*Item); ok && i.Selected {
						m.Selections = append(m.Selections, i.Title)
					}
				}
				if len(m.Selections) == 0 {
					if i, ok := m.List.SelectedItem().(*Item); ok {
						m.Selections = append(m.Selections, i.Title)
					}
				}
			} else {
				if i, ok := m.List.SelectedItem().(*Item); ok {
					m.Selections = []string{i.Title}
				}
			}
			m.Quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.List.SetWidth(msg.Width)
		return m, nil
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.Quitting {
		return ""
	}
	return "\n" + m.List.View()
}

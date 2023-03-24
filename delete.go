package main

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	yes = iota
	no
)

var (
	buttonStyle         = lipgloss.NewStyle().Background(lipgloss.Color("241")).Padding(0, 3)
	selectedButtonStyle = buttonStyle.Copy().Background(googleBlue).Underline(true)
)

type DeleteDialog struct {
	calendarId string
	eventId    string
	selection  int
	height     int
	width      int
	success    bool
	pending    bool
	spinner    spinner.Model
	err        error
	help       help.Model
	keys       keyMapDelete
}

func newDeleteDialog(calendarId, eventId string, width, height int) DeleteDialog {
	s := spinner.New()
	s.Spinner = spinner.Points
	return DeleteDialog{
		calendarId: calendarId,
		eventId:    eventId,
		selection:  no,
		height:     height,
		width:      width,
		success:    false,
		pending:    false,
		spinner:    s,
		err:        nil,
		help:       help.New(),
		keys:       deleteKeyMap,
	}
}

func (m DeleteDialog) Init() tea.Cmd {
	return nil
}

func (m DeleteDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		return m, nil
	case errMsg:
		m.err = msg.err
		return m, nil
	case successMsg:
		m.success = true
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if m.success || m.err != nil {
			return m, showCalendarViewCmd
		}
		switch {
		case key.Matches(msg, m.keys.Toggle):
			m.toggleSelection()
		case key.Matches(msg, m.keys.Yes):
			m.selection = yes
		case key.Matches(msg, m.keys.No):
			m.selection = no
		case key.Matches(msg, m.keys.Confirm):
			if m.selection == yes {
				m.pending = true
				var cmds []tea.Cmd
				cmds = append(cmds, deleteEventRequestCmd(m.calendarId, m.eventId))
				cmds = append(cmds, m.spinner.Tick)
				return m, tea.Batch(cmds...)
			} else {
				return m, showCalendarViewCmd
			}
		case key.Matches(msg, m.keys.Cancel):
			return m, showCalendarViewCmd
		}
	}
	return m, nil
}

func (m *DeleteDialog) toggleSelection() {
	if m.selection == yes {
		m.selection = no
	} else {
		m.selection = yes
	}
}

func (m DeleteDialog) View() string {
	var content string
	if m.err != nil {
		content = "Error deleting event. Press any key to return to calendar."
	} else if m.success {
		content = "Successfully deleted event. Press any key to return to calendar."
	} else if m.pending {
		content = m.spinner.View()
	} else {
		content = renderDeleteContent(m)
	}
	helpView := lipgloss.NewStyle().
		Width(m.width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(m.help.View(m.keys))
	container := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-lipgloss.Height(helpView)-3).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialogStyle.Render(content))
	return lipgloss.JoinVertical(lipgloss.Center, container, helpView)
}

func renderDeleteContent(m DeleteDialog) string {
	var (
		yesStyle lipgloss.Style
		noStyle  lipgloss.Style
	)
	if m.selection == yes {
		yesStyle = selectedButtonStyle
		noStyle = buttonStyle
	} else {
		yesStyle = buttonStyle
		noStyle = selectedButtonStyle
	}
	yesButton := yesStyle.Render("Yes")
	noButton := noStyle.Render("No")
	return lipgloss.JoinVertical(
		lipgloss.Center,
		"Delete Event?\n",
		lipgloss.JoinHorizontal(lipgloss.Top, noButton, " ", yesButton),
	)
}

type keyMapDelete struct {
	Toggle  key.Binding
	Yes     key.Binding
	No      key.Binding
	Confirm key.Binding
	Cancel  key.Binding
	Quit    key.Binding
}

var deleteKeyMap = keyMapDelete{
	Toggle: key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "toggle"),
	),
	Yes: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "yes"),
	),
	No: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "no"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

func (k keyMapDelete) ShortHelp() []key.Binding {
	return []key.Binding{k.Toggle, k.Yes, k.No, k.Confirm, k.Cancel}
}

func (k keyMapDelete) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Toggle, k.Confirm},
		{k.Yes, k.Cancel},
		{k.No},
	}
}

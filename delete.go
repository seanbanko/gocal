package main

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type selection int

const (
	yes selection = iota
	no
)

var (
	buttonStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("241")).
			Padding(0, 3)
	selectedButtonStyle = buttonStyle.Copy().
				Background(lipgloss.Color("62")).
				Underline(true)
)

type DeleteDialog struct {
	calendarId string
	eventId    string
	selection  selection
	height     int
	width      int
	success    bool
	err        error
	help       help.Model
	keys       keyMapDelete
}

func newDeleteDialog(calendarId, eventId string, width, height int) DeleteDialog {
	return DeleteDialog{
		calendarId: calendarId,
		eventId:    eventId,
		selection:  no,
		height:     height,
		width:      width,
		success:    false,
		err:        nil,
		help:       help.New(),
		keys:       DeleteKeyMap,
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
	case deleteEventResponseMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.success = true
		}
		return m, nil
	case tea.KeyMsg:
		// Prevents further updates after creating one event
		if m.success {
			return m, exitCreateDialogCmd
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, exitDeleteDialogCmd
		case "enter":
			if m.selection == yes {
				return m, deleteEventRequestCmd(m.calendarId, m.eventId)
			} else {
				return m, exitDeleteDialogCmd
			}
		case "tab", "shift+tab":
			m.toggleSelection()
		case "y":
			m.selection = yes
		case "n":
			m.selection = no
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
		Height(m.height - lipgloss.Height(helpView)).
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
		lipgloss.JoinHorizontal(lipgloss.Top, yesButton, " ", noButton),
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

var DeleteKeyMap = keyMapDelete{
	Toggle: key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "toggle"),
	),
	Yes: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("yes", "yes"),
	),
	No: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("no", "no"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
}

func (k keyMapDelete) ShortHelp() []key.Binding {
	return []key.Binding{k.Toggle, k.Yes, k.No, k.Confirm, k.Cancel, k.Quit}
}

func (k keyMapDelete) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Toggle, k.Confirm},
		{k.Yes, k.Cancel},
		{k.No, k.Quit},
	}
}

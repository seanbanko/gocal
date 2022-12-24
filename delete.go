package main

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type selectionValue int

const (
	yes selectionValue = iota
	no
)

var (
	deletePopupStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				AlignHorizontal(lipgloss.Center).
				AlignVertical(lipgloss.Center)
	rosePineDawnRoseHex    = "#D7827E"
	rosePineDawnOverlayHex = "#F2E9E1"
	rosePineDawnMutedHex   = "#9893A5"
	buttonStyle            = lipgloss.NewStyle().
				Foreground(lipgloss.Color(rosePineDawnOverlayHex)).
				Background(lipgloss.Color(rosePineDawnMutedHex)).
				Padding(0, 3)
	selectedButtonStyle = buttonStyle.Copy().
				Background(lipgloss.Color(rosePineDawnRoseHex)).
				Underline(true)
)

type DeleteEventPopup struct {
	selection  selectionValue
	calendarId string
	eventId    string
	height     int
	width      int
	success    bool
	err        error
	help       help.Model
	keys       keyMapDelete
}

func newDeletePopup(calendarId, eventId string, width, height int) DeleteEventPopup {
	return DeleteEventPopup{
		selection:  no,
		calendarId: calendarId,
		eventId:    eventId,
		height:     height,
		width:      width,
		success:    false,
		err:        nil,
		help:       help.New(),
		keys:       DeleteKeyMap,
	}
}

func (m DeleteEventPopup) Init() tea.Cmd {
	return nil
}

func (m DeleteEventPopup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, exitDeletePopupCmd
		case "enter":
			if m.selection == yes {
				return m, deleteEventRequestCmd(m.calendarId, m.eventId)
			} else {
				return m, exitDeletePopupCmd
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

func (m *DeleteEventPopup) toggleSelection() {
	if m.selection == yes {
		m.selection = no
	} else {
		m.selection = yes
	}
}

func (m DeleteEventPopup) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	deletePopupStyle = deletePopupStyle.Width(m.width / 3)
	deletePopupStyle = deletePopupStyle.Height(m.height / 3)
	var content string
	if m.err != nil {
		content = "Error deleting event. Press esc to return to calendar."
	} else if m.success {
		content = "Successfully deleted event. Press esc to return to calendar."
	} else {
		content = renderDeleteContent(m)
	}
	help := renderHelpDelete(m.help, m.keys, m.width)
	deletePopupContainer := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - lipgloss.Height(help)).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(deletePopupStyle.Render(content))
	return lipgloss.JoinVertical(lipgloss.Center, deletePopupContainer, help)
}

func renderDeleteContent(m DeleteEventPopup) string {
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
	yesButton := yesStyle.Render("yes")
	noButton := noStyle.MarginLeft(2).Render("no")
	return lipgloss.JoinVertical(
		lipgloss.Center,
		"Delete Event?",
		"\n",
		lipgloss.JoinHorizontal(lipgloss.Top, yesButton, noButton),
	)
}

func renderHelpDelete(help help.Model, keys keyMapDelete, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(help.View(keys))
}

// Help

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
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
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

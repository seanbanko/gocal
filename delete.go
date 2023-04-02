package main

import (
	"gocal/common"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/api/calendar/v3"
)

// -----------------------------------------------------------------------------
// Model
// -----------------------------------------------------------------------------

const (
	yes = iota
	no
)

var (
	buttonStyle         = lipgloss.NewStyle().Background(common.Grey).Padding(0, 3)
	selectedButtonStyle = buttonStyle.Copy().Background(common.GoogleBlue).Underline(true)
)

type DeleteDialog struct {
	srv           *calendar.Service
	calendarId    string
	eventId       string
	selection     int
	success       bool
	pending       bool
	spinner       spinner.Model
	err           error
	keys          keyMapDelete
	help          help.Model
	width, height int
}

func newDeleteDialog(srv *calendar.Service, calendarId, eventId string, width, height int) DeleteDialog {
	s := spinner.New()
	s.Spinner = spinner.Points
	return DeleteDialog{
		srv:        srv,
		calendarId: calendarId,
		eventId:    eventId,
		selection:  no,
		success:    false,
		pending:    false,
		spinner:    s,
		err:        nil,
		keys:       deleteKeyMap,
		help:       help.New(),
		width:      width,
		height:     height,
	}
}

// -----------------------------------------------------------------------------
// Init
// -----------------------------------------------------------------------------

func (m DeleteDialog) Init() tea.Cmd {
	return nil
}

// -----------------------------------------------------------------------------
// Update
// -----------------------------------------------------------------------------

func (m DeleteDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case deleteEventSuccessMsg:
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
			if m.selection == no {
				m.selection = yes
			} else {
				m.selection = no
			}
			return m, nil

		case key.Matches(msg, m.keys.Yes):
			m.selection = yes
			return m, nil

		case key.Matches(msg, m.keys.No):
			m.selection = no
			return m, nil

		case key.Matches(msg, m.keys.Confirm):
			if m.selection == yes {
				m.pending = true
				return m, tea.Batch(
					deleteEvent(m.srv, m.calendarId, m.eventId),
					m.spinner.Tick,
				)
			} else {
				return m, showCalendarViewCmd
			}

		case key.Matches(msg, m.keys.Exit):
			return m, showCalendarViewCmd
		}
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// View
// -----------------------------------------------------------------------------

func (m DeleteDialog) View() string {
	var s string
	if m.err != nil {
		s = "Error. Press any key to return to calendar."
	} else if m.success {
		s = "Success. Press any key to return to calendar."
	} else if m.pending {
		s = m.spinner.View()
	} else {
		s = m.viewDialog()
	}
	help := lipgloss.NewStyle().Width(m.width).Padding(1).AlignHorizontal(lipgloss.Center).Render(m.help.View(m.keys))
	body := lipgloss.Place(m.width, m.height-lipgloss.Height(help), lipgloss.Center, lipgloss.Center, s)
	return lipgloss.JoinVertical(lipgloss.Center, body, help)
}

func (m DeleteDialog) viewDialog() string {
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
	s := lipgloss.JoinVertical(
		lipgloss.Center,
		"Delete Event?\n",
		lipgloss.JoinHorizontal(lipgloss.Top, noButton, "  ", yesButton),
	)
	return lipgloss.NewStyle().Padding(1).Border(lipgloss.RoundedBorder()).Align(lipgloss.Center, lipgloss.Center).Render(s)
}

// -----------------------------------------------------------------------------
// Messages and Commands
// -----------------------------------------------------------------------------

type deleteEventSuccessMsg struct{}

func deleteEvent(srv *calendar.Service, calendarId, eventId string) tea.Cmd {
	return func() tea.Msg {
		err := srv.Events.Delete(calendarId, eventId).Do()
		if err != nil {
			return errMsg{err: err}
		}
		return deleteEventSuccessMsg{}
	}
}

// -----------------------------------------------------------------------------
// Keys
// -----------------------------------------------------------------------------

type keyMapDelete struct {
	Toggle  key.Binding
	Yes     key.Binding
	No      key.Binding
	Confirm key.Binding
	Exit    key.Binding
}

var deleteKeyMap = keyMapDelete{
	Yes: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "yes"),
	),
	No: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "no"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "toggle"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Exit: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "exit"),
	),
}

func (k keyMapDelete) ShortHelp() []key.Binding {
	return []key.Binding{k.Yes, k.No, k.Toggle, k.Confirm, k.Exit}
}

func (k keyMapDelete) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Yes, k.Confirm},
		{k.No, k.Exit},
		{k.Confirm},
	}
}

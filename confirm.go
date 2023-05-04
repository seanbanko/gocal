package main

import (
	"gocal/common"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type ConfirmDialog struct {
	prompt        string
	yesCmd        tea.Cmd
	noCmd         tea.Cmd
	selection     int
	success       bool
	pending       bool
	spinner       spinner.Model
	err           error
	keys          keyMapConfirm
	help          help.Model
	width, height int
}

func newConfirmDialog(prompt string, yesCmd, noCmd tea.Cmd, width, height int) ConfirmDialog {
	s := spinner.New()
	s.Spinner = spinner.Points
	return ConfirmDialog{
		prompt:    prompt,
		yesCmd:    yesCmd,
		noCmd:     noCmd,
		selection: no,
		success:   false,
		pending:   false,
		spinner:   s,
		err:       nil,
		keys:      confirmKeyMap,
		help:      help.New(),
		width:     width,
		height:    height,
	}
}

// -----------------------------------------------------------------------------
// Init
// -----------------------------------------------------------------------------

func (m ConfirmDialog) Init() tea.Cmd {
	return nil
}

// -----------------------------------------------------------------------------
// Update
// -----------------------------------------------------------------------------

func (m ConfirmDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m, m.noCmd
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
				return m, tea.Batch(m.yesCmd, m.spinner.Tick)
			} else {
				return m, m.noCmd
			}

		case key.Matches(msg, m.keys.Exit):
			return m, m.noCmd
		}
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// View
// -----------------------------------------------------------------------------

func (m ConfirmDialog) View() string {
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

func (m ConfirmDialog) viewDialog() string {
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
		m.prompt+"\n",
		lipgloss.JoinHorizontal(lipgloss.Top, noButton, "  ", yesButton),
	)
	return lipgloss.NewStyle().Padding(1).Border(lipgloss.RoundedBorder()).Align(lipgloss.Center, lipgloss.Center).Render(s)
}

// -----------------------------------------------------------------------------
// Keys
// -----------------------------------------------------------------------------

type keyMapConfirm struct {
	Toggle  key.Binding
	Yes     key.Binding
	No      key.Binding
	Confirm key.Binding
	Exit    key.Binding
}

var confirmKeyMap = keyMapConfirm{
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

func (k keyMapConfirm) ShortHelp() []key.Binding {
	return []key.Binding{k.Yes, k.No, k.Toggle, k.Confirm, k.Exit}
}

func (k keyMapConfirm) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Yes, k.Confirm},
		{k.No, k.Exit},
		{k.Confirm},
	}
}

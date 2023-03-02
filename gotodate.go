package main

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var gotoDatePopupStyle = lipgloss.NewStyle().
    Padding(1).
    AlignHorizontal(lipgloss.Center).
    AlignVertical(lipgloss.Center).
    Border(lipgloss.RoundedBorder())

type GotoDatePopup struct {
	input   textinput.Model
	height  int
	width   int
	help    help.Model
	keys    keyMapGotoDate
}

func newGotoDatePopup(width, height int) GotoDatePopup {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	input := textinput.New()
	input.Placeholder = today.Format(MMDDYYYY)
	input.CharLimit = 10
	input.Prompt = ""
	input.PlaceholderStyle = textInputPlaceholderStyle
	input.Focus()

	return GotoDatePopup{
		input:   input,
		height:  height,
		width:   width,
		help:    help.New(),
		keys:    GotoDateKeymap,
	}
}

func (m GotoDatePopup) Init() tea.Cmd {
	return textinput.Blink
}

func (m GotoDatePopup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, exitGotoDatePopupCmd
		case "enter", "ctrl+s":
			date := m.input.Value()
            // must be sequential to ensure gotoDateResponseMsg reaches calendar
			return m, tea.Sequence(exitGotoDatePopupCmd, gotoDateRequestCmd(date))
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m GotoDatePopup) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
    content := lipgloss.JoinHorizontal(
        lipgloss.Top,
        "Go to Date: ",
        dateStyle.Render(m.input.View()),
    )
	help := renderHelpGotoDate(m.help, m.keys, m.width)
	popupContainer := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - lipgloss.Height(help)).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(gotoDatePopupStyle.Render(content))
	return lipgloss.JoinVertical(lipgloss.Center, popupContainer, help)
}

func renderHelpGotoDate(help help.Model, keys keyMapGotoDate, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(help.View(keys))
}

// Help

type keyMapGotoDate struct {
	Go     key.Binding
	Cancel key.Binding
	Quit   key.Binding
}

var GotoDateKeymap = keyMapGotoDate{
	Go: key.NewBinding(
		key.WithKeys("enter", "ctrl+s"),
		key.WithHelp("enter/ctrl+s", "go"),
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

func (k keyMapGotoDate) ShortHelp() []key.Binding {
	return []key.Binding{k.Go, k.Cancel, k.Quit}
}

func (k keyMapGotoDate) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Go},
		{k.Cancel},
		{k.Quit},
	}
}

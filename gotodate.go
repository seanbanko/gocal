package main

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GotoDialog struct {
	input  textinput.Model
	height int
	width  int
	help   help.Model
	keys   keyMapGoto
}

func newGotoDialog(width, height int) GotoDialog {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	input := textinput.New()
	input.Placeholder = today.Format(AbbreviatedTextDate)
	input.CharLimit = 11
	input.Prompt = ""
	input.PlaceholderStyle = textInputPlaceholderStyle
	input.Focus()

	return GotoDialog{
		input:  input,
		height: height,
		width:  width,
		help:   help.New(),
		keys:   GotoKeymap,
	}
}

func (m GotoDialog) Init() tea.Cmd {
	return textinput.Blink
}

func (m GotoDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, exitGotoDialogCmd
		case "enter", "ctrl+s":
			date := m.input.Value()
			return m, tea.Sequence(exitGotoDialogCmd, gotoDateRequestCmd(date))
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m GotoDialog) View() string {
	content := "Go to Date: " + dateStyle.Render(m.input.View())
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

type keyMapGoto struct {
	Go     key.Binding
	Cancel key.Binding
	Quit   key.Binding
}

var GotoKeymap = keyMapGoto{
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

func (k keyMapGoto) ShortHelp() []key.Binding {
	return []key.Binding{k.Go, k.Cancel, k.Quit}
}

func (k keyMapGoto) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Go},
		{k.Cancel},
		{k.Quit},
	}
}

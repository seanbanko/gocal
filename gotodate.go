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

func newGotoDialog(today time.Time, width, height int) GotoDialog {
	input := newDateTextInput()
	input.Placeholder = today.Format(AbbreviatedTextDate)
	input.Focus()
	return GotoDialog{
		input:  input,
		height: height,
		width:  width,
		help:   help.New(),
		keys:   gotoKeymap,
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
		switch {
		case key.Matches(msg, m.keys.Go):
			text := m.input.Value()
			date, err := time.ParseInLocation(AbbreviatedTextDate, text, time.Local)
			if err != nil {
				return m, showCalendarViewCmd
			}
			return m, tea.Batch(showCalendarViewCmd, gotoDateCmd(date))
		case key.Matches(msg, m.keys.Cancel):
			return m, showCalendarViewCmd
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m GotoDialog) View() string {
	content := "Go to Date: " + textInputDateStyle.Render(m.input.View())
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

type keyMapGoto struct {
	Go     key.Binding
	Cancel key.Binding
}

var gotoKeymap = keyMapGoto{
	Go: key.NewBinding(
		key.WithKeys("enter", "ctrl+s"),
		key.WithHelp("enter", "go"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

func (k keyMapGoto) ShortHelp() []key.Binding {
	return []key.Binding{k.Go, k.Cancel}
}

func (k keyMapGoto) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Go}, {k.Cancel}}
}

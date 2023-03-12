package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	month = iota
	day
	year
)

type GotoDialog struct {
	inputs     []textinput.Model
	focusIndex int
	height     int
	width      int
	help       help.Model
	keys       keyMapGoto
}

func newGotoDialog(focusedDate time.Time, width, height int) GotoDialog {
	inputs := make([]textinput.Model, 3)
	inputs[month] = newMonthTextInput()
	inputs[month].Placeholder = focusedDate.Month().String()[:3]
	inputs[day] = newDayTextInput()
	inputs[day].Placeholder = fmt.Sprintf("%02d", focusedDate.Day())
	inputs[year] = newYearTextInput()
	inputs[year].Placeholder = fmt.Sprintf("%d", focusedDate.Year())
	focusIndex := month
	refocus(inputs, focusIndex)
	return GotoDialog{
		inputs:     inputs,
		focusIndex: focusIndex,
		height:     height,
		width:      width,
		help:       help.New(),
		keys:       gotoKeymap,
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
		case key.Matches(msg, m.keys.Next):
			m.focusIndex = focusNext(m.inputs, m.focusIndex)
		case key.Matches(msg, m.keys.Prev):
			m.focusIndex = focusPrev(m.inputs, m.focusIndex)
		case key.Matches(msg, m.keys.Go):
			autofillPlaceholders(m.inputs)
			text := fmt.Sprintf(
				"%s %s %s",
				m.inputs[month].Value(),
				m.inputs[day].Value(),
				m.inputs[year].Value(),
			)
			date, err := time.ParseInLocation(AbbreviatedTextDate, text, time.Local)
			if err != nil {
				return m, showCalendarViewCmd
			}
			return m, tea.Batch(showCalendarViewCmd, gotoDateCmd(date))
		case key.Matches(msg, m.keys.Cancel):
			return m, showCalendarViewCmd
		}
	}
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m GotoDialog) View() string {
	content := lipgloss.JoinHorizontal(
		lipgloss.Center,
		"Go to Date: ",
		textInputMonthStyle.Render(m.inputs[month].View()),
		" ",
		textInputDayStyle.Render(m.inputs[day].View()),
		" ",
		textInputYearStyle.Render(m.inputs[year].View()),
	)
	helpView := lipgloss.NewStyle().
		Width(m.width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(m.help.View(m.keys))
	container := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-lipgloss.Height(helpView)-3).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
	return lipgloss.JoinVertical(lipgloss.Center, container, helpView)
}

type keyMapGoto struct {
	Next   key.Binding
	Prev   key.Binding
	Go     key.Binding
	Cancel key.Binding
}

var gotoKeymap = keyMapGoto{
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous field"),
	),
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

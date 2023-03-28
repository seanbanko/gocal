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
	inputs        []textinput.Model
	focusIndex    int
	keys          keyMapGoto
	help          help.Model
	width, height int
}

func newGotoDialog(focusedDate time.Time, width, height int) GotoDialog {
	inputs := make([]textinput.Model, 3)
	inputs[month] = newTextInput(monthWidth)
	inputs[day] = newTextInput(dayWidth)
	inputs[year] = newTextInput(yearWidth)
	monthText, dayText, yearText := toDateFields(focusedDate)
	inputs[month].Placeholder = monthText
	inputs[day].Placeholder = dayText
	inputs[year].Placeholder = yearText
	inputs[month].SetValue(monthText)
	inputs[day].SetValue(dayText)
	inputs[year].SetValue(yearText)
	focusIndex := month
	refocus(inputs, focusIndex)
	return GotoDialog{
		inputs:     inputs,
		focusIndex: focusIndex,
		keys:       gotoKeymap,
		help:       help.New(),
		width:      width,
		height:     height,
	}
}

func (m GotoDialog) Init() tea.Cmd {
	return textinput.Blink
}

func (m GotoDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Next):
			m.autoformatInputs()
			m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
			refocus(m.inputs, m.focusIndex)
			return m, nil

		case key.Matches(msg, m.keys.Prev):
			m.autoformatInputs()
			m.focusIndex = m.focusIndex - 1
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			refocus(m.inputs, m.focusIndex)
			return m, nil

		case key.Matches(msg, m.keys.Increment):
			text := fmt.Sprintf("%s %s %s", m.inputs[month].Value(), m.inputs[day].Value(), m.inputs[year].Value())
			date, err := time.ParseInLocation(AbbreviatedTextDate, text, time.Local)
			if err != nil {
				return m, nil
			}
			date = date.AddDate(0, 0, 1)
			populateDateInputs(date, &m.inputs[month], &m.inputs[day], &m.inputs[year])
			return m, nil

		case key.Matches(msg, m.keys.Decrement):
			text := fmt.Sprintf("%s %s %s", m.inputs[month].Value(), m.inputs[day].Value(), m.inputs[year].Value())
			date, err := time.ParseInLocation(AbbreviatedTextDate, text, time.Local)
			if err != nil {
				return m, nil
			}
			date = date.AddDate(0, 0, -1)
			populateDateInputs(date, &m.inputs[month], &m.inputs[day], &m.inputs[year])
			return m, nil

		case key.Matches(msg, m.keys.Confirm):
			autofillEmptyInputs(m.inputs)
			text := fmt.Sprintf("%s %s %s", m.inputs[month].Value(), m.inputs[day].Value(), m.inputs[year].Value())
			date, err := time.ParseInLocation(AbbreviatedTextDate, text, time.Local)
			if err != nil {
				return m, showCalendarViewCmd
			}
			return m, tea.Batch(showCalendarViewCmd, gotoDateCmd(date))

		case key.Matches(msg, m.keys.Exit):
			return m, showCalendarViewCmd
		}
	}

	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *GotoDialog) autoformatInputs() {
	if m.focusIndex == month {
		autoformatMonthInput(&m.inputs[m.focusIndex])
	} else if m.focusIndex == day {
		autoformatDayInput(&m.inputs[m.focusIndex])
	} else if m.focusIndex == year {
		autoformatYearInput(&m.inputs[m.focusIndex])
	}
}

func (m GotoDialog) View() string {
	inputs := renderDateInputs(m.inputs[month], m.inputs[day], m.inputs[year])
	s := lipgloss.JoinHorizontal(lipgloss.Center, "Go to Date: ", inputs)
	helpView := lipgloss.NewStyle().
		Width(m.width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(m.help.View(m.keys))
	container := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-lipgloss.Height(helpView)-3).
		Align(lipgloss.Center, lipgloss.Center).
		Render(s)
	return lipgloss.JoinVertical(lipgloss.Center, container, helpView)
}

// -----------------------------------------------------------------------------
// Messages and Commands
// -----------------------------------------------------------------------------

type gotoDateMsg struct{ date time.Time }

func gotoDateCmd(date time.Time) tea.Cmd {
	return func() tea.Msg {
		return gotoDateMsg{date: date}
	}
}

// -----------------------------------------------------------------------------
// Keys
// -----------------------------------------------------------------------------

type keyMapGoto struct {
	Next      key.Binding
	Prev      key.Binding
	Increment key.Binding
	Decrement key.Binding
	Confirm   key.Binding
	Exit      key.Binding
}

var gotoKeymap = keyMapGoto{
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev"),
	),
	Increment: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "increment"),
	),
	Decrement: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "decrement"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter", "ctrl+s"),
		key.WithHelp("enter", "go"),
	),
	Exit: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "exit"),
	),
}

func (k keyMapGoto) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Prev, k.Increment, k.Decrement, k.Confirm, k.Exit}
}

func (k keyMapGoto) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next},
		{k.Prev},
		{k.Increment},
		{k.Decrement},
		{k.Confirm},
		{k.Exit},
	}
}

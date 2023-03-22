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
			if m.focusIndex == month {
				autoformatMonthInput(&m.inputs[m.focusIndex])
			} else if m.focusIndex == day {
				autoformatDayInput(&m.inputs[m.focusIndex])
			} else if m.focusIndex == year {
				autoformatYearInput(&m.inputs[m.focusIndex])
			}
			m.focusIndex = focusNext(m.inputs, m.focusIndex)
		case key.Matches(msg, m.keys.Prev):
			if m.focusIndex == month {
				autoformatMonthInput(&m.inputs[m.focusIndex])
			} else if m.focusIndex == day {
				autoformatDayInput(&m.inputs[m.focusIndex])
			} else if m.focusIndex == year {
				autoformatYearInput(&m.inputs[m.focusIndex])
			}
			m.focusIndex = focusPrev(m.inputs, m.focusIndex)
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
		case key.Matches(msg, m.keys.Go):
			autofillEmptyInputs(m.inputs)
			text := fmt.Sprintf("%s %s %s", m.inputs[month].Value(), m.inputs[day].Value(), m.inputs[year].Value())
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
	dateInputs := renderDateInputs(m.inputs[month], m.inputs[day], m.inputs[year])
	content := lipgloss.JoinHorizontal(lipgloss.Center, "Go to Date: ", dateInputs)
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
	Next      key.Binding
	Prev      key.Binding
	Increment key.Binding
	Decrement key.Binding
	Go        key.Binding
	Cancel    key.Binding
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
	Increment: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "increment date"),
	),
	Decrement: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "decrement date"),
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
	return []key.Binding{k.Next, k.Prev, k.Increment, k.Decrement, k.Go, k.Cancel}
}

func (k keyMapGoto) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next},
		{k.Go},
		{k.Prev},
		{k.Cancel},
		{k.Increment},
		{k.Decrement},
	}
}

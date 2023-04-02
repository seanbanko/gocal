package main

import (
	"fmt"
	"time"

	"gocal/common"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// -----------------------------------------------------------------------------
// Model
// -----------------------------------------------------------------------------

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
	inputs[month] = common.NewTextInput(common.MonthWidth)
	inputs[day] = common.NewTextInput(common.DayWidth)
	inputs[year] = common.NewTextInput(common.YearWidth)
	monthText, dayText, yearText := common.ToDateFields(focusedDate)
	inputs[month].Placeholder = monthText
	inputs[day].Placeholder = dayText
	inputs[year].Placeholder = yearText
	inputs[month].SetValue(monthText)
	inputs[day].SetValue(dayText)
	inputs[year].SetValue(yearText)
	focusIndex := month
	common.Refocus(inputs, focusIndex)
	return GotoDialog{
		inputs:     inputs,
		focusIndex: focusIndex,
		keys:       gotoKeymap,
		help:       help.New(),
		width:      width,
		height:     height,
	}
}

// -----------------------------------------------------------------------------
// Init
// -----------------------------------------------------------------------------

func (m GotoDialog) Init() tea.Cmd {
	return textinput.Blink
}

// -----------------------------------------------------------------------------
// Update
// -----------------------------------------------------------------------------

func (m GotoDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Next):
			m.autoformatInputs()
			m.focusIndex = (m.focusIndex + 1 + len(m.inputs)) % len(m.inputs)
			common.Refocus(m.inputs, m.focusIndex)
			return m, nil

		case key.Matches(msg, m.keys.Prev):
			m.autoformatInputs()
			m.focusIndex = (m.focusIndex - 1 + len(m.inputs)) % len(m.inputs)
			common.Refocus(m.inputs, m.focusIndex)
			return m, nil

		case key.Matches(msg, m.keys.Increment):
			m.adjustDate(1)
			return m, nil

		case key.Matches(msg, m.keys.Decrement):
			m.adjustDate(-1)
			return m, nil

		case key.Matches(msg, m.keys.Confirm):
			common.AutofillEmptyInputs(m.inputs)
			text := fmt.Sprintf("%s %s %s", m.inputs[month].Value(), m.inputs[day].Value(), m.inputs[year].Value())
			date, err := time.ParseInLocation(common.AbbreviatedTextDate, text, time.Local)
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

func (m *GotoDialog) adjustDate(days int) {
	text := fmt.Sprintf("%s %s %s", m.inputs[month].Value(), m.inputs[day].Value(), m.inputs[year].Value())
	date, err := time.ParseInLocation(common.AbbreviatedTextDate, text, time.Local)
	if err != nil {
		return
	}
	date = date.AddDate(0, 0, days)
	common.PopulateDateInputs(date, &m.inputs[month], &m.inputs[day], &m.inputs[year])
}

func (m *GotoDialog) autoformatInputs() {
	if m.focusIndex == month {
		common.AutoformatMonthInput(&m.inputs[m.focusIndex])
	} else if m.focusIndex == day {
		common.AutoformatDayInput(&m.inputs[m.focusIndex])
	} else if m.focusIndex == year {
		common.AutoformatYearInput(&m.inputs[m.focusIndex])
	}
}

// -----------------------------------------------------------------------------
// View
// -----------------------------------------------------------------------------

func (m GotoDialog) View() string {
	s := lipgloss.JoinHorizontal(
		lipgloss.Center,
		"Go to Date: ",
		common.RenderDateInputs(m.inputs[month], m.inputs[day], m.inputs[year]),
	)
	help := lipgloss.NewStyle().Width(m.width).Padding(1).AlignHorizontal(lipgloss.Center).Render(m.help.View(m.keys))
	body := lipgloss.Place(m.width, m.height-lipgloss.Height(help), lipgloss.Center, lipgloss.Center, s)
	return lipgloss.JoinVertical(lipgloss.Center, body, help)
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
// Keys and Help
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

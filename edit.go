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
	summary = iota
	startMonth
	startDay
	startYear
	startHour
	startMinute
	endHour
	endMinute
	endMonth
	endDay
	endYear
)

type EditDialog struct {
	inputs     []textinput.Model
	focusIndex int
	calendarId string
	eventId    string
	height     int
	width      int
	success    bool
	err        error
	help       help.Model
	keys       keyMapEdit
}

func newEditDialog(event *Event, focusedDate time.Time, width, height int) EditDialog {
	inputs := make([]textinput.Model, 11)

	inputs[summary] = textinput.New()
	inputs[summary].Placeholder = "Add title"
	inputs[summary].Width = summaryWidth
	inputs[summary].Prompt = ""
	inputs[summary].PlaceholderStyle = textInputPlaceholderStyle

	inputs[startMonth] = newMonthTextInput()
	inputs[startMonth].Placeholder = focusedDate.Month().String()[:3]

	inputs[startDay] = newDayTextInput()
	inputs[startDay].Placeholder = fmt.Sprintf("%02d", focusedDate.Day())

	inputs[startYear] = newYearTextInput()
	inputs[startYear].Placeholder = fmt.Sprintf("%d", focusedDate.Year())

	inputs[startHour] = newTimeTextInput()
	inputs[startHour].Placeholder = fmt.Sprintf("%02d", time.Now().Hour())

	inputs[startMinute] = newTimeTextInput()
	inputs[startMinute].Placeholder = "00"

	inputs[endMonth] = newMonthTextInput()
	inputs[endMonth].Placeholder = focusedDate.Month().String()[:3]

	inputs[endDay] = newDayTextInput()
	inputs[endDay].Placeholder = fmt.Sprintf("%02d", focusedDate.Day())

	inputs[endYear] = newYearTextInput()
	inputs[endYear].Placeholder = fmt.Sprintf("%d", focusedDate.Year())

	inputs[endHour] = newTimeTextInput()
	inputs[endHour].Placeholder = fmt.Sprintf("%02d", time.Now().Add(time.Hour).Hour())

	inputs[endMinute] = newTimeTextInput()
	inputs[endMinute].Placeholder = "00"

	var calendarId, eventId string
	if event != nil {
		calendarId = event.calendarId
		eventId = event.event.Id
		inputs[summary].SetValue(event.event.Summary)

		start, err := time.Parse(time.RFC3339, event.event.Start.DateTime)
		var sMonth, sDay, sYear, sHour, sMin string
		if err == nil {
			sMonth = start.Month().String()[:3]
			sDay = fmt.Sprintf("%02d", start.Day())
			sYear = fmt.Sprintf("%d", start.Year())
			sHour = fmt.Sprintf("%02d", start.Hour())
			sMin = fmt.Sprintf("%02d", start.Minute())
		}
		inputs[startMonth].SetValue(sMonth)
		inputs[startDay].SetValue(sDay)
		inputs[startYear].SetValue(sYear)
		inputs[startHour].SetValue(sHour)
		inputs[startMinute].SetValue(sMin)

		end, err := time.Parse(time.RFC3339, event.event.End.DateTime)
		var eMonth, eDay, eYear, eHour, eMin string
		if err == nil {
			eMonth = end.Month().String()[:3]
			eDay = fmt.Sprintf("%02d", end.Day())
			eYear = fmt.Sprintf("%d", end.Year())
			eHour = fmt.Sprintf("%02d", end.Hour())
			eMin = fmt.Sprintf("%02d", end.Minute())
		}
		inputs[endMonth].SetValue(eMonth)
		inputs[endDay].SetValue(eDay)
		inputs[endYear].SetValue(eYear)
		inputs[endHour].SetValue(eHour)
		inputs[endMinute].SetValue(eMin)
	} else {
		calendarId = "primary"
	}

	focusIndex := summary
	refocus(inputs, focusIndex)

	return EditDialog{
		inputs:     inputs,
		focusIndex: focusIndex,
		calendarId: calendarId,
		eventId:    eventId,
		height:     height,
		width:      width,
		success:    false,
		err:        nil,
		help:       help.New(),
		keys:       editKeyMap,
	}
}

func newMonthTextInput() textinput.Model {
	input := textinput.New()
	input.CharLimit = monthWidth
	input.PlaceholderStyle = textInputPlaceholderStyle
	input.Prompt = ""
	return input
}

func newDayTextInput() textinput.Model {
	input := textinput.New()
	input.CharLimit = dayWidth
	input.PlaceholderStyle = textInputPlaceholderStyle
	input.Prompt = ""
	return input
}

func newYearTextInput() textinput.Model {
	input := textinput.New()
	input.CharLimit = yearWidth
	input.PlaceholderStyle = textInputPlaceholderStyle
	input.Prompt = ""
	return input
}

func newTimeTextInput() textinput.Model {
	input := textinput.New()
	input.CharLimit = timeWidth
	input.PlaceholderStyle = textInputPlaceholderStyle
	input.Prompt = ""
	return input
}

func (m EditDialog) Init() tea.Cmd {
	return textinput.Blink
}

func (m EditDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		return m, nil
	case editEventResponseMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.success = true
		}
		return m, nil
	case tea.KeyMsg:
		if m.success || m.err != nil {
			return m, showCalendarViewCmd
		}
		switch {
		case key.Matches(msg, m.keys.Next):
			newIndex := focusNext(m.inputs, m.focusIndex)
			if m.focusIndex == startMonth {
				m.inputs[endMonth].SetValue(m.inputs[startMonth].Value())
			}
			if m.focusIndex == startDay {
				m.inputs[endDay].SetValue(m.inputs[startDay].Value())
			}
			if m.focusIndex == startYear {
				m.inputs[endYear].SetValue(m.inputs[startYear].Value())
			}
			m.focusIndex = newIndex
		case key.Matches(msg, m.keys.Prev):
			m.focusIndex = focusPrev(m.inputs, m.focusIndex)
		case key.Matches(msg, m.keys.Save):
			autofillPlaceholders(m.inputs)
			startDate := fmt.Sprintf(
				"%s %s %s",
				m.inputs[startMonth].Value(),
				m.inputs[startDay].Value(),
				m.inputs[startYear].Value(),
			)
			startTime := fmt.Sprintf(
				"%s:%s",
				m.inputs[startHour].Value(),
				m.inputs[startMinute].Value(),
			)
			endDate := fmt.Sprintf(
				"%s %s %s",
				m.inputs[endMonth].Value(),
				m.inputs[endDay].Value(),
				m.inputs[endYear].Value(),
			)
			endTime := fmt.Sprintf(
				"%s:%s",
				m.inputs[endHour].Value(),
				m.inputs[endMinute].Value(),
			)
			return m, editEventRequestCmd(
				m.calendarId,
				m.eventId,
				m.inputs[summary].Value(),
				startDate,
				startTime,
				endDate,
				endTime,
			)
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

func focusNext(inputs []textinput.Model, focusIndex int) int {
	if len(inputs[focusIndex].Value()) == 0 {
		inputs[focusIndex].SetValue(inputs[focusIndex].Placeholder)
	}
	newIndex := (focusIndex + 1) % len(inputs)
	refocus(inputs, newIndex)
	return newIndex
}

func focusPrev(inputs []textinput.Model, focusIndex int) int {
	newIndex := focusIndex - 1
	if newIndex < 0 {
		newIndex = len(inputs) - 1
	}
	refocus(inputs, newIndex)
	return newIndex
}

func autofillPlaceholders(inputs []textinput.Model) {
	for i := range inputs {
		if len(inputs[i].Value()) == 0 {
			inputs[i].SetValue(inputs[i].Placeholder)
		}
	}
}

func refocus(inputs []textinput.Model, focusIndex int) {
	for i := range inputs {
		inputs[i].Blur()
	}
	inputs[focusIndex].Focus()
}

func (m EditDialog) View() string {
	var content string
	if m.err != nil {
		content = "Error. Press any key to return to calendar."
	} else if m.success {
		content = "Success. Press any key to return to calendar."
	} else {
		content = renderEditContent(m)
	}
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

func renderEditContent(m EditDialog) string {
	return lipgloss.JoinVertical(
		lipgloss.Center,
		"Create/Edit Event\n",
		textInputSummaryStyle.Render(m.inputs[summary].View())+"\n",
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			textInputMonthStyle.Render(m.inputs[startMonth].View()),
			" ",
			textInputDayStyle.Render(m.inputs[startDay].View()),
			" ",
			textInputYearStyle.Render(m.inputs[startYear].View()),
			" at ",
			textInputTimeStyle.Copy().BorderRight(false).Render(m.inputs[startHour].View()),
			lipgloss.NewStyle().BorderTop(true).BorderBottom(true).BorderStyle(lipgloss.RoundedBorder()).Render(":"),
			textInputTimeStyle.Copy().BorderLeft(false).Render(m.inputs[startMinute].View()),
			" to ",
			textInputTimeStyle.Copy().BorderRight(false).Render(m.inputs[endHour].View()),
			lipgloss.NewStyle().BorderTop(true).BorderBottom(true).BorderStyle(lipgloss.RoundedBorder()).Render(":"),
			textInputTimeStyle.Copy().BorderLeft(false).Render(m.inputs[endMinute].View()),
			" on ",
			textInputMonthStyle.Render(m.inputs[endMonth].View()),
			" ",
			textInputDayStyle.Render(m.inputs[endDay].View()),
			" ",
			textInputYearStyle.Render(m.inputs[endYear].View()),
		),
	)
}

type keyMapEdit struct {
	Next   key.Binding
	Prev   key.Binding
	Save   key.Binding
	Cancel key.Binding
}

var editKeyMap = keyMapEdit{
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous field"),
	),
	Save: key.NewBinding(
		key.WithKeys("enter", "ctrl+s"),
		key.WithHelp("enter/ctrl+s", "save"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

func (k keyMapEdit) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Prev, k.Save, k.Cancel}
}

func (k keyMapEdit) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Next}, {k.Prev}, {k.Save}, {k.Cancel}}
}

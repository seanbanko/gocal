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
	duration   time.Duration
	allDay     bool
	height     int
	width      int
	success    bool
	err        error
	help       help.Model
	keys       keyMapEdit
}

func newEditDialog(event *Event, focusedDate time.Time, width, height int) EditDialog {
	var calendarId, eventId string
	if event == nil {
		calendarId = "primary"
		eventId = ""
	} else {
		calendarId = event.calendarId
		eventId = event.event.Id
	}

	inputs := make([]textinput.Model, 11)

	inputs[summary] = textinput.New()
	inputs[summary].Placeholder = "Add title"
	inputs[summary].Width = summaryWidth
	inputs[summary].Prompt = ""
	inputs[summary].PlaceholderStyle = textInputPlaceholderStyle

	inputs[startMonth] = newTextInput(monthWidth)
	inputs[startDay] = newTextInput(dayWidth)
	inputs[startYear] = newTextInput(yearWidth)
	inputs[startHour] = newTextInput(timeWidth)
	inputs[startMinute] = newTextInput(timeWidth)
	inputs[endMonth] = newTextInput(monthWidth)
	inputs[endDay] = newTextInput(dayWidth)
	inputs[endYear] = newTextInput(yearWidth)
	inputs[endHour] = newTextInput(timeWidth)
	inputs[endMinute] = newTextInput(timeWidth)

	var start, end time.Time
	var allDay bool
	if event == nil {
		allDay = false
		start = time.Date(focusedDate.Year(), focusedDate.Month(), focusedDate.Day(), time.Now().Hour(), time.Now().Minute(), 0, 0, time.Local).Round(time.Hour)
		end = start.Add(time.Hour)
	} else {
		var eventStart, eventEnd time.Time
		// TODO handle errors
		if isAllDay(event.event) {
			allDay = true
			eventStart, _ = time.Parse(YYYYMMDD, event.event.Start.Date)
			eventStart = time.Date(eventStart.Year(), eventStart.Month(), eventStart.Day(), 0, 0, 0, 0, time.Local)
			eventEnd, _ = time.Parse(YYYYMMDD, event.event.End.Date)
			eventEnd = time.Date(eventEnd.Year(), eventEnd.Month(), eventEnd.Day(), 0, 0, 0, 0, time.Local)
		} else {
			allDay = false
			eventStart, _ = time.Parse(time.RFC3339, event.event.Start.DateTime)
			eventEnd, _ = time.Parse(time.RFC3339, event.event.End.DateTime)
		}
		start = eventStart.In(time.Local)
		end = eventEnd.In(time.Local)
	}

	var (
		startMonthText, startDayText, startYearText = toMonthDayYear(start)
		startHourText                               = fmt.Sprintf("%02d", start.Hour())
		startMinuteText                             = fmt.Sprintf("%02d", start.Minute())
		endMonthText, endDayText, endYearText       = toMonthDayYear(end)
		endHourText                                 = fmt.Sprintf("%02d", end.Hour())
		endMinuteText                               = fmt.Sprintf("%02d", end.Minute())
	)

	inputs[startMonth].Placeholder = startMonthText
	inputs[startDay].Placeholder = startDayText
	inputs[startYear].Placeholder = startYearText
	inputs[startHour].Placeholder = startHourText
	inputs[startMinute].Placeholder = startMinuteText
	inputs[endMonth].Placeholder = endMonthText
	inputs[endDay].Placeholder = endDayText
	inputs[endYear].Placeholder = endYearText
	inputs[endHour].Placeholder = endHourText
	inputs[endMinute].Placeholder = endMinuteText

	if event != nil {
		inputs[summary].SetValue(event.event.Summary)
		inputs[startMonth].SetValue(startMonthText)
		inputs[startDay].SetValue(startDayText)
		inputs[startYear].SetValue(startYearText)
		inputs[startHour].SetValue(startHourText)
		inputs[startMinute].SetValue(startMinuteText)
		inputs[endMonth].SetValue(endMonthText)
		inputs[endDay].SetValue(endDayText)
		inputs[endYear].SetValue(endYearText)
		inputs[endHour].SetValue(endHourText)
		inputs[endMinute].SetValue(endMinuteText)
	}

	duration := end.Sub(start)

	focusIndex := summary
	refocus(inputs, focusIndex)

	return EditDialog{
		inputs:     inputs,
		focusIndex: focusIndex,
		calendarId: calendarId,
		eventId:    eventId,
		duration:   duration,
		allDay:     allDay,
		height:     height,
		width:      width,
		success:    false,
		err:        nil,
		help:       help.New(),
		keys:       editKeyMap,
	}
}

func (m EditDialog) Init() tea.Cmd {
	return textinput.Blink
}

func (m EditDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		return m, nil
	case errMsg:
		m.err = msg.err
		return m, nil
	case successMsg:
		m.success = true
		return m, nil
	case tea.KeyMsg:
		if m.success || m.err != nil {
			return m, showCalendarViewCmd
		}
		switch {
		case key.Matches(msg, m.keys.Next):
			if len(m.inputs[m.focusIndex].Value()) == 0 {
				autofillPlaceholder(&m.inputs[m.focusIndex])
			}
			if isOnStartInput(m.focusIndex) {
				m.autofillEndInputs()
			} else if isOnEndInput(m.focusIndex) {
				m.updateDuration()
			}
			m.focusIndex = focusNext(m.inputs, m.focusIndex)
			if m.allDay && isOnTimeInput(m.focusIndex) {
				m.focusIndex = endMonth
				refocus(m.inputs, m.focusIndex)
			}
			return m, nil
		case key.Matches(msg, m.keys.Prev):
			if isOnEndInput(m.focusIndex) {
				m.updateDuration()
			}
			m.focusIndex = focusPrev(m.inputs, m.focusIndex)
			if m.allDay && isOnTimeInput(m.focusIndex) {
				m.focusIndex = startYear
				refocus(m.inputs, m.focusIndex)
			}
			return m, nil
		case key.Matches(msg, m.keys.ToggleAllDay):
			m.allDay = !m.allDay
			if m.allDay && isOnTimeInput(m.focusIndex) {
				m.focusIndex = startYear
			}
			refocus(m.inputs, m.focusIndex)
			return m, nil
		case key.Matches(msg, m.keys.Save):
			autofillAllPlaceholders(m.inputs)
			summary := m.inputs[summary].Value()
			start, err := toDate(
				m.inputs[startMonth].Value(),
				m.inputs[startDay].Value(),
				m.inputs[startYear].Value(),
				m.inputs[startHour].Value(),
				m.inputs[startMinute].Value(),
			)
			if err != nil {
				return m, func() tea.Msg { return errMsg{err: err} }
			}
			end, err := toDate(
				m.inputs[endMonth].Value(),
				m.inputs[endDay].Value(),
				m.inputs[endYear].Value(),
				m.inputs[endHour].Value(),
				m.inputs[endMinute].Value(),
			)
			if err != nil {
				return m, func() tea.Msg { return errMsg{err: err} }
			}
			return m, editEventRequestCmd(m.calendarId, m.eventId, summary, start, end, m.allDay)
		case key.Matches(msg, m.keys.Cancel):
			return m, showCalendarViewCmd
		case msg.Type == tea.KeySpace && (m.focusIndex == startMonth || m.focusIndex == startDay || m.focusIndex == endMonth || m.focusIndex == endDay):
			m.focusIndex = focusNext(m.inputs, m.focusIndex)
			return m, nil
		case msg.Type == tea.KeyBackspace && m.inputs[m.focusIndex].Cursor() == 0 && m.focusIndex != 0:
			m.focusIndex = focusPrev(m.inputs, m.focusIndex)
			return m, nil
		case ((m.focusIndex == startHour && isFull(m.inputs[startHour])) || (m.focusIndex == endHour && isFull(m.inputs[endHour]))) &&
			!(msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete):
			m.focusIndex = focusNext(m.inputs, m.focusIndex)
		}
	}
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m *EditDialog) updateDuration() {
	start, err := toDate(
		m.inputs[startMonth].Value(),
		m.inputs[startDay].Value(),
		m.inputs[startYear].Value(),
		m.inputs[startHour].Value(),
		m.inputs[startMinute].Value(),
	)
	if err != nil {
		return
	}
	end, err := toDate(
		m.inputs[endMonth].Value(),
		m.inputs[endDay].Value(),
		m.inputs[endYear].Value(),
		m.inputs[endHour].Value(),
		m.inputs[endMinute].Value(),
	)
	if err != nil {
		return
	}
	m.duration = end.Sub(start)
}

func (m *EditDialog) autofillEndInputs() {
	start, err := toDate(
		m.inputs[startMonth].Value(),
		m.inputs[startDay].Value(),
		m.inputs[startYear].Value(),
		m.inputs[startHour].Value(),
		m.inputs[startMinute].Value(),
	)
	if err != nil {
		return
	}
	end := start.Add(m.duration)
	month, day, year, hour, minute := toMonthDayYearHourMinute(end)
	m.inputs[endMonth].SetValue(month)
	m.inputs[endDay].SetValue(day)
	m.inputs[endYear].SetValue(year)
	m.inputs[endHour].SetValue(hour)
	m.inputs[endMinute].SetValue(minute)
}

func isOnStartInput(focusIndex int) bool {
	return focusIndex == startDay ||
		focusIndex == startMonth ||
		focusIndex == startYear ||
		focusIndex == startHour ||
		focusIndex == startMinute
}

func isOnEndInput(focusIndex int) bool {
	return focusIndex == endDay ||
		focusIndex == endMonth ||
		focusIndex == endYear ||
		focusIndex == endHour ||
		focusIndex == endMinute
}

func isOnTimeInput(focusIndex int) bool {
	return focusIndex == startHour || focusIndex == startMinute || focusIndex == endHour || focusIndex == endMinute
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
	var duration, startTimeInputs, endTimeInputs string
	if m.allDay {
		duration = "[X] all day"
		startTimeInputs = ""
		endTimeInputs = ""
	} else {
		duration = m.duration.String()
		startTimeInputs = lipgloss.JoinHorizontal(lipgloss.Center,
			" at ",
			textInputTimeStyle.Copy().BorderRight(false).Render(m.inputs[startHour].View()),
			lipgloss.NewStyle().BorderTop(true).BorderBottom(true).BorderStyle(lipgloss.RoundedBorder()).Render(":"),
			textInputTimeStyle.Copy().BorderLeft(false).Render(m.inputs[startMinute].View()),
		)
		endTimeInputs = lipgloss.JoinHorizontal(
			lipgloss.Center,
			textInputTimeStyle.Copy().BorderRight(false).Render(m.inputs[endHour].View()),
			lipgloss.NewStyle().BorderTop(true).BorderBottom(true).BorderStyle(lipgloss.RoundedBorder()).Render(":"),
			textInputTimeStyle.Copy().BorderLeft(false).Render(m.inputs[endMinute].View()),
			" on ",
		)
	}
	return lipgloss.JoinVertical(
		lipgloss.Center,
		"Create/Edit Event",
		"\n",
		textInputSummaryStyle.Render(m.inputs[summary].View()),
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			textInputMonthStyle.Render(m.inputs[startMonth].View()),
			" ",
			textInputDayStyle.Render(m.inputs[startDay].View()),
			" ",
			textInputYearStyle.Render(m.inputs[startYear].View()),
			startTimeInputs,
			" to ",
			endTimeInputs,
			textInputMonthStyle.Render(m.inputs[endMonth].View()),
			" ",
			textInputDayStyle.Render(m.inputs[endDay].View()),
			" ",
			textInputYearStyle.Render(m.inputs[endYear].View()),
		),
		duration,
		"", // TODO the last line is not being centered properly so this is just here for that
	)
}

type keyMapEdit struct {
	Next         key.Binding
	Prev         key.Binding
	ToggleAllDay key.Binding
	Save         key.Binding
	Cancel       key.Binding
}

var editKeyMap = keyMapEdit{
	Next: key.NewBinding(
		key.WithKeys("tab", "enter"),
		key.WithHelp("tab", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous field"),
	),
	ToggleAllDay: key.NewBinding(
		key.WithKeys("ctrl+a"),
		key.WithHelp("ctrl+a", "toggle all day"),
	),
	Save: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

func (k keyMapEdit) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Prev, k.ToggleAllDay, k.Save, k.Cancel}
}

func (k keyMapEdit) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Next}, {k.Prev}, {k.ToggleAllDay}, {k.Save}, {k.Cancel}}
}

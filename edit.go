package main

import (
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
	startTime
	endTime
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

	inputs := make([]textinput.Model, 9)

	inputs[summary] = textinput.New()
	inputs[summary].Placeholder = "Add title"
	inputs[summary].Width = summaryWidth
	inputs[summary].Prompt = ""
	inputs[summary].PlaceholderStyle = textInputPlaceholderStyle

	inputs[startMonth] = newTextInput(monthWidth)
	inputs[startDay] = newTextInput(dayWidth)
	inputs[startYear] = newTextInput(yearWidth)
	inputs[startTime] = newTextInput(timeWidth)
	inputs[endTime] = newTextInput(timeWidth)
	inputs[endMonth] = newTextInput(monthWidth)
	inputs[endDay] = newTextInput(dayWidth)
	inputs[endYear] = newTextInput(yearWidth)

	var start, end time.Time
	var allDay bool
	if event == nil {
		allDay = false
		start = time.Date(focusedDate.Year(), focusedDate.Month(), focusedDate.Day(), time.Now().Hour(), time.Now().Minute(), 0, 0, time.Local).Truncate(30*time.Minute).Add(30*time.Minute)
		end = start.Add(time.Hour)
	} else {
		var eventStart, eventEnd time.Time
		if isAllDay(event.event) {
			allDay = true
			// TODO handle errors
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
		startMonthText, startDayText, startYearText = toDateFields(start)
		startTimeText                               = start.Format(HH_MM_PM)
		endMonthText, endDayText, endYearText       = toDateFields(end)
		endTimeText                                 = end.Format(HH_MM_PM)
	)

	inputs[startMonth].Placeholder = startMonthText
	inputs[startDay].Placeholder = startDayText
	inputs[startYear].Placeholder = startYearText
	inputs[startTime].Placeholder = startTimeText
	inputs[endTime].Placeholder = endTimeText
	inputs[endMonth].Placeholder = endMonthText
	inputs[endDay].Placeholder = endDayText
	inputs[endYear].Placeholder = endYearText

	if event != nil {
		inputs[summary].SetValue(event.event.Summary)
		inputs[startMonth].SetValue(startMonthText)
		inputs[startDay].SetValue(startDayText)
		inputs[startYear].SetValue(startYearText)
		inputs[startTime].SetValue(startTimeText)
		inputs[endTime].SetValue(endTimeText)
		inputs[endMonth].SetValue(endMonthText)
		inputs[endDay].SetValue(endDayText)
		inputs[endYear].SetValue(endYearText)
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
			if isEmpty(m.inputs[m.focusIndex]) {
				autofillPlaceholder(&m.inputs[m.focusIndex])
			}
			if m.isOnStartInput() {
				m.adjustEndInputs()
				autoformatDateTimeInputs(&m.inputs[startMonth], &m.inputs[startDay], &m.inputs[startYear], &m.inputs[startTime])
			} else if m.isOnEndInput() {
				m.updateDuration()
				autoformatDateTimeInputs(&m.inputs[endMonth], &m.inputs[endDay], &m.inputs[endYear], &m.inputs[endTime])
			}
			m.focusIndex = focusNext(m.inputs, m.focusIndex)
			if m.allDay && m.isOnTimeInput() {
				m.focusIndex = endMonth
				refocus(m.inputs, m.focusIndex)
			}
			m.inputs[m.focusIndex].CursorEnd()
			return m, nil
		case key.Matches(msg, m.keys.Prev):
			if m.isOnEndInput() {
				m.updateDuration()
				autoformatDateTimeInputs(&m.inputs[endMonth], &m.inputs[endDay], &m.inputs[endYear], &m.inputs[endTime])
			}
			m.focusIndex = focusPrev(m.inputs, m.focusIndex)
			if m.allDay && m.isOnTimeInput() {
				m.focusIndex = startYear
				refocus(m.inputs, m.focusIndex)
			}
			m.inputs[m.focusIndex].CursorEnd()
			return m, nil
		case key.Matches(msg, m.keys.ToggleAllDay):
			m.allDay = !m.allDay
			if m.allDay && m.isOnTimeInput() {
				m.focusIndex = startYear
			}
			refocus(m.inputs, m.focusIndex)
			return m, nil
		case key.Matches(msg, m.keys.Save):
			autofillEmptyInputs(m.inputs)
			summary := m.inputs[summary].Value()
			start, err := parseDateTimeInputs(
				m.inputs[startMonth].Value(),
				m.inputs[startDay].Value(),
				m.inputs[startYear].Value(),
				m.inputs[startTime].Value(),
			)
			if err != nil {
				return m, func() tea.Msg { return errMsg{err: err} }
			}
			end, err := parseDateTimeInputs(
				m.inputs[endMonth].Value(),
				m.inputs[endDay].Value(),
				m.inputs[endYear].Value(),
				m.inputs[endTime].Value(),
			)
			if err != nil {
				return m, func() tea.Msg { return errMsg{err: err} }
			}
			return m, editEventRequestCmd(m.calendarId, m.eventId, summary, start, end, m.allDay)
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

func (m *EditDialog) isOnStartInput() bool {
	return m.focusIndex == startDay || m.focusIndex == startMonth || m.focusIndex == startYear || m.focusIndex == startTime
}

func (m *EditDialog) isOnEndInput() bool {
	return m.focusIndex == endDay || m.focusIndex == endMonth || m.focusIndex == endYear || m.focusIndex == endTime
}

func (m *EditDialog) isOnTimeInput() bool {
	return m.focusIndex == startTime || m.focusIndex == endTime
}

func (m *EditDialog) updateDuration() {
	start, err := parseDateTimeInputs(
		m.inputs[startMonth].Value(),
		m.inputs[startDay].Value(),
		m.inputs[startYear].Value(),
		m.inputs[startTime].Value(),
	)
	if err != nil {
		return
	}
	end, err := parseDateTimeInputs(
		m.inputs[endMonth].Value(),
		m.inputs[endDay].Value(),
		m.inputs[endYear].Value(),
		m.inputs[endTime].Value(),
	)
	if err != nil {
		return
	}
	m.duration = end.Sub(start)
}

func (m *EditDialog) adjustEndInputs() {
	start, err := parseDateTimeInputs(
		m.inputs[startMonth].Value(),
		m.inputs[startDay].Value(),
		m.inputs[startYear].Value(),
		m.inputs[startTime].Value(),
	)
	if err != nil {
		return
	}
	end := start.Add(m.duration)
	populateDateTimeInputs(
		end,
		&m.inputs[endMonth],
		&m.inputs[endDay],
		&m.inputs[endYear],
		&m.inputs[endTime],
	)
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
			textInputBaseStyle.Copy().Width(timeWidth+2).Render(m.inputs[startTime].View()),
		)
		endTimeInputs = lipgloss.JoinHorizontal(
			lipgloss.Center,
			textInputBaseStyle.Copy().Width(timeWidth+2).Render(m.inputs[endTime].View()),
			" on ",
		)
	}
	startDateInputs := renderDateInputs(m.inputs[startMonth], m.inputs[startDay], m.inputs[startYear])
	endDateInputs := renderDateInputs(m.inputs[endMonth], m.inputs[endDay], m.inputs[endYear])
	return lipgloss.JoinVertical(
		lipgloss.Center,
		"Create/Edit Event\n",
		textInputBaseStyle.Copy().Width(summaryWidth+2).Render(m.inputs[summary].View()),
		lipgloss.JoinHorizontal(lipgloss.Center, startDateInputs, startTimeInputs, " to ", endTimeInputs, endDateInputs),
		duration,
		"", // TODO the last line is not being centered properly so this is just here for that
	)
}

func renderDateInputs(month, day, year textinput.Model) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		textInputBaseStyle.Copy().Width(monthWidth+2).Render(month.View()),
		" ",
		textInputBaseStyle.Copy().Width(dayWidth+2).Render(day.View()),
		" ",
		textInputBaseStyle.Copy().Width(yearWidth+2).Render(year.View()),
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

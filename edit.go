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
		startMonthText  = start.Month().String()[:3]
		startDayText    = fmt.Sprintf("%02d", start.Day())
		startYearText   = fmt.Sprintf("%d", start.Year())
		startHourText   = fmt.Sprintf("%02d", start.Hour())
		startMinuteText = fmt.Sprintf("%02d", start.Minute())
		endMonthText    = end.Month().String()[:3]
		endDayText      = fmt.Sprintf("%02d", end.Day())
		endYearText     = fmt.Sprintf("%d", end.Year())
		endHourText     = fmt.Sprintf("%02d", end.Hour())
		endMinuteText   = fmt.Sprintf("%02d", end.Minute())
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
				autofill(&m.inputs[m.focusIndex])
			}
			if m.focusIndex == startMonth {
				m.inputs[endMonth].SetValue(m.inputs[startMonth].Value())
			} else if m.focusIndex == startDay {
				m.inputs[endDay].SetValue(m.inputs[startDay].Value())
			} else if m.focusIndex == startYear {
				m.inputs[endYear].SetValue(m.inputs[startYear].Value())
			} else if m.focusIndex == startHour || m.focusIndex == startMinute {
				startTime := fmt.Sprintf("%s:%s", m.inputs[startHour].Value(), m.inputs[startMinute].Value())
				start, err := time.Parse(HHMM24h, startTime)
				if err == nil {
					m.inputs[endHour].SetValue(fmt.Sprintf("%02d", start.Add(m.duration).Hour()))
					m.inputs[endMinute].SetValue(fmt.Sprintf("%02d", start.Add(m.duration).Minute()))
				}
			} else if m.focusIndex == endHour || m.focusIndex == endMinute {
				startTime := fmt.Sprintf("%s:%s", m.inputs[startHour].Value(), m.inputs[startMinute].Value())
				endTime := fmt.Sprintf("%s:%s", m.inputs[endHour].Value(), m.inputs[endMinute].Value())
				m.duration = updateDuration(m.duration, startTime, endTime)
			}
            m.focusIndex = focusNext(m.inputs, m.focusIndex)
            // skip time inputs
			if m.allDay && startHour <= m.focusIndex && m.focusIndex <= endMinute {
				m.focusIndex = endMonth
				refocus(m.inputs, m.focusIndex)
            }
		case key.Matches(msg, m.keys.Prev):
			if m.focusIndex == endHour || m.focusIndex == endMinute {
				startTime := fmt.Sprintf("%s:%s", m.inputs[startHour].Value(), m.inputs[startMinute].Value())
				endTime := fmt.Sprintf("%s:%s", m.inputs[endHour].Value(), m.inputs[endMinute].Value())
				m.duration = updateDuration(m.duration, startTime, endTime)
			}
            m.focusIndex = focusPrev(m.inputs, m.focusIndex)
            // skip time inputs
			if m.allDay && startHour <= m.focusIndex && m.focusIndex <= endMinute {
				m.focusIndex = startMonth
				refocus(m.inputs, m.focusIndex)
            }
		case key.Matches(msg, m.keys.Save):
			autofillAll(m.inputs)
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
				m.allDay,
			)
		case key.Matches(msg, m.keys.Cancel):
			return m, showCalendarViewCmd
		case key.Matches(msg, m.keys.ToggleAllDay):
			if m.allDay {
				m.allDay = false
				start := time.Now().Round(time.Hour)
				end := start.Add(time.Hour)
				var (
					startHourText   = fmt.Sprintf("%02d", start.Hour())
					startMinuteText = fmt.Sprintf("%02d", start.Minute())
					endHourText     = fmt.Sprintf("%02d", end.Hour())
					endMinuteText   = fmt.Sprintf("%02d", end.Minute())
				)
				m.inputs[startHour].SetValue(startHourText)
				m.inputs[startMinute].SetValue(startMinuteText)
				m.inputs[endHour].SetValue(endHourText)
				m.inputs[endMinute].SetValue(endMinuteText)
				startTime := fmt.Sprintf("%s:%s", m.inputs[startHour].Value(), m.inputs[startMinute].Value())
				endTime := fmt.Sprintf("%s:%s", m.inputs[endHour].Value(), m.inputs[endMinute].Value())
				m.duration = updateDuration(m.duration, startTime, endTime)
			} else {
				m.allDay = true
			}
			if m.allDay && startHour <= m.focusIndex && m.focusIndex <= endMinute {
				m.focusIndex = endMonth
            }
            refocus(m.inputs, m.focusIndex)
			return m, nil
		case !(msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete) && ((m.focusIndex == startHour && len(m.inputs[startHour].Value()) == m.inputs[startHour].CharLimit) ||
			(m.focusIndex == endHour && len(m.inputs[endHour].Value()) == m.inputs[endHour].CharLimit)):
			m.focusIndex = focusNext(m.inputs, m.focusIndex)
		case msg.Type == tea.KeySpace && (m.focusIndex == startMonth || m.focusIndex == startDay || m.focusIndex == endMonth || m.focusIndex == endDay):
			m.focusIndex = focusNext(m.inputs, m.focusIndex)
			return m, nil
		case msg.Type == tea.KeyBackspace && (m.inputs[m.focusIndex].Cursor() == 0) && m.focusIndex != 0:
			m.focusIndex = focusPrev(m.inputs, m.focusIndex)
			return m, nil
		}
	}
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func updateDuration(fallback time.Duration, startTime, endTime string) time.Duration {
	start, err := time.Parse(HHMM24h, startTime)
	if err != nil {
		return fallback
	}
	end, err := time.Parse(HHMM24h, endTime)
	if err != nil {
		return fallback
	}
	return end.Sub(start)
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
	var allDayBox, duration, startTimeInputs, endTimeInputs string
	if m.allDay {
		allDayBox = "[X]"
		duration = "all day"
		startTimeInputs = ""
		endTimeInputs = ""
	} else {
		allDayBox = "[ ]"
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
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			textInputSummaryStyle.Render(m.inputs[summary].View()),
			" ",
			allDayBox,
			// lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(allDayBox),
			" All Day",
		),
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

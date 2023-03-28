package main

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/api/calendar/v3"
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
	calId
)

type EditPage struct {
	srv           *calendar.Service
	inputs        []textinput.Model
	focusIndex    int
	calendarId    string
	eventId       string
	duration      time.Duration
	allDay        bool
	calendars     []*calendar.CalendarListEntry
	calendarIndex int
	height        int
	width         int
	success       bool
	pending       bool
	spinner       spinner.Model
	err           error
	help          help.Model
	keys          keyMapEdit
}

func newEditPage(srv *calendar.Service, event *Event, focusedDate time.Time, calendars []*calendar.CalendarListEntry, width, height int) EditPage {
	var calendarId, eventId string
	if event != nil {
		calendarId = event.calendarId
		eventId = event.event.Id
	}

	inputs := make([]textinput.Model, 10)

	inputs[summary] = textinput.New()
	inputs[summary].Placeholder = "Add title"
	inputs[summary].Width = summaryWidth
	inputs[summary].Prompt = " "

	inputs[startMonth] = newTextInput(monthWidth)
	inputs[startDay] = newTextInput(dayWidth)
	inputs[startYear] = newTextInput(yearWidth)
	inputs[startTime] = newTextInput(timeWidth)
	inputs[endTime] = newTextInput(timeWidth)
	inputs[endMonth] = newTextInput(monthWidth)
	inputs[endDay] = newTextInput(dayWidth)
	inputs[endYear] = newTextInput(yearWidth)

	inputs[calId] = textinput.New()
	inputs[calId].Prompt = " "
	inputs[calId].SetCursorMode(textinput.CursorHide)

	var start, end time.Time
	var allDay bool
	if event == nil {
		allDay = false
		start = time.Date(focusedDate.Year(), focusedDate.Month(), focusedDate.Day(), time.Now().Hour(), time.Now().Minute(), 0, 0, time.Local).Truncate(30 * time.Minute).Add(30 * time.Minute)
		end = start.Add(time.Hour)
	} else {
		var eventStart, eventEnd time.Time
		if isAllDay(*event) {
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

	var calendarIndex int
	if event != nil {
		inputs[summary].SetValue(event.event.Summary)
		for i, calendar := range calendars {
			if calendar.Id == event.calendarId {
				calendarIndex = i
			}
		}
	}
	inputs[startMonth].SetValue(startMonthText)
	inputs[startDay].SetValue(startDayText)
	inputs[startYear].SetValue(startYearText)
	inputs[startTime].SetValue(startTimeText)
	inputs[endTime].SetValue(endTimeText)
	inputs[endMonth].SetValue(endMonthText)
	inputs[endDay].SetValue(endDayText)
	inputs[endYear].SetValue(endYearText)
	inputs[calId].SetValue(calendars[calendarIndex].Summary + " ⏷")

	duration := end.Sub(start)

	focusIndex := summary
	refocus(inputs, focusIndex)

	s := spinner.New()
	s.Spinner = spinner.Points

	return EditPage{
		srv:           srv,
		inputs:        inputs,
		focusIndex:    focusIndex,
		calendarId:    calendarId,
		eventId:       eventId,
		duration:      duration,
		allDay:        allDay,
		calendars:     calendars,
		calendarIndex: 0,
		height:        height,
		width:         width,
		success:       false,
		pending:       false,
		spinner:       s,
		err:           nil,
		help:          help.New(),
		keys:          editKeyMap,
	}
}

func (m EditPage) Init() tea.Cmd {
	return textinput.Blink
}

func (m EditPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.success || m.err != nil {
			return m, tea.Sequence(flushCacheCmd, showCalendarViewCmd)
		}
		switch {
		case key.Matches(msg, m.keys.Next):
			if m.isOnDateTimeInput() {
				m.autoformatInputs()
				m.adjustDuration()
			}
			m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
			if m.allDay && m.isOnTimeInput() {
				m.focusIndex = endMonth
			}
			refocus(m.inputs, m.focusIndex)
			m.inputs[m.focusIndex].CursorEnd()
			return m, nil

		case key.Matches(msg, m.keys.Prev):
			if m.isOnDateTimeInput() {
				m.autoformatInputs()
				m.adjustDuration()
			}
			m.focusIndex = m.focusIndex - 1
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			if m.allDay && m.isOnTimeInput() {
				m.focusIndex = startYear
			}
			refocus(m.inputs, m.focusIndex)
			m.inputs[m.focusIndex].CursorEnd()
			return m, nil

		case key.Matches(msg, m.keys.ToggleAllDay):
			m.allDay = !m.allDay
			if m.allDay && m.isOnTimeInput() {
				m.focusIndex = startYear
			}
			refocus(m.inputs, m.focusIndex)
			return m, nil

		case key.Matches(msg, m.keys.NextCal) && m.focusIndex == calId:
			m.calendarIndex = (m.calendarIndex + 1) % len(m.calendars)
			m.inputs[calId].SetValue(m.calendars[m.calendarIndex].Summary + " ⏷")
			return m, nil

		case key.Matches(msg, m.keys.PrevCal) && m.focusIndex == calId:
			m.calendarIndex = m.calendarIndex - 1
			if m.calendarIndex < 0 {
				m.calendarIndex = len(m.calendars) - 1
			}
			m.inputs[calId].SetValue(m.calendars[m.calendarIndex].Summary + " ⏷")
			return m, nil

		case key.Matches(msg, m.keys.Save):
			autofillEmptyInputs(m.inputs)
			start, err := m.parseStart()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err: err} }
			}
			end, err := m.parseEnd()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err: err} }
			}
			m.pending = true
			editEventRequestMsg := editEventRequestMsg{
				calendarId: m.calendars[m.calendarIndex].Id,
				eventId:    m.eventId,
				summary:    m.inputs[summary].Value(),
				start:      start,
				end:        end,
				allDay:     m.allDay,
			}
			return m, tea.Batch(
				editEvent(m.srv, editEventRequestMsg),
				m.spinner.Tick,
			)
		case key.Matches(msg, m.keys.Exit):
			return m, showCalendarViewCmd
		}

	case errMsg:
		m.err = msg.err
		m.pending = false
		return m, nil

	case successMsg:
		m.success = true
		m.pending = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	}

	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == calId {
			continue
		}
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m *EditPage) autoformatInputs() {
	if m.isOnMonthInput() {
		autoformatMonthInput(&m.inputs[m.focusIndex])
	} else if m.isOnDayInput() {
		autoformatDayInput(&m.inputs[m.focusIndex])
	} else if m.isOnYearInput() {
		autoformatYearInput(&m.inputs[m.focusIndex])
	} else if m.isOnTimeInput() {
		autoformatTimeInput(&m.inputs[m.focusIndex])
	}
}

func (m *EditPage) adjustDuration() {
	if m.isOnStartInput() {
		m.adjustEndInputs()
	} else if m.isOnEndInput() {
		m.updateDuration()
	}
}

func (m EditPage) isOnDateTimeInput() bool {
	return m.isOnStartInput() || m.isOnEndInput()
}

func (m EditPage) isOnStartInput() bool {
	return m.focusIndex == startDay || m.focusIndex == startMonth || m.focusIndex == startYear || m.focusIndex == startTime
}

func (m EditPage) isOnEndInput() bool {
	return m.focusIndex == endDay || m.focusIndex == endMonth || m.focusIndex == endYear || m.focusIndex == endTime
}

func (m EditPage) isOnMonthInput() bool {
	return m.focusIndex == startMonth || m.focusIndex == endMonth
}

func (m EditPage) isOnDayInput() bool {
	return m.focusIndex == startDay || m.focusIndex == endDay
}

func (m EditPage) isOnYearInput() bool {
	return m.focusIndex == startYear || m.focusIndex == endYear
}

func (m EditPage) isOnTimeInput() bool {
	return m.focusIndex == startTime || m.focusIndex == endTime
}

func (m EditPage) parseStart() (time.Time, error) {
	return parseDateTime(
		m.inputs[startMonth].Value(),
		m.inputs[startDay].Value(),
		m.inputs[startYear].Value(),
		m.inputs[startTime].Value(),
	)
}

func (m EditPage) parseEnd() (time.Time, error) {
	return parseDateTime(
		m.inputs[endMonth].Value(),
		m.inputs[endDay].Value(),
		m.inputs[endYear].Value(),
		m.inputs[endTime].Value(),
	)
}

func (m *EditPage) updateDuration() {
	start, err := m.parseStart()
	if err != nil {
		return
	}
	end, err := m.parseEnd()
	if err != nil {
		return
	}
	m.duration = end.Sub(start)
}

func (m *EditPage) adjustEndInputs() {
	start, err := m.parseStart()
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

func (m EditPage) View() string {
	var s string
	if m.err != nil {
		s = "Error. Press any key to return to calendar."
	} else if m.success {
		s = "Success. Press any key to return to calendar."
	} else if m.pending {
		s = m.spinner.View()
	} else {
		s = renderEditContent(m)
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
		Render(s)
	return lipgloss.JoinVertical(lipgloss.Center, container, helpView)
}

func renderEditContent(m EditPage) string {
	var title string
	if m.eventId == "" {
		title = "Create Event"
	} else {
		title = "Edit Event"
	}
	var duration, startTimeInputs, endTimeInputs string
	if m.allDay {
		duration = "[X] all day"
		startTimeInputs = ""
		endTimeInputs = ""
	} else {
		duration = m.duration.String()
		startTimeInputs = lipgloss.JoinHorizontal(lipgloss.Center,
			" at ",
			lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(timeWidth+2).Render(m.inputs[startTime].View()),
		)
		endTimeInputs = lipgloss.JoinHorizontal(
			lipgloss.Center,
			lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(timeWidth+2).Render(m.inputs[endTime].View()),
			" on ",
		)
	}
	startDateInputs := renderDateInputs(m.inputs[startMonth], m.inputs[startDay], m.inputs[startYear])
	endDateInputs := renderDateInputs(m.inputs[endMonth], m.inputs[endDay], m.inputs[endYear])
	return lipgloss.JoinVertical(
		lipgloss.Center,
		title+"\n",
		lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(summaryWidth+2).Render(m.inputs[summary].View()),
		lipgloss.JoinHorizontal(lipgloss.Center, startDateInputs, startTimeInputs, " to ", endTimeInputs, endDateInputs),
		duration,
		m.renderCalendarDrowpdown(),
		"", // TODO the last line is not being centered properly so this is just here for that
	)
}

func (m EditPage) renderCalendarDrowpdown() string {
	style := lipgloss.NewStyle().Padding(0, 1)
	if m.focusIndex == calId {
		return style.Border(lipgloss.DoubleBorder()).Render(m.inputs[calId].View())
	}
	return style.Border(lipgloss.RoundedBorder()).Render(m.inputs[calId].View())
}

// -----------------------------------------------------------------------------
// Messages and Commands
// -----------------------------------------------------------------------------

type editEventRequestMsg struct {
	calendarId string
	eventId    string
	summary    string
	start      time.Time
	end        time.Time
	allDay     bool
}

func editEvent(srv *calendar.Service, msg editEventRequestMsg) tea.Cmd {
	return func() tea.Msg {
		var startDate, startDateTime, endDate, endDateTime string
		if msg.allDay {
			startDate = msg.start.Format(YYYYMMDD)
			endDate = msg.end.Format(YYYYMMDD)
			startDateTime = ""
			endDateTime = ""
		} else {
			startDate = ""
			endDate = ""
			startDateTime = msg.start.Format(time.RFC3339)
			endDateTime = msg.end.Format(time.RFC3339)
		}
		if msg.eventId == "" {
			var startEventDateTime, endEventDatetime *calendar.EventDateTime
			if msg.allDay {
				startEventDateTime = &calendar.EventDateTime{Date: startDate}
				endEventDatetime = &calendar.EventDateTime{Date: endDate}
			} else {
				startEventDateTime = &calendar.EventDateTime{DateTime: startDateTime}
				endEventDatetime = &calendar.EventDateTime{DateTime: endDateTime}
			}
			event := &calendar.Event{
				Summary: msg.summary,
				Start:   startEventDateTime,
				End:     endEventDatetime,
			}
			_, err := srv.Events.Insert(msg.calendarId, event).Do()
			if err != nil {
				return errMsg{err: err}
			}
		} else {
			event, err := srv.Events.Get(msg.calendarId, msg.eventId).Do()
			if err != nil {
				return errMsg{err: err}
			}
			event.Summary = msg.summary
			event.Start.Date = startDate
			event.End.Date = endDate
			event.Start.DateTime = startDateTime
			event.End.DateTime = endDateTime
			_, err = srv.Events.Update(msg.calendarId, msg.eventId, event).Do()
			if err != nil {
				return errMsg{err: err}
			}
		}
		return successMsg{}
	}
}

// -----------------------------------------------------------------------------
// Keys
// -----------------------------------------------------------------------------

type keyMapEdit struct {
	Next         key.Binding
	Prev         key.Binding
	ToggleAllDay key.Binding
	NextCal      key.Binding
	PrevCal      key.Binding
	Save         key.Binding
	Exit         key.Binding
}

var editKeyMap = keyMapEdit{
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev field"),
	),
	ToggleAllDay: key.NewBinding(
		key.WithKeys("ctrl+a"),
		key.WithHelp("ctrl+a", "toggle all day"),
	),
	NextCal: key.NewBinding(
		key.WithKeys("j", "ctrl+n", "down"),
		key.WithHelp("↑/↓", "select calendar"),
	),
	PrevCal: key.NewBinding(
		key.WithKeys("k", "ctrl+p", "up"),
	),
	Save: key.NewBinding(
		key.WithKeys("enter", "ctrl+s"),
		key.WithHelp("enter/ctrl+s", "save"),
	),
	Exit: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "exit"),
	),
}

func (k keyMapEdit) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Prev, k.ToggleAllDay, k.NextCal, k.Save, k.Exit}
}

func (k keyMapEdit) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Next}, {k.Prev}, {k.ToggleAllDay}, {k.NextCal}, {k.Save}, {k.Exit}}
}

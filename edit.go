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
	eventId       string
	start         time.Time
	end           time.Time
	duration      time.Duration
	allDay        bool
	calendars     []*calendar.CalendarListEntry
	calendarIndex int
	success       bool
	pending       bool
	spinner       spinner.Model
	err           error
	keys          keyMapEdit
	help          help.Model
	width, height int
}

func newEditPage(srv *calendar.Service, event *Event, focusedDate time.Time, calendars []*calendar.CalendarListEntry, width, height int) EditPage {
	inputs := make([]textinput.Model, 10)

	inputs[summary] = textinput.New()
	inputs[summary].Width = summaryWidth
	inputs[summary].Prompt = " "
	inputs[summary].Placeholder = "Add title"
	if event != nil {
		inputs[summary].SetValue(event.event.Summary)
	}

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

	var (
		eventId    string
		allDay     bool
		start, end time.Time
	)

	if event == nil {
		eventId = ""
		allDay = false
		start = time.Date(focusedDate.Year(), focusedDate.Month(), focusedDate.Day(), time.Now().Hour(), time.Now().Minute(), 0, 0, time.Local).Truncate(30 * time.Minute).Add(30 * time.Minute)
		end = start.Add(time.Hour)
	} else {
		eventId = event.event.Id
		if isAllDay(*event) {
			allDay = true
			// TODO handle errors
			start, _ = time.Parse(YYYYMMDD, event.event.Start.Date)
			end, _ = time.Parse(YYYYMMDD, event.event.End.Date)
		} else {
			allDay = false
			// TODO handle errors
			start, _ = time.Parse(time.RFC3339, event.event.Start.DateTime)
			end, _ = time.Parse(time.RFC3339, event.event.End.DateTime)
		}
		start = start.In(time.Local)
		end = end.In(time.Local)
	}

	duration := end.Sub(start)

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

	inputs[startMonth].SetValue(startMonthText)
	inputs[startDay].SetValue(startDayText)
	inputs[startYear].SetValue(startYearText)
	inputs[startTime].SetValue(startTimeText)
	inputs[endTime].SetValue(endTimeText)
	inputs[endMonth].SetValue(endMonthText)
	inputs[endDay].SetValue(endDayText)
	inputs[endYear].SetValue(endYearText)

	var calendarIndex int
	if event != nil {
		for i, calendar := range calendars {
			if calendar.Id == event.calendarId {
				calendarIndex = i
			}
		}
	}

	inputs[calId].SetValue(calendars[calendarIndex].Summary + " ⏷")

	focusIndex := summary
	refocus(inputs, focusIndex)

	s := spinner.New()
	s.Spinner = spinner.Points

	return EditPage{
		srv:           srv,
		inputs:        inputs,
		focusIndex:    focusIndex,
		eventId:       eventId,
		start:         start,
		end:           end,
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
			return m, showCalendarViewCmd
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
			var cmd tea.Cmd
			if m.eventId == "" {
				cmd = createEvent(m.srv, editEventRequestMsg)
			} else {
				cmd = editEvent(m.srv, editEventRequestMsg)
			}
			return m, tea.Batch(cmd, m.spinner.Tick)

		case key.Matches(msg, m.keys.Exit):
			return m, showCalendarViewCmd
		}

	case errMsg:
		m.err = msg.err
		m.pending = false
		return m, nil

	case createEventSuccessMsg, editEventSuccessMsg:
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
		// s = "Error. Press any key to return to calendar."
		s = m.err.Error()
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

type (
	createEventSuccessMsg struct{}
	editEventSuccessMsg   struct{}
)

func createEvent(srv *calendar.Service, msg editEventRequestMsg) tea.Cmd {
	return func() tea.Msg {
		var start, end *calendar.EventDateTime
		if msg.allDay {
			start = &calendar.EventDateTime{Date: msg.start.Format(YYYYMMDD)}
			end = &calendar.EventDateTime{Date: msg.end.Format(YYYYMMDD)}
		} else {
			start = &calendar.EventDateTime{DateTime: msg.start.Format(time.RFC3339)}
			end = &calendar.EventDateTime{DateTime: msg.end.Format(time.RFC3339)}
		}
		event := &calendar.Event{
			Summary: msg.summary,
			Start:   start,
			End:     end,
		}
		_, err := srv.Events.Insert(msg.calendarId, event).Do()
		if err != nil {
			return errMsg{err: err}
		}
		return createEventSuccessMsg{}
	}
}

func editEvent(srv *calendar.Service, msg editEventRequestMsg) tea.Cmd {
	return func() tea.Msg {
		event, err := srv.Events.Get(msg.calendarId, msg.eventId).Do()
		if err != nil {
			return errMsg{err: err}
		}
		event.Summary = msg.summary
		if msg.allDay {
			event.Start.Date = msg.start.Format(YYYYMMDD)
			event.End.Date = msg.end.Format(YYYYMMDD)
			event.Start.DateTime = ""
			event.End.DateTime = ""
		} else {
			event.Start.Date = ""
			event.End.Date = ""
			event.Start.DateTime = msg.start.Format(time.RFC3339)
			event.End.DateTime = msg.end.Format(time.RFC3339)
		}
		_, err = srv.Events.Update(msg.calendarId, msg.eventId, event).Do()
		if err != nil {
			return errMsg{err: err}
		}
		return editEventSuccessMsg{}
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
		key.WithHelp("shift+tab", "prev"),
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

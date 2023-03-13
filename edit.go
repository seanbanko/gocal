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
	inputs           []textinput.Model
	focusIndex       int
	calendarId       string
	eventId          string
	autofillDuration time.Duration
	height           int
	width            int
	success          bool
	err              error
	help             help.Model
	keys             keyMapEdit
}

func newEditDialog(event *Event, focusedDate time.Time, width, height int) EditDialog {
	inputs := make([]textinput.Model, 11)

	inputs[summary] = textinput.New()
	inputs[summary].Placeholder = "Add title"
	inputs[summary].Width = summaryWidth
	inputs[summary].Prompt = ""
	inputs[summary].PlaceholderStyle = textInputPlaceholderStyle

	inputs[startMonth] = newMonthTextInput()
	inputs[startDay] = newDayTextInput()
	inputs[startYear] = newYearTextInput()
	inputs[startHour] = newTimeTextInput()
	inputs[startMinute] = newTimeTextInput()
	inputs[endMonth] = newMonthTextInput()
	inputs[endDay] = newDayTextInput()
	inputs[endYear] = newYearTextInput()
	inputs[endHour] = newTimeTextInput()
	inputs[endMinute] = newTimeTextInput()

	var start, end time.Time
	if event == nil {
		start = time.Date(focusedDate.Year(), focusedDate.Month(), focusedDate.Day(), time.Now().Hour(), time.Now().Minute(), 0, 0, time.Local)
		end = start.Add(time.Hour)
	} else {
		// TODO handle errors
		eventStart, _ := time.Parse(time.RFC3339, event.event.Start.DateTime)
		start = eventStart.In(time.Local)
		eventEnd, _ := time.Parse(time.RFC3339, event.event.End.DateTime)
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

	var calendarId, eventId string
	var autofillDuration time.Duration
	if event == nil {
		calendarId = "primary"
		eventId = ""
		autofillDuration = time.Hour
	} else {
		calendarId = event.calendarId
		eventId = event.event.Id
		autofillDuration = end.Sub(start)
	}

	focusIndex := summary
	refocus(inputs, focusIndex)

	return EditDialog{
		inputs:           inputs,
		focusIndex:       focusIndex,
		calendarId:       calendarId,
		eventId:          eventId,
		autofillDuration: autofillDuration,
		height:           height,
		width:            width,
		success:          false,
		err:              nil,
		help:             help.New(),
		keys:             editKeyMap,
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
			prevIndex := m.focusIndex
			m.focusIndex = focusNext(m.inputs, m.focusIndex)
			if prevIndex == startMonth {
				m.inputs[endMonth].SetValue(m.inputs[startMonth].Value())
			} else if prevIndex == startDay {
				m.inputs[endDay].SetValue(m.inputs[startDay].Value())
			} else if prevIndex == startYear {
				m.inputs[endYear].SetValue(m.inputs[startYear].Value())
			} else if prevIndex == startHour || prevIndex == startMinute {
				startTime := fmt.Sprintf("%s:%s", m.inputs[startHour].Value(), m.inputs[startMinute].Value())
				start, err := time.Parse(HHMM24h, startTime)
				if err != nil {
					break
				}
				m.inputs[endHour].SetValue(fmt.Sprintf("%02d", start.Add(m.autofillDuration).Hour()))
				m.inputs[endMinute].SetValue(fmt.Sprintf("%02d", start.Add(m.autofillDuration).Minute()))
			} else if prevIndex == endHour || prevIndex == endMinute {
				startTime := fmt.Sprintf("%s:%s", m.inputs[startHour].Value(), m.inputs[startMinute].Value())
				endTime := fmt.Sprintf("%s:%s", m.inputs[endHour].Value(), m.inputs[endMinute].Value())
				m.autofillDuration = updateAutofillDuration(startTime, endTime)
			}
		case key.Matches(msg, m.keys.Prev):
			prevIndex := m.focusIndex
			m.focusIndex = focusPrev(m.inputs, m.focusIndex)
			if prevIndex == endHour || prevIndex == endMinute {
				startTime := fmt.Sprintf("%s:%s", m.inputs[startHour].Value(), m.inputs[startMinute].Value())
				endTime := fmt.Sprintf("%s:%s", m.inputs[endHour].Value(), m.inputs[endMinute].Value())
				m.autofillDuration = updateAutofillDuration(startTime, endTime)
			}
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

func refocus(inputs []textinput.Model, focusIndex int) {
	for i := range inputs {
		inputs[i].Blur()
	}
	inputs[focusIndex].Focus()
}

func autofillPlaceholders(inputs []textinput.Model) {
	for i := range inputs {
		if len(inputs[i].Value()) == 0 {
			inputs[i].SetValue(inputs[i].Placeholder)
		}
	}
}

func updateAutofillDuration(startTime, endTime string) time.Duration {
	start, err := time.Parse(HHMM24h, startTime)
	if err != nil {
		return time.Hour
	}
	end, err := time.Parse(HHMM24h, endTime)
	if err != nil {
		return time.Hour
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

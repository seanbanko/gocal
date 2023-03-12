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
	startDate
	startTime
	endDate
	endTime
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

func newEditDialog(event *Event, today time.Time, width, height int) EditDialog {
	inputs := make([]textinput.Model, 5)

	inputs[summary] = textinput.New()
	inputs[summary].Placeholder = "Add title"
	inputs[summary].CharLimit = 40
	inputs[summary].Prompt = ""
	inputs[summary].PlaceholderStyle = textInputPlaceholderStyle

	inputs[startDate] = newDateTextInput()
	inputs[startDate].Placeholder = today.Format(AbbreviatedTextDate)

	inputs[startTime] = newTimeTextInput()
	inputs[startTime].Placeholder = today.Format(HHMM24h)

	inputs[endDate] = newDateTextInput()
	inputs[endDate].Placeholder = today.Format(AbbreviatedTextDate)

	inputs[endTime] = newTimeTextInput()
	inputs[endTime].Placeholder = today.Format(HHMM24h)

	var calendarId, eventId string
	if event != nil {
		calendarId = event.calendarId
		eventId = event.event.Id
		inputs[summary].SetValue(event.event.Summary)
		start, err := time.Parse(time.RFC3339, event.event.Start.DateTime)
		var startD, startT string
		if err == nil {
			startD = start.In(time.Local).Format(AbbreviatedTextDate)
			startT = start.In(time.Local).Format(HHMM24h)
		}
		end, err := time.Parse(time.RFC3339, event.event.End.DateTime)
		var endD, endT string
		if err == nil {
			endD = end.In(time.Local).Format(AbbreviatedTextDate)
			endT = end.In(time.Local).Format(HHMM24h)
		}
		inputs[startDate].SetValue(startD)
		inputs[startTime].SetValue(startT)
		inputs[endDate].SetValue(endD)
		inputs[endTime].SetValue(endT)
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
		keys:       EditKeyMap,
	}
}

func newDateTextInput() textinput.Model {
	input := textinput.New()
	input.CharLimit = 11
	input.PlaceholderStyle = textInputPlaceholderStyle
	input.Prompt = ""
	return input
}

func newTimeTextInput() textinput.Model {
	input := textinput.New()
	input.CharLimit = 5
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
		// Prevents further updates after creating one event
		if m.success {
			return m, showCalendarViewCmd
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, showCalendarViewCmd
		case "enter", "ctrl+s":
			return m, editEventRequestCmd(
				m.calendarId,
				m.eventId,
				m.inputs[summary].Value(),
				m.inputs[startDate].Value(),
				m.inputs[startTime].Value(),
				m.inputs[endDate].Value(),
				m.inputs[endTime].Value(),
			)
		case "tab":
			m.focusNext()
		case "shift+tab":
			m.focusPrev()
		}
	}
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m *EditDialog) focusNext() {
	if len(m.inputs[m.focusIndex].Value()) == 0 {
		m.inputs[m.focusIndex].SetValue(m.inputs[m.focusIndex].Placeholder)
	}
	m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
	refocus(m.inputs, m.focusIndex)
}

func (m *EditDialog) focusPrev() {
	if len(m.inputs[m.focusIndex].Value()) == 0 {
		m.inputs[m.focusIndex].SetValue(m.inputs[m.focusIndex].Placeholder)
	}
	m.focusIndex--
	if m.focusIndex < 0 {
		m.focusIndex = len(m.inputs) - 1
	}
	refocus(m.inputs, m.focusIndex)
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
		Render(dialogStyle.Render(content))
	return lipgloss.JoinVertical(lipgloss.Center, container, helpView)
}

func renderEditContent(m EditDialog) string {
	return lipgloss.JoinVertical(
		lipgloss.Center,
		"Edit Event\n",
		textInputSummaryStyle.Render(m.inputs[summary].View())+"\n",
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			textInputDateStyle.Render(m.inputs[startDate].View()),
			"at ",
			textInputTimeStyle.Render(m.inputs[startTime].View()),
			"to ",
			textInputDateStyle.Render(m.inputs[endDate].View()),
			"at ",
			textInputTimeStyle.Render(m.inputs[endTime].View()),
		),
	)
}

type keyMapEdit struct {
	Next   key.Binding
	Prev   key.Binding
	Save   key.Binding
	Cancel key.Binding
	Quit   key.Binding
}

var EditKeyMap = keyMapEdit{
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
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
}

func (k keyMapEdit) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Prev, k.Save, k.Cancel, k.Quit}
}

func (k keyMapEdit) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next, k.Cancel},
		{k.Prev, k.Quit},
		{k.Save},
	}
}

package main

import (
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	title = iota
	startDate
	startTime
	endDate
	endTime
)

var (
	textInputPlaceholderStyle = lipgloss.NewStyle().Faint(true)
	textInputTextStyle        = lipgloss.NewStyle().AlignHorizontal(lipgloss.Center)
)

type CreateEventPopup struct {
	inputs          []textinput.Model
	focusIndex      int
	calendarService *calendar.Service
	height          int
	width           int
	err             error
}

func newPopup(srv *calendar.Service, height, width int) CreateEventPopup {
	inputs := make([]textinput.Model, 5)

	inputs[title] = textinput.New()
	inputs[title].Placeholder = "Title"
	inputs[title].Width = 40 // TODO make not arbitrary
	inputs[title].Prompt = ""
	inputs[title].PlaceholderStyle = textInputPlaceholderStyle
	inputs[title].Focus()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	inputs[startDate] = textinput.New()
	inputs[startDate].Placeholder = today.Format(MMDDYYYY)
	inputs[startDate].CharLimit = 10
	inputs[startDate].Prompt = ""
	inputs[startDate].PlaceholderStyle = textInputPlaceholderStyle

	inputs[startTime] = textinput.New()
	inputs[startTime].Placeholder = today.Format(HHMM24h)
	inputs[startTime].CharLimit = 5
	inputs[startTime].Prompt = ""
	inputs[startTime].PlaceholderStyle = textInputPlaceholderStyle

	inputs[endDate] = textinput.New()
	inputs[endDate].Placeholder = today.Format(MMDDYYYY)
	inputs[endDate].CharLimit = 10
	inputs[endDate].Prompt = ""
	inputs[endDate].PlaceholderStyle = textInputPlaceholderStyle

	inputs[endTime] = textinput.New()
	inputs[endTime].Placeholder = today.Format(HHMM24h)
	inputs[endTime].CharLimit = 5
	inputs[endTime].Width = 5
	inputs[endTime].Prompt = ""
	inputs[endTime].PlaceholderStyle = textInputPlaceholderStyle

	return CreateEventPopup{
		calendarService: srv,
		inputs:          inputs,
		focusIndex:      title,
		height:          height,
		width:           width,
	}
}

func (m CreateEventPopup) Init() tea.Cmd {
	return textinput.Blink
}

type enterCreatePopupMsg struct{}

func enterCreatePopupCmd() tea.Msg {
	return enterCreatePopupMsg{}
}

type exitCreatePopupMsg struct{}

func exitCreatePopupCmd() tea.Msg {
	return exitCreatePopupMsg{}
}

func (m CreateEventPopup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		return m, nil
	case createEventMsg:
		if msg.err != nil {
			m.err = msg.err
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, exitCreatePopupCmd
		case "enter", "ctrl+s":
			title := m.inputs[title].Value()
			startDate := m.inputs[startDate].Value()
			startTime := m.inputs[startTime].Value()
			endDate := m.inputs[endDate].Value()
			endTime := m.inputs[endTime].Value()
			cmd := createEventCmd(m.calendarService, title, startDate, startTime, endDate, endTime)
			return m, cmd
		case "tab", "ctrl+n":
			m.focusNext()
		case "shift+tab", "ctrl+p":
			m.focusPrev()
		}
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		m.inputs[m.focusIndex].Focus()
	}
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m *CreateEventPopup) focusNext() {
	m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
}

func (m *CreateEventPopup) focusPrev() {
	m.focusIndex--
	if m.focusIndex < 0 {
		m.focusIndex = len(m.inputs) - 1
	}
}

func (m CreateEventPopup) View() string {
    popupStyle := lipgloss.NewStyle().
        Width(m.width / 2).
        Height(m.height / 2).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Border(lipgloss.NormalBorder())
	if m.err != nil {
		s := "Error creating event. Press esc to return to calendar."
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popupStyle.Render(s))
	}
	titleStyle := lipgloss.NewStyle().AlignHorizontal(lipgloss.Center)
	dateStyle := lipgloss.NewStyle().Width(11)
	timeStyle := lipgloss.NewStyle().Width(6)
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"Create Event",
		"\n",
		titleStyle.Render(m.inputs[title].View()),
		"\n",
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			dateStyle.Render(m.inputs[startDate].View()),
			"at ",
			timeStyle.Render(m.inputs[startTime].View()),
			"to ",
			dateStyle.Render(m.inputs[endDate].View()),
			"at ",
			timeStyle.Render(m.inputs[endTime].View()),
			".", // TODO This is just here to the overall width doesn't change
		),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popupStyle.Render(content))
}

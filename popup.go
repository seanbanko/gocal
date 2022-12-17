package main

import (
	"time"

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

type CreateEventPopup struct {
	inputs     []textinput.Model
	focusIndex int
	height     int
	width      int
}

var (
	textInputPlaceholderStyle = lipgloss.NewStyle().Faint(true)
	textInputTextStyle        = lipgloss.NewStyle().AlignHorizontal(lipgloss.Center)
)

func newPopup() CreateEventPopup {
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
		inputs:     inputs,
		focusIndex: title,
	}
}

func updateCreateEventPopup(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.creatingEvent = false
			return m, nil
		case "enter", "ctrl+s":
			// TODO this should return a CreateEventCmd of some kind
			m.creatingEvent = false
			return m, nil
		case "tab", "ctrl+n":
			m.createEventPopup.focusNext()
		case "shift+tab", "ctrl+p":
			m.createEventPopup.focusPrev()
		}
		for i := range m.createEventPopup.inputs {
			m.createEventPopup.inputs[i].Blur()
		}
		m.createEventPopup.inputs[m.createEventPopup.focusIndex].Focus()
	}
	cmds := make([]tea.Cmd, len(m.createEventPopup.inputs))
	for i := range m.createEventPopup.inputs {
		m.createEventPopup.inputs[i], cmds[i] = m.createEventPopup.inputs[i].Update(msg)
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

func viewPopup(m model) string {
	popupStyle := lipgloss.NewStyle().
		Width(m.width / 2).
		Height(m.height / 2).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Border(lipgloss.NormalBorder())
	titleStyle := lipgloss.NewStyle().AlignHorizontal(lipgloss.Center)
	dateStyle := lipgloss.NewStyle().Width(11)
	timeStyle := lipgloss.NewStyle().Width(6)
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"Create Event",
		"\n",
		titleStyle.Render(m.createEventPopup.inputs[title].View()),
		"\n",
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			dateStyle.Render(m.createEventPopup.inputs[startDate].View()),
			"at ",
			timeStyle.Render(m.createEventPopup.inputs[startTime].View()),
			"to ",
			dateStyle.Render(m.createEventPopup.inputs[endDate].View()),
			"at ",
			timeStyle.Render(m.createEventPopup.inputs[endTime].View()),
			".", // TODO This is just here to the overall width doesn't change
		),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popupStyle.Render(content))
}

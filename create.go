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
	title = iota
	startDate
	startTime
	endDate
	endTime
)

type CreateDialog struct {
	inputs     []textinput.Model
	focusIndex int
	height     int
	width      int
	success    bool
	err        error
	help       help.Model
	keys       keyMapCreate
}

func newCreateDialog(width, height int) CreateDialog {
	inputs := make([]textinput.Model, 5)

	inputs[title] = textinput.New()
	inputs[title].Placeholder = "Title"
	inputs[title].Width = 40 // TODO make not arbitrary
	inputs[title].Prompt = ""
	inputs[title].PlaceholderStyle = textInputPlaceholderStyle

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	inputs[startDate] = textinput.New()
	inputs[startDate].Placeholder = today.Format(AbbreviatedTextDate)
	inputs[startDate].CharLimit = 11
	inputs[startDate].Prompt = ""
	inputs[startDate].PlaceholderStyle = textInputPlaceholderStyle

	inputs[startTime] = textinput.New()
	inputs[startTime].Placeholder = today.Format(HHMM24h)
	inputs[startTime].CharLimit = 5
	inputs[startTime].Prompt = ""
	inputs[startTime].PlaceholderStyle = textInputPlaceholderStyle

	inputs[endDate] = textinput.New()
	inputs[endDate].Placeholder = today.Format(AbbreviatedTextDate)
	inputs[endDate].CharLimit = 11
	inputs[endDate].Prompt = ""
	inputs[endDate].PlaceholderStyle = textInputPlaceholderStyle

	inputs[endTime] = textinput.New()
	inputs[endTime].Placeholder = today.Format(HHMM24h)
	inputs[endTime].CharLimit = 5
	inputs[endTime].Width = 5
	inputs[endTime].Prompt = ""
	inputs[endTime].PlaceholderStyle = textInputPlaceholderStyle

	focusIndex := title
	inputs[focusIndex].Focus()

	return CreateDialog{
		inputs:     inputs,
		focusIndex: focusIndex,
		height:     height,
		width:      width,
		success:    false,
		err:        nil,
		help:       help.New(),
		keys:       CreateKeyMap,
	}
}

func (m CreateDialog) Init() tea.Cmd {
	return textinput.Blink
}

func (m CreateDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		return m, nil
	case createEventResponseMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.success = true
		}
		return m, nil
	case tea.KeyMsg:
		// Prevents further updates after creating one event
		if m.success {
			return m, exitCreateDialogCmd
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, exitCreateDialogCmd
		case "enter", "ctrl+s":
			return m, createEventRequestCmd(
				"primary",
				m.inputs[title].Value(),
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

func (m *CreateDialog) focusNext() {
	if len(m.inputs[m.focusIndex].Value()) == 0 {
		m.inputs[m.focusIndex].SetValue(m.inputs[m.focusIndex].Placeholder)
	}
	m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
    refocus(m.inputs, m.focusIndex)
}

func (m *CreateDialog) focusPrev() {
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

func (m CreateDialog) View() string {
	var content string
	if m.err != nil {
		content = "Error creating event. Press any key to return to calendar."
	} else if m.success {
		content = "Successfully created event. Press any key to return to calendar."
	} else {
		content = renderCreateContent(m)
	}
	helpView := lipgloss.NewStyle().
        Width(m.width).
        Padding(1).
        AlignHorizontal(lipgloss.Center).
        Render(m.help.View(m.keys))
	container := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - lipgloss.Height(helpView)).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(dialogStyle.Render(content))
	return lipgloss.JoinVertical(lipgloss.Center, container, helpView)
}

func renderCreateContent(m CreateDialog) string {
	return lipgloss.JoinVertical(
		lipgloss.Center,
		"Create Event",
		"\n",
		titleStyle.Render(m.inputs[title].View()),
		"\n",
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			dateStyle.Render(m.inputs[startDate].View()),
			" at ",
			timeStyle.Render(m.inputs[startTime].View()),
			" to ",
			dateStyle.Render(m.inputs[endDate].View()),
			" at ",
			timeStyle.Render(m.inputs[endTime].View()),
			".", // TODO This is just here to the overall width doesn't change
		),
	)
}

type keyMapCreate struct {
	Next   key.Binding
	Prev   key.Binding
	Create key.Binding
	Cancel key.Binding
	Quit   key.Binding
}

var CreateKeyMap = keyMapCreate{
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous field"),
	),
	Create: key.NewBinding(
		key.WithKeys("enter", "ctrl+s"),
		key.WithHelp("enter/ctrl+s", "create event"),
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

func (k keyMapCreate) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Prev, k.Create, k.Cancel, k.Quit}
}

func (k keyMapCreate) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next, k.Cancel},
		{k.Prev, k.Quit},
		{k.Create},
	}
}

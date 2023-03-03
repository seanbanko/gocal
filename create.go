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

var createPopupStyle = lipgloss.NewStyle().
    Padding(1).
    AlignHorizontal(lipgloss.Center).
    AlignVertical(lipgloss.Center).
    Border(lipgloss.RoundedBorder())

type CreateEventPopup struct {
	inputs     []textinput.Model
	focusIndex int
	height     int
	width      int
	success    bool
	err        error
	help       help.Model
	keys       keyMapCreate
}

func newCreatePopup(width, height int) CreateEventPopup {
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

	return CreateEventPopup{
		inputs:     inputs,
		focusIndex: title,
		height:     height,
		width:      width,
		success:    false,
		err:        nil,
		help:       help.New(),
		keys:       CreateKeyMap,
	}
}

func (m CreateEventPopup) Init() tea.Cmd {
	return textinput.Blink
}

func (m CreateEventPopup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		// Prevent further updates after creating one event
		if m.success == true {
			return m, exitCreatePopupCmd
		}
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
			cmd := createEventRequestCmd("primary", title, startDate, startTime, endDate, endTime)
			return m, cmd
		case "tab":
			m.focusNext()
		case "shift+tab":
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
    if len(m.inputs[m.focusIndex].Value()) == 0 {
        m.inputs[m.focusIndex].SetValue(m.inputs[m.focusIndex].Placeholder)
    }
	m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
}

func (m *CreateEventPopup) focusPrev() {
    if len(m.inputs[m.focusIndex].Value()) == 0 {
        m.inputs[m.focusIndex].SetValue(m.inputs[m.focusIndex].Placeholder)
    }
	m.focusIndex--
	if m.focusIndex < 0 {
		m.focusIndex = len(m.inputs) - 1
	}
}

func (m CreateEventPopup) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	var content string
	if m.err != nil {
		content = "Error creating event. Press any key to return to calendar."
	} else if m.success {
		content = "Successfully created event. Press any key to return to calendar."
	} else {
		content = renderForm(m)
	}
	help := renderHelpCreate(m.help, m.keys, m.width)
	popupContainer := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - lipgloss.Height(help)).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(createPopupStyle.Render(content))
	return lipgloss.JoinVertical(lipgloss.Center, popupContainer, help)
}

func renderForm(m CreateEventPopup) string {
	return lipgloss.JoinVertical(
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
}

func renderHelpCreate(help help.Model, keys keyMapCreate, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(help.View(keys))
}

// Help

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

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/calendar/v3"
)

const (
	calendarView = iota
	deleteDialog
	gotoDateDialog
	editPage
	calendarList
)

type Event struct {
	calendarId string
	event      *calendar.Event
}

// Implement list.Item interface
func (event Event) FilterValue() string { return event.event.Summary }
func (event Event) Title() string       { return event.event.Summary }
func (event Event) Description() string {
	if isAllDay(event.event) {
		return "all day"
	}
	start, err := time.Parse(time.RFC3339, event.event.Start.DateTime)
	if err != nil {
		return err.Error()
	}
	s := start.In(time.Local).Format(time.Kitchen)
	end, err := time.Parse(time.RFC3339, event.event.End.DateTime)
	if err != nil {
		return err.Error()
	}
	e := end.In(time.Local).Format(time.Kitchen)
	return fmt.Sprintf("%v - %v", s, e)
}

type model struct {
	srv           *calendar.Service
	cache         *cache.Cache
	currentDate   time.Time
	focusedDate   time.Time
	events        []*Event
	calendars     []*calendar.CalendarListEntry
	focusedModel  int
	eventsList    list.Model
	gotoDialog    tea.Model
	editPage      tea.Model
	deleteDialog  tea.Model
	calendarList  tea.Model
	keys          keyMapDefault
	help          help.Model
	width, height int
}

func newModel(service *calendar.Service, cache *cache.Cache) model {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle.Foreground(googleBlue)
	delegate.Styles.SelectedTitle.BorderForeground(googleBlue)
	delegate.Styles.SelectedDesc.BorderForeground(googleBlue)
	delegate.Styles.SelectedDesc.Foreground(googleBlue)
	eventsList := list.New(nil, delegate, 0, 0)
	eventsList.SetShowStatusBar(false)
	eventsList.SetStatusBarItemName("event", "events")
	eventsList.SetShowHelp(false)
	eventsList.DisableQuitKeybindings()
	eventsList.Title = today.Format(AbbreviatedTextDateWithWeekday)
	eventsList.Styles.Title.Background(googleBlue)
	return model{
		srv:          service,
		cache:        cache,
		currentDate:  today,
		focusedDate:  today,
		focusedModel: calendarView,
		eventsList:   eventsList,
		keys:         defaultKeyMap,
		help:         help.New(),
	}
}

func (m model) Init() tea.Cmd {
	return calendarListCmd(m.srv)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("%T %v", msg, msg)
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.Width = m.width
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case errMsg:
		// TODO make sure this doesn't require further action
		// Currently the assumption is that sub-models handle and display errors
		log.Printf("Error: %v", msg.err)
	case showCalendarMsg:
		m.focusedModel = calendarView
		return m, tea.Batch(tea.ClearScreen, refreshEventsCmd)
	case calendarListMsg:
		m.calendars = msg.calendars
		if m.focusedModel == calendarList {
			m.calendarList, cmd = m.calendarList.Update(msg)
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, refreshEventsCmd)
		return m, tea.Batch(cmds...)
	case eventsMsg:
		m.events = msg.events
		var items []list.Item
		for _, event := range msg.events {
			items = append(items, event)
		}
		m.eventsList.SetItems(items)
		return m, nil
	case gotoDateMsg:
		m.focusedDate = msg.date
		m.eventsList.Title = m.focusedDate.Format(AbbreviatedTextDateWithWeekday)
		return m, refreshEventsCmd
	case refreshEventsMsg:
		var calendars []*calendar.CalendarListEntry
		for _, calendar := range m.calendars {
			if !calendar.Selected {
				continue
			}
			calendars = append(calendars, calendar)
		}
		return m, eventsListCmd(m.srv, m.cache, calendars, m.focusedDate)
	case editEventRequestMsg:
		return m, editEventResponseCmd(m.srv, msg)
	case deleteEventRequestMsg:
		return m, deleteEventResponseCmd(m.srv, msg)
	case updateCalendarRequestMsg:
		return m, updateCalendarResponseCmd(m.srv, msg)
	case successMsg:
		switch m.focusedModel {
		case editPage, deleteDialog:
			m.cache.Flush()
		case calendarList:
			return m, calendarListCmd(m.srv)
		}
	}
	switch m.focusedModel {
	case calendarView:
		if m.eventsList.SettingFilter() {
			var cmd tea.Cmd
			m.eventsList, cmd = m.eventsList.Update(msg)
			return m, cmd
		}
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keys.Next):
				return m, gotoDateCmd(m.focusedDate.AddDate(0, 0, 1))
			case key.Matches(msg, m.keys.Prev):
				return m, gotoDateCmd(m.focusedDate.AddDate(0, 0, -1))
			case key.Matches(msg, m.keys.Today):
				return m, gotoDateCmd(m.currentDate)
			case key.Matches(msg, m.keys.GotoDate):
				m.focusedModel = gotoDateDialog
				m.gotoDialog = newGotoDialog(m.focusedDate, m.width, m.height)
				return m, nil
			case key.Matches(msg, m.keys.Create):
                if len(m.calendars) <= 0 {
                    return m, nil
                }
				m.focusedModel = editPage
                var modifiableCalendars []*calendar.CalendarListEntry
                for _, calendar := range m.calendars {
                    if (calendar.AccessRole == "writer" || calendar.AccessRole == "owner") {
                        modifiableCalendars = append(modifiableCalendars, calendar)
                    }
                }
				m.editPage = newEditPage(nil, m.focusedDate, modifiableCalendars, m.width, m.height)
				return m, nil
			case key.Matches(msg, m.keys.Edit):
                if len(m.calendars) <= 0 {
                    return m, nil
                }
				event, ok := m.eventsList.SelectedItem().(*Event)
				if !ok {
					return m, func() tea.Msg { return errMsg{err: fmt.Errorf("Type assertion failed")} }
				}
				m.focusedModel = editPage
				m.editPage = newEditPage(event, m.focusedDate, m.calendars, m.width, m.height)
				return m, nil
			case key.Matches(msg, m.keys.Delete):
				event, ok := m.eventsList.SelectedItem().(*Event)
				if !ok {
					return m, func() tea.Msg { return errMsg{err: fmt.Errorf("Type assertion failed")} }
				}
				m.focusedModel = deleteDialog
				m.deleteDialog = newDeleteDialog(event.calendarId, event.event.Id, m.width, m.height)
				return m, nil
			case key.Matches(msg, m.keys.CalendarList):
				m.focusedModel = calendarList
				m.calendarList = newCalendarListDialog(m.calendars, m.width, m.height)
				return m, nil
			case key.Matches(msg, m.keys.Help):
				m.help.ShowAll = !m.help.ShowAll
				return m, nil
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			default:
				var cmd tea.Cmd
				m.eventsList, cmd = m.eventsList.Update(msg)
				return m, cmd
			}
		}
	case gotoDateDialog:
		m.gotoDialog, cmd = m.gotoDialog.Update(msg)
		return m, cmd
	case editPage:
		m.editPage, cmd = m.editPage.Update(msg)
		return m, cmd
	case deleteDialog:
		m.deleteDialog, cmd = m.deleteDialog.Update(msg)
		return m, cmd
	case calendarList:
		m.calendarList, cmd = m.calendarList.Update(msg)
		return m, cmd
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	titleContainer := lipgloss.NewStyle().
		Width(m.width - 2).
		MaxWidth(m.width - 2).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render("GoCal")
	var body string
	switch m.focusedModel {
	case calendarView:
		helpContainer := lipgloss.NewStyle().
			Width(m.width).
			Padding(1).
			AlignHorizontal(lipgloss.Center).
			Render(m.help.View(m.keys))
		m.eventsList.SetSize(m.width, m.height-lipgloss.Height(titleContainer)-lipgloss.Height(helpContainer))
		eventsView := lipgloss.PlaceHorizontal(m.width, lipgloss.Left, m.eventsList.View())
		body = lipgloss.JoinVertical(lipgloss.Center, eventsView, helpContainer)
	case gotoDateDialog:
		body = m.gotoDialog.View()
	case editPage:
		body = m.editPage.View()
	case deleteDialog:
		body = m.deleteDialog.View()
	case calendarList:
		body = m.calendarList.View()
	}
	bodyContainer := lipgloss.NewStyle().
		MaxWidth(m.width).
		MaxHeight(m.height - lipgloss.Height(titleContainer)).
		Render(body)
	return lipgloss.JoinVertical(lipgloss.Center, titleContainer, bodyContainer)
}

type keyMapDefault struct {
	Next         key.Binding
	Prev         key.Binding
	Today        key.Binding
	GotoDate     key.Binding
	Create       key.Binding
	Edit         key.Binding
	Delete       key.Binding
	CalendarList key.Binding
	Help         key.Binding
	Quit         key.Binding
}

var defaultKeyMap = keyMapDefault{
	Next: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next period"),
	),
	Prev: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "previous period"),
	),
	Today: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "today"),
	),
	GotoDate: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "go to date"),
	),
	Create: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "create event"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit event"),
	),
	Delete: key.NewBinding(
		key.WithKeys("backspace", "delete"),
		key.WithHelp("del", "delete event"),
	),
	CalendarList: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "show calendar list"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
	),
}

func (k keyMapDefault) ShortHelp() []key.Binding {
	return []key.Binding{k.Help}
}

func (k keyMapDefault) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next, k.Delete},
		{k.Prev, k.GotoDate},
		{k.Today, k.CalendarList},
		{k.GotoDate, k.Help},
		{k.Create, k.Quit},
		{k.Edit},
	}
}

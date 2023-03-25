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

type viewType int

const (
	dayView viewType = iota
	weekView
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
	calendars     []*calendar.CalendarListEntry
	focusedModel  int
	viewType      viewType
	dayLists      []list.Model
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
	return model{
		srv:          service,
		cache:        cache,
		currentDate:  today,
		focusedDate:  today,
		focusedModel: calendarView,
		dayLists:     newWeekLists(today),
		keys:         defaultKeyMap,
		help:         help.New(),
	}
}

func newFocusedDateDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle.Foreground(googleBlue)
	delegate.Styles.SelectedTitle.BorderForeground(googleBlue)
	delegate.Styles.SelectedDesc.Foreground(googleBlue)
	delegate.Styles.SelectedDesc.BorderForeground(googleBlue)
	return delegate
}

func newUnfocusedDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle.Foreground(delegate.Styles.NormalTitle.GetForeground())
	delegate.Styles.SelectedTitle.BorderForeground(delegate.Styles.NormalTitle.GetBorderLeftForeground())
	delegate.Styles.SelectedDesc.Foreground(delegate.Styles.NormalDesc.GetForeground())
	delegate.Styles.SelectedDesc.BorderForeground(delegate.Styles.NormalDesc.GetBorderLeftForeground())
	return delegate
}

func newDayList(date time.Time) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle.Foreground(googleBlue)
	delegate.Styles.SelectedTitle.BorderForeground(googleBlue)
	delegate.Styles.SelectedDesc.BorderForeground(googleBlue)
	delegate.Styles.SelectedDesc.Foreground(googleBlue)
	dayList := list.New(nil, delegate, 0, 0)
	dayList.SetShowStatusBar(false)
	dayList.SetStatusBarItemName("event", "events")
	dayList.SetShowHelp(false)
	dayList.DisableQuitKeybindings()
	dayList.Title = date.Format(AbbreviatedTextDateWithWeekday)
	dayList.Styles.Title.UnsetForeground()
	dayList.Styles.Title.UnsetBackground()
	return dayList
}

func newWeekLists(focusedDate time.Time) []list.Model {
	lists := make([]list.Model, 7)
	startOfWeek := focusedDate.AddDate(0, 0, -1*int(focusedDate.Weekday()))
	for i := range lists {
		lists[i] = newDayList(startOfWeek.AddDate(0, 0, i))
	}
	return lists
}

func (m model) Init() tea.Cmd {
	return tea.Batch(calendarListCmd(m.srv), m.dayLists[m.focusedDate.Weekday()].StartSpinner())
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
		return m, nil
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
		m.dayLists[msg.date.Weekday()].StopSpinner()
		setEventsListItems(&m.dayLists[msg.date.Weekday()], msg.events)
		return m, nil
	case gotoDateMsg:
		m.dayLists[msg.date.Weekday()].Select(m.dayLists[msg.date.Weekday()].Index())
		m.dayLists[m.focusedDate.Weekday()].ResetSelected()
		log.Printf("focused date: %v, msg date: %v", m.focusedDate, msg.date)
		switch m.viewType {
		case dayView:
			m.dayLists[msg.date.Weekday()].Title = msg.date.Format(AbbreviatedTextDateWithWeekday)
			m.focusedDate = msg.date
			return m, refreshEventsCmd
		case weekView:
			if areInDifferentWeeks(m.focusedDate, msg.date) {
				log.Printf("decided different weeks")
				startOfWeek := msg.date.AddDate(0, 0, -1*int(msg.date.Weekday()))
				for i := range m.dayLists {
					m.dayLists[i].Title = startOfWeek.AddDate(0, 0, i).Format(AbbreviatedTextDateWithWeekday)
					cmds = append(cmds, eventsListCmd(m.srv, m.cache, selectedCalendars(m), startOfWeek.AddDate(0, 0, i)))
					cmds = append(cmds, m.dayLists[i].StartSpinner())
				}
			}
			m.focusedDate = msg.date
			return m, tea.Batch(cmds...)
		}
	case refreshEventsMsg:
		cmds = append(cmds, eventsListCmd(m.srv, m.cache, selectedCalendars(m), m.focusedDate))
		cmds = append(cmds, m.dayLists[m.focusedDate.Weekday()].StartSpinner())
		return m, tea.Batch(cmds...)
	case flushCacheMsg:
		m.cache.Flush()
		return m, nil
	case editEventRequestMsg:
		return m, editEventResponseCmd(m.srv, msg)
	case deleteEventRequestMsg:
		return m, deleteEventResponseCmd(m.srv, msg)
	case updateCalendarRequestMsg:
		return m, updateCalendarResponseCmd(m.srv, msg)
	case successMsg:
		if m.focusedModel == calendarList {
			return m, calendarListCmd(m.srv)
		}
	}
	switch m.focusedModel {
	case calendarView:
		m, cmd = m.updateCalendarView(msg)
		return m, cmd
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

func areInDifferentWeeks(a, b time.Time) bool {
	firstDayOfWeek := a.AddDate(0, 0, -1*int(a.Weekday()))
	lastDayOfWeek := firstDayOfWeek.AddDate(0, 0, 6)
	log.Printf("%v %v %v %v", a, b, firstDayOfWeek, lastDayOfWeek)
	return b.After(lastDayOfWeek) || b.Before(firstDayOfWeek)
}

func (m model) updateCalendarView(msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
	if m.dayLists[m.focusedDate.Weekday()].SettingFilter() {
		m.dayLists[m.focusedDate.Weekday()], cmd = m.dayLists[m.focusedDate.Weekday()].Update(msg)
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
				if calendar.AccessRole == "writer" || calendar.AccessRole == "owner" {
					modifiableCalendars = append(modifiableCalendars, calendar)
				}
			}
			m.editPage = newEditPage(nil, m.focusedDate, modifiableCalendars, m.width, m.height)
			return m, nil
		case key.Matches(msg, m.keys.Edit):
			if len(m.calendars) <= 0 {
				return m, nil
			}
			event, ok := m.dayLists[m.focusedDate.Weekday()].SelectedItem().(*Event)
			if !ok {
				return m, func() tea.Msg { return errMsg{err: fmt.Errorf("Type assertion failed")} }
			}
			m.focusedModel = editPage
			m.editPage = newEditPage(event, m.focusedDate, m.calendars, m.width, m.height)
			return m, nil
		case key.Matches(msg, m.keys.Delete):
			event, ok := m.dayLists[m.focusedDate.Weekday()].SelectedItem().(*Event)
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
		case key.Matches(msg, m.keys.DayView):
			m.viewType = dayView
			return m, refreshEventsCmd
		case key.Matches(msg, m.keys.WeekView):
			m.viewType = weekView
			var cmds []tea.Cmd
			startOfWeek := m.focusedDate.AddDate(0, 0, -1*int(m.focusedDate.Weekday()))
			for i := 0; i < 7; i++ {
				date := startOfWeek.AddDate(0, 0, i)
				m.dayLists[date.Weekday()].Title = date.Format(AbbreviatedTextDateWithWeekday)
				cmds = append(cmds, eventsListCmd(m.srv, m.cache, selectedCalendars(m), startOfWeek.AddDate(0, 0, i)))
				cmds = append(cmds, m.dayLists[date.Weekday()].StartSpinner())
			}
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}
	m.dayLists[m.focusedDate.Weekday()], cmd = m.dayLists[m.focusedDate.Weekday()].Update(msg)
	return m, cmd
}

func selectedCalendars(m model) []*calendar.CalendarListEntry {
	var selected []*calendar.CalendarListEntry
	for _, calendar := range m.calendars {
		if !calendar.Selected {
			continue
		}
		selected = append(selected, calendar)
	}
	return selected
}

func setEventsListItems(l *list.Model, events []*Event) {
	var items []list.Item
	for _, event := range events {
		items = append(items, event)
	}
	l.SetItems(items)
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
		var calendar string
		width, height := m.width-2, m.height-lipgloss.Height(titleContainer)-lipgloss.Height(helpContainer)
		switch m.viewType {
		case dayView:
			m.dayLists[m.focusedDate.Weekday()].SetSize(width, height)
			calendar = lipgloss.PlaceHorizontal(width, lipgloss.Left, m.dayLists[m.focusedDate.Weekday()].View())
		case weekView:
			calendar = m.viewWeek(width, height)
		}
		body = lipgloss.JoinVertical(lipgloss.Center, lipgloss.NewStyle().Padding(0, 1).Render(calendar), helpContainer)
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

func (m *model) viewWeek(width, height int) string {
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(width / 8)
	var dayViews []string
	startOfWeek := m.focusedDate.AddDate(0, 0, -1*int(m.focusedDate.Weekday()))
	for i := 0; i < 7; i++ {
		date := startOfWeek.AddDate(0, 0, i)
		m.dayLists[date.Weekday()].SetSize(width/8, height)
		if date.Equal(m.focusedDate) {
			m.dayLists[date.Weekday()].SetDelegate(newFocusedDateDelegate())
			style = style.BorderForeground(googleBlue)
		} else {
			m.dayLists[date.Weekday()].SetDelegate(newUnfocusedDelegate())
			style = style.UnsetBorderForeground()
		}
        if date.Equal(m.currentDate) {
            m.dayLists[date.Weekday()].Styles.Title.Background(googleBlue)
        } else {
            m.dayLists[date.Weekday()].Styles.Title.UnsetBackground()
        }
		dayViews = append(dayViews, style.Render(m.dayLists[date.Weekday()].View()))
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, lipgloss.JoinHorizontal(lipgloss.Top, dayViews...))
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
	DayView      key.Binding
	WeekView     key.Binding
	Help         key.Binding
	Quit         key.Binding
}

var defaultKeyMap = keyMapDefault{
	Next: key.NewBinding(
		key.WithKeys("n", "l"),
		key.WithHelp("n", "next day"),
	),
	Prev: key.NewBinding(
		key.WithKeys("p", "h"),
		key.WithHelp("p", "previous day"),
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
	DayView: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "day view"),
	),
	WeekView: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "week view"),
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

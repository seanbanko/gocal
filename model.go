package main

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/calendar/v3"
)

type state int

const (
	initializing = iota
	ready
	editing
	deleting
	gotodate
	calendarList
)

type calendarPeriod int

const (
	dayView calendarPeriod = iota
	weekView
)

type model struct {
	srv           *calendar.Service
	cache         *cache.Cache
	currentDate   time.Time
	focusedDate   time.Time
	calendars     []*calendar.CalendarListEntry
	state         state
	viewType      calendarPeriod
	dayLists      []list.Model
	gotoDialog    tea.Model
	editPage      tea.Model
	deleteDialog  tea.Model
	calendarList  tea.Model
	spinner       spinner.Model
	keys          keyMapDefault
	help          help.Model
	width, height int
}

func newModel(service *calendar.Service, cache *cache.Cache, now time.Time) model {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	s := spinner.New()
	s.Spinner = spinner.Points
	return model{
		srv:         service,
		cache:       cache,
		currentDate: today,
		focusedDate: today,
		state:       initializing,
		viewType:    weekView,
		dayLists:    newWeekLists(today),
		spinner:     s,
		keys:        defaultKeyMap,
		help:        help.New(),
	}
}

func newBaseDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.NormalTitle.UnsetForeground()
	d.Styles.NormalTitle.UnsetBorderForeground()
	d.Styles.NormalDesc.UnsetForeground()
	d.Styles.NormalDesc.UnsetBorderForeground()
	return d
}

func newFocusedDelegate() list.DefaultDelegate {
	d := newBaseDelegate()
	d.Styles.SelectedTitle.Foreground(googleBlue)
	d.Styles.SelectedTitle.BorderForeground(googleBlue)
	d.Styles.SelectedDesc.Foreground(googleBlue)
	d.Styles.SelectedDesc.BorderForeground(googleBlue)
	return d
}

func newUnfocusedDelegate() list.DefaultDelegate {
	d := newBaseDelegate()
	d.Styles.SelectedTitle.Foreground(d.Styles.NormalTitle.GetForeground())
	d.Styles.SelectedTitle.BorderStyle(lipgloss.HiddenBorder())
	d.Styles.SelectedDesc.Foreground(d.Styles.NormalDesc.GetForeground())
	d.Styles.SelectedDesc.BorderStyle(lipgloss.HiddenBorder())
	return d
}

func newDayList(date time.Time) list.Model {
	l := list.New(nil, newUnfocusedDelegate(), 0, 0)
	l.DisableQuitKeybindings()
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetStatusBarItemName("event", "events")
	l.Title = date.Format(AbbreviatedTextDateWithWeekday)
	l.Styles.Title.Bold(true)
	l.Styles.Title.UnsetForeground()
	return l
}

func newWeekLists(focusedDate time.Time) []list.Model {
	lists := make([]list.Model, 7)
	startOfWeek := focusedDate.AddDate(0, 0, -1*int(focusedDate.Weekday()))
	for i := range lists {
		lists[i] = newDayList(startOfWeek.AddDate(0, 0, i))
		lists[focusedDate.Weekday()].SetDelegate(newFocusedDelegate())
	}
	return lists
}

func (m model) Init() tea.Cmd {
	return tea.Batch(getCalendarList(m.srv), m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.Width = m.width
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case showCalendarMsg:
		m.state = ready
		return m, tea.Batch(tea.ClearScreen, m.refreshEvents())
	case calendarListMsg:
		if len(m.calendars) == 0 {
			m.state = ready
		}
		m.calendars = msg.calendars
		if m.state == calendarList {
			m.calendarList, cmd = m.calendarList.Update(msg)
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, m.refreshEvents())
		return m, tea.Batch(cmds...)
	case eventsMsg:
		m.dayLists[msg.date.Weekday()].StopSpinner()
		m.dayLists[msg.date.Weekday()].SetItems(eventsToItems(msg.events))
		return m, nil
	case gotoDateMsg:
		return m, m.focus(msg.date)
	case flushCacheMsg:
		m.cache.Flush()
		return m, nil
	case successMsg:
		if m.state == calendarList {
			return m, getCalendarList(m.srv)
		}
	}
	m, cmd = m.updateSubModels(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) updateSubModels(msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.state {
	case ready:
		m, cmd = m.updateCalendarView(msg)
		return m, cmd
	case gotodate:
		m.gotoDialog, cmd = m.gotoDialog.Update(msg)
		return m, cmd
	case editing:
		m.editPage, cmd = m.editPage.Update(msg)
		return m, cmd
	case deleting:
		m.deleteDialog, cmd = m.deleteDialog.Update(msg)
		return m, cmd
	case calendarList:
		m.calendarList, cmd = m.calendarList.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m *model) focus(date time.Time) tea.Cmd {
	prevDate := m.focusedDate
	m.focusedDate = date
	m.refocusDayLists()
	if m.isOutOfView(prevDate, m.focusedDate) {
		m.resetTitles()
		return m.refreshEvents()
	}
	return nil
}

func (m *model) refocusDayLists() {
	for i := range m.dayLists {
		m.dayLists[i].ResetSelected()
		m.dayLists[i].SetDelegate(newUnfocusedDelegate())
	}
	m.dayLists[m.focusedDate.Weekday()].ResetSelected()
	m.dayLists[m.focusedDate.Weekday()].SetDelegate(newFocusedDelegate())
}

func (m *model) resetTitles() {
	switch m.viewType {
	case dayView:
		m.dayLists[m.focusedDate.Weekday()].Title = m.focusedDate.Format(AbbreviatedTextDateWithWeekday)
	case weekView:
		startOfWeek := m.focusedDate.AddDate(0, 0, -1*int(m.focusedDate.Weekday()))
		for i := range m.dayLists {
			date := startOfWeek.AddDate(0, 0, i)
			m.dayLists[date.Weekday()].Title = date.Format(AbbreviatedTextDateWithWeekday)
		}
	}
}

func (m model) isOutOfView(prev, curr time.Time) bool {
	switch m.viewType {
	case dayView:
		return !prev.Equal(curr)
	case weekView:
		return areInDifferentWeeks(prev, curr)
	default:
		return true
	}
}

func areInDifferentWeeks(a, b time.Time) bool {
	firstDayOfWeek := a.AddDate(0, 0, -1*int(a.Weekday()))
	lastDayOfWeek := firstDayOfWeek.AddDate(0, 0, 6)
	return b.After(lastDayOfWeek) || b.Before(firstDayOfWeek)
}

func (m model) updateCalendarView(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Next):
			return m, m.focus(m.focusedDate.AddDate(0, 0, daysIn(m.viewType)))

		case key.Matches(msg, m.keys.Prev):
			return m, m.focus(m.focusedDate.AddDate(0, 0, -daysIn(m.viewType)))

		case key.Matches(msg, m.keys.NextDay):
			return m, m.focus(m.focusedDate.AddDate(0, 0, 1))

		case key.Matches(msg, m.keys.PrevDay):
			return m, m.focus(m.focusedDate.AddDate(0, 0, -1))

		case key.Matches(msg, m.keys.Today):
			return m, m.focus(m.currentDate)

		case key.Matches(msg, m.keys.DayView):
			m.viewType = dayView
			return m, m.refreshDate(m.focusedDate)

		case key.Matches(msg, m.keys.WeekView):
			m.viewType = weekView
			return m, m.refreshWeek()

		case key.Matches(msg, m.keys.GotoDate):
			m.state = gotodate
			m.gotoDialog = newGotoDialog(m.focusedDate, m.width, m.height)
			return m, nil

		case key.Matches(msg, m.keys.Create):
			m.state = editing
			m.editPage = newEditPage(m.srv, nil, m.focusedDate, filterModifiable(m.calendars), m.width, m.height)
			return m, nil

		case key.Matches(msg, m.keys.Edit):
			event, ok := m.dayLists[m.focusedDate.Weekday()].SelectedItem().(*Event)
			if !ok {
				return m, nil
			}
			m.state = editing
			m.editPage = newEditPage(m.srv, event, m.focusedDate, m.calendars, m.width, m.height)
			return m, nil

		case key.Matches(msg, m.keys.Delete):
			event, ok := m.dayLists[m.focusedDate.Weekday()].SelectedItem().(*Event)
			if !ok {
				return m, nil
			}
			m.state = deleting
			m.deleteDialog = newDeleteDialog(m.srv, event.calendarId, event.event.Id, m.width, m.height)
			return m, nil

		case key.Matches(msg, m.keys.CalendarList):
			m.state = calendarList
			m.calendarList = newCalendarListDialog(m.srv, m.calendars, m.width, m.height)
			return m, nil

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.dayLists[m.focusedDate.Weekday()], cmd = m.dayLists[m.focusedDate.Weekday()].Update(msg)
	return m, cmd
}

func daysIn(p calendarPeriod) int {
	switch p {
	case dayView:
		return 1
	case weekView:
		return 7
	default:
		return 1
	}
}

func filterSelected(calendars []*calendar.CalendarListEntry) []*calendar.CalendarListEntry {
	var selected []*calendar.CalendarListEntry
	for _, c := range calendars {
		if c.Selected {
			selected = append(selected, c)
		}
	}
	return selected
}

func filterModifiable(calendars []*calendar.CalendarListEntry) []*calendar.CalendarListEntry {
	var modifiable []*calendar.CalendarListEntry
	for _, c := range calendars {
		if c.AccessRole == "writer" || c.AccessRole == "owner" {
			modifiable = append(modifiable, c)
		}
	}
	return modifiable
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	title := lipgloss.NewStyle().Padding(0, 1).Bold(true).Background(googleBlue).Render("GoCal")
	header := lipgloss.NewStyle().
		Width(m.width - 2).
		MaxWidth(m.width - 2).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(title)
	var body string
	switch m.state {
	case initializing:
		body = lipgloss.Place(m.width-2, m.height-lipgloss.Height(header), lipgloss.Center, lipgloss.Center, m.spinner.View())
	case ready:
		body = m.viewCalendar(m.width-2, m.height-lipgloss.Height(header))
	case gotodate:
		body = m.gotoDialog.View()
	case editing:
		body = m.editPage.View()
	case deleting:
		body = m.deleteDialog.View()
	case calendarList:
		body = m.calendarList.View()
	}
	bodyContainer := lipgloss.NewStyle().
		MaxWidth(m.width).
		MaxHeight(m.height - lipgloss.Height(header)).
		Render(body)
	return lipgloss.JoinVertical(lipgloss.Center, header, bodyContainer)
}

func (m *model) viewCalendar(width, height int) string {
	helpContainer := lipgloss.NewStyle().
		Width(m.width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(m.help.View(m.keys))
	var calendar string
	switch m.viewType {
	case dayView:
		calendar = m.viewDay(width, height-lipgloss.Height(helpContainer)-2)
	case weekView:
		calendar = m.viewWeek(width, height-lipgloss.Height(helpContainer)-2)
	}
	return lipgloss.JoinVertical(lipgloss.Center, lipgloss.NewStyle().Padding(0, 1).Render(calendar), helpContainer)
}

func (m *model) viewDay(width, height int) string {
	m.dayLists[m.focusedDate.Weekday()].SetSize(width, height)
	if m.focusedDate.Equal(m.currentDate) {
		m.dayLists[m.focusedDate.Weekday()].Styles.Title.Background(googleBlue)
	} else {
		m.dayLists[m.focusedDate.Weekday()].Styles.Title.Background(grey)
	}
	style := lipgloss.NewStyle().Border(lipgloss.HiddenBorder())
	return lipgloss.PlaceHorizontal(width, lipgloss.Left, style.Render(m.dayLists[m.focusedDate.Weekday()].View()))
}

func (m *model) viewWeek(width, height int) string {
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(width / 8)
	var dayViews []string
	startOfWeek := m.focusedDate.AddDate(0, 0, -1*int(m.focusedDate.Weekday()))
	for i := 0; i < 7; i++ {
		date := startOfWeek.AddDate(0, 0, i)
		m.dayLists[date.Weekday()].SetSize(width/8, height)
		if date.Equal(m.focusedDate) {
			style = style.BorderForeground(googleBlue)
		} else {
			style = style.UnsetBorderForeground()
		}
		if date.Equal(m.currentDate) {
			m.dayLists[date.Weekday()].Styles.Title.Background(googleBlue)
		} else {
			m.dayLists[date.Weekday()].Styles.Title.Background(grey)
		}
		dayViews = append(dayViews, style.Render(m.dayLists[date.Weekday()].View()))
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, lipgloss.JoinHorizontal(lipgloss.Top, dayViews...))
}

// -----------------------------------------------------------------------------
// Messages and Commands
// -----------------------------------------------------------------------------

type (
	calendarListMsg struct{ calendars []*calendar.CalendarListEntry }
	eventsMsg       struct {
		date   time.Time
		events []*Event
	}
	errMsg          struct{ err error }
	successMsg      struct{}
	showCalendarMsg struct{}
	flushCacheMsg   struct{}
)

func showCalendarViewCmd() tea.Msg {
	return showCalendarMsg{}
}

func flushCacheCmd() tea.Msg {
	return flushCacheMsg{}
}

func getCalendarList(srv *calendar.Service) tea.Cmd {
	return func() tea.Msg {
		response, err := srv.CalendarList.List().Do()
		if err != nil {
			return errMsg{err: err}
		}
		sort.Slice(response.Items, func(i, j int) bool {
			return response.Items[i].Summary < response.Items[j].Summary
		})
		return calendarListMsg{calendars: response.Items}
	}
}

func (m model) refreshEvents() tea.Cmd {
	switch m.viewType {
	case dayView:
		return m.refreshDate(m.focusedDate)
	case weekView:
		return m.refreshWeek()
	default:
		return nil
	}
}

func (m model) refreshDate(date time.Time) tea.Cmd {
	return tea.Batch(
		getEvents(m.srv, m.cache, filterSelected(m.calendars), date),
		m.dayLists[date.Weekday()].StartSpinner(),
	)
}

func (m model) refreshWeek() tea.Cmd {
	var cmds []tea.Cmd
	startOfWeek := m.focusedDate.AddDate(0, 0, -1*int(m.focusedDate.Weekday()))
	for i := range m.dayLists {
		date := startOfWeek.AddDate(0, 0, i)
		m.dayLists[date.Weekday()].Title = date.Format(AbbreviatedTextDateWithWeekday)
		cmds = append(cmds, m.refreshDate(date))
	}
	return tea.Batch(cmds...)
}

func getEvents(srv *calendar.Service, cache *cache.Cache, calendars []*calendar.CalendarListEntry, date time.Time) tea.Cmd {
	return func() tea.Msg {
		eventCh := make(chan *Event)
		errCh := make(chan error)
		done := make(chan struct{})
		defer close(done)
		var wg sync.WaitGroup
		wg.Add(len(calendars))
		oneDayLater := date.AddDate(0, 0, 1)
		for _, cal := range calendars {
			go func(id string) {
				forwardEvents(srv, cache, id, date, oneDayLater, eventCh, errCh, done)
				wg.Done()
			}(cal.Id)
		}
		go func() {
			wg.Wait()
			close(eventCh)
			close(errCh)
		}()
		var events []*Event
		for event := range eventCh {
			events = append(events, event)
		}
		var errs []error
		for err := range errCh {
			errs = append(errs, err)
		}
		if len(errs) >= 1 {
			return errMsg{err: errs[0]}
		}

		var allDayEvents []*Event
		var timeEvents []*Event
		for _, event := range events {
			if event.event.Start.Date != "" {
				allDayEvents = append(allDayEvents, event)
			} else {
				timeEvents = append(timeEvents, event)
			}
		}
		sort.Slice(allDayEvents, func(i, j int) bool {
			return allDayEvents[i].event.Summary < allDayEvents[j].event.Summary
		})
		sort.Sort(eventsSlice(timeEvents))
		allEvents := append(allDayEvents, timeEvents...)
		return eventsMsg{date: date, events: allEvents}
	}
}

type eventsSlice []*Event

func (events eventsSlice) Len() int {
	return len(events)
}

func (events eventsSlice) Less(i, j int) bool {
	dateI, err := time.Parse(time.RFC3339, events[i].event.Start.DateTime)
	if err != nil {
		return true
	}
	dateJ, err := time.Parse(time.RFC3339, events[j].event.Start.DateTime)
	if err != nil {
		return true
	}
	return dateI.Before(dateJ)
}

func (events eventsSlice) Swap(i, j int) {
	events[i], events[j] = events[j], events[i]
}

func cacheKey(ss ...string) string {
	return strings.Join(ss, "-")
}

func forwardEvents(
	srv *calendar.Service,
	cache *cache.Cache,
	calendarId string,
	timeMin, timeMax time.Time,
	eventCh chan<- *Event,
	errCh chan<- error,
	done <-chan struct{},
) {
	var events []*Event
	key := cacheKey(calendarId, timeMin.Format(time.RFC3339), timeMax.Format(time.RFC3339))
	x, found := cache.Get(key)
	if found {
		events = x.([]*Event)
	} else {
		response, err := srv.Events.
			List(calendarId).
			SingleEvents(true).
			TimeMin(timeMin.Format(time.RFC3339)).
			TimeMax(timeMax.Format(time.RFC3339)).
			OrderBy("startTime").
			Do()
		if err != nil {
			errCh <- err
			return
		}
		for _, event := range response.Items {
			events = append(events, &Event{calendarId: calendarId, event: event})
		}
		cache.SetDefault(key, events)
	}
	for _, event := range events {
		select {
		case eventCh <- event:
		case <-done:
			return
		}
	}
}

// -----------------------------------------------------------------------------
// Keys
// -----------------------------------------------------------------------------

type keyMapDefault struct {
	Next         key.Binding
	Prev         key.Binding
	NextDay      key.Binding
	PrevDay      key.Binding
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
		key.WithKeys("n"),
		key.WithHelp("n", "next period"),
	),
	Prev: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "prev period"),
	),
	NextDay: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "next day"),
	),
	PrevDay: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "prev day"),
	),
	Today: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "today"),
	),
	DayView: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "day view"),
	),
	WeekView: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "week view"),
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

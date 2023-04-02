package main

import (
	"sort"
	"strings"
	"sync"
	"time"

	"gocal/common"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/calendar/v3"
)

// -----------------------------------------------------------------------------
// Model
// -----------------------------------------------------------------------------

type state int

const (
	initializing = iota
	ready
	editing
	deleting
	gotodate
	showCalendarList
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
	state         state
	viewType      calendarPeriod
	eventLists    []list.Model
	calendarList  CalendarList
	gotoDialog    tea.Model
	editPage      tea.Model
	deleteDialog  tea.Model
	spinner       spinner.Model
	keys          CalendarKeyMap
	help          help.Model
	width, height int
}

func newModel(srv *calendar.Service, cache *cache.Cache, now time.Time) model {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	s := spinner.New()
	s.Spinner = spinner.Points
	return model{
		srv:          srv,
		cache:        cache,
		currentDate:  today,
		focusedDate:  today,
		state:        initializing,
		viewType:     weekView,
		eventLists:   newWeekLists(today),
		calendarList: newCalendarList(srv, nil, 0, 0),
		gotoDialog:   newGotoDialog(today, 0, 0),
		editPage:     newEditPage(srv, nil, today, nil, 0, 0),
		deleteDialog: newDeleteDialog(srv, "", "", 0, 0),
		spinner:      s,
		keys:         calendarKeyMap(),
		help:         help.New(),
		width:        0,
		height:       0,
	}
}

func newBaseDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.NormalTitle.UnsetForeground().UnsetBorderForeground()
	d.Styles.NormalDesc.UnsetForeground().UnsetBorderForeground()
	return d
}

func newFocusedDelegate() list.DefaultDelegate {
	d := newBaseDelegate()
	d.Styles.SelectedTitle.Foreground(common.GoogleBlue).BorderForeground(common.GoogleBlue)
	d.Styles.SelectedDesc.Foreground(common.GoogleBlue).BorderForeground(common.GoogleBlue)
	return d
}

func newUnfocusedDelegate() list.DefaultDelegate {
	d := newBaseDelegate()
	d.Styles.SelectedTitle.Foreground(d.Styles.NormalTitle.GetForeground()).BorderStyle(lipgloss.HiddenBorder())
	d.Styles.SelectedDesc.Foreground(d.Styles.NormalDesc.GetForeground()).BorderStyle(lipgloss.HiddenBorder())
	return d
}

func newDayList(date time.Time) list.Model {
	l := list.New(nil, newUnfocusedDelegate(), 0, 0)
	l.DisableQuitKeybindings()
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetStatusBarItemName("event", "events")
	l.Title = date.Format(common.AbbreviatedTextDateWithWeekday)
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

// -----------------------------------------------------------------------------
// Init
// -----------------------------------------------------------------------------

func (m model) Init() tea.Cmd {
	return tea.Batch(getCalendarList(m.srv), m.spinner.Tick)
}

// -----------------------------------------------------------------------------
// Update
// -----------------------------------------------------------------------------

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
		const headerHeight = 3
		m, cmd = m.updateAllSubModels(tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - headerHeight})
		return m, cmd

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		m, cmd = m.updateFocusedSubModel(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case showCalendarMsg:
		m.state = ready
		return m, tea.Batch(tea.ClearScreen, m.refreshEvents())

	case calendarListMsg:
		if m.state == initializing {
			m.state = ready
		}
		m.calendarList.StopSpinner()
		m.calendarList.SetItems(calendarsToItems(msg.calendars))
		return m, m.refreshEvents()

	case eventsMsg:
		m.eventLists[msg.date.Weekday()].StopSpinner()
		m.eventLists[msg.date.Weekday()].SetItems(eventsToItems(msg.events))
		return m, nil

	case gotoDateMsg:
		return m, m.focus(msg.date)

	case updateCalendarListSuccessMsg:
		return m, getCalendarList(m.srv)

	case createEventSuccessMsg, editEventSuccessMsg, deleteEventSuccessMsg:
		m.cache.Flush()
		m, cmd = m.updateFocusedSubModel(msg)
		return m, cmd

	}
	m, cmd = m.updateFocusedSubModel(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) updateAllSubModels(msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	m, cmd = m.updateCalendarView(msg)
	cmds = append(cmds, cmd)
	m.calendarList, cmd = m.calendarList.Update(msg)
	cmds = append(cmds, cmd)
	m.gotoDialog, cmd = m.gotoDialog.Update(msg)
	cmds = append(cmds, cmd)
	m.editPage, cmd = m.editPage.Update(msg)
	cmds = append(cmds, cmd)
	m.deleteDialog, cmd = m.deleteDialog.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) updateFocusedSubModel(msg tea.Msg) (model, tea.Cmd) {
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
	case showCalendarList:
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
	for i := range m.eventLists {
		m.eventLists[i].ResetSelected()
		m.eventLists[i].SetDelegate(newUnfocusedDelegate())
	}
	m.eventLists[m.focusedDate.Weekday()].ResetSelected()
	m.eventLists[m.focusedDate.Weekday()].SetDelegate(newFocusedDelegate())
}

func (m *model) resetTitles() {
	switch m.viewType {
	case dayView:
		m.eventLists[m.focusedDate.Weekday()].Title = m.focusedDate.Format(common.AbbreviatedTextDateWithWeekday)
	case weekView:
		startOfWeek := m.focusedDate.AddDate(0, 0, -1*int(m.focusedDate.Weekday()))
		for i := range m.eventLists {
			date := startOfWeek.AddDate(0, 0, i)
			m.eventLists[date.Weekday()].Title = date.Format(common.AbbreviatedTextDateWithWeekday)
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
		case key.Matches(msg, m.keys.NextPeriod):
			return m, m.focus(m.focusedDate.AddDate(0, 0, daysIn(m.viewType)))

		case key.Matches(msg, m.keys.PrevPeriod):
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
			const headerHeight = 3
			m.gotoDialog = newGotoDialog(m.focusedDate, m.width, m.height-headerHeight)
			return m, nil

		case key.Matches(msg, m.keys.Create):
			m.state = editing
			const headerHeight = 3
			m.editPage = newEditPage(m.srv, nil, m.focusedDate, filterModifiable(itemsToCalendars(m.calendarList.Items())), m.width, m.height-headerHeight)
			return m, nil

		case key.Matches(msg, m.keys.Edit):
			event, ok := m.eventLists[m.focusedDate.Weekday()].SelectedItem().(*EventItem)
			if !ok {
				return m, nil
			}
			m.state = editing
			const headerHeight = 3
			m.editPage = newEditPage(m.srv, event, m.focusedDate, itemsToCalendars(m.calendarList.Items()), m.width, m.height-headerHeight)
			return m, nil

		case key.Matches(msg, m.keys.Delete):
			event, ok := m.eventLists[m.focusedDate.Weekday()].SelectedItem().(*EventItem)
			if !ok {
				return m, nil
			}
			m.state = deleting
			const headerHeight = 3
			m.deleteDialog = newDeleteDialog(m.srv, event.calendarId, event.Id, m.width, m.height-headerHeight)
			return m, nil

		case key.Matches(msg, m.keys.CalendarList):
			m.state = showCalendarList
			return m, nil

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.eventLists[m.focusedDate.Weekday()], cmd = m.eventLists[m.focusedDate.Weekday()].Update(msg)
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

// -----------------------------------------------------------------------------
// View
// -----------------------------------------------------------------------------

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	const title = "GoCal"
	titleStyle := lipgloss.NewStyle().Padding(0, 1).Bold(true).Background(common.GoogleBlue)
	header := lipgloss.NewStyle().
		Width(m.width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(titleStyle.Render(title))
	var body string
	switch m.state {
	case initializing:
		body = lipgloss.Place(m.width, m.height-lipgloss.Height(header), lipgloss.Center, lipgloss.Center, m.spinner.View())
	case ready:
		body = m.viewCalendar(m.width, m.height-lipgloss.Height(header))
	case gotodate:
		body = m.gotoDialog.View()
	case editing:
		body = m.editPage.View()
	case deleting:
		body = m.deleteDialog.View()
	case showCalendarList:
		body = m.calendarList.View()
	}
	bodyContainer := lipgloss.NewStyle().
		MaxWidth(m.width).
		MaxHeight(m.height - lipgloss.Height(header)).
		Render(body)
	return lipgloss.JoinVertical(lipgloss.Center, header, bodyContainer)
}

func (m *model) viewCalendar(width, height int) string {
	help := lipgloss.NewStyle().
		Width(m.width).
		Padding(1).
		AlignHorizontal(lipgloss.Center).
		Render(m.help.View(m.keys))
	var calendar string
	switch m.viewType {
	case dayView:
		calendar = m.viewDay(width, height-lipgloss.Height(help)-2)
	case weekView:
		calendar = m.viewWeek(width, height-lipgloss.Height(help)-2)
	}
	return lipgloss.JoinVertical(lipgloss.Center, lipgloss.NewStyle().Padding(0, 1).Render(calendar), help)
}

func (m *model) viewDay(width, height int) string {
	m.eventLists[m.focusedDate.Weekday()].SetSize(width, height)
	updateDayListTitles(m.eventLists, m.focusedDate, m.currentDate)
	style := lipgloss.NewStyle().Border(lipgloss.HiddenBorder())
	return lipgloss.PlaceHorizontal(width, lipgloss.Left, style.Render(m.eventLists[m.focusedDate.Weekday()].View()))
}

func (m *model) viewWeek(width, height int) string {
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(width / 8)
	var dayViews []string
	startOfWeek := m.focusedDate.AddDate(0, 0, -1*int(m.focusedDate.Weekday()))
	for i := 0; i < 7; i++ {
		date := startOfWeek.AddDate(0, 0, i)
		m.eventLists[date.Weekday()].SetSize(width/8, height)
		updateDayListTitles(m.eventLists, date, m.currentDate)
		if date.Equal(m.focusedDate) {
			style = style.BorderForeground(common.GoogleBlue)
		} else {
			style = style.UnsetBorderForeground()
		}
		dayViews = append(dayViews, style.Render(m.eventLists[date.Weekday()].View()))
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, lipgloss.JoinHorizontal(lipgloss.Top, dayViews...))
}

func updateDayListTitles(dayLists []list.Model, focusedDate, currentDate time.Time) {
	if focusedDate.Equal(currentDate) {
		dayLists[focusedDate.Weekday()].Styles.Title.Background(common.GoogleBlue)
	} else {
		dayLists[focusedDate.Weekday()].Styles.Title.Background(common.Grey)
	}
}

// -----------------------------------------------------------------------------
// Messages and Commands
// -----------------------------------------------------------------------------

type (
	calendarListMsg struct{ calendars []*calendar.CalendarListEntry }
	eventsMsg       struct {
		date   time.Time
		events []*EventItem
	}
	errMsg          struct{ err error }
	showCalendarMsg struct{}
)

func showCalendarViewCmd() tea.Msg {
	return showCalendarMsg{}
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
		getEvents(m.srv, m.cache, filterSelected(itemsToCalendars(m.calendarList.Items())), date),
		m.eventLists[date.Weekday()].StartSpinner(),
	)
}

func (m model) refreshWeek() tea.Cmd {
	var cmds []tea.Cmd
	startOfWeek := m.focusedDate.AddDate(0, 0, -1*int(m.focusedDate.Weekday()))
	for i := range m.eventLists {
		date := startOfWeek.AddDate(0, 0, i)
		m.eventLists[date.Weekday()].Title = date.Format(common.AbbreviatedTextDateWithWeekday)
		cmds = append(cmds, m.refreshDate(date))
	}
	return tea.Batch(cmds...)
}

func getEvents(srv *calendar.Service, cache *cache.Cache, calendars []*calendar.CalendarListEntry, date time.Time) tea.Cmd {
	return func() tea.Msg {
		eventCh := make(chan *EventItem)
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
		var events []*EventItem
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

		// TODO can probably adjust sort interface impl to do this automatically
		var allDayEvents EventItems
		var timeEvents EventItems
		for _, event := range events {
			if event.Start.Date != "" {
				allDayEvents = append(allDayEvents, event)
			} else {
				timeEvents = append(timeEvents, event)
			}
		}
		sort.Slice(allDayEvents, func(i, j int) bool {
			return allDayEvents[i].Summary < allDayEvents[j].Summary
		})
		sort.Sort(timeEvents)
		allEvents := append(allDayEvents, timeEvents...)

		return eventsMsg{date: date, events: allEvents}
	}
}

func cacheKey(ss ...string) string {
	return strings.Join(ss, "-")
}

func forwardEvents(
	srv *calendar.Service,
	cache *cache.Cache,
	calendarId string,
	timeMin, timeMax time.Time,
	eventCh chan<- *EventItem,
	errCh chan<- error,
	done <-chan struct{},
) {
	var events []*EventItem
	key := cacheKey(calendarId, timeMin.Format(time.RFC3339), timeMax.Format(time.RFC3339))
	x, found := cache.Get(key)
	if found {
		events = x.([]*EventItem)
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
			events = append(events, &EventItem{*event, calendarId})
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

func (k CalendarKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help}
}

func (k CalendarKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextPeriod, k.Delete},
		{k.PrevPeriod, k.GotoDate},
		{k.Today, k.CalendarList},
		{k.GotoDate, k.Help},
		{k.Create, k.Quit},
		{k.Edit},
	}
}

package main

import "github.com/charmbracelet/bubbles/key"

type CalendarKeyMap struct {
	NextDay      key.Binding
	PrevDay      key.Binding
	NextPeriod   key.Binding
	PrevPeriod   key.Binding
	Today        key.Binding
	GotoDate     key.Binding
	DayView      key.Binding
	WeekView     key.Binding
	Create       key.Binding
	Edit         key.Binding
	Delete       key.Binding
	CalendarList key.Binding
	Help         key.Binding
	Quit         key.Binding
}

func calendarKeyMap() CalendarKeyMap {
	return CalendarKeyMap{
		NextPeriod: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next period"),
		),
		PrevPeriod: key.NewBinding(
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
			key.WithKeys("backspace", "delete", "x"),
			key.WithHelp("x/del", "delete event"),
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
}

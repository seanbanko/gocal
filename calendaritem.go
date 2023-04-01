package main

import (
	"github.com/charmbracelet/bubbles/list"
	"google.golang.org/api/calendar/v3"
)

type CalendarItem struct {
	calendar *calendar.CalendarListEntry
}

// Implement list.Item interface
func (c CalendarItem) FilterValue() string { return c.Title() }
func (c CalendarItem) Title() string       { return checkbox(c.calendar.Summary, c.calendar.Selected) }
func (c CalendarItem) Description() string { return c.calendar.Description }

func checkbox(label string, checked bool) string {
	if checked {
		return "[X] " + label
	} else {
		return "[ ] " + label
	}
}

func calendarsToItems(calendars []*calendar.CalendarListEntry) []list.Item {
	var items []list.Item
	for _, c := range calendars {
		items = append(items, CalendarItem{calendar: c})
	}
	return items
}

func itemsToCalendars(items []list.Item) []*calendar.CalendarListEntry {
	var calendars []*calendar.CalendarListEntry
	for _, i := range items {
		calendarItem, ok := i.(CalendarItem)
		if !ok {
			continue
		}
		calendars = append(calendars, calendarItem.calendar)
	}
	return calendars
}

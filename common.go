package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/api/calendar/v3"
)

const (
	YYYYMMDD                       = "2006-01-02"
	MMDDYYYY                       = "01/02/2006"
	HHMMSS24h                      = "15:04:05"
	HHMM24h                        = "15:04"
	HHMMSS12h                      = "3:04:05 PM"
	HHMM12h                        = "3:04 PM"
	MMDDYYYYHHMM24h                = "01/02/2006 15:04"
	TextDate                       = "January 2, 2006"
	AbbreviatedTextDate            = "Jan 2 2006"
	TextDateWithWeekday            = "Monday, January 2, 2006"
	AbbreviatedTextDateWithWeekday = "Mon Jan 2"
	AbbreviatedTextDate24h         = "Jan 2 2006 15:04"
)

const (
	googleBlue = lipgloss.Color("#4285F4")
)

const (
	summaryWidth = 40
	monthWidth   = 3
	dayWidth     = 2
	yearWidth    = 4
	timeWidth    = 2
)

var (
	dialogStyle = lipgloss.NewStyle().
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			Align(lipgloss.Center, lipgloss.Center)
	textInputPlaceholderStyle = lipgloss.NewStyle().Faint(true)
	textInputStyle            = lipgloss.NewStyle().
					PaddingLeft(1).
					Border(lipgloss.RoundedBorder())
	textInputSummaryStyle = lipgloss.NewStyle().
				Width(summaryWidth + 2).
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder())
	textInputMonthStyle = lipgloss.NewStyle().
				Width(monthWidth + 2).
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder())
	textInputDayStyle = lipgloss.NewStyle().
				Width(dayWidth + 2).
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder())
	textInputYearStyle = lipgloss.NewStyle().
				Width(yearWidth + 2).
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder())
	textInputTimeStyle = lipgloss.NewStyle().
				Width(timeWidth + 2).
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder())
)

func abbreviatedMonthDayYear(date time.Time) (string, string, string) {
	month := date.Month().String()[:3]
	day := fmt.Sprintf("%02d", date.Day())
	year := fmt.Sprintf("%d", date.Year())
	return month, day, year
}

func checkbox(label string, checked bool) string {
	if checked {
		return "[X] " + label
	} else {
		return "[ ] " + label
	}
}

func isAllDay(event *calendar.Event) bool {
	return event.Start.Date != ""
}

func newTextInput(charLimit int) textinput.Model {
	input := textinput.New()
	input.CharLimit = charLimit
	input.PlaceholderStyle = textInputPlaceholderStyle
	input.Prompt = ""
	return input
}

func focusNext(inputs []textinput.Model, focusIndex int) int {
	newIndex := (focusIndex + 1) % len(inputs)
	refocus(inputs, newIndex)
	return newIndex
}

func focusPrev(inputs []textinput.Model, focusIndex int) int {
	newIndex := focusIndex - 1
	if newIndex < 0 {
		newIndex = len(inputs) - 1
	}
	refocus(inputs, newIndex)
	return newIndex
}

func refocus(inputs []textinput.Model, focusIndex int) {
	for i := range inputs {
		inputs[i].Blur()
	}
	inputs[focusIndex].Focus()
}

func autofill(input *textinput.Model) {
	input.SetValue(input.Placeholder)
}

func autofillAll(inputs []textinput.Model) {
	for i := range inputs {
		if len(inputs[i].Value()) == 0 {
			autofill(&inputs[i])
		}
	}
}

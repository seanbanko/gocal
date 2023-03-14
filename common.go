package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/api/calendar/v3"
)

const (
	HHMM12h                        = "3:04 PM"
	HHMM24h                        = "15:04"
	YYYYMMDD                       = "2006-01-02"
	AbbreviatedTextDate            = "Jan 2 2006"
	AbbreviatedTextDateWithWeekday = "Mon Jan 2"
	TextDateWithWeekday            = "Monday, January 2, 2006"
	AbbreviatedTextDate24h         = "Jan 2 2006 15:04"
	AbbreviatedTextDate12h         = "Jan 2 2006 03:04 PM"
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

func toDateTime(month, day, year, hour, minute, ampm string) (time.Time, error) {
	text := fmt.Sprintf("%s %s %s %s:%s %s", month, day, year, hour, minute, strings.ToUpper(ampm))
	return time.ParseInLocation(AbbreviatedTextDate12h, text, time.Local)
}

func toDateFields(date time.Time) (string, string, string) {
	month := date.Month().String()[:3]
	day := fmt.Sprintf("%02d", date.Day())
	year := fmt.Sprintf("%d", date.Year())
	return month, day, year
}

func toTimeFields(date time.Time) (string, string, string) {
	var hour string
	if date.Hour()%12 == 0 {
		hour = "12"
	} else {
		hour = fmt.Sprintf("%02d", date.Hour()%12)
	}
	minute := fmt.Sprintf("%02d", date.Minute())
	var ampm string
	if date.Hour() < 12 {
		ampm = "am"
	} else {
		ampm = "pm"
	}
	return hour, minute, ampm
}

func toDateTimeFields(date time.Time) (string, string, string, string, string, string) {
	month, day, year := toDateFields(date)
	hour, minute, ampm := toTimeFields(date)
	return month, day, year, hour, minute, ampm
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

func autofillPlaceholder(input *textinput.Model) {
	if len(input.Value()) == 0 {
		input.SetValue(input.Placeholder)
	}
}

func autofillAllPlaceholders(inputs []textinput.Model) {
	for i := range inputs {
		autofillPlaceholder(&inputs[i])
	}
}

func isFull(input textinput.Model) bool {
	return len(input.Value()) == input.CharLimit
}

package common

import (
	"fmt"
	"strings"
	"time"
)

const (
	H                              = "3"
	HPM                            = "3PM"
	H_PM                           = "3 PM"
	HHMM24h                        = "15:04"
	KitchenWithSpace               = "3:04 PM"
	HH_MM_PM                       = "03:04 PM"
	AbbreviatedTextDate            = "Jan 2 2006"
	AbbreviatedTextDateWithWeekday = "Mon Jan 2"
	TextDateWithWeekday            = "Monday, January 2, 2006"
)

const (
	SummaryWidth = 40
	MonthWidth   = len("Jan")
	DayWidth     = len("02")
	YearWidth    = len("2006")
	TimeWidth    = len(HH_MM_PM)
)

func ParseDateTime(month, day, year, tme string) (time.Time, error) {
	d, err := time.Parse(AbbreviatedTextDate, month+" "+day+" "+year)
	if err != nil {
		return d, fmt.Errorf("Failed to parse date: %v", err)
	}
	t, err := ParseTime(tme)
	if err != nil {
		return d, fmt.Errorf("Failed to parse time: %v", err)
	}
	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location()), nil
}

func ParseTime(t string) (time.Time, error) {
	t = strings.ToUpper(t)
	t = strings.TrimSpace(t)
	if !strings.Contains(t, ":") && !strings.ContainsAny(t, "APM") && len(t) >= 3 {
		t = t[:len(t)-2] + ":" + t[len(t)-2:]
	}
	var d time.Time
	formats := []string{time.Kitchen, KitchenWithSpace, HHMM24h, H, HPM, H_PM}
	for _, f := range formats {
		if d, err := time.ParseInLocation(f, t, time.Local); err == nil {
			return d, nil
		}
	}
	return d, fmt.Errorf("Failed to parse time")
}

func ToDateFields(date time.Time) (string, string, string) {
	m := date.Month().String()[:3]
	d := fmt.Sprintf("%02d", date.Day())
	y := fmt.Sprintf("%d", date.Year())
	return m, d, y
}

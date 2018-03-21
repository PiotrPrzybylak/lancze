package time

import (
	"fmt"
	"strings"
	"time"
)

// LocalTimeSpan represents a span between two LocalTime instances
type LocalTimeSpan struct {
	from LocalTime
	to   LocalTime
}

// NewTimeSpan creates new LocalTimeSpan
func NewTimeSpan(from LocalTime, to LocalTime) LocalTimeSpan {
	if to != Midnight && from.After(to) {
		panic(fmt.Sprintf("Start can't be after end of LocalTimeSpan."+
			" Got: start: %v end: %v", from, to))
	}
	return LocalTimeSpan{from, to}
}

// OverlapsDateTimeSpan checks if LocalTimeSpan overlaps DateTimeSpan
func (ts LocalTimeSpan) OverlapsDateTimeSpan(dateTimeSpan DateTimeSpan) bool {
	fromDay := dateTimeSpan.from.Date()
	container := ts.DateTimeSpanWithinOneDay(fromDay)
	return container.Overlaps(dateTimeSpan)
}

// Overlaps checks if LocalTimeSpan overlaps other LocalTimeSpan
func (ts LocalTimeSpan) Overlaps(other LocalTimeSpan) bool {

	if ts.to == Midnight && ts.from.Before(other.to) {
		return true
	}

	if !ts.from.Before(other.to) || !ts.to.After(other.from) {
		return false
	}

	return true
}

// Contains check if LocalTimeSpan contains DateTimeSpan
func (ts LocalTimeSpan) Contains(dateTimeSpan DateTimeSpan) bool {
	fromDay := dateTimeSpan.from.Date()
	container := ts.DateTimeSpanWithinOneDay(fromDay)
	return container.Contains(dateTimeSpan)
}

// From returns start of LocalTimeSpan
func (ts LocalTimeSpan) From() LocalTime {
	return ts.from
}

// To returns end of LocalTimeSpan
func (ts LocalTimeSpan) To() LocalTime {
	return ts.to
}

// DateTimeSpanWithinOneDay creates DateTimeSpan with start and end on the
// given day and with time corresponding to LocalTimeSpan start and end.
func (ts LocalTimeSpan) DateTimeSpanWithinOneDay(date LocalDate) DateTimeSpan {
	start := date.WithTime(ts.from)
	var end LocalDateTime
	if ts.to == Midnight {
		end = date.Next().Start()
	} else {
		end = date.WithTime(ts.to)
	}
	return NewDateTimeSpan(start, end)
}

// MustParseTimeSpan parses string in form of "15:04-15:04" into LocalTimeSpan
func MustParseTimeSpan(value string) LocalTimeSpan {
	times := strings.Split(value, "-")
	from := MustParseLocalTime(times[0])
	to := MustParseLocalTime(times[1])
	return NewTimeSpan(from, to)
}

func (ts LocalTimeSpan) Duration() Duration {
	from := time.Date(0, 0, 0, ts.from.hour, ts.from.minute, 0, 0, time.UTC)
	day := 0
	if ts.to == Midnight {
		day++
	}
	to := time.Date(0, 0, day, ts.to.hour, ts.to.minute, 0, 0, time.UTC)
	return Duration(to.Sub(from))
}

func (ts LocalTimeSpan) Valid() bool {
	return ts.Duration() > 0
}

package time

import (
	"encoding/json"
	"fmt"
)

// DateTimeSpan represents time span between two moments
type DateTimeSpan struct {
	from LocalDateTime
	to   LocalDateTime
}

//CalculateDuration returns time duration between two DateTimes
func CalculateDuration(from LocalDateTime, to LocalDateTime) Duration {
	return to.Sub(from)

}

// NewDateTimeSpan creates DateTimeSpan instance
func NewDateTimeSpan(from LocalDateTime, to LocalDateTime) DateTimeSpan {
	if from.After(to) {
		panic(fmt.Sprintf("Start can't be after end of DateTimeSpan."+
			" Got: start: %v end: %v", from, to))
	}
	return DateTimeSpan{from, to}
}

// NewOpenDateTimeSpanFrom creates DateTimeSpan that represents a set of days between "from" (inclusive) and eternity
func NewOpenDateTimeSpanFrom(from LocalDateTime) (DateTimeSpan, error) {
	if from.IsNull() {
		return DateTimeSpan{from: NullLocalDateTime, to: NullLocalDateTime}, ErrPeriodInvalidParamNull
	}
	return DateTimeSpan{from: from, to: NullLocalDateTime}, nil
}

func MustNewOpenDateTimeSpanFrom(from LocalDateTime) DateTimeSpan {
	span, err := NewOpenDateTimeSpanFrom(from)
	if err != nil {
		panic(err)
	}
	return span
}

// From returns start of DateTimeSpan
func (ts DateTimeSpan) From() LocalDateTime {
	return ts.from
}

// To returns end of DateTimeSpan
func (ts DateTimeSpan) To() LocalDateTime {
	return ts.to
}

// TimeSpan creates LocalTimeSpan based on this object start and stop time of a day.
// Start and stop must be on the same day and start can't be after end.
func (ts DateTimeSpan) TimeSpan() LocalTimeSpan {
	if ts.from.Date() != ts.from.Date() {
		panic(fmt.Sprintf("Start and end of DateTimeSpan "+
			"must be on the same day to produce LocalTimeSpan."+
			" Got: start: %v end: %v", ts.from, ts.to))
	}
	return NewTimeSpan(ts.from.Time(), ts.to.Time())
}

// Empty checks if DateTimeStamp has zero duration
func (ts DateTimeSpan) Empty() bool {
	return ts.from == ts.to
}

// Overlaps checks if DateTimeSpan overlaps other DateTimeSpan
func (ts DateTimeSpan) Overlaps(other DateTimeSpan) bool {
	return ts.from.Before(other.to) && ts.to.After(other.from)
}

// NullableTo returns last day of a period or null
func (ts DateTimeSpan) NullableTo() NullableLocalDateTime {
	if ts.to.IsNull() {
		return NullableLocalDateTime{}
	} else {
		return NullableLocalDateTime{DateTime: ts.to, Valid: true}
	}
}

// Contains checks if DateTimeSpan fully covers other DateTimeSpan
func (ts DateTimeSpan) Contains(other DateTimeSpan) bool {
	return !other.from.Before(ts.from) && !other.to.After(ts.to)
}

func (ts DateTimeSpan) MarshalJSON() ([]byte, error) {
	type JSONLocalDateTime struct {
		From LocalDateTime `json:"from"`
		To   LocalDateTime `json:"to"`
	}
	return json.Marshal(JSONLocalDateTime{From: ts.from, To: ts.to})
}

func (ts DateTimeSpan) Subtract(span DateTimeSpan) []DateTimeSpan {

	if !ts.Overlaps(span) {
		return []DateTimeSpan{ts}
	}

	result := []DateTimeSpan{}
	if ts.From().Before(span.From()) {
		beforeSlot := NewDateTimeSpan(ts.From(), span.From())
		result = append(result, beforeSlot)
	}
	if ts.To().After(span.To()) {
		afterSlot := NewDateTimeSpan(span.To(), ts.To())
		result = append(result, afterSlot)
	}
	return result
}

package time

import (
	"errors"
)

// Period represents a set of days between two dates (inclusive)
type Period struct {
	from LocalDate
	to   LocalDate
}

// ErrPeriodInvalidFromAfterTo is returned when Period creating is called
// with invalid parameters: from > to
var ErrPeriodInvalidFromAfterTo = errors.New(`Invalid time period - "from" is after "to"`)

// ErrPeriodInvalidParamNull is returned when Period creating is called
// with invalid parameters: from or to is NullLocalDate
var ErrPeriodInvalidParamNull = errors.New(`Invalid time period - date is null`)

// NewPeriod creates Period that represents a set of days between two dates: "from" (inclusive) and "to" (inclusive)
func NewPeriod(from LocalDate, to LocalDate) (Period, error) {
	if from.IsNull() || to.IsNull() {
		return Period{from: NullLocalDate, to: NullLocalDate}, ErrPeriodInvalidParamNull
	}

	if from.After(to) {
		return Period{from: NullLocalDate, to: NullLocalDate}, ErrPeriodInvalidFromAfterTo
	}
	return Period{from: from, to: to}, nil
}

// NewOneDayPeriod creates Period that represents a single day
func NewOneDayPeriod(date LocalDate) (Period, error) {
	return NewPeriod(date, date)
}

// NewOpenPeriodFrom creates Period that represents a set of days between "from" (inclusive) and eternity
func NewOpenPeriodFrom(from LocalDate) (Period, error) {
	if from.IsNull() {
		return Period{from: NullLocalDate, to: NullLocalDate}, ErrPeriodInvalidParamNull
	}
	return Period{from: from, to: NullLocalDate}, nil
}

func MustNewOpenPeriodFrom(from LocalDate) Period {
	period, err := NewOpenPeriodFrom(from)
	if err != nil {
		panic(err)
	}
	return period
}

func MustNewPeriod(from LocalDate, to LocalDate) Period {
	period, err := NewPeriod(from, to)
	if err != nil {
		panic(err)
	}
	return period
}

func MustNewOneDayPeriod(date LocalDate) Period {
	return MustNewPeriod(date, date)
}

// Days returns a slice of all days in a period in order
func (p Period) Days() []LocalDate {
	var days []LocalDate
	for current := p.from; current.BeforeOrEqual(p.to); current = current.Next() {
		days = append(days, current)
	}
	return days
}

// Contains tells if period contains given date
func (p Period) Contains(date LocalDate) bool {
	return p.from.BeforeOrEqual(date) && (date.BeforeOrEqual(p.to) || p.to.IsNull())
}

// From returns first day of a period
func (p Period) From() LocalDate {
	return p.from
}

// To returns last day of a period
func (p Period) To() LocalDate {
	return p.to
}

// NullableTo returns last day of a period or null
func (p Period) NullableTo() NullableLocalDate {
	if p.to.IsNull() {
		return NullableLocalDate{}
	} else {
		return NullableLocalDate{Date: p.to, Valid: true}
	}
}

func (p Period) String() string {
	if p.to == NullLocalDate {
		return "[" + p.from.t.Format(LocalDateFormat) + " - indefinitely]"
	}
	return "[" + p.from.t.Format(LocalDateFormat) + " - " + p.to.t.Format(LocalDateFormat) + "]"
}

func (p Period) IsOpen() bool {
	return p.to.IsNull()
}

func (p Period) ToDateTimeSpan() DateTimeSpan {
	if p.IsOpen() {
		return MustNewOpenDateTimeSpanFrom(p.from.Start())
	}
	return NewDateTimeSpan(p.from.Start(), p.to.End())
}

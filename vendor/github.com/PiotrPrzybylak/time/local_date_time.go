package time

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/mailru/easyjson/jlexer"
	"github.com/pkg/errors"
)

// LocalDateTime represents a date and time without taking into account a timezone
type LocalDateTime struct {
	t time.Time
}

// NullableLocalDateTime represents a LocalDateTime that may be null.
// It implements the sql.Scanner interface so it can be used
// as a scan destination, similar to sql.NullString.
type NullableLocalDateTime struct {
	DateTime LocalDateTime
	Valid    bool
}

// Scan implements the Scanner interface.
func (d *NullableLocalDateTime) Scan(value interface{}) error {
	d.DateTime.t, d.Valid = value.(time.Time)
	return nil
}

// Value implements the sql driver Valuer interface.
func (d NullableLocalDateTime) Value() (driver.Value, error) {
	if !d.Valid {
		return nil, nil
	}
	return d.DateTime.String(), nil
}

// NewLocalDateTime creates instances of LocalDateTime
func NewLocalDateTime(t time.Time) LocalDateTime {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	return LocalDateTime{t: time.Date(year, month, day, hour, min, sec, t.Nanosecond(), time.UTC)}
}

// Date returns the date component from LocalDateTime
func (ldt LocalDateTime) Date() LocalDate {
	return NewLocalDate(ldt.t.Date())
}

// Time returns the time component from LocalDateTime
func (ldt LocalDateTime) Time() LocalTime {
	return ToLocalTime(ldt.t)
}

// After reports whether this object if after method argument
func (ldt LocalDateTime) After(other LocalDateTime) bool {
	return ldt.t.After(other.t)
}

// Before reports whether this object if before method argument
func (ldt LocalDateTime) Before(other LocalDateTime) bool {
	return ldt.t.Before(other.t)
}

// Add returns the date-time equal to ldt+d.
func (ldt LocalDateTime) Add(d Duration) LocalDateTime {
	return NewLocalDateTime(ldt.t.Add(time.Duration(d)))
}

// AddDate returns the date-time increased by the given number of years, months and days.
func (ldt LocalDateTime) AddDate(years, months, days int) LocalDateTime {
	return NewLocalDateTime(ldt.t.AddDate(years, months, days))
}

// Sub returns the duration ldt-u. If the result exceeds the maximum (or minimum)
// value that can be stored in a Duration, the maximum (or minimum) duration
// will be returned. To compute ldt-d for a duration d, use Add(-d).
func (ldt LocalDateTime) Sub(u LocalDateTime) Duration {
	return Duration(ldt.t.Sub(u.t))
}

// GoTime returns converts LocalDateTime to time.Time,
// keeping same numeric date and time values and sets timezone
func (ldt LocalDateTime) GoTime(location *time.Location) time.Time {
	year, month, day := ldt.t.Date()
	hour, minute, second := ldt.t.Clock()
	nanosecond := ldt.t.Nanosecond()
	return time.Date(year, month, day, hour, minute, second, nanosecond, location)
}

// Scan implements the Scanner interface.
func (ldt *LocalDateTime) Scan(value interface{}) error {
	if value == nil {
		return errors.New("sql: failed to scan into LocalDateTime: column is NULL")
	}

	tt, ok := value.(time.Time)
	if !ok {
		return errors.New(
			"sql: failed to scan into LocalDateTime: column type is neither time nor date")
	}
	*ldt = NewLocalDateTime(tt)

	return nil
}

// Value implements the sql driver Valuer interface.
func (ldt LocalDateTime) Value() (driver.Value, error) {
	return ldt.t.Format("2006-01-02 15:04:05.999999999"), nil
}

func (ldt LocalDateTime) String() string {
	return ldt.Date().String() + " " + ldt.Time().String()
}

// MarshalJSON marshals date to JSON
func (ldt LocalDateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ldt.String())
}

// UnmarshalJSON parses JSON string into LocalDateTime
func (ldt *LocalDateTime) UnmarshalJSON(data []byte) error {
	l := jlexer.Lexer{Data: data}
	ldt.unmarshalEasyJSON(&l)
	return l.Error()
}

func (ldt *LocalDateTime) unmarshalEasyJSON(in *jlexer.Lexer) {
	if data := in.String(); in.Ok() {
		parsedLocalDateTime, err := ParseLocalDateTime(data)
		if err != nil {
			in.AddError(err)
			return
		}

		*ldt = parsedLocalDateTime
	}
}

// ParseLocalDateTime parses string into LocalDateTime
func ParseLocalDateTime(datetime string) (LocalDateTime, error) {
	t, err := time.Parse(LocalDateFormat+" 15:04", datetime)
	if err != nil {
		return NullLocalDateTime, err
	}
	return NewLocalDateTime(t), nil
}

// MustParseLocalDateTime parses string into LocalDateTime
func MustParseLocalDateTime(datetime string) LocalDateTime {
	t, err := ParseLocalDateTime(datetime)
	if err != nil {
		panic(err)
	}
	return t
}

// NullLocalDateTime is used to represent missing LocalDateTime value
var NullLocalDateTime = LocalDateTime{t: time.Time{}}

// Weekday returns weekday of a given date
func (ldt LocalDateTime) Weekday() Weekday {
	return Weekday(ldt.t.Weekday())
}

// IsNull checks whether this variable represents missing value
func (ldt LocalDateTime) IsNull() bool {
	return ldt == NullLocalDateTime
}

func (ldt LocalDateTime) Earlier(other LocalDateTime) LocalDateTime {
	return Earlier(ldt, other)
}

func Earlier(dateTime1, dateTime2 LocalDateTime) LocalDateTime {
	if dateTime1.Before(dateTime2) {
		return dateTime1
	}
	return dateTime2
}

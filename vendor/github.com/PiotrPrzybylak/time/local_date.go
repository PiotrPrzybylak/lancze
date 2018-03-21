package time

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mailru/easyjson/jlexer"
)

// LocalDate represents a date without taking into account a timezone
type LocalDate struct {
	t time.Time
}

// NullableLocalDate represents a LocalDate that may be null.
// It implements the sql.Scanner interface so it can be used
// as a scan destination, similar to sql.NullString.
type NullableLocalDate struct {
	Date  LocalDate
	Valid bool
}

// LocalDateFormat is used to render LocalDate as string
const LocalDateFormat = "2006-01-02" // yyyy-mm-dd

// NullLocalDate is used to represent missing date value
var NullLocalDate = LocalDate{t: time.Time{}}

// NewLocalDate creates instances of LocalDate
func NewLocalDate(year int, month time.Month, day int) LocalDate {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return LocalDate{t: t}
}

// ToLocalDate converts date from go's time.Time to LocalDate.
func ToLocalDate(date time.Time) LocalDate {
	return NewLocalDate(date.Date())
}

// BeforeOrEqual reports wheter the date d is before or equal to u.
func (d LocalDate) BeforeOrEqual(u LocalDate) bool {
	return !d.t.After(u.t)
}

// Equal reports wheter the date d is equal to u.
func (d LocalDate) Equal(u LocalDate) bool {
	return d.t.Equal(u.t)
}

// After reports whether date d is after u.
func (d LocalDate) After(u LocalDate) bool {
	return d.t.After(u.t)
}

// Next returns the next day
func (d LocalDate) Next() LocalDate {
	next := d.t.AddDate(0, 0, 1)
	return ToLocalDate(next)
}

// Date returns the year, month, and day in which d occurs.
func (d LocalDate) Date() (year int, month time.Month, day int) {
	return d.t.Date()
}

// AddDate returns the date corresponding to adding the given number of years,
// months and days to d.
func (d LocalDate) AddDate(years, months, days int) LocalDate {
	result := d.t.AddDate(years, months, days)
	return ToLocalDate(result)
}

// IsNull checks whether this variable represents missing value
func (d LocalDate) IsNull() bool {
	return d == NullLocalDate
}

// Weekday returns weekday of a given date
func (d LocalDate) Weekday() Weekday {
	return Weekday(d.t.Weekday())
}

// GetStartOfDayUTC returns Time value that represent beginning of a day (00:00 AM) at UTC timezone
func (d LocalDate) GetStartOfDayUTC() time.Time {
	return d.t
}

// MarshalJSON marshals date to JSON
func (d LocalDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d LocalDate) String() string {
	return d.t.Format(LocalDateFormat)
}

// MarshalText serializes this date type to string
func (d LocalDate) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// MustParseLocalDate parses string into date using LocalDateFormat and panics if it is impossible
func MustParseLocalDate(date string) LocalDate {
	localDate, err := ParseLocalDate(date)
	if err != nil {
		panic(fmt.Sprintf("Wrong date format: %v", date))
	}
	return localDate
}

// ParseLocalDate parses string into date using LocalDateFormat
func ParseLocalDate(date string) (LocalDate, error) {
	t, err := time.Parse(LocalDateFormat, date)
	if err != nil {
		return NullLocalDate, err
	}
	return NewLocalDate(t.Date()), nil
}

// UnmarshalText parses string into date using LocalDateFormat
func (d *LocalDate) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	dd, err := ParseLocalDate(string(text))
	if err != nil {
		return err
	}
	*d = dd
	return nil
}

// UnmarshalJSON parses JSON string into date using LocalDateFormat
func (d *LocalDate) UnmarshalJSON(data []byte) error {
	l := jlexer.Lexer{Data: data}
	d.unmarshalEasyJSON(&l)
	return l.Error()
}

func (d *LocalDate) unmarshalEasyJSON(in *jlexer.Lexer) {
	if data := in.String(); in.Ok() {
		dd, err := ParseLocalDate(data)
		if err != nil {
			in.AddError(err)
			return
		}
		*d = dd
	}
}

// Scan implements the Scanner interface.
func (d *LocalDate) Scan(value interface{}) error {
	if value == nil {
		return errors.New("sql: failed to scan into LocalDate: column is NULL")
	}

	tt, ok := value.(time.Time)
	if !ok {
		return errors.New(
			"sql: failed to scan into LocalDate: column type is neither time nor date")
	}
	*d = ToLocalDate(tt)

	return nil
}

// Value implements the sql driver Valuer interface.
func (d LocalDate) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan implements the Scanner interface.
func (d *NullableLocalDate) Scan(value interface{}) error {
	d.Date.t, d.Valid = value.(time.Time)
	return nil
}

// Value implements the sql driver Valuer interface.
func (d NullableLocalDate) Value() (driver.Value, error) {
	if !d.Valid {
		return nil, nil
	}

	return d.Date.String(), nil
}

// WithTime creates time.Time instance when combined with LocalTime
func (d LocalDate) WithTime(localTime LocalTime) LocalDateTime {
	return NewLocalDateTime(time.Date(d.t.Year(), d.t.Month(), d.t.Day(), localTime.hour, localTime.minute, 0, 0, time.UTC))
}

func (d LocalDate) Year() int {
	return d.t.Year()
}

func (d LocalDate) Start() LocalDateTime {
	return d.WithTime(MustCreateNewLocalTime(0, 0))
}

func (d LocalDate) End() LocalDateTime {
	return d.Next().Start()
}

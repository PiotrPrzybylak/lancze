package time

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mailru/easyjson/jlexer"
	"github.com/pkg/errors"
)

var Midnight = MustCreateNewLocalTime(0, 0)

// LocalTime represents time of a day without taking into account a timezone
type LocalTime struct {
	hour   int
	minute int
}

// NewLocalTime creates instances of LocalTime
func NewLocalTime(hour, minute int) (LocalTime, error) {
	if hour < 0 || hour >= 24 {
		return NullLocalTime, fmt.Errorf(
			"hour must be non-negative and smaller than 24! Was: %v", hour)
	}

	if minute < 0 || minute >= 60 {
		return NullLocalTime, fmt.Errorf(
			"minute must be non-negative and smaller than 60! Was: %v", minute)
	}
	return LocalTime{hour: hour, minute: minute}, nil
}

// MustCreateNewLocalTime is like NewLocalTime but panics on error
func MustCreateNewLocalTime(hour, minute int) LocalTime {
	localTime, err := NewLocalTime(hour, minute)
	if err != nil {
		panic(err)
	}
	return localTime
}

// MustParseLocalTime parses string in form of "15:04"
func MustParseLocalTime(value string) LocalTime {
	parts := strings.Split(value, ":")
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		panic(err)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		panic(err)
	}
	localTime := MustCreateNewLocalTime(hour, minute)
	return localTime
}

// NullLocalTime is used to represent missing date value
var NullLocalTime = LocalTime{hour: -1, minute: -1}

// ToLocalTime converts time from go's time.Time to LocalTime.
func ToLocalTime(t time.Time) LocalTime {
	hour, minute, _ := t.Clock()
	localTime, err := NewLocalTime(hour, minute)
	if err != nil {
		panic(err)
	}
	return localTime
}

func (t LocalTime) String() string {
	return fmt.Sprintf("%02v:%02v", t.hour, t.minute)
}

// ParseLocalTime parses string into LocalTime
func ParseLocalTime(date string) (LocalTime, error) {
	t, err := time.Parse("15:04", date)
	if err != nil {
		return NullLocalTime, err
	}
	return ToLocalTime(t), nil
}

// MarshalJSON marshals locat time to JSON
func (t LocalTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON parses JSON string into LocalTime
func (t *LocalTime) UnmarshalJSON(data []byte) error {
	l := jlexer.Lexer{Data: data}
	t.unmarshalEasyJSON(&l)
	return l.Error()
}

func (t *LocalTime) unmarshalEasyJSON(in *jlexer.Lexer) {
	if data := in.String(); in.Ok() {
		tt, err := ParseLocalTime(data)
		if err != nil {
			in.AddError(err)
			return
		}
		*t = tt
	}
}

// Scan implements the Scanner interface.
func (t *LocalTime) Scan(value interface{}) error {
	if value == nil {
		return errors.New("sql: failed to scan into LocalTime: column is NULL")
	}

	tt, ok := value.(time.Time)
	if !ok {
		return errors.New(
			"sql: failed to scan into LocalTime: column type is neither time nor date")
	}
	*t = ToLocalTime(tt)

	return nil
}

// Value implements the sql driver Valuer interface.
func (t LocalTime) Value() (driver.Value, error) {
	return t.String(), nil
}

// Hour returns hour part of time
func (t LocalTime) Hour() int {
	return t.hour
}

// Minutes returns minutes part of time
func (t LocalTime) Minutes() int {
	return t.minute
}

// After checks if given local time is after method argument
func (t LocalTime) After(other LocalTime) bool {
	if t.hour > other.hour {
		return true
	}

	if t.hour == other.hour && t.minute > other.minute {
		return true
	}

	return false
}

// Before checks if given local time is before method argument
func (t LocalTime) Before(other LocalTime) bool {
	if t.After(other) || t == other {
		return false
	}
	return true
}

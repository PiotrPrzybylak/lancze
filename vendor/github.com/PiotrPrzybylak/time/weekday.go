package time

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Weekday represents day of a week
type Weekday time.Weekday

// Constants representing days of a week
const (
	NotAWeekday Weekday = -1 + iota
	Sunday
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

var weekdays = []Weekday{
	Monday,
	Tuesday,
	Wednesday,
	Thursday,
	Friday,
	Saturday,
	Sunday,
}

// MarshalJSON marshals date to JSON
func (w Weekday) MarshalJSON() ([]byte, error) {
	return json.Marshal(w.String())
}

func (w Weekday) String() string {
	return strings.ToLower(time.Weekday(w).String())
}

// MarshalText serializes this date type to string
func (w Weekday) MarshalText() ([]byte, error) {
	return []byte(w.String()), nil
}

// UnmarshalText parses string into weekday
func (w *Weekday) UnmarshalText(text []byte) error {

	if len(text) == 0 {
		return nil
	}

	parsedWeekday, err := ParseWeekday(string(text))
	if err != nil {
		*w = NotAWeekday
		return errors.Wrap(err, "Weekday.UnmarshalText() failed")
	}
	*w = parsedWeekday
	return nil
}

// ParseWeekday parses string into Weekday
func ParseWeekday(weekdayString string) (Weekday, error) {
	for _, weekday := range weekdays {
		if weekday.String() == strings.ToLower(weekdayString) {
			return weekday, nil
		}
	}
	return NotAWeekday, errors.Errorf("Wrong Weekday format: %v", weekdayString)
}

// Value implements the sql driver Valuer interface.
func (w Weekday) Value() (driver.Value, error) {
	return strings.ToUpper(w.String()), nil
}

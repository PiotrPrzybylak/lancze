package time

import (
	"context"
	"database/sql/driver"
	"time"
)

// UnixTimeStamp represents number of seconds
// elapsed since January 1, 1970 UTC.
type UnixTimeStamp int64

// Value implements the sql driver Valuer interface.
func (uts UnixTimeStamp) Value() (driver.Value, error) {
	return int64(uts), nil
}

// Scan implements the Scanner interface.
func (uts *UnixTimeStamp) Scan(value interface{}) error {
	*uts = UnixTimeStamp(value.(int64))
	return nil
}

func (uts UnixTimeStamp) SecondsTo(t time.Time) int64 {
	return int64(uts) - t.Unix()
}

// UserTypeCtxKey is type to keep time in context
type TimeCtxKey string

// TimeKey is key to keep current time in context
var CurrentTimeKey = TimeCtxKey("current_time")

// Clock provides information about current time
type Clock interface {
	Today(ctx context.Context) LocalDate
	Now(ctx context.Context) LocalDateTime
	GoTime(ctx context.Context) time.Time
	ToUnixTimestamp(ctx context.Context, localDateTime LocalDateTime) UnixTimeStamp
}

// NewClock creates new Clock
func NewClock() Clock {
	timeZoneWarsaw, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		panic(err)
	}
	return clock{timeZoneWarsaw}
}

type clock struct {
	timeZone *time.Location
}

func (c clock) Today(ctx context.Context) LocalDate {
	return ToLocalDate(c.GoTime(ctx))
}

func (c clock) Now(ctx context.Context) LocalDateTime {
	return NewLocalDateTime(c.GoTime(ctx))
}

func (c clock) ToUnixTimestamp(ctx context.Context, localDateTime LocalDateTime) UnixTimeStamp {
	return UnixTimeStamp(localDateTime.GoTime(c.timeZone).Unix())
}

func (c clock) GoTime(ctx context.Context) time.Time {
	return (*(ctx.Value(CurrentTimeKey).(*time.Time))).In(c.timeZone)
}

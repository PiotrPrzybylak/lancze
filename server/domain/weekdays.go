package domain

import "github.com/PiotrPrzybylak/time"

var Weekdays = map[time.Weekday]string {
	time.Monday: "P",
	time.Tuesday: "W",
	time.Wednesday: "Ś",
	time.Thursday: "C",
	time.Friday: "P",
	time.Saturday: "S",
	time.Sunday: "N",

}

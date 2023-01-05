package config

import (
	"strconv"
	"time"
)

func GetDurationTime(StartDate time.Time, EndDate time.Time) string {
	diff := EndDate.Sub(StartDate)

	months := int64(diff.Hours() / 24 / 30)
	days := int64(diff.Hours() / 24)

	if days%30 >= 0 {
		days = days % 30
	}

	var duration string
	if months >= 1 && days >= 1 {
		duration = strconv.FormatInt(months, 10) + " month " + strconv.FormatInt(days, 10) + " days"
	} else if months >= 1 && days <= 0 {
		duration = strconv.FormatInt(months, 10) + " month"
	} else if months < 1 && days >= 0 {
		duration = strconv.FormatInt(days, 10) + " days"
	} else {
		duration = "0 days"
	}
	return duration
}

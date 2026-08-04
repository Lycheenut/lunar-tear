package gametime

import "time"

const utcPlusNineOffsetSeconds = 9 * 60 * 60

var businessLocation = time.FixedZone("UTC+9", utcPlusNineOffsetSeconds)

// BusinessLocation returns the fixed timezone used for game calendar boundaries.
func BusinessLocation() *time.Location {
	return businessLocation
}

func Now() time.Time {
	return time.Now().UTC()
}

func NowMillis() int64 {
	return Now().UnixMilli()
}

func InBusinessLocation(millis int64) time.Time {
	return time.UnixMilli(millis).In(businessLocation)
}

func StartOfBusinessDayMillis() int64 {
	return StartOfBusinessDayAtMillis(NowMillis())
}

func StartOfBusinessDayAtMillis(millis int64) int64 {
	n := InBusinessLocation(millis)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, businessLocation).UnixMilli()
}

// BusinessWeeklyVersion returns Monday 00:00 UTC+9 as a stable weekly identifier.
func BusinessWeeklyVersion(millis int64) int64 {
	t := InBusinessLocation(millis)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := time.Date(t.Year(), t.Month(), t.Day()-(weekday-1), 0, 0, 0, 0, businessLocation)
	return monday.UnixMilli()
}

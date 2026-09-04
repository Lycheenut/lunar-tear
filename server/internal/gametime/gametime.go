package gametime

import "time"

const utcMinusEightOffsetSeconds = -8 * 60 * 60

var businessLocation = time.FixedZone("UTC-8", utcMinusEightOffsetSeconds)

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

func BusinessDayKey(millis int64) int32 {
	n := InBusinessLocation(millis)
	return int32(n.Year()*10000 + int(n.Month())*100 + n.Day())
}

// BusinessMonthKey returns a compact key for the calendar month in UTC-8.
func BusinessMonthKey(millis int64) int32 {
	n := InBusinessLocation(millis)
	return int32(n.Year()*100 + int(n.Month()))
}

func StartOfNextBusinessMonthAtMillis(millis int64) int64 {
	n := InBusinessLocation(millis)
	return time.Date(n.Year(), n.Month()+1, 1, 0, 0, 0, 0, businessLocation).UnixMilli()
}

// BusinessWeeklyVersion returns Monday 00:00 UTC-8 (08:00 UTC) as a stable weekly identifier.
func BusinessWeeklyVersion(millis int64) int64 {
	t := InBusinessLocation(millis)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := time.Date(t.Year(), t.Month(), t.Day()-(weekday-1), 0, 0, 0, 0, businessLocation)
	return monday.UnixMilli()
}

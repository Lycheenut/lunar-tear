package gametime

import "time"

func Now() time.Time {
	return time.Now().UTC()
}

func NowMillis() int64 {
	return Now().UnixMilli()
}

func StartOfDayMillis() int64 {
	return StartOfDayAtMillis(NowMillis())
}

func StartOfDayAtMillis(millis int64) int64 {
	n := time.UnixMilli(millis).UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC).UnixMilli()
}

// WeeklyVersion returns a stable weekly identifier (start-of-week timestamp in millis, Monday 00:00 UTC).
func WeeklyVersion(millis int64) int64 {
	t := time.UnixMilli(millis).UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := time.Date(t.Year(), t.Month(), t.Day()-(weekday-1), 0, 0, 0, 0, time.UTC)
	return monday.UnixMilli()
}

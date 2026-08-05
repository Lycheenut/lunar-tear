package gametime

import (
	"testing"
	"time"
)

func TestBusinessLocationUsesUTCPlusNine(t *testing.T) {
	name, offset := InBusinessLocation(0).Zone()
	if name != "UTC+9" || offset != 9*60*60 {
		t.Fatalf("business timezone = %s (%d), want UTC+9 (%d)", name, offset, 9*60*60)
	}
}

func TestNowRemainsUTC(t *testing.T) {
	if got := Now().Location(); got != time.UTC {
		t.Fatalf("Now location = %s, want UTC", got)
	}
}

func TestStartOfBusinessDayAtMillisUsesUTCPlusNine(t *testing.T) {
	now := time.Date(2026, time.August, 4, 16, 30, 0, 0, time.UTC)
	want := time.Date(2026, time.August, 4, 15, 0, 0, 0, time.UTC).UnixMilli()
	if got := StartOfBusinessDayAtMillis(now.UnixMilli()); got != want {
		t.Fatalf("UTC+9 day start = %d, want %d", got, want)
	}
}

func TestBusinessWeeklyVersionUsesUTCPlusNineMonday(t *testing.T) {
	now := time.Date(2026, time.August, 2, 16, 30, 0, 0, time.UTC)
	want := time.Date(2026, time.August, 2, 15, 0, 0, 0, time.UTC).UnixMilli()
	if got := BusinessWeeklyVersion(now.UnixMilli()); got != want {
		t.Fatalf("UTC+9 weekly version = %d, want %d", got, want)
	}
}

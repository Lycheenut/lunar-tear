package gametime

import (
	"testing"
	"time"
)

func TestBusinessLocationUsesUTCMinusEight(t *testing.T) {
	name, offset := InBusinessLocation(0).Zone()
	if name != "UTC-8" || offset != -8*60*60 {
		t.Fatalf("business timezone = %s (%d), want UTC-8 (%d)", name, offset, -8*60*60)
	}
}

func TestNowRemainsUTC(t *testing.T) {
	if got := Now().Location(); got != time.UTC {
		t.Fatalf("Now location = %s, want UTC", got)
	}
}

func TestStartOfBusinessDayAtMillisUsesUTC0800Boundary(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{name: "before boundary", now: time.Date(2026, time.August, 4, 7, 59, 0, 0, time.UTC), want: time.Date(2026, time.August, 3, 8, 0, 0, 0, time.UTC)},
		{name: "at boundary", now: time.Date(2026, time.August, 4, 8, 0, 0, 0, time.UTC), want: time.Date(2026, time.August, 4, 8, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := StartOfBusinessDayAtMillis(tc.now.UnixMilli()); got != tc.want.UnixMilli() {
				t.Fatalf("business day start = %d, want %d", got, tc.want.UnixMilli())
			}
		})
	}
}

func TestBusinessDayKeyUsesUTC0800Boundary(t *testing.T) {
	before := time.Date(2026, time.September, 4, 7, 59, 0, 0, time.UTC)
	at := time.Date(2026, time.September, 4, 8, 0, 0, 0, time.UTC)
	if got := BusinessDayKey(before.UnixMilli()); got != 20260903 {
		t.Fatalf("day before boundary = %d, want 20260903", got)
	}
	if got := BusinessDayKey(at.UnixMilli()); got != 20260904 {
		t.Fatalf("day at boundary = %d, want 20260904", got)
	}
}

func TestBusinessWeeklyVersionUsesMondayUTC0800Boundary(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{name: "before boundary", now: time.Date(2026, time.August, 3, 7, 59, 0, 0, time.UTC), want: time.Date(2026, time.July, 27, 8, 0, 0, 0, time.UTC)},
		{name: "at boundary", now: time.Date(2026, time.August, 3, 8, 0, 0, 0, time.UTC), want: time.Date(2026, time.August, 3, 8, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BusinessWeeklyVersion(tc.now.UnixMilli()); got != tc.want.UnixMilli() {
				t.Fatalf("weekly version = %d, want %d", got, tc.want.UnixMilli())
			}
		})
	}
}

func TestBusinessMonthUsesUTC0800Boundary(t *testing.T) {
	before := time.Date(2026, time.September, 1, 7, 59, 0, 0, time.UTC)
	at := time.Date(2026, time.September, 1, 8, 0, 0, 0, time.UTC)
	if got := BusinessMonthKey(before.UnixMilli()); got != 202608 {
		t.Fatalf("month before boundary = %d, want 202608", got)
	}
	if got := BusinessMonthKey(at.UnixMilli()); got != 202609 {
		t.Fatalf("month at boundary = %d, want 202609", got)
	}
	wantNext := time.Date(2026, time.October, 1, 8, 0, 0, 0, time.UTC).UnixMilli()
	if got := StartOfNextBusinessMonthAtMillis(at.UnixMilli()); got != wantNext {
		t.Fatalf("next month start = %d, want %d", got, wantNext)
	}
}

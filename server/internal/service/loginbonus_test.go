package service

import (
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"
)

func TestApplyLoginBonusStampTreatsSameBusinessDayAsIdempotentReplay(t *testing.T) {
	now := time.Date(2026, 8, 4, 8, 1, 0, 0, time.UTC).UnixMilli()
	user := store.UserState{
		LoginBonus: store.UserLoginBonusState{
			LoginBonusId:                1,
			CurrentPageNumber:           2,
			CurrentStampNumber:          3,
			LatestRewardReceiveDatetime: now - int64(time.Minute/time.Millisecond),
			LatestVersion:               now - int64(time.Minute/time.Millisecond),
		},
	}
	before := user

	receipt, applied, err := applyLoginBonusStamp(nil, &user, now)
	if err != nil {
		t.Fatalf("idempotent replay returned error: %v", err)
	}
	if applied {
		t.Fatal("idempotent replay applied another login bonus stamp")
	}
	if receipt != (loginBonusReceipt{}) {
		t.Fatalf("idempotent replay receipt = %+v, want empty", receipt)
	}
	if !reflect.DeepEqual(user, before) {
		t.Fatalf("idempotent replay changed user state\nbefore: %+v\nafter:  %+v", before, user)
	}
}

func TestIsLoginBonusStampReceivedTodayUsesUTC0800Boundary(t *testing.T) {
	now := time.Date(2026, 8, 4, 8, 1, 0, 0, time.UTC)
	cases := []struct {
		name   string
		latest time.Time
		want   bool
	}{
		{name: "previous business day", latest: time.Date(2026, 8, 4, 7, 59, 59, 999_000_000, time.UTC), want: false},
		{name: "at business day boundary", latest: time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC), want: true},
		{name: "current business day", latest: time.Date(2026, 8, 4, 8, 0, 30, 0, time.UTC), want: true},
		{name: "future timestamp", latest: now.Add(time.Millisecond), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lb := store.UserLoginBonusState{LatestRewardReceiveDatetime: tc.latest.UnixMilli()}
			if got := isLoginBonusStampReceivedToday(lb, now.UnixMilli()); got != tc.want {
				t.Fatalf("isLoginBonusStampReceivedToday() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateLoginBonusTerm(t *testing.T) {
	day := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC).UnixMilli()
	term := masterdata.LoginBonusTerm{
		StartDatetime:           day - int64(time.Hour/time.Millisecond),
		EndDatetime:             day + int64(time.Hour/time.Millisecond),
		StampReceiveEndDatetime: day + int64(2*time.Hour/time.Millisecond),
	}
	lb := store.UserLoginBonusState{LoginBonusId: 1}

	if err := validateLoginBonusTerm(term, lb, day); err != nil {
		t.Fatalf("valid term rejected: %v", err)
	}

	cases := []struct {
		name string
		term masterdata.LoginBonusTerm
		lb   store.UserLoginBonusState
		now  int64
	}{
		{name: "not started", term: term, lb: lb, now: term.StartDatetime - 1},
		{name: "ended", term: term, lb: lb, now: term.EndDatetime},
		{name: "stamp receive ended", term: masterdata.LoginBonusTerm{StampReceiveEndDatetime: day}, lb: lb, now: day},
		{name: "already received today", term: term, lb: store.UserLoginBonusState{LoginBonusId: 1, LatestRewardReceiveDatetime: day - 1000}, now: day},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateLoginBonusTerm(tc.term, tc.lb, tc.now); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("status = %v, want FailedPrecondition (err=%v)", status.Code(err), err)
			}
		})
	}
}

func TestValidateLoginBonusTermUsesUTC0800DayBoundary(t *testing.T) {
	now := time.Date(2026, 8, 4, 8, 1, 0, 0, time.UTC)
	previousServerDay := time.Date(2026, 8, 4, 7, 59, 0, 0, time.UTC)
	sameServerDay := time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC)

	term := masterdata.LoginBonusTerm{}
	lb := store.UserLoginBonusState{
		LoginBonusId:                1,
		LatestRewardReceiveDatetime: previousServerDay.UnixMilli(),
	}
	if err := validateLoginBonusTerm(term, lb, now.UnixMilli()); err != nil {
		t.Fatalf("previous business day rejected: %v", err)
	}

	lb.LatestRewardReceiveDatetime = sameServerDay.UnixMilli()
	if err := validateLoginBonusTerm(term, lb, now.UnixMilli()); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("same business day status = %v, want FailedPrecondition (err=%v)", status.Code(err), err)
	}
}

func TestAdvanceLoginStateCountsCalendarDays(t *testing.T) {
	day := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	sameDay := store.UserLoginState{
		TotalLoginCount:        5,
		ContinualLoginCount:    3,
		MaxContinualLoginCount: 4,
		LastLoginDatetime:      day.Add(-time.Hour).UnixMilli(),
	}
	advanceLoginState(&sameDay, day.UnixMilli())
	assertLoginCounts(t, sameDay, 5, 3, 4)

	consecutive := store.UserLoginState{
		TotalLoginCount:        5,
		ContinualLoginCount:    4,
		MaxContinualLoginCount: 4,
		LastLoginDatetime:      day.Add(-24 * time.Hour).UnixMilli(),
	}
	advanceLoginState(&consecutive, day.UnixMilli())
	assertLoginCounts(t, consecutive, 6, 5, 5)

	broken := store.UserLoginState{
		TotalLoginCount:        5,
		ContinualLoginCount:    4,
		MaxContinualLoginCount: 7,
		LastLoginDatetime:      day.Add(-48 * time.Hour).UnixMilli(),
	}
	advanceLoginState(&broken, day.UnixMilli())
	assertLoginCounts(t, broken, 6, 1, 7)

	utc0800Rollover := store.UserLoginState{
		TotalLoginCount:        5,
		ContinualLoginCount:    4,
		MaxContinualLoginCount: 4,
		LastLoginDatetime:      time.Date(2026, 8, 4, 7, 59, 0, 0, time.UTC).UnixMilli(),
	}
	advanceLoginState(&utc0800Rollover, time.Date(2026, 8, 4, 8, 1, 0, 0, time.UTC).UnixMilli())
	assertLoginCounts(t, utc0800Rollover, 6, 5, 5)
}

func assertLoginCounts(t *testing.T, login store.UserLoginState, total, continual, maximum int32) {
	t.Helper()
	if login.TotalLoginCount != total || login.ContinualLoginCount != continual || login.MaxContinualLoginCount != maximum {
		t.Fatalf("login counts = (%d,%d,%d), want (%d,%d,%d)",
			login.TotalLoginCount, login.ContinualLoginCount, login.MaxContinualLoginCount,
			total, continual, maximum)
	}
}

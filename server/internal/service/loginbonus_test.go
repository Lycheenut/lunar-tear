package service

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"
)

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

func TestValidateLoginBonusTermUsesUTCPlusNineDayBoundary(t *testing.T) {
	// 2026-08-04 16:30 UTC is already 2026-08-05 01:30 in UTC+9.
	now := time.Date(2026, 8, 4, 16, 30, 0, 0, time.UTC)
	previousServerDay := time.Date(2026, 8, 4, 8, 30, 0, 0, time.UTC)
	sameServerDay := time.Date(2026, 8, 4, 15, 30, 0, 0, time.UTC)

	term := masterdata.LoginBonusTerm{}
	lb := store.UserLoginBonusState{
		LoginBonusId:                1,
		LatestRewardReceiveDatetime: previousServerDay.UnixMilli(),
	}
	if err := validateLoginBonusTerm(term, lb, now.UnixMilli()); err != nil {
		t.Fatalf("previous UTC+9 day rejected: %v", err)
	}

	lb.LatestRewardReceiveDatetime = sameServerDay.UnixMilli()
	if err := validateLoginBonusTerm(term, lb, now.UnixMilli()); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("same UTC+9 day status = %v, want FailedPrecondition (err=%v)", status.Code(err), err)
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

	utcPlusNineRollover := store.UserLoginState{
		TotalLoginCount:        5,
		ContinualLoginCount:    4,
		MaxContinualLoginCount: 4,
		LastLoginDatetime:      time.Date(2026, 8, 4, 14, 30, 0, 0, time.UTC).UnixMilli(),
	}
	advanceLoginState(&utcPlusNineRollover, time.Date(2026, 8, 4, 15, 30, 0, 0, time.UTC).UnixMilli())
	assertLoginCounts(t, utcPlusNineRollover, 6, 5, 5)
}

func assertLoginCounts(t *testing.T, login store.UserLoginState, total, continual, maximum int32) {
	t.Helper()
	if login.TotalLoginCount != total || login.ContinualLoginCount != continual || login.MaxContinualLoginCount != maximum {
		t.Fatalf("login counts = (%d,%d,%d), want (%d,%d,%d)",
			login.TotalLoginCount, login.ContinualLoginCount, login.MaxContinualLoginCount,
			total, continual, maximum)
	}
}

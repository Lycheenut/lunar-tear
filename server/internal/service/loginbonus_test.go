package service

import (
	"maps"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestApplyLoginBonusStampsTreatsSameBusinessDayAsIdempotentReplay(t *testing.T) {
	now := time.Date(2026, 8, 4, 8, 1, 0, 0, time.UTC).UnixMilli()
	catalog, _ := loadCampaignAndLoginBonusCatalogs(t)
	user := store.UserState{
		LoginBonuses: map[int32]store.UserLoginBonusState{1: {
			LoginBonusId:                1,
			CurrentPageNumber:           2,
			CurrentStampNumber:          3,
			LatestRewardReceiveDatetime: now - int64(time.Minute/time.Millisecond),
			LatestVersion:               now - int64(time.Minute/time.Millisecond),
		}},
	}
	before := user
	before.LoginBonuses = maps.Clone(user.LoginBonuses)

	receipts, err := applyLoginBonusStamps(catalog, &user, nil, now)
	if err != nil {
		t.Fatalf("idempotent replay returned error: %v", err)
	}
	if len(receipts) != 0 {
		t.Fatalf("idempotent replay receipts = %+v, want empty", receipts)
	}
	if !reflect.DeepEqual(user, before) {
		t.Fatalf("idempotent replay changed user state\nbefore: %+v\nafter:  %+v", before, user)
	}
}

func TestLoginBonusStartConditionsCoverCurrentMasterData(t *testing.T) {
	loginBonuses, campaigns := loadCampaignAndLoginBonusCatalogs(t)
	seen := make(map[int32]bool)
	for _, definition := range loginBonuses.Definitions() {
		seen[definition.LoginBonusStartConditionId] = true
		switch definition.LoginBonusStartConditionId {
		case loginBonusStartConditionAll, loginBonusStartConditionComeback,
			loginBonusStartConditionBeginner, loginBonusStartConditionComebackGrade1:
		default:
			t.Fatalf("login bonus %d references unsupported start condition %d",
				definition.LoginBonusId, definition.LoginBonusStartConditionId)
		}
	}
	for _, conditionId := range []int32{0, 4, 5, 6} {
		if !seen[conditionId] {
			t.Fatalf("current master data no longer references expected start condition %d", conditionId)
		}
	}

	cleared := map[int32]store.UserQuestState{1: {QuestStateType: model.UserQuestStateTypeCleared}}
	beginnerNow := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC).UnixMilli()
	beginner := store.UserState{
		BeginnerCampaign: store.UserBeginnerCampaignState{
			BeginnerCampaignId:       1,
			CampaignRegisterDatetime: beginnerNow - int64(24*time.Hour/time.Millisecond),
		},
		Quests: cleared,
	}
	if !loginBonusStartConditionEligible(loginBonusStartConditionAll, campaigns, &beginner, beginnerNow) {
		t.Fatal("condition 0 did not accept an ordinary active user")
	}
	if !loginBonusStartConditionEligible(loginBonusStartConditionBeginner, campaigns, &beginner, beginnerNow) {
		t.Fatal("condition 5 did not accept a beginner campaign user")
	}
	if loginBonusStartConditionEligible(loginBonusStartConditionComeback, campaigns, &beginner, beginnerNow) {
		t.Fatal("condition 4 accepted a non-comeback user")
	}

	comeback := store.UserState{
		ComebackCampaign: store.UserComebackCampaignState{
			ComebackCampaignId: 2,
			ComebackDatetime:   beginnerNow - int64(24*time.Hour/time.Millisecond),
		},
		Quests: cleared,
	}
	if !loginBonusStartConditionEligible(loginBonusStartConditionComeback, campaigns, &comeback, beginnerNow) {
		t.Fatal("condition 4 did not accept a comeback campaign user")
	}
	if loginBonusStartConditionEligible(loginBonusStartConditionComebackGrade1, campaigns, &comeback, beginnerNow) {
		t.Fatal("condition 6 accepted comeback grade group 0")
	}

	grade1Now := int64(1659924000000) + int64(24*time.Hour/time.Millisecond)
	grade1 := store.UserState{
		ComebackCampaign: store.UserComebackCampaignState{
			ComebackCampaignId: 3,
			ComebackDatetime:   grade1Now - int64(time.Hour/time.Millisecond),
		},
		Quests: cleared,
	}
	if !loginBonusStartConditionEligible(loginBonusStartConditionComebackGrade1, campaigns, &grade1, grade1Now) {
		t.Fatal("condition 6 did not accept comeback grade group 1")
	}
}

func TestSyncLoginBonusesCreatesEveryEligibleActiveBonus(t *testing.T) {
	loginBonuses, campaigns := loadCampaignAndLoginBonusCatalogs(t)
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC).UnixMilli()
	user := store.UserState{
		BeginnerCampaign: store.UserBeginnerCampaignState{
			BeginnerCampaignId:       1,
			CampaignRegisterDatetime: now - int64(24*time.Hour/time.Millisecond),
		},
		Quests: map[int32]store.UserQuestState{1: {QuestStateType: model.UserQuestStateTypeCleared}},
	}

	syncLoginBonuses(loginBonuses, campaigns, &user, now, false)
	for _, id := range []int32{1, 91, 97} {
		if _, ok := user.LoginBonuses[id]; !ok {
			t.Fatalf("eligible active login bonus %d was not created", id)
		}
	}
	if _, ok := user.LoginBonuses[24]; ok {
		t.Fatal("comeback login bonus 24 was created for a beginner")
	}
}

func TestApplyLoginBonusStampsReceivesAllActiveBonuses(t *testing.T) {
	loginBonuses, _ := loadCampaignAndLoginBonusCatalogs(t)
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC).UnixMilli()
	user := store.UserState{
		LoginBonuses: map[int32]store.UserLoginBonusState{
			1:  {LoginBonusId: 1, CurrentPageNumber: 1},
			91: {LoginBonusId: 91, CurrentPageNumber: 1},
		},
	}

	receipts, err := applyLoginBonusStamps(loginBonuses, &user, nil, now)
	if err != nil {
		t.Fatalf("receive all active login bonuses: %v", err)
	}
	if len(receipts) != 2 || len(user.Gifts.NotReceived) != 2 {
		t.Fatalf("received %d stamps and %d gifts, want 2 each", len(receipts), len(user.Gifts.NotReceived))
	}
	if user.LoginBonuses[1].CurrentStampNumber != 1 || user.LoginBonuses[91].CurrentStampNumber != 1 {
		t.Fatalf("login bonus stamps were not advanced: %+v", user.LoginBonuses)
	}

	replayed, err := applyLoginBonusStamps(loginBonuses, &user, nil, now)
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if len(replayed) != 0 || len(user.Gifts.NotReceived) != 2 {
		t.Fatalf("idempotent replay added rewards: receipts=%d gifts=%d", len(replayed), len(user.Gifts.NotReceived))
	}
}

func loadCampaignAndLoginBonusCatalogs(t *testing.T) (*masterdata.LoginBonusCatalog, *campaign.Catalog) {
	t.Helper()
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatalf("init master data: %v", err)
	}
	campaigns, err := campaign.Load()
	if err != nil {
		t.Fatalf("load campaigns: %v", err)
	}
	return masterdata.LoadLoginBonusCatalog(), campaigns
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

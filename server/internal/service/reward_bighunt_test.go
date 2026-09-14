package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func TestReceiveBigHuntRewardClaimsCompletedWeekOnce(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("weekly-reward", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	masterData, err := os.ReadFile(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if err != nil {
		t.Fatal(err)
	}
	masterDataPath := filepath.Join(t.TempDir(), "master-data.bin.e")
	if err := os.WriteFile(masterDataPath, masterData, 0o600); err != nil {
		t.Fatal(err)
	}
	holder, err := runtime.NewHolder(masterDataPath)
	if err != nil {
		t.Fatal(err)
	}
	holder.Get().BigHunt = bigHuntWeeklyRewardTestCatalog()
	currentWeek := gametime.BusinessWeeklyVersion(gametime.NowMillis())
	lastWeek := currentWeek - 7*24*60*60*1000
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: lastWeek, AttributeType: 2}] = store.BigHuntWeeklyMaxScore{MaxScore: 100}
		user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: currentWeek, AttributeType: 2}] = store.BigHuntWeeklyMaxScore{MaxScore: 200}
		user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{BigHuntScheduleId: 1, BigHuntBossId: 1}] = store.BigHuntScheduleMaxScore{MaxScore: 100}
	}); err != nil {
		t.Fatal(err)
	}
	rewardServer := NewRewardServiceServer(repo, repo, holder)
	topServer := NewBigHuntServiceServer(repo, repo, holder)
	top, err := topServer.GetBigHuntTopData(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if top.IsReceivedWeeklyScoreReward {
		t.Fatal("unclaimed completed week is shown as received")
	}
	for attempt := 0; attempt < 2; attempt++ {
		response, err := rewardServer.ReceiveBigHuntReward(context.Background(), &emptypb.Empty{})
		if err != nil {
			t.Fatal(err)
		}
		user, err := repo.LoadUser(userId)
		if err != nil {
			t.Fatal(err)
		}
		if got := user.Materials[999]; got != 3 {
			t.Fatalf("attempt %d: weekly reward count = %d, want 3 from the completed week", attempt, got)
		}
		if got := user.Materials[998]; got != 2 {
			t.Fatalf("attempt %d: daily reward count = %d, want 2", attempt, got)
		}
		if got := len(response.WeeklyScoreReward); got != 1-attempt {
			t.Fatalf("attempt %d: granted weekly reward entries = %d, want %d", attempt, got, 1-attempt)
		}
		if !user.BigHuntWeeklyStatuses[lastWeek].IsReceivedWeeklyReward || user.BigHuntWeeklyStatuses[currentWeek].IsReceivedWeeklyReward {
			t.Fatalf("wrong week marked as received: %+v", user.BigHuntWeeklyStatuses)
		}
		if !response.IsReceivedWeeklyScoreReward || len(response.LastWeekWeeklyScoreReward) != 1 || response.LastWeekWeeklyScoreReward[0].Count != 3 {
			t.Fatalf("incorrect completed-week response: %+v", response)
		}
		top, err := topServer.GetBigHuntTopData(context.Background(), &emptypb.Empty{})
		if err != nil {
			t.Fatal(err)
		}
		if !top.IsReceivedWeeklyScoreReward || len(top.LastWeekWeeklyScoreReward) != 1 || top.LastWeekWeeklyScoreReward[0].Count != 3 {
			t.Fatalf("top data disagrees with completed-week claim: %+v", top)
		}
	}
}

func TestReceiveBigHuntWeeklyRewardRollover(t *testing.T) {
	rollover := time.Date(2026, time.September, 14, 8, 0, 0, 0, time.UTC).UnixMilli()
	lastWeek := rollover - 7*24*60*60*1000
	for _, tc := range []struct {
		name           string
		now            int64
		score          int64
		alreadyClaimed bool
		futureOnly     bool
		wantCount      int32
		wantReceived   bool
	}{
		{name: "before rollover", now: rollover - 1, score: 100},
		{name: "at rollover", now: rollover, score: 100, wantCount: 3, wantReceived: true},
		{name: "late in claim week", now: rollover + 6*24*60*60*1000, score: 100, wantCount: 3, wantReceived: true},
		{name: "no completed-week score", now: rollover},
		{name: "below reward threshold", now: rollover, score: 99},
		{name: "already claimed", now: rollover, score: 100, alreadyClaimed: true, wantReceived: true},
		{name: "reward schedule not yet active", now: rollover, score: 100, futureOnly: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := bigHuntWeeklyRewardTestCatalog()
			// The new reward group must not replace the completed week's rewards.
			catalog.WeeklyRewardSchedulesByAttr[2] = []masterdata.ScoreRewardScheduleEntry{
				{BigHuntScoreRewardGroupId: 2, StartDatetime: rollover},
				{BigHuntScoreRewardGroupId: 1, StartDatetime: lastWeek},
			}
			catalog.ScoreRewardThresholds[2] = []masterdata.ScoreRewardThreshold{{NecessaryScore: 100, BigHuntRewardGroupId: 2}}
			if tc.futureOnly {
				catalog.WeeklyRewardSchedulesByAttr[2] = catalog.WeeklyRewardSchedulesByAttr[2][:1]
			}
			cat := &runtime.Catalogs{BigHunt: catalog, QuestHandler: &questflow.QuestHandler{Granter: &store.PossessionGranter{}}}
			user := store.SeedUserState(1, "weekly", 1, model.ClientPlatform{})
			user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: lastWeek, AttributeType: 2}] = store.BigHuntWeeklyMaxScore{MaxScore: tc.score}
			user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: rollover, AttributeType: 2}] = store.BigHuntWeeklyMaxScore{MaxScore: 200}
			if tc.alreadyClaimed {
				user.BigHuntWeeklyStatuses[lastWeek] = store.BigHuntWeeklyStatus{IsReceivedWeeklyReward: true, LatestVersion: rollover}
			}
			_, received := receiveBigHuntWeeklyReward(cat, user, tc.now)
			if user.Materials[999] != tc.wantCount || received != tc.wantReceived {
				t.Fatalf("reward count=%d, received=%v; want %d, %v", user.Materials[999], received, tc.wantCount, tc.wantReceived)
			}
			if _, ok := user.BigHuntWeeklyStatuses[rollover]; ok {
				t.Fatal("unfinished week was marked as received")
			}
			if tc.wantReceived {
				ws := user.BigHuntWeeklyStatuses[lastWeek]
				wantVersion := tc.now
				if tc.alreadyClaimed {
					wantVersion = rollover
				}
				if !ws.IsReceivedWeeklyReward || ws.LatestVersion != wantVersion {
					t.Fatalf("incorrect persisted claim: %+v", ws)
				}
			} else if len(user.BigHuntWeeklyStatuses) != 0 {
				t.Fatalf("empty claim wrote received statuses: %+v", user.BigHuntWeeklyStatuses)
			}
		})
	}
}

func TestReceiveBigHuntWeeklyRewardEmptyClaimDoesNotConsumeFollowingWeek(t *testing.T) {
	rollover := time.Date(2026, time.September, 14, 8, 0, 0, 0, time.UTC).UnixMilli()
	cat := &runtime.Catalogs{BigHunt: bigHuntWeeklyRewardTestCatalog(), QuestHandler: &questflow.QuestHandler{Granter: &store.PossessionGranter{}}}
	user := store.SeedUserState(1, "weekly", 1, model.ClientPlatform{})
	receiveBigHuntWeeklyReward(cat, user, rollover)
	user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: rollover, AttributeType: 2}] = store.BigHuntWeeklyMaxScore{MaxScore: 200}
	for range 2 {
		_, received := receiveBigHuntWeeklyReward(cat, user, rollover+7*24*60*60*1000)
		if !received || user.Materials[999] != 8 {
			t.Fatalf("next week's reward count=%d, received=%v; want 8, true", user.Materials[999], received)
		}
	}
}

func bigHuntWeeklyRewardTestCatalog() *masterdata.BigHuntCatalog {
	return &masterdata.BigHuntCatalog{
		ActiveScheduleId: 1,
		BossQuestById: map[int32]masterdata.BigHuntBossQuestRow{
			1: {BigHuntBossQuestId: 1, BigHuntBossId: 1, BigHuntScoreRewardGroupScheduleId: 3},
		},
		BossByBossId: map[int32]masterdata.BigHuntBossRow{
			1: {BigHuntBossId: 1, AttributeType: 2},
		},
		WeeklyRewardSchedulesByAttr: map[int32][]masterdata.ScoreRewardScheduleEntry{
			2: {{BigHuntScoreRewardGroupId: 1}},
		},
		ScoreRewardSchedules: map[int32][]masterdata.ScoreRewardScheduleEntry{
			3: {{BigHuntScoreRewardGroupId: 3}},
		},
		ScoreRewardThresholds: map[int32][]masterdata.ScoreRewardThreshold{
			1: {{NecessaryScore: 100, BigHuntRewardGroupId: 1}, {NecessaryScore: 200, BigHuntRewardGroupId: 2}},
			3: {{NecessaryScore: 100, BigHuntRewardGroupId: 3}},
		},
		RewardItems: map[int32][]masterdata.RewardItem{
			1: {{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 999, Count: 3}},
			2: {{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 999, Count: 5}},
			3: {{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 998, Count: 2}},
		},
	}
}

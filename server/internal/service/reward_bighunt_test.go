package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "lunar-tear/server/gen/proto"
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
	// Existing accounts have scores but no pending status. The client will not
	// call ReceiveBigHuntReward until login repairs the status sent by GetUserData.
	userServer := NewUserServiceServer(repo, repo, holder, "", false)
	if _, err := userServer.Auth(context.Background(), &pb.AuthUserRequest{Uuid: "weekly-reward"}); err != nil {
		t.Fatal(err)
	}
	data, err := NewDataServiceServer(repo, repo).GetUserData(context.Background(), &pb.UserDataGetRequest{
		TableName: []string{"IUserBigHuntWeeklyStatus"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var statuses []struct {
		Week     int64 `json:"bigHuntWeeklyVersion"`
		Received bool  `json:"isReceivedWeeklyReward"`
	}
	if err := json.Unmarshal([]byte(data.UserDataJson["IUserBigHuntWeeklyStatus"]), &statuses); err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 2 || statuses[0].Week != lastWeek || statuses[0].Received || statuses[1].Week != currentWeek || statuses[1].Received {
		t.Fatalf("client cannot discover the unclaimed completed week: %+v", statuses)
	}
	top, err := topServer.GetBigHuntTopData(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if top.IsReceivedWeeklyScoreReward {
		t.Fatal("unclaimed completed week is shown as received")
	}
	assertBigHuntWeeklyScoreResults(t, top.WeeklyScoreResult, 0, 100, 100)
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := userServer.Auth(context.Background(), &pb.AuthUserRequest{Uuid: "weekly-reward"}); err != nil {
			t.Fatal(err)
		}
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
		assertBigHuntWeeklyScoreResults(t, response.WeeklyScoreResult, 0, 100, 100)
		top, err := topServer.GetBigHuntTopData(context.Background(), &emptypb.Empty{})
		if err != nil {
			t.Fatal(err)
		}
		if !top.IsReceivedWeeklyScoreReward || len(top.LastWeekWeeklyScoreReward) != 1 || top.LastWeekWeeklyScoreReward[0].Count != 3 {
			t.Fatalf("top data disagrees with completed-week claim: %+v", top)
		}
	}
}

func assertBigHuntWeeklyScoreResults(t *testing.T, results []*pb.WeeklyScoreResult, before, after, current int64) {
	t.Helper()
	if len(results) != 1 || results[0].AttributeType != 2 || results[0].BeforeMaxScore != before || results[0].AfterMaxScore != after || results[0].CurrentMaxScore != current {
		t.Fatalf("incorrect completed-week comparison: %+v", results)
	}
}

func TestBigHuntWeeklyScoreResultsCompareCompletedWeeks(t *testing.T) {
	rollover := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC).UnixMilli()
	catalog := bigHuntWeeklyRewardTestCatalog()
	catalog.BossByBossId[1] = masterdata.BigHuntBossRow{BigHuntBossId: 1, AttributeType: 2, BigHuntBossGradeGroupId: 1}
	catalog.GradeThresholds = map[int32][]masterdata.GradeThreshold{1: {
		{NecessaryScore: 0, AssetGradeIconId: 1},
		{NecessaryScore: 100, AssetGradeIconId: 2},
		{NecessaryScore: 200, AssetGradeIconId: 3},
	}}
	user := store.SeedUserState(1, "weekly", 1, model.ClientPlatform{})
	user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: rollover - 2*bigHuntWeekMillis, AttributeType: 2}] = store.BigHuntWeeklyMaxScore{MaxScore: 50}
	user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: rollover - bigHuntWeekMillis, AttributeType: 2}] = store.BigHuntWeeklyMaxScore{MaxScore: 100}
	user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{BigHuntScheduleId: 1, BigHuntBossId: 1}] = store.BigHuntScheduleMaxScore{MaxScore: 200}
	results := buildBigHuntWeeklyScoreResults(catalog, *user, rollover)
	assertBigHuntWeeklyScoreResults(t, results, 50, 100, 200)
	if result := results[0]; result.BeforeAssetGradeIconId != 1 || result.AfterAssetGradeIconId != 2 || result.CurrentAssetGradeIconId != 3 {
		t.Fatalf("grade icons do not match the score periods: %+v", result)
	}
	results = buildBigHuntWeeklyScoreResults(catalog, *store.SeedUserState(1, "empty", 1, model.ClientPlatform{}), rollover)
	assertBigHuntWeeklyScoreResults(t, results, 0, 0, 0)
	if result := results[0]; result.BeforeAssetGradeIconId != 0 || result.AfterAssetGradeIconId != 0 || result.CurrentAssetGradeIconId != 0 {
		t.Fatalf("missing score was assigned a grade: %+v", result)
	}
}

func TestEnsureBigHuntWeeklyStatusesPreservesReceipts(t *testing.T) {
	rollover := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC).UnixMilli()
	user := store.SeedUserState(1, "weekly", 1, model.ClientPlatform{})
	for _, week := range []int64{rollover, rollover - bigHuntWeekMillis, rollover - 2*bigHuntWeekMillis} {
		for _, attr := range []int32{2, 3} {
			user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: week, AttributeType: attr}] = store.BigHuntWeeklyMaxScore{MaxScore: 100}
		}
	}
	user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: rollover - 3*bigHuntWeekMillis, AttributeType: 2}] = store.BigHuntWeeklyMaxScore{}
	claimed := store.BigHuntWeeklyStatus{IsReceivedWeeklyReward: true, LatestVersion: rollover - 1}
	user.BigHuntWeeklyStatuses[rollover-2*bigHuntWeekMillis] = claimed
	for _, now := range []int64{rollover, rollover + 1} {
		ensureBigHuntWeeklyStatuses(user, now)
		if len(user.BigHuntWeeklyStatuses) != 3 || user.BigHuntWeeklyStatuses[rollover-2*bigHuntWeekMillis] != claimed {
			t.Fatalf("receipt changed or empty score got a status: %+v", user.BigHuntWeeklyStatuses)
		}
		for _, week := range []int64{rollover, rollover - bigHuntWeekMillis} {
			if got := user.BigHuntWeeklyStatuses[week]; got.IsReceivedWeeklyReward || got.LatestVersion != rollover {
				t.Fatalf("pending status was not created idempotently: %+v", got)
			}
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
		if !received || user.Materials[999] != 5 {
			t.Fatalf("next week's reward count=%d, received=%v; want 5, true", user.Materials[999], received)
		}
	}
}

func TestBigHuntWeeklyRewardUsesHighestTierPerAttribute(t *testing.T) {
	rollover := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC).UnixMilli()
	lastWeek := rollover - bigHuntWeekMillis
	for _, tc := range []struct {
		name      string
		score     int64
		wantCount int32
	}{
		{name: "zero-score tier", score: 1, wantCount: 1},
		{name: "first tier", score: 100, wantCount: 3},
		{name: "highest tier", score: 200, wantCount: 5},
		{name: "above highest tier", score: 1000, wantCount: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := bigHuntWeeklyRewardTestCatalog()
			catalog.ScoreRewardThresholds[1] = append([]masterdata.ScoreRewardThreshold{{NecessaryScore: 0, BigHuntRewardGroupId: 4}}, catalog.ScoreRewardThresholds[1]...)
			catalog.RewardItems[4] = []masterdata.RewardItem{{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 999, Count: 1}}
			// Multiple bosses can refer to an attribute; weekly rewards belong to the attribute.
			catalog.BossByBossId[2] = masterdata.BigHuntBossRow{BigHuntBossId: 2, AttributeType: 2}
			catalog.WeeklyRewardSchedulesByAttr[3] = catalog.WeeklyRewardSchedulesByAttr[2]
			catalog.BossByBossId[3] = masterdata.BigHuntBossRow{BigHuntBossId: 3, AttributeType: 3}
			user := store.SeedUserState(1, "weekly", 1, model.ClientPlatform{})
			user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: lastWeek, AttributeType: 2}] = store.BigHuntWeeklyMaxScore{MaxScore: tc.score}
			cat := &runtime.Catalogs{BigHunt: catalog, QuestHandler: &questflow.QuestHandler{Granter: &store.PossessionGranter{}}}
			for attempt := 0; attempt < 2; attempt++ {
				_, received := receiveBigHuntWeeklyReward(cat, user, rollover)
				if !received || user.Materials[999] != tc.wantCount {
					t.Fatalf("attempt %d: count=%d, received=%v; want %d, true", attempt, user.Materials[999], received, tc.wantCount)
				}
			}
		})
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

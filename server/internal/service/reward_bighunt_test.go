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

func TestReceiveBigHuntRewardClaimsSeasonBestOncePerWeek(t *testing.T) {
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
		user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{BigHuntScheduleId: 1, BigHuntBossId: 1}] = store.BigHuntScheduleMaxScore{MaxScore: 100}
	}); err != nil {
		t.Fatal(err)
	}
	rewardServer := NewRewardServiceServer(repo, repo, holder)
	topServer := NewBigHuntServiceServer(repo, repo, holder)
	// Season-only accounts must discover the weekly reward on login, without
	// replaying or skipping a battle to register a weekly score.
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
	if len(statuses) != 1 || statuses[0].Week != lastWeek || statuses[0].Received {
		t.Fatalf("client cannot discover the weekly reward: %+v", statuses)
	}
	top, err := topServer.GetBigHuntTopData(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if top.IsReceivedWeeklyScoreReward {
		t.Fatal("unclaimed weekly reward is shown as received")
	}
	assertBigHuntWeeklyScoreResults(t, top.WeeklyScoreResult, 100, 100, 100)
	if len(top.WeeklyScoreReward) != 1 || top.WeeklyScoreReward[0].Count != 3 || len(top.LastWeekWeeklyScoreReward) != 1 || top.LastWeekWeeklyScoreReward[0].Count != 3 {
		t.Fatalf("reward previews do not use the season best: %+v", top)
	}
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
			t.Fatalf("attempt %d: weekly reward count = %d, want 3 from the season best", attempt, got)
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
		if len(user.BigHuntWeeklyMaxScores) != 0 {
			t.Fatalf("weekly scores were unexpectedly registered: %+v", user.BigHuntWeeklyMaxScores)
		}
		if !response.IsReceivedWeeklyScoreReward || len(response.LastWeekWeeklyScoreReward) != 1 || response.LastWeekWeeklyScoreReward[0].Count != 3 {
			t.Fatalf("incorrect weekly reward response: %+v", response)
		}
		assertBigHuntWeeklyScoreResults(t, response.WeeklyScoreResult, 100, 100, 100)
		top, err := topServer.GetBigHuntTopData(context.Background(), &emptypb.Empty{})
		if err != nil {
			t.Fatal(err)
		}
		if !top.IsReceivedWeeklyScoreReward || len(top.LastWeekWeeklyScoreReward) != 1 || top.LastWeekWeeklyScoreReward[0].Count != 3 {
			t.Fatalf("top data disagrees with weekly claim: %+v", top)
		}
	}
}

func assertBigHuntWeeklyScoreResults(t *testing.T, results []*pb.WeeklyScoreResult, before, after, current int64) {
	t.Helper()
	if len(results) != 1 || results[0].AttributeType != 2 || results[0].BeforeMaxScore != before || results[0].AfterMaxScore != after || results[0].CurrentMaxScore != current {
		t.Fatalf("incorrect completed-week comparison: %+v", results)
	}
}

func TestBigHuntWeeklyScoreResultsUseSeasonBest(t *testing.T) {
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
	results := buildBigHuntWeeklyScoreResults(catalog, *user)
	assertBigHuntWeeklyScoreResults(t, results, 200, 200, 200)
	if result := results[0]; result.BeforeAssetGradeIconId != 3 || result.AfterAssetGradeIconId != 3 || result.CurrentAssetGradeIconId != 3 {
		t.Fatalf("grade icons do not match the season score: %+v", result)
	}
	results = buildBigHuntWeeklyScoreResults(catalog, *store.SeedUserState(1, "empty", 1, model.ClientPlatform{}))
	assertBigHuntWeeklyScoreResults(t, results, 0, 0, 0)
	if result := results[0]; result.BeforeAssetGradeIconId != 0 || result.AfterAssetGradeIconId != 0 || result.CurrentAssetGradeIconId != 0 {
		t.Fatalf("missing score was assigned a grade: %+v", result)
	}
}

func TestEnsureBigHuntWeeklyStatusesPreservesReceipts(t *testing.T) {
	rollover := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC).UnixMilli()
	catalog := bigHuntWeeklyRewardTestCatalog()
	user := store.SeedUserState(1, "weekly", 1, model.ClientPlatform{})
	user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{BigHuntScheduleId: 1, BigHuntBossId: 1}] = store.BigHuntScheduleMaxScore{MaxScore: 100}
	claimed := store.BigHuntWeeklyStatus{IsReceivedWeeklyReward: true, LatestVersion: rollover - 1}
	user.BigHuntWeeklyStatuses[rollover-2*bigHuntWeekMillis] = claimed
	for _, now := range []int64{rollover - 1, rollover, rollover + 1} {
		ensureBigHuntWeeklyStatuses(catalog, user, now)
		if user.BigHuntWeeklyStatuses[rollover-2*bigHuntWeekMillis] != claimed {
			t.Fatal("existing receipt was changed")
		}
	}
	pending := user.BigHuntWeeklyStatuses[rollover-bigHuntWeekMillis]
	if len(user.BigHuntWeeklyStatuses) != 2 || pending.IsReceivedWeeklyReward || pending.LatestVersion != rollover || len(user.BigHuntWeeklyMaxScores) != 0 {
		t.Fatalf("pending status was not created idempotently: %+v", user.BigHuntWeeklyStatuses)
	}
	user.BigHuntWeeklyStatuses[rollover-bigHuntWeekMillis] = claimed
	ensureBigHuntWeeklyStatuses(catalog, user, rollover+1)
	if user.BigHuntWeeklyStatuses[rollover-bigHuntWeekMillis] != claimed {
		t.Fatal("login reset the current claim window's receipt")
	}
}

func TestBigHuntWeeklyRewardRequiresEligibleActiveSeasonScore(t *testing.T) {
	rollover := time.Date(2026, time.October, 12, 8, 0, 0, 0, time.UTC).UnixMilli()
	for _, tc := range []struct {
		name   string
		season int32
		score  int64
	}{
		{name: "no score"},
		{name: "below threshold", season: 1, score: 99},
		{name: "previous season only", season: 2, score: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := bigHuntWeeklyRewardTestCatalog()
			user := store.SeedUserState(1, "weekly", 1, model.ClientPlatform{})
			user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{BigHuntScheduleId: tc.season, BigHuntBossId: 1}] = store.BigHuntScheduleMaxScore{MaxScore: tc.score}
			// Old weekly records must not make an account eligible.
			user.BigHuntWeeklyMaxScores[store.BigHuntWeeklyScoreKey{BigHuntWeeklyVersion: rollover - bigHuntWeekMillis, AttributeType: 2}] = store.BigHuntWeeklyMaxScore{MaxScore: 200}
			ensureBigHuntWeeklyStatuses(catalog, user, rollover)
			cat := &runtime.Catalogs{BigHunt: catalog, QuestHandler: &questflow.QuestHandler{Granter: &store.PossessionGranter{}}}
			rewards, received := receiveBigHuntWeeklyReward(cat, user, rollover)
			if len(user.BigHuntWeeklyStatuses) != 0 || len(rewards) != 0 || received || user.Materials[999] != 0 {
				t.Fatalf("ineligible account received a weekly reward: rewards=%v received=%v statuses=%v", rewards, received, user.BigHuntWeeklyStatuses)
			}
		})
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
		{name: "before rollover", now: rollover - 1, score: 100, wantCount: 3, wantReceived: true},
		{name: "at rollover", now: rollover, score: 100, wantCount: 5, wantReceived: true},
		{name: "late in claim week", now: rollover + 6*24*60*60*1000, score: 100, wantCount: 5, wantReceived: true},
		{name: "no season score", now: rollover},
		{name: "below reward threshold", now: rollover, score: 99},
		{name: "already claimed", now: rollover, score: 100, alreadyClaimed: true, wantReceived: true},
		{name: "reward schedule not yet active", now: rollover, score: 100, futureOnly: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := bigHuntWeeklyRewardTestCatalog()
			// Resolve the reward schedule active when the weekly reward is claimed.
			catalog.WeeklyRewardSchedulesByAttr[2] = []masterdata.ScoreRewardScheduleEntry{
				{BigHuntScoreRewardGroupId: 2, StartDatetime: rollover},
				{BigHuntScoreRewardGroupId: 1, StartDatetime: lastWeek},
			}
			catalog.ScoreRewardThresholds[2] = []masterdata.ScoreRewardThreshold{{NecessaryScore: 100, BigHuntRewardGroupId: 2}}
			if tc.futureOnly {
				catalog.WeeklyRewardSchedulesByAttr[2] = []masterdata.ScoreRewardScheduleEntry{{BigHuntScoreRewardGroupId: 2, StartDatetime: rollover + 1}}
			}
			cat := &runtime.Catalogs{BigHunt: catalog, QuestHandler: &questflow.QuestHandler{Granter: &store.PossessionGranter{}}}
			user := store.SeedUserState(1, "weekly", 1, model.ClientPlatform{})
			user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{BigHuntScheduleId: 1, BigHuntBossId: 1}] = store.BigHuntScheduleMaxScore{MaxScore: tc.score}
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
				ws := user.BigHuntWeeklyStatuses[gametime.BusinessWeeklyVersion(tc.now)-bigHuntWeekMillis]
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

func TestReceiveBigHuntWeeklyRewardEmptyClaimDoesNotConsumeWeek(t *testing.T) {
	rollover := time.Date(2026, time.September, 14, 8, 0, 0, 0, time.UTC).UnixMilli()
	cat := &runtime.Catalogs{BigHunt: bigHuntWeeklyRewardTestCatalog(), QuestHandler: &questflow.QuestHandler{Granter: &store.PossessionGranter{}}}
	user := store.SeedUserState(1, "weekly", 1, model.ClientPlatform{})
	receiveBigHuntWeeklyReward(cat, user, rollover)
	user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{BigHuntScheduleId: 1, BigHuntBossId: 1}] = store.BigHuntScheduleMaxScore{MaxScore: 200}
	for range 2 {
		_, received := receiveBigHuntWeeklyReward(cat, user, rollover+1)
		if !received || user.Materials[999] != 5 {
			t.Fatalf("reward after earning a season score: count=%d, received=%v; want 5, true", user.Materials[999], received)
		}
	}
}

func TestBigHuntWeeklyRewardRefreshesAtMondayMidnightPST(t *testing.T) {
	for _, month := range []time.Month{time.January, time.July} {
		t.Run(month.String(), func(t *testing.T) {
			day := 5
			if month == time.July {
				day = 6
			}
			rollover := time.Date(2026, month, day, 8, 0, 0, 0, time.UTC).UnixMilli()
			cat := &runtime.Catalogs{BigHunt: bigHuntWeeklyRewardTestCatalog(), QuestHandler: &questflow.QuestHandler{Granter: &store.PossessionGranter{}}}
			user := store.SeedUserState(1, "weekly", 1, model.ClientPlatform{})
			key := store.BigHuntScheduleScoreKey{BigHuntScheduleId: 1, BigHuntBossId: 1}
			user.BigHuntScheduleMaxScores[key] = store.BigHuntScheduleMaxScore{MaxScore: 100}
			for _, tc := range []struct {
				now       int64
				wantCount int32
				wantGrant int
			}{
				{now: rollover - 1, wantCount: 3, wantGrant: 1},
				{now: rollover, wantCount: 6, wantGrant: 1},
				{now: rollover + 1, wantCount: 6},
				{now: rollover + bigHuntWeekMillis - 1, wantCount: 6},
				{now: rollover + bigHuntWeekMillis, wantCount: 9, wantGrant: 1},
				{now: rollover + bigHuntWeekMillis + 1, wantCount: 9},
			} {
				ensureBigHuntWeeklyStatuses(cat.BigHunt, user, tc.now)
				rewards, received := receiveBigHuntWeeklyReward(cat, user, tc.now)
				if !received || len(rewards) != tc.wantGrant || user.Materials[999] != tc.wantCount {
					t.Fatalf("at %s: reward entries=%d total=%d received=%v, want %d/%d/true", time.UnixMilli(tc.now), len(rewards), user.Materials[999], received, tc.wantGrant, tc.wantCount)
				}
			}
			// Improving the season score or changing seasons cannot grant twice
			// within the same weekly claim window.
			user.BigHuntScheduleMaxScores[key] = store.BigHuntScheduleMaxScore{MaxScore: 200}
			cat.BigHunt.ActiveScheduleId = 2
			user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{BigHuntScheduleId: 2, BigHuntBossId: 1}] = store.BigHuntScheduleMaxScore{MaxScore: 200}
			ensureBigHuntWeeklyStatuses(cat.BigHunt, user, rollover+bigHuntWeekMillis+2)
			rewards, received := receiveBigHuntWeeklyReward(cat, user, rollover+bigHuntWeekMillis+2)
			if !received || len(rewards) != 0 || user.Materials[999] != 9 {
				t.Fatal("season score change allowed a duplicate weekly claim")
			}
			rewards, received = receiveBigHuntWeeklyReward(cat, user, rollover+2*bigHuntWeekMillis)
			if !received || len(rewards) != 1 || user.Materials[999] != 14 || len(user.BigHuntWeeklyMaxScores) != 0 {
				t.Fatal("next week did not use the new season best without weekly scores")
			}
		})
	}
}

func TestBigHuntWeeklyRewardUsesHighestTierPerAttribute(t *testing.T) {
	rollover := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC).UnixMilli()
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
			user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{BigHuntScheduleId: 1, BigHuntBossId: 1}] = store.BigHuntScheduleMaxScore{MaxScore: tc.score}
			user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{BigHuntScheduleId: 1, BigHuntBossId: 2}] = store.BigHuntScheduleMaxScore{MaxScore: tc.score / 2}
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

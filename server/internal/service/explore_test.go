package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestExploreUnlockUsesServerProgress(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	catalog := &masterdata.ExploreCatalog{UnlockConditions: map[int32]masterdata.EntityMExploreUnlockCondition{1: {ExploreUnlockConditionId: 1, ExploreUnlockConditionType: 1, ConditionValue: 31}, 2: {ExploreUnlockConditionId: 2, ExploreUnlockConditionType: 2, ConditionValue: 100000}}, LowerDifficulty: map[int32]int32{11: 1}}
	if exploreUnlocked(user, catalog, masterdata.EntityMExplore{ExploreId: 1, ExploreUnlockConditionId: 1}) {
		t.Fatal("uncleared quest unlocked explore")
	}
	user.Quests[500] = store.UserQuestState{QuestStateType: model.UserQuestStateTypeCleared}
	if exploreUnlocked(user, catalog, masterdata.EntityMExplore{ExploreId: 1, ExploreUnlockConditionId: 1}) {
		t.Fatal("quest from a main quest sequence id unlocked explore")
	}
	user.Quests[31] = store.UserQuestState{QuestStateType: model.UserQuestStateTypeCleared}
	if !exploreUnlocked(user, catalog, masterdata.EntityMExplore{ExploreId: 1, ExploreUnlockConditionId: 1}) {
		t.Fatal("cleared quest did not unlock explore")
	}
	user.ExploreScores[1] = store.ExploreScoreState{MaxScore: 99999}
	if exploreUnlocked(user, catalog, masterdata.EntityMExplore{ExploreId: 11, ExploreUnlockConditionId: 2}) {
		t.Fatal("insufficient normal score unlocked hard explore")
	}
}

func TestCalculateExploreRewardUsesGradeAndLotteryCount(t *testing.T) {
	gradeScores := []masterdata.EntityMExploreGradeScore{
		{ExploreId: 1, NecessaryScore: 100_000, ExploreGradeId: 101},
		{ExploreId: 1, NecessaryScore: 50_000, ExploreGradeId: 102},
		{ExploreId: 1, NecessaryScore: 20_000, ExploreGradeId: 103},
		{ExploreId: 1, NecessaryScore: 10_000, ExploreGradeId: 104},
		{ExploreId: 1, NecessaryScore: 5_000, ExploreGradeId: 105},
		{ExploreId: 1, NecessaryScore: 0, ExploreGradeId: 106},
	}
	catalog := &masterdata.ExploreCatalog{GradeScores: map[int32][]masterdata.EntityMExploreGradeScore{1: gradeScores}}

	tests := []struct {
		name       string
		score      int32
		multiplier int32
		want       exploreReward
	}{
		{name: "SS", score: 100_000, multiplier: 1, want: exploreReward{staminaCount: 50, goldCount: 100_000}},
		{name: "S", score: 50_000, multiplier: 1, want: exploreReward{staminaCount: 50, goldCount: 100_000}},
		{name: "A", score: 20_000, multiplier: 1, want: exploreReward{staminaCount: 40, goldCount: 80_000}},
		{name: "B", score: 10_000, multiplier: 1, want: exploreReward{staminaCount: 30, goldCount: 60_000}},
		{name: "C", score: 5_000, multiplier: 1, want: exploreReward{staminaCount: 20, goldCount: 40_000}},
		{name: "D", score: 0, multiplier: 1, want: exploreReward{staminaCount: 10, goldCount: 20_000}},
		{name: "hard SS", score: 100_000, multiplier: 11, want: exploreReward{staminaCount: 550, goldCount: 1_100_000}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			explore := masterdata.EntityMExplore{ExploreId: 1, RewardLotteryCount: test.multiplier}
			if got := calculateExploreReward(catalog, explore, test.score); got != test.want {
				t.Fatalf("reward = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestFinishExploreGrantsHardModeStaminaAndGold(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("explore-reward", model.ClientPlatform{})
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
	goldItemId := holder.Get().GameConfig.ConsumableItemIdForGold

	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.Explore.PlayingExploreId = 11
		user.Status.StaminaMilliValue = 12_000
		user.ConsumableItems[goldItemId] = 7
	}); err != nil {
		t.Fatal(err)
	}

	server := NewExploreServiceServer(repo, repo, holder)
	response, err := server.FinishExplore(context.Background(), &pb.FinishExploreRequest{ExploreId: 11, Score: 20_000})
	if err != nil {
		t.Fatal(err)
	}
	if response.AcquireStaminaCount != 440 {
		t.Fatalf("acquired stamina = %d, want 440", response.AcquireStaminaCount)
	}
	if len(response.ExploreReward) != 1 {
		t.Fatalf("explore rewards = %d, want 1", len(response.ExploreReward))
	}
	goldReward := response.ExploreReward[0]
	if goldReward.PossessionType != int32(model.PossessionTypeConsumableItem) || goldReward.PossessionId != goldItemId || goldReward.Count != 880_000 {
		t.Fatalf("gold reward = %#v", goldReward)
	}

	user, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if user.Status.StaminaMilliValue != 452_000 {
		t.Fatalf("stamina milli value = %d, want 452000", user.Status.StaminaMilliValue)
	}
	if user.ConsumableItems[goldItemId] != 880_007 {
		t.Fatalf("gold = %d, want 880007", user.ConsumableItems[goldItemId])
	}
	if user.Materials[100001] != 0 {
		t.Fatalf("legacy material reward remains: %d", user.Materials[100001])
	}
	if user.Status.Exp != 0 {
		t.Fatalf("experience = %d, want unchanged", user.Status.Exp)
	}
}

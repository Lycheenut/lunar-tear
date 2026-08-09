package service

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
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

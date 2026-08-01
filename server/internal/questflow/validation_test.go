package questflow

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestHandleQuestSkipBulkAggregatesDuplicateQuestLimits(t *testing.T) {
	h := &QuestHandler{
		QuestCatalog: &masterdata.QuestCatalog{
			QuestById:            map[int32]masterdata.EntityMQuest{10: {QuestId: 10, Stamina: 1, IsUsableSkipTicket: true, DailyClearableCount: 5}},
			MaxStaminaByLevel:    map[int32]int32{1: 100},
			MissionIdsByQuestId:  map[int32][]int32{},
			BattleDropsByQuestId: map[int32][]masterdata.BattleDropInfo{},
		},
		Config:  &masterdata.GameConfig{ConsumableItemIdForQuestSkipTicket: 7},
		Granter: &store.PossessionGranter{},
	}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Status.Level = 1
	user.Status.StaminaMilliValue = 100_000
	user.Status.StaminaUpdateDatetime = 100
	user.ConsumableItems[7] = 10
	user.Quests[10] = store.UserQuestState{QuestId: 10, QuestStateType: model.UserQuestStateTypeCleared, DailyClearCount: 2}

	if _, err := h.HandleQuestSkipBulk(user, []int32{10, 10}, []int32{2, 2}, 100); err == nil {
		t.Fatal("duplicate quest entries bypassed the aggregate daily limit")
	}
	if user.Quests[10].DailyClearCount != 2 || user.ConsumableItems[7] != 10 || user.Status.StaminaMilliValue != 100_000 {
		t.Fatal("rejected bulk skip partially mutated user state")
	}
}

func TestEventChapterAvailableUsesNormalizedUnlockQuests(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		EventChapterById:          map[int32]masterdata.EntityMEventQuestChapter{20: {EventQuestChapterId: 20, EventQuestType: 3, StartDatetime: 10, EndDatetime: 30}},
		EventUnlockQuestIdsByType: map[int32][]int32{3: {11}},
	}}
	user := &store.UserState{}
	user.EnsureMaps()
	if err := h.EventChapterAvailable(user, 20, 20); err == nil {
		t.Fatal("locked event chapter was accepted")
	}
	user.Quests[11] = store.UserQuestState{QuestId: 11, QuestStateType: model.UserQuestStateTypeCleared}
	if err := h.EventChapterAvailable(user, 20, 20); err != nil {
		t.Fatalf("cleared normalized unlock quest did not unlock event: %v", err)
	}
}

package questflow

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestStrayScarecrowUnlockFlow(t *testing.T) {
	h := &QuestHandler{
		QuestCatalog: &masterdata.QuestCatalog{
			QuestById: map[int32]masterdata.EntityMQuest{
				334: {QuestId: 334},
				382: {QuestId: 382, QuestReleaseConditionListId: 385},
				385: {QuestId: 385, QuestReleaseConditionListId: 120140},
				434: {QuestId: 434},
			},
			QuestReleaseConditionsByListId: map[int32]masterdata.QuestReleaseConditionGroup{
				385: {
					OperationType: model.ConditionOperationTypeAnd,
					Conditions: []masterdata.QuestReleaseCondition{{
						ConditionType: model.QuestReleaseConditionTypeQuestChallenge,
						QuestId:       385,
					}},
				},
				120140: {
					OperationType: model.ConditionOperationTypeOr,
					Conditions: []masterdata.QuestReleaseCondition{
						{ConditionType: model.QuestReleaseConditionTypeQuestClear, QuestId: 334},
						{ConditionType: model.QuestReleaseConditionTypeQuestClear, QuestId: 434},
					},
				},
			},
			MaxStaminaByLevel:    map[int32]int32{1: 100},
			MissionIdsByQuestId:  map[int32][]int32{},
			BattleDropsByQuestId: map[int32][]masterdata.BattleDropInfo{},
		},
	}
	user := store.SeedUserState(1, "scarecrow", 1, model.ClientPlatform{})
	user.Status.Level = 1
	user.Status.StaminaMilliValue = 100_000

	if err := h.HandleExtraQuestStart(user, 385, 1, 100); err == nil {
		t.Fatal("quest 385 started before either prerequisite quest was cleared")
	}
	if user.ExtraQuest.CurrentQuestId != 0 {
		t.Fatal("rejected quest start mutated the current extra quest")
	}

	user.Quests[434] = store.UserQuestState{QuestId: 434, QuestStateType: model.UserQuestStateTypeCleared}
	if err := h.HandleExtraQuestStart(user, 385, 1, 100); err != nil {
		t.Fatalf("quest 385 did not start after clearing OR prerequisite 434: %v", err)
	}
	if err := h.HandleExtraQuestStart(user, 382, 1, 100); err == nil {
		t.Fatal("quest 382 started before quest 385 was challenged")
	}

	h.HandleExtraQuestFinish(user, 385, true, false, 200)
	if got := user.Quests[385].QuestStateType; got != model.UserQuestStateTypeChallenged {
		t.Fatalf("retired quest 385 state = %d, want challenged", got)
	}
	if err := h.HandleExtraQuestStart(user, 382, 1, 200); err != nil {
		t.Fatalf("quest 382 did not start after quest 385 was challenged: %v", err)
	}
}

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

	if _, err := h.HandleQuestSkipBulk(user, []int32{10, 10}, []int32{1, 1}, []int32{0, 0}, []int32{2, 2}, 100); err == nil {
		t.Fatal("duplicate quest entries bypassed the aggregate daily limit")
	}
	if user.Quests[10].DailyClearCount != 2 || user.ConsumableItems[7] != 10 || user.Status.StaminaMilliValue != 100_000 {
		t.Fatal("rejected bulk skip partially mutated user state")
	}
}

func TestDailyClearLimitResetsAfterServerDateChanges(t *testing.T) {
	h := &QuestHandler{
		QuestCatalog: &masterdata.QuestCatalog{
			QuestById:            map[int32]masterdata.EntityMQuest{10: {QuestId: 10, Stamina: 1, IsUsableSkipTicket: true, DailyClearableCount: 2}},
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
	user.Status.StaminaUpdateDatetime = 2 * 24 * 60 * 60 * 1000
	user.ConsumableItems[7] = 10
	user.Quests[10] = store.UserQuestState{
		QuestId:           10,
		QuestStateType:    model.UserQuestStateTypeCleared,
		ClearCount:        2,
		DailyClearCount:   2,
		LastClearDatetime: 100,
	}

	now := int64(2 * 24 * 60 * 60 * 1000)
	if _, err := h.HandleQuestSkip(user, 10, int32(model.QuestTypeMain), 0, 1, now); err != nil {
		t.Fatalf("previous day's clears still consumed today's limit: %v", err)
	}
	if got := user.Quests[10].DailyClearCount; got != 1 {
		t.Fatalf("daily count = %d, want 1 after date rollover", got)
	}
}

func TestNormalQuestStartHonorsDailyClearLimit(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById:         map[int32]masterdata.EntityMQuest{10: {QuestId: 10, DailyClearableCount: 1}},
		MaxStaminaByLevel: map[int32]int32{1: 100},
	}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Status.Level = 1
	user.Quests[10] = store.UserQuestState{QuestId: 10, DailyClearCount: 1, LastClearDatetime: 100}
	if err := h.validateQuestStart(user, 10, 100); err == nil {
		t.Fatal("normal start bypassed daily clear limit")
	}
}

func TestValidateQuestContinuationAllowsClearedMenuReplay(t *testing.T) {
	const (
		questId int32 = 13
		sceneId int32 = 22
	)
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById:                      map[int32]masterdata.EntityMQuest{questId: {QuestId: questId, IsCountedAsQuest: true}},
		SceneById:                      map[int32]masterdata.EntityMQuestScene{sceneId: {QuestSceneId: sceneId, QuestId: questId}},
		SceneIdsByQuestId:              map[int32][]int32{questId: {sceneId}},
		BattleOnlyTargetSceneByQuestId: map[int32]int32{questId: sceneId},
		MaxStaminaByLevel:              map[int32]int32{1: 100},
	}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Status.Level = 1
	user.Quests[questId] = store.UserQuestState{
		QuestId:        questId,
		QuestStateType: model.UserQuestStateTypeCleared,
		ClearCount:     1,
	}

	if err := h.HandleQuestStart(user, questId, true, false, 1, 100); err != nil {
		t.Fatalf("start cleared menu replay: %v", err)
	}
	if got := user.Quests[questId].QuestStateType; got != model.UserQuestStateTypeCleared {
		t.Fatalf("stored quest state = %d, want cleared", got)
	}
	if err := h.ValidateMainQuestContinuation(user, questId); err != nil {
		t.Fatalf("validate cleared menu replay: %v", err)
	}
}

func TestValidateMainQuestContinuationAllowsMatchingActiveProgress(t *testing.T) {
	const (
		questId int32 = 13
		sceneId int32 = 22
	)
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			questId: {QuestId: questId},
		},
		SceneById: map[int32]masterdata.EntityMQuestScene{
			sceneId: {QuestSceneId: sceneId, QuestId: questId},
		},
	}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[questId] = store.UserQuestState{QuestId: questId, QuestStateType: model.UserQuestStateTypeActive}
	user.MainQuest.ProgressQuestSceneId = sceneId

	if err := h.ValidateMainQuestContinuation(user, questId); err != nil {
		t.Fatalf("validate matching active progress: %v", err)
	}
}

func TestHandleQuestRestartRejectsMismatchedProgressWithoutMutation(t *testing.T) {
	const (
		requestedQuestId int32 = 100021
		progressQuestId  int32 = 210019
		progressSceneId  int32 = 210019
	)
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			requestedQuestId: {QuestId: requestedQuestId},
			progressQuestId:  {QuestId: progressQuestId},
		},
		SceneById: map[int32]masterdata.EntityMQuestScene{
			progressSceneId: {QuestSceneId: progressSceneId, QuestId: progressQuestId},
		},
	}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[requestedQuestId] = store.UserQuestState{
		QuestId:             requestedQuestId,
		QuestStateType:      model.UserQuestStateTypeActive,
		LatestStartDatetime: 10,
	}
	user.MainQuest.ProgressQuestSceneId = progressSceneId

	if err := h.HandleQuestRestart(user, requestedQuestId, 100); err == nil {
		t.Fatal("mismatched progress quest was restarted")
	}
	if got := user.Quests[requestedQuestId].LatestStartDatetime; got != 10 {
		t.Fatalf("latest start datetime = %d, want unchanged value 10", got)
	}
}

func TestValidateQuestContinuationRejectsUnrelatedClearedQuest(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			13: {QuestId: 13},
			14: {QuestId: 14},
		},
		SceneById: map[int32]masterdata.EntityMQuestScene{
			22: {QuestSceneId: 22, QuestId: 13},
		},
	}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[14] = store.UserQuestState{QuestId: 14, QuestStateType: model.UserQuestStateTypeCleared}
	user.MainQuest.SavedContext.Active = true
	user.MainQuest.ProgressQuestSceneId = 22

	if err := h.ValidateQuestContinuation(user, 14); err == nil {
		t.Fatal("unrelated cleared quest bypassed continuation validation")
	}
}

func TestHandleQuestRestartRejectsClearedQuestWithoutMutation(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{10: {QuestId: 10}},
	}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[10] = store.UserQuestState{QuestId: 10, QuestStateType: model.UserQuestStateTypeCleared, ClearCount: 1}

	if err := h.HandleQuestRestart(user, 10, 100); err == nil {
		t.Fatal("cleared main quest was restarted")
	}
	if got := user.Quests[10].QuestStateType; got != model.UserQuestStateTypeCleared {
		t.Fatalf("quest state = %d, want cleared", got)
	}
}

func TestHandleExtraQuestRestartRejectsClearedQuestWithoutMutation(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{10: {QuestId: 10}},
	}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[10] = store.UserQuestState{QuestId: 10, QuestStateType: model.UserQuestStateTypeCleared, ClearCount: 1}

	if err := h.HandleExtraQuestRestart(user, 10, 100); err == nil {
		t.Fatal("cleared extra quest was restarted")
	}
	if got := user.Quests[10].QuestStateType; got != model.UserQuestStateTypeCleared {
		t.Fatalf("quest state = %d, want cleared", got)
	}
	if got := user.ExtraQuest.CurrentQuestId; got != 0 {
		t.Fatalf("current extra quest = %d, want 0", got)
	}
}

func TestEventChapterAvailableUsesNormalizedUnlockQuests(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		EventChapterById:      map[int32]masterdata.EntityMEventQuestChapter{20: {EventQuestChapterId: 20, EventQuestType: 3, StartDatetime: 10, EndDatetime: 30}},
		EventUnlockConditions: []masterdata.EventQuestUnlockCondition{{EventQuestType: 3, RequiredQuestId: 11}},
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

func TestEventQuestValidationAllowsQuestSharedWithExpiredChapter(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			100: {QuestId: 100},
			200: {QuestId: 200},
		},
		EventChapterById: map[int32]masterdata.EntityMEventQuestChapter{
			6:  {EventQuestChapterId: 6, StartDatetime: 10, EndDatetime: 30},
			10: {EventQuestChapterId: 10, StartDatetime: 1, EndDatetime: 9},
		},
		EventQuestIdsByChapterId: map[int32][]int32{
			6:  {100},
			10: {100},
			11: {200},
		},
	}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.EventQuest.CurrentEventQuestChapterId = 6
	user.EventQuest.CurrentQuestId = 100
	user.Quests[100] = store.UserQuestState{QuestId: 100, QuestStateType: model.UserQuestStateTypeActive}

	if err := h.validateEventQuest(user, 6, 100, 20); err != nil {
		t.Fatalf("start shared event quest: %v", err)
	}
	if err := h.ValidateEventQuestContinuation(user, 6, 100, 20); err != nil {
		t.Fatalf("continue shared event quest: %v", err)
	}
	if err := h.validateEventQuest(user, 6, 200, 20); err == nil {
		t.Fatal("quest from another chapter was accepted")
	}
}

func TestEventUnlockConditionsKeepCharacterAndQuestScope(t *testing.T) {
	catalog := &masterdata.QuestCatalog{
		EventChapterById:             map[int32]masterdata.EntityMEventQuestChapter{20: {EventQuestChapterId: 20, EventQuestType: 3}},
		EventQuestIdsByChapterId:     map[int32][]int32{20: {100}},
		EventCharacterIdsByChapterId: map[int32]map[int32]bool{20: {5: true}},
		EventUnlockConditions: []masterdata.EventQuestUnlockCondition{
			{EventQuestType: 3, CharacterId: 5, RequiredQuestId: 11},
			{EventQuestType: 3, CharacterId: 6, RequiredQuestId: 12},
			{EventQuestType: 3, QuestId: 100, RequiredQuestId: 13},
			{EventQuestType: 3, QuestId: 101, RequiredQuestId: 14},
		},
	}
	got := catalog.EventUnlockQuestIdsForChapter(20)
	if len(got) != 2 || got[0] != 11 || got[1] != 13 {
		t.Fatalf("scoped unlock quests = %v, want [11 13]", got)
	}
}

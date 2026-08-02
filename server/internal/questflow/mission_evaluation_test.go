package questflow

import (
	"testing"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestQuestMissionUsesBattleResultAndFailureDoesNotClear(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById:                         map[int32]masterdata.EntityMQuest{10: {QuestId: 10}},
		MissionById:                       map[int32]masterdata.EntityMQuestMission{20: {QuestMissionId: 20, QuestMissionConditionType: int32(model.QuestMissionConditionTypeCriticalCountGe), ConditionValue: 3}},
		MissionIdsByQuestId:               map[int32][]int32{10: {20}},
		MissionRewardsByMissionId:         map[int32][]masterdata.EntityMQuestMissionReward{},
		FirstClearRewardsByGroupId:        map[int32][]masterdata.EntityMQuestFirstClearRewardGroup{},
		FirstClearRewardSwitchesByQuestId: map[int32][]masterdata.EntityMQuestFirstClearRewardSwitch{},
		ReplayFlowRewardsByGroupId:        map[int32][]masterdata.EntityMQuestReplayFlowRewardGroup{},
		PickupRewardIdsByGroupId:          map[int32][]int32{},
		BattleDropRewardById:              map[int32]masterdata.EntityMBattleDropReward{},
	}, Config: &masterdata.GameConfig{}, Granter: &store.PossessionGranter{}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[10] = store.UserQuestState{QuestId: 10, QuestStateType: model.UserQuestStateTypeActive, LatestStartDatetime: 10}
	user.Battle.LastFinishedAt = 11
	user.Battle.MissionDetail = store.BattleMissionDetailState{IsValid: true, CriticalCount: 2}
	if got := h.evaluateFinishOutcome(user, 10, campaign.QuestTarget{}, 12); len(got.ClearedQuestMissionIds) != 0 {
		t.Fatalf("mission cleared below threshold: %v", got.ClearedQuestMissionIds)
	}
	user.Battle.MissionDetail.CriticalCount = 3
	if got := h.evaluateFinishOutcome(user, 10, campaign.QuestTarget{}, 12); len(got.ClearedQuestMissionIds) != 1 || got.ClearedQuestMissionIds[0] != 20 {
		t.Fatalf("mission did not clear at threshold: %v", got.ClearedQuestMissionIds)
	}
	h.HandleQuestFinish(user, 10, true, false, 13)
	if user.QuestMissions[store.QuestMissionKey{QuestId: 10, QuestMissionId: 20}].IsClear {
		t.Fatal("retired quest cleared its mission")
	}
}

func TestClearedQuestMissionCountUsesStoredState(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{MissionIdsByQuestId: map[int32][]int32{10: {1, 2}, 20: {3}}}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.QuestMissions[store.QuestMissionKey{QuestId: 10, QuestMissionId: 1}] = store.UserQuestMissionState{IsClear: true}
	user.QuestMissions[store.QuestMissionKey{QuestId: 20, QuestMissionId: 3}] = store.UserQuestMissionState{IsClear: true}
	if got := h.ClearedQuestMissionCount(user, []int32{10}); got != 1 {
		t.Fatalf("single-quest count = %d, want 1", got)
	}
	if got := h.ClearedQuestMissionCount(user, []int32{10, 20}); got != 2 {
		t.Fatalf("multi-quest count = %d, want 2", got)
	}
}

func TestQuestMissionRejectsMissingOrIncompleteBattleDetail(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{MissionConditionValuesByGroupId: map[int32][]int32{}}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[10] = store.UserQuestState{QuestId: 10, LatestStartDatetime: 10}
	user.Battle.LastFinishedAt = 11
	noDeaths := masterdata.EntityMQuestMission{QuestMissionConditionType: int32(model.QuestMissionConditionTypeLessThanOrEqualXPeopleNotAlive), ConditionValue: 0}
	minHp := masterdata.EntityMQuestMission{QuestMissionConditionType: int32(model.QuestMissionConditionTypeMinHpPercentageGe), ConditionValue: 100}

	if h.questMissionSatisfied(user, 10, noDeaths) || h.questMissionSatisfied(user, 10, minHp) {
		t.Fatal("missing battle detail satisfied a battle-derived mission")
	}
	user.Battle.MissionDetail.IsValid = true
	if !h.questMissionSatisfied(user, 10, noDeaths) {
		t.Fatal("valid zero-death battle did not satisfy zero-death mission")
	}
	if h.questMissionSatisfied(user, 10, minHp) {
		t.Fatal("HP mission passed without any valid costume result")
	}
}

func TestReplayRewardGroupIsClaimedOnce(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById:                  map[int32]masterdata.EntityMQuest{10: {QuestId: 10, QuestReplayFlowRewardGroupId: 7}},
		ReplayFlowRewardsByGroupId: map[int32][]masterdata.EntityMQuestReplayFlowRewardGroup{7: {{QuestReplayFlowRewardGroupId: 7, PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 1, Count: 1}}},
		FirstClearRewardsByGroupId: map[int32][]masterdata.EntityMQuestFirstClearRewardGroup{}, FirstClearRewardSwitchesByQuestId: map[int32][]masterdata.EntityMQuestFirstClearRewardSwitch{}, PickupRewardIdsByGroupId: map[int32][]int32{}, BattleDropRewardById: map[int32]masterdata.EntityMBattleDropReward{},
	}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.EnsureMaps()
	user.Quests[10] = store.UserQuestState{QuestId: 10}
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	first := h.evaluateFinishOutcome(user, 10, campaign.QuestTarget{}, 2)
	if first.ReplayRewardGroupId != 7 || len(first.ReplayFlowFirstClearRewards) != 1 {
		t.Fatal("first replay reward was not produced")
	}
	user.QuestReplayFlowRewards[7] = store.QuestReplayFlowRewardState{QuestReplayFlowRewardGroupId: 7}
	second := h.evaluateFinishOutcome(user, 10, campaign.QuestTarget{}, 3)
	if second.ReplayRewardGroupId != 0 || len(second.ReplayFlowFirstClearRewards) != 0 {
		t.Fatal("replay reward was produced twice")
	}
}

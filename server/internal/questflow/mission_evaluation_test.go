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

func TestQuestMissionPowerBonusBoundaryAndExclusions(t *testing.T) {
	tests := []struct {
		name             string
		deckPower        int32
		isBigWinTarget   bool
		restrictionGroup int32
		deckType         model.DeckType
		wantClear        bool
	}{
		{name: "below threshold", deckPower: 129999, isBigWinTarget: true, deckType: model.DeckTypeQuest},
		{name: "at threshold", deckPower: 130000, isBigWinTarget: true, deckType: model.DeckTypeQuest, wantClear: true},
		{name: "challenge content excluded", deckPower: 130000, deckType: model.DeckTypeQuest},
		{name: "restricted quest deck", deckPower: 130000, isBigWinTarget: true, restrictionGroup: 7, deckType: model.DeckTypeRestrictedQuest, wantClear: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := questMissionPowerBonusTestHandler(tt.isBigWinTarget, tt.restrictionGroup)
			user := questMissionPowerBonusTestUser(tt.deckType, tt.deckPower)

			got := h.evaluateFinishOutcome(user, 10, campaign.QuestTarget{}, 12)
			if tt.wantClear {
				if len(got.ClearedQuestMissionIds) != 2 || got.ClearedQuestMissionIds[0] != 20 || got.ClearedQuestMissionIds[1] != 21 {
					t.Fatalf("cleared mission ids = %v, want [20 21]", got.ClearedQuestMissionIds)
				}
				if len(got.MissionClearRewards) != 1 || len(got.MissionClearCompleteRewards) != 1 {
					t.Fatalf("mission rewards = %d complete rewards = %d, want 1/1", len(got.MissionClearRewards), len(got.MissionClearCompleteRewards))
				}
				if !got.IsBigWin || len(got.BigWinClearedQuestMissionIds) != 1 || got.BigWinClearedQuestMissionIds[0] != 21 {
					t.Fatalf("big win = %v ids = %v, want true/[21]", got.IsBigWin, got.BigWinClearedQuestMissionIds)
				}
				return
			}

			if len(got.ClearedQuestMissionIds) != 0 || got.IsBigWin {
				t.Fatalf("power bonus unexpectedly applied: cleared=%v bigWin=%v", got.ClearedQuestMissionIds, got.IsBigWin)
			}
		})
	}
}

func TestQuestMissionPowerBonusPersistsMissionsAndGrantsRewards(t *testing.T) {
	h := questMissionPowerBonusTestHandler(true, 0)
	user := questMissionPowerBonusTestUser(model.DeckTypeQuest, 130000)

	outcome := h.HandleQuestFinish(user, 10, false, false, 12)
	if !outcome.IsBigWin {
		t.Fatal("power bonus finish was not marked as big win")
	}
	for _, missionId := range []int32{20, 21} {
		mission := user.QuestMissions[store.QuestMissionKey{QuestId: 10, QuestMissionId: missionId}]
		if !mission.IsClear || mission.ProgressValue != 1 {
			t.Fatalf("mission %d state = %+v, want cleared", missionId, mission)
		}
	}
	if user.ConsumableItems[100] != 2 || user.ConsumableItems[101] != 3 {
		t.Fatalf("mission reward counts = %d/%d, want 2/3", user.ConsumableItems[100], user.ConsumableItems[101])
	}
}

func TestQuestFinishCountsBossesFromQuestMaster(t *testing.T) {
	h := questMissionPowerBonusTestHandler(false, 0)
	h.BossCountByQuestId = map[int32]int32{10: 2}
	user := questMissionPowerBonusTestUser(model.DeckTypeQuest, 100000)

	h.HandleQuestFinish(user, 10, false, false, 12)
	for _, event := range user.PendingMissionEvents {
		if event.ConditionType == int32(model.MissionClearConditionTypeDefeatBossCount) {
			if event.Count != 2 || event.TargetId != 10 {
				t.Fatalf("boss mission event = %+v, want count 2 for quest 10", event)
			}
			return
		}
	}
	t.Fatal("quest finish did not emit a boss mission event")
}

func questMissionPowerBonusTestHandler(isBigWinTarget bool, restrictionGroup int32) *QuestHandler {
	return &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{10: {
			QuestId: 10, RecommendedDeckPower: 100000, QuestDeckRestrictionGroupId: restrictionGroup,
			QuestMissionGroupId: 1, IsBigWinTarget: isBigWinTarget,
		}},
		MissionById: map[int32]masterdata.EntityMQuestMission{
			20: {QuestMissionId: 20, QuestMissionConditionType: int32(model.QuestMissionConditionTypeCriticalCountGe), ConditionValue: 99, QuestMissionRewardId: 200},
			21: {QuestMissionId: 21, QuestMissionConditionType: int32(model.QuestMissionConditionTypeComplete), QuestMissionRewardId: 201},
		},
		MissionIdsByQuestId: map[int32][]int32{10: {20, 21}},
		MissionRewardsByMissionId: map[int32][]masterdata.EntityMQuestMissionReward{
			200: {{QuestMissionRewardId: 200, PossessionType: int32(model.PossessionTypeConsumableItem), PossessionId: 100, Count: 2}},
			201: {{QuestMissionRewardId: 201, PossessionType: int32(model.PossessionTypeConsumableItem), PossessionId: 101, Count: 3}},
		},
		FirstClearRewardsByGroupId:        map[int32][]masterdata.EntityMQuestFirstClearRewardGroup{},
		FirstClearRewardSwitchesByQuestId: map[int32][]masterdata.EntityMQuestFirstClearRewardSwitch{},
		ReplayFlowRewardsByGroupId:        map[int32][]masterdata.EntityMQuestReplayFlowRewardGroup{},
		PickupRewardIdsByGroupId:          map[int32][]int32{},
		BattleDropRewardById:              map[int32]masterdata.EntityMBattleDropReward{},
		UserExpThresholds:                 []int32{0},
	}, Config: &masterdata.GameConfig{QuestMissionBigWinBonusPower: 30000}, Granter: &store.PossessionGranter{}}
}

func questMissionPowerBonusTestUser(deckType model.DeckType, deckPower int32) *store.UserState {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[10] = store.UserQuestState{
		QuestId: 10, QuestStateType: model.UserQuestStateTypeActive, UserDeckNumber: 2, LatestStartDatetime: 10,
	}
	user.Decks[store.DeckKey{DeckType: deckType, UserDeckNumber: 2}] = store.DeckState{Power: deckPower}
	return user
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

func TestRecordQuestClearCapturesDeckCharacters(t *testing.T) {
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		CostumeById: map[int32]masterdata.EntityMCostume{900: {CostumeId: 900, CharacterId: 1015}},
	}}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 7}] = store.DeckState{
		DeckType: model.DeckTypeQuest, UserDeckNumber: 7, UserDeckCharacterUuid01: "dc",
	}
	user.DeckCharacters["dc"] = store.DeckCharacterState{UserDeckCharacterUuid: "dc", UserCostumeUuid: "costume"}
	user.Costumes["costume"] = store.CostumeState{UserCostumeUuid: "costume", CostumeId: 900}
	state := store.UserQuestState{QuestId: 20034, UserDeckNumber: 7}

	h.recordQuestClears(user, &state, 20034, 1, true, 100)
	if len(user.PendingMissionEvents) != 2 {
		t.Fatalf("mission event count = %d, want 2", len(user.PendingMissionEvents))
	}
	contextEvent := user.PendingMissionEvents[0]
	if contextEvent.ConditionType != int32(model.MissionClearConditionTypeQuestClearByCount) || !contextEvent.QuestClearWithDeck {
		t.Fatalf("contextual quest-clear event = %+v", contextEvent)
	}
	if len(contextEvent.DeckCostumeIds) != 1 || contextEvent.DeckCostumeIds[0] != 900 {
		t.Fatalf("recorded quest-clear costume context = %+v", contextEvent)
	}
	event := user.PendingMissionEvents[1]
	if event.TargetId != 20034 || len(event.DeckCharacterIds) != 1 || event.DeckCharacterIds[0] != 1015 {
		t.Fatalf("recorded quest-clear context = %+v", event)
	}
}

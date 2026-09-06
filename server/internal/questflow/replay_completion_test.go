package questflow

import (
	"fmt"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func replayCompletionFixture(replayId, completionId int32) (*QuestHandler, *store.UserState) {
	quest := masterdata.EntityMQuest{
		QuestId: completionId, IsCountedAsQuest: true, RecommendedDeckPower: 57000,
		IsBigWinTarget: true, QuestFirstClearRewardGroupId: 10,
		UserExp: 342, CharacterExp: 17, CostumeExp: 1540, Gold: 850,
	}
	replay := quest
	replay.QuestId = replayId
	replay.QuestReplayFlowRewardGroupId = 20
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById:                     map[int32]masterdata.EntityMQuest{completionId: quest, replayId: replay},
		MainFlowQuestIdByQuestId:      map[int32]int32{16: 16, completionId: 16, replayId: 16},
		SubFlowQuestIdByReplayQuestId: map[int32]int32{replayId: completionId},
		ReplayQuestIdsByMainQuestId:   map[int32][]int32{16: {replayId}},
		SceneIdsByQuestId:             map[int32][]int32{16: {266, 267}},
		SceneById: map[int32]masterdata.EntityMQuestScene{
			266: {QuestSceneId: 266, QuestId: 16, SortOrder: 1},
			267: {QuestSceneId: 267, QuestId: 16, SortOrder: 2},
		},
		MissionIdsByQuestId: map[int32][]int32{completionId: {103, 21001, 2, 1}, replayId: {103, 21001, 2, 1}},
		MissionById: map[int32]masterdata.EntityMQuestMission{
			103:   {QuestMissionId: 103, QuestMissionConditionType: int32(model.QuestMissionConditionTypeSpecifiedAttributeMainWeaponIsInDeck), ConditionValue: 6, QuestMissionRewardId: 1},
			21001: {QuestMissionId: 21001, QuestMissionConditionType: int32(model.QuestMissionConditionTypeGreaterThanOrEqualXWeaponSkillUseCount), ConditionValue: 1, QuestMissionRewardId: 1},
			2:     {QuestMissionId: 2, QuestMissionConditionType: int32(model.QuestMissionConditionTypeLessThanOrEqualXPeopleNotAlive), QuestMissionRewardId: 1},
			1:     {QuestMissionId: 1, QuestMissionConditionType: int32(model.QuestMissionConditionTypeComplete), QuestMissionRewardId: 1},
		},
		MissionRewardsByMissionId: map[int32][]masterdata.EntityMQuestMissionReward{
			1: {{PossessionType: int32(model.PossessionTypeConsumableItem), PossessionId: 102, Count: 10}},
		},
		FirstClearRewardsByGroupId: map[int32][]masterdata.EntityMQuestFirstClearRewardGroup{
			10: {{PossessionType: int32(model.PossessionTypeConsumableItem), PossessionId: 100, Count: 1}},
		},
		ReplayFlowRewardsByGroupId: map[int32][]masterdata.EntityMQuestReplayFlowRewardGroup{
			20: {{PossessionType: int32(model.PossessionTypeConsumableItem), PossessionId: 101, Count: 30}},
		},
		CostumeById:            map[int32]masterdata.EntityMCostume{10: {CostumeId: 10, CharacterId: 1}},
		WeaponById:             map[int32]masterdata.EntityMWeapon{10: {WeaponId: 10, AttributeType: 6}},
		UserExpThresholds:      []int32{0, 100000},
		CharacterExpThresholds: []int32{0, 1000},
		BossCountByQuestId:     map[int32]int32{replayId: 1},
	}, Config: &masterdata.GameConfig{QuestMissionBigWinBonusPower: 30000, ConsumableItemIdForGold: 99}, Granter: &store.PossessionGranter{}}
	user := store.SeedUserState(1, "replay-test", 1, model.ClientPlatform{})
	user.MainQuest.CurrentQuestSceneId = 888
	user.MainQuest.HeadQuestSceneId = 999
	user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 2}] = store.DeckState{
		UserDeckCharacterUuid01: "unit", Power: 57000,
	}
	user.DeckCharacters["unit"] = store.DeckCharacterState{UserCostumeUuid: "costume", MainUserWeaponUuid: "weapon"}
	user.Costumes["costume"] = store.CostumeState{CostumeId: 10}
	user.Weapons["weapon"] = store.WeaponState{WeaponId: 10}
	return h, user
}

func startReplayCompletion(t *testing.T, h *QuestHandler, user *store.UserState, replayId int32, now int64) {
	t.Helper()
	if err := h.HandleQuestStartReplay(user, replayId, true, 2, now); err != nil {
		t.Fatal(err)
	}
	// Replay uses the shared main-flow scene, even on Hard / Very Hard.
	user.MainQuest.ProgressQuestSceneId = 266
	user.MainQuest.ProgressHeadQuestSceneId = 266
	user.MainQuest.ProgressQuestFlowType = user.MainQuest.CurrentQuestFlowType
	user.Battle.LastFinishedAt = now + 1
	user.Battle.MissionDetail = store.BattleMissionDetailState{IsValid: true, WeaponSkillUseCount: 1}
	if err := h.ValidateMainQuestContinuation(user, replayId); err != nil {
		t.Fatal(err)
	}
}

func TestMapReplayCreditsMatchingDifficultyAndMissions(t *testing.T) {
	for replayId, completionId := range map[int32]int32{30009: 16, 40009: 10009, 50009: 20009} {
		for _, flow := range []model.QuestFlowType{model.QuestFlowTypeReplayFlow, model.QuestFlowTypeAnotherRouteReplayFlow} {
			t.Run(fmt.Sprintf("%d/%s", replayId, flow), func(t *testing.T) {
				h, user := replayCompletionFixture(replayId, completionId)
				user.MainQuest.CurrentQuestFlowType = int32(flow)
				for run := int64(1); run <= 2; run++ {
					startReplayCompletion(t, h, user, replayId, run*10)
					if state := user.Quests[replayId]; state.UserDeckNumber != 2 || !state.IsBattleOnly {
						t.Errorf("replay lost selected deck / battle-only setting: %+v", state)
					}
					outcome := h.HandleQuestFinish(user, replayId, false, false, run*10+2)
					for _, id := range []int32{completionId, replayId} {
						if state := user.Quests[id]; state.QuestStateType != model.UserQuestStateTypeCleared || state.ClearCount != int32(run) || !state.IsRewardGranted {
							t.Errorf("quest %d after run %d: %+v", id, run, state)
						}
					}
					for _, id := range []int32{103, 21001, 2, 1} {
						if !user.QuestMissions[store.QuestMissionKey{QuestId: completionId, QuestMissionId: id}].IsClear {
							t.Errorf("completion quest %d mission %d was not cleared", completionId, id)
						}
					}
					if run == 1 && (!outcome.IsBigWin || len(outcome.FirstClearRewards) != 1 || len(outcome.ReplayFlowFirstClearRewards) != 1) {
						t.Errorf("first finish is missing completion / replay rewards: %+v", outcome)
					}
					if run == 2 && (outcome.IsBigWin || len(outcome.ClearedQuestMissionIds)+len(outcome.FirstClearRewards)+len(outcome.ReplayFlowFirstClearRewards) != 0) {
						t.Errorf("repeat finish awarded one-time rewards again: %+v", outcome)
					}
				}
				if user.ConsumableItems[100] != 1 || user.ConsumableItems[101] != 30 || user.ConsumableItems[102] != 40 || user.ConsumableItems[99] != 1700 {
					t.Errorf("incorrect reward totals: %+v", user.ConsumableItems)
				}
				if user.Status.Exp != 684 || user.Characters[1].Exp != 34 || user.Costumes["costume"].Exp != 3080 {
					t.Errorf("experience missing or granted twice: user=%d character=%d costume=%d", user.Status.Exp, user.Characters[1].Exp, user.Costumes["costume"].Exp)
				}
				for key := range user.QuestMissions {
					if key.QuestId == replayId {
						t.Errorf("created invalid replay mission row: %+v", key)
					}
				}
				for _, id := range []int32{16, 10009, 20009} {
					if id != completionId && user.Quests[id].ClearCount != 0 {
						t.Errorf("credited wrong difficulty %d", id)
					}
				}
				if user.MainQuest.CurrentQuestSceneId != 888 || user.MainQuest.HeadQuestSceneId != 999 || user.MainQuest.ReplayFlowCurrentQuestSceneId != 267 {
					t.Errorf("incorrect scene progression: %+v", user.MainQuest)
				}
				clearEvents := 0
				for _, event := range user.PendingMissionEvents {
					condition := model.MissionClearConditionType(event.ConditionType)
					if condition != model.MissionClearConditionTypeQuestClearByCount && condition != model.MissionClearConditionTypeQuestClearByCountWithoutSkip && condition != model.MissionClearConditionTypeDefeatBossCount {
						continue
					}
					clearEvents++
					if event.TargetId != completionId {
						t.Errorf("mission event targets wrong quest: %+v", event)
					}
				}
				if clearEvents != 6 {
					t.Errorf("clear events = %d, want 6", clearEvents)
				}
			})
		}
	}
}

func TestMapReplayDoesNotCreditUnmetMissionsOrFailedRuns(t *testing.T) {
	for _, result := range []string{"unmet missions", "retired", "annihilated"} {
		t.Run(result, func(t *testing.T) {
			h, user := replayCompletionFixture(50009, 20009)
			startReplayCompletion(t, h, user, 50009, 10)
			user.Battle.MissionDetail.WeaponSkillUseCount = 0
			user.Battle.MissionDetail.CharacterDeathCount = 1
			outcome := h.HandleQuestFinish(user, 50009, result == "retired", result == "annihilated", 12)
			if result == "unmet missions" {
				if len(outcome.ClearedQuestMissionIds) != 1 || outcome.ClearedQuestMissionIds[0] != 103 || outcome.IsBigWin {
					t.Fatalf("unmet missions awarded: %+v", outcome)
				}
			} else if user.Quests[20009].ClearCount != 0 || len(user.QuestMissions) != 0 || len(outcome.FirstClearRewards)+len(outcome.ReplayFlowFirstClearRewards) != 0 {
				t.Fatalf("failed run credited progress or rewards: %+v", outcome)
			}
		})
	}
}

func TestMapReplayUsesExistingCompletionRewardHistory(t *testing.T) {
	h, user := replayCompletionFixture(50009, 20009)
	user.Quests[20009] = store.UserQuestState{QuestId: 20009, ClearCount: 5, IsRewardGranted: true, QuestStateType: model.UserQuestStateTypeCleared}
	user.QuestMissions[store.QuestMissionKey{QuestId: 20009, QuestMissionId: 103}] = store.UserQuestMissionState{QuestId: 20009, QuestMissionId: 103, IsClear: true}
	startReplayCompletion(t, h, user, 50009, 10)
	outcome := h.HandleQuestFinish(user, 50009, false, false, 12)
	if len(outcome.FirstClearRewards) != 0 || user.ConsumableItems[100] != 0 || user.ConsumableItems[102] != 30 || user.Quests[20009].ClearCount != 6 {
		t.Fatalf("existing progress / rewards not respected: outcome=%+v state=%+v rewards=%+v", outcome, user.Quests[20009], user.ConsumableItems)
	}
}

func TestMapReplayEvaluatesHiddenMissionsOnCompletionQuest(t *testing.T) {
	h, user := replayCompletionFixture(50009, 20009)
	h.InvisibleMissionIdsByQuestId = map[int32][]int32{20009: {50}}
	h.MissionById[50] = masterdata.EntityMQuestMission{
		QuestMissionId: 50, QuestMissionConditionType: int32(model.QuestMissionConditionTypeCriticalCountGe), ConditionValue: 3,
	}
	deckKey := store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 2}
	deck := user.Decks[deckKey]
	deck.Power = 220999
	user.Decks[deckKey] = deck
	key := store.QuestMissionKey{QuestId: 20009, QuestMissionId: 50}
	for run := int64(1); run <= 3; run++ {
		startReplayCompletion(t, h, user, 50009, run*10)
		if run > 1 {
			user.Battle.MissionDetail.CriticalCount = 3
		}
		outcome := h.HandleQuestFinish(user, 50009, false, false, run*10+2)
		if got := user.QuestMissions[key].IsClear; got != (run > 1) {
			t.Fatalf("hidden mission clear = %v after run %d", got, run)
		}
		if run > 1 && (outcome.IsBigWin || len(outcome.MissionClearRewards)+len(outcome.MissionClearCompleteRewards) != 0) {
			t.Fatalf("hidden mission awarded visible mission rewards: %+v", outcome)
		}
		if run == 3 && len(outcome.ClearedQuestMissionIds) != 0 {
			t.Fatalf("hidden mission awarded again: %+v", outcome)
		}
	}
}

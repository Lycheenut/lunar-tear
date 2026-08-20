package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestGimmickOrnamentRewardsLoadFromMasterData(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	if rewards := loadGimmickOrnamentRewards(LoadCageOrnamentCatalog()); len(rewards) == 0 {
		t.Fatal("gimmick ornament reward mapping was empty")
	}
}

func TestReportGimmickUnlockAndProgressUseDifferentConditions(t *testing.T) {
	key := store.GimmickSequenceKey{GimmickSequenceScheduleId: 1, GimmickSequenceId: 2}
	resolver := &ConditionResolver{
		conditionsById: map[int32]EntityMEvaluateCondition{
			10: {EvaluateConditionId: 10, EvaluateConditionFunctionType: int32(model.EvaluateConditionFunctionTypeMissionClear), EvaluateConditionValueGroupId: 20},
		},
		valuesByGroupId: map[int32][]EntityMEvaluateConditionValueGroup{
			20: {{GroupIndex: 1, Value: 30}},
		},
	}
	catalog := &GimmickCatalog{
		scheduleByKey:           map[store.GimmickSequenceKey]gimmickScheduleEntry{key: {ScheduleId: 1, FirstSequenceId: 2, EndDatetime: 100}},
		gimmicksBySequence:      map[int32]map[int32]bool{2: {3: true}},
		gimmickTypes:            map[int32]model.GimmickType{3: model.GimmickTypeReport},
		clearConditionByGimmick: map[int32]int32{3: 10},
		conditions:              resolver,
	}
	user := store.SeedUserState(1, "report", 1, model.ClientPlatform{})
	if !catalog.GimmickUnlockAvailable(user, 1, 2, 3, 50) {
		t.Fatal("report gimmick unlock required its clear condition")
	}
	if catalog.GimmickAvailable(user, 1, 2, 3, 50) {
		t.Fatal("report gimmick progress was available before its mission cleared")
	}
	user.Missions[30] = store.UserMissionState{MissionId: 30, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
	if !catalog.GimmickAvailable(user, 1, 2, 3, 50) {
		t.Fatal("cleared mission did not allow report gimmick progress")
	}
}

func TestGimmickConditionsUseSupportedRoots(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	conditions, err := memorydb.ReadTable[EntityMEvaluateCondition]("m_evaluate_condition")
	if err != nil {
		t.Fatal(err)
	}
	conditionByID := make(map[int32]EntityMEvaluateCondition, len(conditions))
	for _, condition := range conditions {
		conditionByID[condition.EvaluateConditionId] = condition
	}
	schedules, err := memorydb.ReadTable[EntityMGimmickSequenceSchedule]("m_gimmick_sequence_schedule")
	if err != nil {
		t.Fatal(err)
	}
	for _, schedule := range schedules {
		if schedule.ReleaseEvaluateConditionId != 0 {
			conditionType := model.EvaluateConditionFunctionType(conditionByID[schedule.ReleaseEvaluateConditionId].EvaluateConditionFunctionType)
			if conditionType != model.EvaluateConditionFunctionTypeQuestClear {
				t.Fatalf("schedule %d has unsupported release condition type %d", schedule.GimmickSequenceScheduleId, conditionType)
			}
		}
	}
	gimmicks, err := memorydb.ReadTable[EntityMGimmick]("m_gimmick")
	if err != nil {
		t.Fatal(err)
	}
	for _, gimmick := range gimmicks {
		if model.GimmickType(gimmick.GimmickType) == model.GimmickTypeReport && gimmick.ClearEvaluateConditionId != 0 {
			conditionType := model.EvaluateConditionFunctionType(conditionByID[gimmick.ClearEvaluateConditionId].EvaluateConditionFunctionType)
			switch conditionType {
			case model.EvaluateConditionFunctionTypeRecursion,
				model.EvaluateConditionFunctionTypeMissionClear,
				model.EvaluateConditionFunctionTypeQuestMissionClear:
			default:
				t.Fatalf("report gimmick %d has unsupported clear condition type %d", gimmick.GimmickId, conditionType)
			}
		}
	}
}

func TestGimmickOrnamentRewardLookupHasNoFallback(t *testing.T) {
	catalog := &GimmickCatalog{ornamentRewards: map[GimmickOrnamentRef]SequenceReward{{GimmickId: 7, OrnamentIndex: 1}: {PossessionType: 5, PossessionId: 9, Count: 2}}}
	if reward, ok := catalog.OrnamentReward(7, 1); !ok || reward.PossessionId != 9 {
		t.Fatalf("mapped reward = %+v, %v", reward, ok)
	}
	if _, ok := catalog.OrnamentReward(8, 1); ok {
		t.Fatal("unmapped ornament received a fallback")
	}
}

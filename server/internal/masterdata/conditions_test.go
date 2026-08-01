package masterdata

import (
	"testing"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestConditionResolverUsesPersistedMissionState(t *testing.T) {
	resolver := &ConditionResolver{
		conditionsById: map[int32]EntityMEvaluateCondition{
			1: {EvaluateConditionId: 1, EvaluateConditionFunctionType: int32(model.EvaluateConditionFunctionTypeRecursion), EvaluateConditionEvaluateType: int32(model.EvaluateConditionEvaluateTypeAnd), EvaluateConditionValueGroupId: 10},
			2: {EvaluateConditionId: 2, EvaluateConditionFunctionType: int32(model.EvaluateConditionFunctionTypeMissionClear), EvaluateConditionValueGroupId: 20},
			3: {EvaluateConditionId: 3, EvaluateConditionFunctionType: int32(model.EvaluateConditionFunctionTypeQuestMissionClear), EvaluateConditionValueGroupId: 30},
		},
		valuesByGroupId: map[int32][]EntityMEvaluateConditionValueGroup{
			10: {{GroupIndex: 1, Value: 2}, {GroupIndex: 2, Value: 3}},
			20: {{GroupIndex: 1, Value: 100}},
			30: {{GroupIndex: 1, Value: 200}, {GroupIndex: 2, Value: 300}},
		},
	}
	user := store.SeedUserState(1, "conditions", 1, model.ClientPlatform{})
	if resolver.Satisfied(1, user) {
		t.Fatal("condition was satisfied by absent mission state")
	}
	user.Missions[100] = store.UserMissionState{MissionId: 100, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
	if resolver.Satisfied(1, user) {
		t.Fatal("AND condition ignored an uncleared quest mission")
	}
	key := store.QuestMissionKey{QuestId: 200, QuestMissionId: 300}
	user.QuestMissions[key] = store.UserQuestMissionState{QuestId: 200, QuestMissionId: 300, IsClear: true}
	if !resolver.Satisfied(1, user) {
		t.Fatal("real mission states did not satisfy condition")
	}
}

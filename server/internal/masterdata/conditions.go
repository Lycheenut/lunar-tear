package masterdata

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

const defaultGroupIndex = 1

type ConditionResolver struct {
	requiredQuestByCondId map[int32]int32
	conditionsById        map[int32]EntityMEvaluateCondition
	valuesByGroupId       map[int32][]EntityMEvaluateConditionValueGroup
}

func LoadConditionResolver() (*ConditionResolver, error) {
	conditions, err := utils.ReadTable[EntityMEvaluateCondition]("m_evaluate_condition")
	if err != nil {
		return nil, fmt.Errorf("load evaluate condition table: %w", err)
	}
	valueGroups, err := utils.ReadTable[EntityMEvaluateConditionValueGroup]("m_evaluate_condition_value_group")
	if err != nil {
		return nil, fmt.Errorf("load evaluate condition value group table: %w", err)
	}

	condById := make(map[int32]EntityMEvaluateCondition, len(conditions))
	for _, c := range conditions {
		condById[c.EvaluateConditionId] = c
	}

	type vgKey struct {
		GroupId    int32
		GroupIndex int32
	}
	vgByKey := make(map[vgKey]int64, len(valueGroups))
	for _, vg := range valueGroups {
		vgByKey[vgKey{vg.EvaluateConditionValueGroupId, vg.GroupIndex}] = vg.Value
	}

	resolved := make(map[int32]int32)
	for _, c := range conditions {
		if model.EvaluateConditionFunctionType(c.EvaluateConditionFunctionType) == model.EvaluateConditionFunctionTypeQuestClear &&
			model.EvaluateConditionEvaluateType(c.EvaluateConditionEvaluateType) == model.EvaluateConditionEvaluateTypeIdContain {
			if questId, ok := vgByKey[vgKey{c.EvaluateConditionValueGroupId, defaultGroupIndex}]; ok {
				resolved[c.EvaluateConditionId] = int32(questId)
			}
		}
	}
	valuesByGroupId := make(map[int32][]EntityMEvaluateConditionValueGroup)
	for _, value := range valueGroups {
		valuesByGroupId[value.EvaluateConditionValueGroupId] = append(valuesByGroupId[value.EvaluateConditionValueGroupId], value)
	}
	for groupId := range valuesByGroupId {
		sort.Slice(valuesByGroupId[groupId], func(i, j int) bool {
			return valuesByGroupId[groupId][i].GroupIndex < valuesByGroupId[groupId][j].GroupIndex
		})
	}

	return &ConditionResolver{
		requiredQuestByCondId: resolved,
		conditionsById:        condById,
		valuesByGroupId:       valuesByGroupId,
	}, nil
}

func (r *ConditionResolver) Satisfied(conditionId int32, user *store.UserState) bool {
	return r.satisfied(conditionId, user, make(map[int32]bool), 0)
}

func (r *ConditionResolver) satisfied(conditionId int32, user *store.UserState, visiting map[int32]bool, depth int) bool {
	if conditionId == 0 {
		return true
	}
	if depth > 32 || visiting[conditionId] {
		return false
	}
	condition, ok := r.conditionsById[conditionId]
	if !ok {
		return false
	}
	visiting[conditionId] = true
	defer delete(visiting, conditionId)
	values := r.valuesByGroupId[condition.EvaluateConditionValueGroupId]
	valueAt := func(index int32) (int64, bool) {
		for _, value := range values {
			if value.GroupIndex == index {
				return value.Value, true
			}
		}
		return 0, false
	}

	switch model.EvaluateConditionFunctionType(condition.EvaluateConditionFunctionType) {
	case model.EvaluateConditionFunctionTypeRecursion:
		if len(values) == 0 {
			return false
		}
		if model.EvaluateConditionEvaluateType(condition.EvaluateConditionEvaluateType) == model.EvaluateConditionEvaluateTypeOr {
			for _, value := range values {
				if r.satisfied(int32(value.Value), user, visiting, depth+1) {
					return true
				}
			}
			return false
		}
		for _, value := range values {
			if !r.satisfied(int32(value.Value), user, visiting, depth+1) {
				return false
			}
		}
		return true
	case model.EvaluateConditionFunctionTypeQuestClear:
		questId, ok := valueAt(defaultGroupIndex)
		return ok && user.Quests[int32(questId)].QuestStateType == model.UserQuestStateTypeCleared
	case model.EvaluateConditionFunctionTypeQuestNotClear:
		questId, ok := valueAt(defaultGroupIndex)
		return ok && user.Quests[int32(questId)].QuestStateType != model.UserQuestStateTypeCleared
	case model.EvaluateConditionFunctionTypeMissionClear:
		missionId, ok := valueAt(defaultGroupIndex)
		return ok && user.Missions[int32(missionId)].MissionProgressStatusType >= int32(model.MissionProgressStatusTypeClear)
	case model.EvaluateConditionFunctionTypeQuestMissionClear:
		questId, questOK := valueAt(1)
		missionId, missionOK := valueAt(2)
		return questOK && missionOK && user.QuestMissions[store.QuestMissionKey{QuestId: int32(questId), QuestMissionId: int32(missionId)}].IsClear
	case model.EvaluateConditionFunctionTypeWeaponAcquisition:
		weaponId, ok := valueAt(defaultGroupIndex)
		if !ok {
			return false
		}
		for _, weapon := range user.Weapons {
			if weapon.WeaponId == int32(weaponId) {
				return true
			}
		}
		return false
	case model.EvaluateConditionFunctionTypeTutorial:
		tutorialType, ok := valueAt(defaultGroupIndex)
		_, completed := user.Tutorials[int32(tutorialType)]
		return ok && completed
	case model.EvaluateConditionFunctionTypeQuestSceneChoice:
		questSceneId, sceneOK := valueAt(1)
		questFlowType, flowOK := valueAt(2)
		choiceNumber, choiceOK := valueAt(3)
		if !sceneOK || !flowOK || !choiceOK {
			return false
		}
		choice, exists := user.QuestSceneChoices[store.QuestSceneChoiceKey{QuestSceneId: int32(questSceneId), QuestFlowType: int32(questFlowType)}]
		return exists && choice.ChoiceNumber == int32(choiceNumber)
	default:
		return false
	}
}

func (r *ConditionResolver) RequiredQuestId(conditionId int32) (int32, bool) {
	qid, ok := r.requiredQuestByCondId[conditionId]
	return qid, ok
}

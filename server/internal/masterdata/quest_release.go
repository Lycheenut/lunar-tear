package masterdata

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

type QuestReleaseCondition struct {
	ConditionType model.QuestReleaseConditionType
	QuestId       int32
}

type QuestReleaseConditionGroup struct {
	OperationType model.ConditionOperationType
	Conditions    []QuestReleaseCondition
}

func loadQuestReleaseConditions() (map[int32]QuestReleaseConditionGroup, error) {
	lists, err := utils.ReadTable[EntityMQuestReleaseConditionList]("m_quest_release_condition_list")
	if err != nil {
		return nil, fmt.Errorf("load quest release condition lists: %w", err)
	}
	groups, err := utils.ReadTable[EntityMQuestReleaseConditionGroup]("m_quest_release_condition_group")
	if err != nil {
		return nil, fmt.Errorf("load quest release condition groups: %w", err)
	}
	questClears, err := utils.ReadTable[EntityMQuestReleaseConditionQuestClear]("m_quest_release_condition_quest_clear")
	if err != nil {
		return nil, fmt.Errorf("load quest-clear release conditions: %w", err)
	}
	questChallenges, err := utils.ReadTable[EntityMQuestReleaseConditionQuestChallenge]("m_quest_release_condition_quest_challenge")
	if err != nil {
		return nil, fmt.Errorf("load quest-challenge release conditions: %w", err)
	}

	questClearByConditionId := make(map[int32]int32, len(questClears))
	for _, condition := range questClears {
		questClearByConditionId[condition.QuestReleaseConditionId] = condition.QuestId
	}
	questChallengeByConditionId := make(map[int32]int32, len(questChallenges))
	for _, condition := range questChallenges {
		questChallengeByConditionId[condition.QuestReleaseConditionId] = condition.QuestId
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].QuestReleaseConditionGroupId != groups[j].QuestReleaseConditionGroupId {
			return groups[i].QuestReleaseConditionGroupId < groups[j].QuestReleaseConditionGroupId
		}
		return groups[i].SortOrder < groups[j].SortOrder
	})
	conditionsByGroupId := make(map[int32][]QuestReleaseCondition)
	for _, condition := range groups {
		conditionType := model.QuestReleaseConditionType(condition.QuestReleaseConditionType)
		var questId int32
		switch conditionType {
		case model.QuestReleaseConditionTypeQuestClear:
			questId = questClearByConditionId[condition.QuestReleaseConditionId]
		case model.QuestReleaseConditionTypeQuestChallenge:
			questId = questChallengeByConditionId[condition.QuestReleaseConditionId]
		}
		conditionsByGroupId[condition.QuestReleaseConditionGroupId] = append(
			conditionsByGroupId[condition.QuestReleaseConditionGroupId],
			QuestReleaseCondition{ConditionType: conditionType, QuestId: questId},
		)
	}

	result := make(map[int32]QuestReleaseConditionGroup, len(lists))
	for _, list := range lists {
		result[list.QuestReleaseConditionListId] = QuestReleaseConditionGroup{
			OperationType: model.ConditionOperationType(list.ConditionOperationType),
			Conditions:    conditionsByGroupId[list.QuestReleaseConditionGroupId],
		}
	}
	return result, nil
}

func (c *QuestCatalog) QuestReleased(user *store.UserState, quest EntityMQuest) bool {
	if quest.QuestReleaseConditionListId == 0 {
		return true
	}
	group, ok := c.QuestReleaseConditionsByListId[quest.QuestReleaseConditionListId]
	if !ok || len(group.Conditions) == 0 {
		return false
	}

	satisfied := func(condition QuestReleaseCondition) bool {
		state, ok := user.Quests[condition.QuestId]
		if !ok || condition.QuestId == 0 {
			return false
		}
		switch condition.ConditionType {
		case model.QuestReleaseConditionTypeQuestClear:
			return state.QuestStateType == model.UserQuestStateTypeCleared
		case model.QuestReleaseConditionTypeQuestChallenge:
			return state.QuestStateType == model.UserQuestStateTypeCleared ||
				state.QuestStateType == model.UserQuestStateTypeChallenged
		default:
			return false
		}
	}

	switch group.OperationType {
	case model.ConditionOperationTypeAnd:
		for _, condition := range group.Conditions {
			if !satisfied(condition) {
				return false
			}
		}
		return true
	case model.ConditionOperationTypeOr:
		for _, condition := range group.Conditions {
			if satisfied(condition) {
				return true
			}
		}
	}
	return false
}

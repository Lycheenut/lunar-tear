package masterdataadmin

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/masterdata/memorydb"
)

type MissionSourceCatalog struct {
	Groups   []MissionGroupSource `json:"groups"`
	Missions []MissionSource      `json:"missions"`
}

type MissionGroupSource struct {
	MissionGroupID      int64             `json:"missionGroupId"`
	MissionCategoryType int64             `json:"missionCategoryType"`
	SortOrder           int64             `json:"sortOrder"`
	Names               map[string]string `json:"names,omitempty"`
}

type MissionSource struct {
	Row              int               `json:"row"`
	MissionID        int64             `json:"missionId"`
	MissionGroupID   int64             `json:"missionGroupId"`
	MissionRewardID  int64             `json:"missionRewardId"`
	MissionTermID    int64             `json:"missionTermId"`
	RequirementCount int64             `json:"requirementCount"`
	SortOrder        int64             `json:"sortOrder"`
	Names            map[string]string `json:"names,omitempty"`
}

func loadMissionSources(file *memorydb.File, resolver *titleResolver) MissionSourceCatalog {
	rewardIDs := make(map[int64]struct{})
	for _, row := range readRows(file, "m_mission_reward") {
		if rewardID, ok := integerAt(row, 0); ok {
			rewardIDs[rewardID] = struct{}{}
		}
	}
	termIDs := make(map[int64]struct{})
	for _, row := range readRows(file, "m_mission_term") {
		if termID, ok := integerAt(row, 0); ok {
			termIDs[termID] = struct{}{}
		}
	}

	groupsByID := make(map[int64]MissionGroupSource)
	for _, row := range readRows(file, "m_mission_group") {
		groupID, groupOK := integerAt(row, 0)
		categoryType, categoryOK := integerAt(row, 1)
		labelTextID, labelOK := integerAt(row, 2)
		sortOrder, sortOK := integerAt(row, 3)
		if !groupOK || !categoryOK || !labelOK || !sortOK {
			continue
		}
		groupsByID[groupID] = MissionGroupSource{
			MissionGroupID:      groupID,
			MissionCategoryType: categoryType,
			SortOrder:           sortOrder,
			Names:               resolver.byKey(fmt.Sprintf("mission.label.%d", labelTextID)),
		}
	}

	usedGroups := make(map[int64]struct{})
	var missions []MissionSource
	for rowIndex, row := range readRows(file, "m_mission") {
		missionID, missionOK := integerAt(row, 0)
		groupID, groupOK := integerAt(row, 1)
		sortOrder, sortOK := integerAt(row, 2)
		nameTextID, nameOK := integerAt(row, 5)
		requirementCount, requirementOK := integerAt(row, 9)
		rewardID, rewardOK := integerAt(row, 11)
		termID, termOK := integerAt(row, 12)
		_, hasReward := rewardIDs[rewardID]
		_, hasTerm := termIDs[termID]
		_, hasGroup := groupsByID[groupID]
		if !missionOK || !groupOK || !sortOK || !nameOK || !requirementOK || !rewardOK || !termOK || (!hasReward && !hasTerm) || !hasGroup {
			continue
		}
		missions = append(missions, MissionSource{
			Row:              rowIndex,
			MissionID:        missionID,
			MissionGroupID:   groupID,
			MissionRewardID:  rewardID,
			MissionTermID:    termID,
			RequirementCount: requirementCount,
			SortOrder:        sortOrder,
			Names:            resolver.byKey(fmt.Sprintf("mission.name.%d", nameTextID)),
		})
		usedGroups[groupID] = struct{}{}
	}

	groups := make([]MissionGroupSource, 0, len(usedGroups))
	for groupID := range usedGroups {
		groups = append(groups, groupsByID[groupID])
	}
	sort.SliceStable(groups, func(left, right int) bool {
		if groups[left].MissionCategoryType != groups[right].MissionCategoryType {
			return groups[left].MissionCategoryType < groups[right].MissionCategoryType
		}
		if groups[left].SortOrder != groups[right].SortOrder {
			return groups[left].SortOrder < groups[right].SortOrder
		}
		return groups[left].MissionGroupID < groups[right].MissionGroupID
	})
	sort.SliceStable(missions, func(left, right int) bool {
		leftGroup := groupsByID[missions[left].MissionGroupID]
		rightGroup := groupsByID[missions[right].MissionGroupID]
		if leftGroup.MissionCategoryType != rightGroup.MissionCategoryType {
			return leftGroup.MissionCategoryType < rightGroup.MissionCategoryType
		}
		if leftGroup.SortOrder != rightGroup.SortOrder {
			return leftGroup.SortOrder < rightGroup.SortOrder
		}
		if missions[left].SortOrder != missions[right].SortOrder {
			return missions[left].SortOrder < missions[right].SortOrder
		}
		return missions[left].MissionID < missions[right].MissionID
	})
	return MissionSourceCatalog{Groups: groups, Missions: missions}
}

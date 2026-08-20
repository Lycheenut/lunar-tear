package masterdata

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/utils"
)

type MissionPassCatalog struct {
	Definition EntityMMissionPass
	Levels     []EntityMMissionPassLevelGroup
	Rewards    []EntityMMissionPassRewardGroup
}

type MissionCatalog struct {
	MissionById                    map[int32]EntityMMission
	OrderedMissions                []EntityMMission
	MissionIdsByType               map[int32][]int32
	MeasurableMissionIdsByType     map[int32][]int32
	RewardById                     map[int32]EntityMMissionReward
	TermById                       map[int32]EntityMMissionTerm
	UnlockById                     map[int32]EntityMMissionUnlockCondition
	GroupById                      map[int32]EntityMMissionGroup
	LinkById                       map[int32]EntityMMissionLink
	QuestChapterIdsByClearOption   map[int32][]int32
	CompletePossessionsByMissionId map[int32][]EntityMCompleteMissionGroup
	WebviewPageNumberByPageId      map[int32]int32
	PassById                       map[int32]MissionPassCatalog
	PassIdByMissionId              map[int32]int32
}

func LoadMissionCatalog() (*MissionCatalog, error) {
	missions, err := utils.ReadTable[EntityMMission]("m_mission")
	if err != nil {
		return nil, fmt.Errorf("load missions: %w", err)
	}
	rewards, err := utils.ReadTable[EntityMMissionReward]("m_mission_reward")
	if err != nil {
		return nil, fmt.Errorf("load mission rewards: %w", err)
	}
	terms, err := utils.ReadTable[EntityMMissionTerm]("m_mission_term")
	if err != nil {
		return nil, fmt.Errorf("load mission terms: %w", err)
	}
	unlocks, err := utils.ReadTable[EntityMMissionUnlockCondition]("m_mission_unlock_condition")
	if err != nil {
		return nil, fmt.Errorf("load mission unlocks: %w", err)
	}
	passes, err := utils.ReadTable[EntityMMissionPass]("m_mission_pass")
	if err != nil {
		return nil, fmt.Errorf("load mission passes: %w", err)
	}
	levels, err := utils.ReadTable[EntityMMissionPassLevelGroup]("m_mission_pass_level_group")
	if err != nil {
		return nil, fmt.Errorf("load mission pass levels: %w", err)
	}
	passRewards, err := utils.ReadTable[EntityMMissionPassRewardGroup]("m_mission_pass_reward_group")
	if err != nil {
		return nil, fmt.Errorf("load mission pass rewards: %w", err)
	}
	passGroups, err := utils.ReadTable[EntityMMissionPassMissionGroup]("m_mission_pass_mission_group")
	if err != nil {
		return nil, fmt.Errorf("load mission pass groups: %w", err)
	}
	groups, err := utils.ReadTable[EntityMMissionGroup]("m_mission_group")
	if err != nil {
		return nil, fmt.Errorf("load mission groups: %w", err)
	}
	links, err := utils.ReadTable[EntityMMissionLink]("m_mission_link")
	if err != nil {
		return nil, fmt.Errorf("load mission links: %w", err)
	}
	completePossessions, err := utils.ReadTable[EntityMCompleteMissionGroup]("m_complete_mission_group")
	if err != nil {
		return nil, fmt.Errorf("load complete mission groups: %w", err)
	}
	webviewPages, err := utils.ReadTable[EntityMWebviewPanelMission]("m_webview_panel_mission")
	if err != nil {
		return nil, fmt.Errorf("load webview panel mission pages: %w", err)
	}

	c := &MissionCatalog{
		MissionById: make(map[int32]EntityMMission), OrderedMissions: append([]EntityMMission(nil), missions...),
		MissionIdsByType: make(map[int32][]int32), MeasurableMissionIdsByType: make(map[int32][]int32),
		RewardById: make(map[int32]EntityMMissionReward), TermById: make(map[int32]EntityMMissionTerm),
		UnlockById: make(map[int32]EntityMMissionUnlockCondition), GroupById: make(map[int32]EntityMMissionGroup),
		LinkById:                       make(map[int32]EntityMMissionLink),
		QuestChapterIdsByClearOption:   make(map[int32][]int32),
		CompletePossessionsByMissionId: make(map[int32][]EntityMCompleteMissionGroup),
		WebviewPageNumberByPageId:      make(map[int32]int32), PassById: make(map[int32]MissionPassCatalog),
		PassIdByMissionId: make(map[int32]int32),
	}
	sort.Slice(c.OrderedMissions, func(i, j int) bool { return c.OrderedMissions[i].MissionId < c.OrderedMissions[j].MissionId })
	for _, row := range missions {
		c.MissionById[row.MissionId] = row
		c.MissionIdsByType[row.MissionClearConditionType] = append(c.MissionIdsByType[row.MissionClearConditionType], row.MissionId)
		if row.MissionClearConditionType == 31 || row.MissionClearConditionType == 32 ||
			row.MissionClearConditionType == 55 || row.MissionClearConditionType == 61 {
			c.MeasurableMissionIdsByType[row.MissionClearConditionType] = append(c.MeasurableMissionIdsByType[row.MissionClearConditionType], row.MissionId)
		}
	}
	for _, row := range rewards {
		c.RewardById[row.MissionRewardId] = row
	}
	for _, row := range terms {
		c.TermById[row.MissionTermId] = row
	}
	for _, row := range unlocks {
		c.UnlockById[row.MissionUnlockConditionId] = row
	}
	for _, row := range groups {
		c.GroupById[row.MissionGroupId] = row
	}
	for _, row := range links {
		c.LinkById[row.MissionLinkId] = row
	}
	seenQuestChaptersByOption := make(map[int32]map[int32]bool)
	for _, mission := range missions {
		link, ok := c.LinkById[mission.MissionLinkId]
		if !ok || link.DestinationDomainType != 4 || link.DestinationDomainId == 0 || mission.MissionClearConditionOptionGroupId == 0 {
			continue
		}
		option := mission.MissionClearConditionOptionGroupId
		if seenQuestChaptersByOption[option] == nil {
			seenQuestChaptersByOption[option] = make(map[int32]bool)
		}
		if !seenQuestChaptersByOption[option][link.DestinationDomainId] {
			seenQuestChaptersByOption[option][link.DestinationDomainId] = true
			c.QuestChapterIdsByClearOption[option] = append(c.QuestChapterIdsByClearOption[option], link.DestinationDomainId)
		}
	}
	for _, row := range completePossessions {
		c.CompletePossessionsByMissionId[row.MissionId] = append(c.CompletePossessionsByMissionId[row.MissionId], row)
	}
	for _, row := range webviewPages {
		c.WebviewPageNumberByPageId[row.WebviewPanelMissionPageId] = row.Page
	}
	levelsByGroup := make(map[int32][]EntityMMissionPassLevelGroup)
	for _, row := range levels {
		levelsByGroup[row.MissionPassLevelGroupId] = append(levelsByGroup[row.MissionPassLevelGroupId], row)
	}
	rewardsByGroup := make(map[int32][]EntityMMissionPassRewardGroup)
	for _, row := range passRewards {
		rewardsByGroup[row.MissionPassRewardGroupId] = append(rewardsByGroup[row.MissionPassRewardGroupId], row)
	}
	for _, row := range passes {
		ls := levelsByGroup[row.MissionPassLevelGroupId]
		sort.Slice(ls, func(i, j int) bool { return ls[i].Level < ls[j].Level })
		rs := rewardsByGroup[row.MissionPassRewardGroupId]
		sort.Slice(rs, func(i, j int) bool {
			if rs[i].Level != rs[j].Level {
				return rs[i].Level < rs[j].Level
			}
			if rs[i].IsPremium != rs[j].IsPremium {
				return !rs[i].IsPremium
			}
			return rs[i].SortOrder < rs[j].SortOrder
		})
		c.PassById[row.MissionPassId] = MissionPassCatalog{Definition: row, Levels: ls, Rewards: rs}
	}
	passIdByGroup := make(map[int32]int32)
	for _, row := range passGroups {
		passIdByGroup[row.MissionGroupId] = row.MissionPassId
	}
	for _, row := range missions {
		if passId := passIdByGroup[row.MissionGroupId]; passId != 0 {
			c.PassIdByMissionId[row.MissionId] = passId
		}
	}
	return c, nil
}

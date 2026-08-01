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
	MissionById                map[int32]EntityMMission
	MeasurableMissionIdsByType map[int32][]int32
	RewardsById                map[int32][]EntityMMissionReward
	TermById                   map[int32]EntityMMissionTerm
	UnlockById                 map[int32]EntityMMissionUnlockCondition
	PassById                   map[int32]MissionPassCatalog
	PassIdByMissionId          map[int32]int32
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

	c := &MissionCatalog{MissionById: make(map[int32]EntityMMission), MeasurableMissionIdsByType: make(map[int32][]int32), RewardsById: make(map[int32][]EntityMMissionReward), TermById: make(map[int32]EntityMMissionTerm), UnlockById: make(map[int32]EntityMMissionUnlockCondition), PassById: make(map[int32]MissionPassCatalog), PassIdByMissionId: make(map[int32]int32)}
	for _, row := range missions {
		c.MissionById[row.MissionId] = row
		if row.MissionClearConditionType >= 36 && row.MissionClearConditionType <= 39 {
			c.MeasurableMissionIdsByType[row.MissionClearConditionType] = append(c.MeasurableMissionIdsByType[row.MissionClearConditionType], row.MissionId)
		}
	}
	for _, row := range rewards {
		c.RewardsById[row.MissionRewardId] = append(c.RewardsById[row.MissionRewardId], row)
	}
	for _, row := range terms {
		c.TermById[row.MissionTermId] = row
	}
	for _, row := range unlocks {
		c.UnlockById[row.MissionUnlockConditionId] = row
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

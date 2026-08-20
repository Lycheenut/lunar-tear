package masterdata

import (
	"log"
	"sort"

	"lunar-tear/server/internal/utils"
)

type TowerTier struct {
	QuestMissionClearCount int32
	Reward                 RewardItem
}

type TowerCatalog struct {
	TiersByChapter map[int32][]TowerTier
}

func (c *TowerCatalog) CollectRewards(chapterId, oldCount, targetCount int32) ([]RewardItem, int32) {
	var items []RewardItem
	highest := int32(0)
	for _, t := range c.TiersByChapter[chapterId] {
		if t.QuestMissionClearCount > oldCount && t.QuestMissionClearCount <= targetCount {
			if t.Reward.Count > 0 {
				items = append(items, t.Reward)
			}
			if t.QuestMissionClearCount > highest {
				highest = t.QuestMissionClearCount
			}
		}
	}
	return items, highest
}

func LoadTowerCatalog() *TowerCatalog {
	// chapterId -> accumulation reward group id
	accumRewardRows, err := utils.ReadTable[EntityMEventQuestTowerAccumulationReward]("m_event_quest_tower_accumulation_reward")
	if err != nil {
		log.Fatalf("load event quest tower accumulation reward table: %v", err)
	}
	groupByChapter := make(map[int32]int32, len(accumRewardRows))
	for _, r := range accumRewardRows {
		groupByChapter[r.EventQuestChapterId] = r.EventQuestTowerAccumulationRewardGroupId
	}

	// reward group id -> reward items
	rewardGroupRows, err := utils.ReadTable[EntityMEventQuestTowerRewardGroup]("m_event_quest_tower_reward_group")
	if err != nil {
		log.Fatalf("load event quest tower reward group table: %v", err)
	}
	itemByRewardGroup := make(map[int32]RewardItem, len(rewardGroupRows))
	for _, r := range rewardGroupRows {
		itemByRewardGroup[r.EventQuestTowerRewardGroupId] = RewardItem{
			PossessionType: r.PossessionType,
			PossessionId:   r.PossessionId,
			Count:          r.Count,
		}
	}

	// accumulation group id -> tiers (threshold + resolved reward items)
	accumGroupRows, err := utils.ReadTable[EntityMEventQuestTowerAccumulationRewardGroup]("m_event_quest_tower_accumulation_reward_group")
	if err != nil {
		log.Fatalf("load event quest tower accumulation reward group table: %v", err)
	}
	tiersByGroup := make(map[int32][]TowerTier)
	for _, r := range accumGroupRows {
		tiersByGroup[r.EventQuestTowerAccumulationRewardGroupId] = append(tiersByGroup[r.EventQuestTowerAccumulationRewardGroupId], TowerTier{
			QuestMissionClearCount: r.QuestMissionClearCount,
			Reward:                 itemByRewardGroup[r.EventQuestTowerRewardGroupId],
		})
	}

	// resolve per-chapter, sorted ascending by threshold
	tiersByChapter := make(map[int32][]TowerTier, len(groupByChapter))
	for chapterId, groupId := range groupByChapter {
		tiers := tiersByGroup[groupId]
		sort.Slice(tiers, func(i, j int) bool {
			return tiers[i].QuestMissionClearCount < tiers[j].QuestMissionClearCount
		})
		tiersByChapter[chapterId] = tiers
	}

	log.Printf("tower catalog loaded: %d chapters", len(tiersByChapter))

	return &TowerCatalog{TiersByChapter: tiersByChapter}
}

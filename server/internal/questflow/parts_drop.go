package questflow

import (
	"math/rand"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questdrop"
	"lunar-tear/server/internal/store"
)

func (h *QuestHandler) partsDropRewards(
	user *store.UserState,
	quest masterdata.EntityMQuest,
	planned masterdata.BattleDropInfo,
	target campaign.QuestTarget,
	nowMillis int64,
	dropRate campaign.DropRateMul,
	dropCount campaign.DropCountMul,
	random *rand.Rand,
) []RewardGrant {
	rewardID := planned.BattleDropRewardId
	reward := h.BattleDropRewardById[rewardID]
	rate, _ := h.battleDropMultipliers(user, reward, target, nowMillis, dropRate, dropCount)
	rolls := rate.Apply(1)
	var pool []questdrop.Reward
	if rolls > 1 {
		pool = h.partsDropPool(quest, planned)
	}
	var drops []RewardGrant
	for roll := int32(0); roll < rolls; roll++ {
		if roll > 0 {
			rewardID = weightedRewardID(random, pool)
			reward = h.BattleDropRewardById[rewardID]
		}
		// Drop rate buys draws; only drop-count bonuses multiply each draw's
		// quantity. Item-specific count bonuses follow the newly drawn item.
		_, count := h.battleDropMultipliers(user, reward, target, nowMillis, dropRate, dropCount)
		drops = append(drops, RewardGrant{
			PossessionType: model.PossessionTypeParts,
			PossessionId:   reward.PossessionId,
			Count:          count.Apply(reward.Count),
			RewardEffectId: planned.BattleDropEffectId,
		})
	}
	return drops
}

func (h *QuestHandler) partsDropPool(quest masterdata.EntityMQuest, planned masterdata.BattleDropInfo) []questdrop.Reward {
	pool, configured := h.DropRewardsByQuestID[quest.QuestId]
	if !configured {
		for _, id := range h.PickupRewardIdsByGroupAndEffectId[quest.QuestPickupRewardGroupId][planned.BattleDropEffectId] {
			// Keep duplicate master-data rows as independent lottery tickets.
			pool = append(pool, questdrop.Reward{BattleDropRewardID: id, Weight: 1})
		}
	}
	original := h.BattleDropRewardById[planned.BattleDropRewardId]
	rarity := h.PartsById[original.PossessionId].RarityType
	var parts []questdrop.Reward
	for _, entry := range pool {
		reward := h.BattleDropRewardById[entry.BattleDropRewardID]
		if !entry.Guaranteed && reward.PossessionType == int32(model.PossessionTypeParts) &&
			h.BattleDropEffectIdByRewardId[entry.BattleDropRewardID] == planned.BattleDropEffectId &&
			h.PartsById[reward.PossessionId].RarityType == rarity {
			parts = append(parts, entry)
		}
	}
	return parts
}

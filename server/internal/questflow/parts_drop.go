package questflow

import (
	"math/rand"

	"google.golang.org/protobuf/encoding/protowire"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questdrop"
	"lunar-tear/server/internal/store"
)

// QuestReward.equipmentData (field 6) is the client's apb.api.gift.Parts
// message: level=1, parts_status_main_id=2, repeated parts_status_sub=3.
// Snapshot the granted instance so result details never reuse another drop.
func partsRewardEquipmentData(user *store.UserState, partsUUID string) []byte {
	part := user.Parts[partsUUID]
	data := protowire.AppendTag(nil, 1, protowire.VarintType)
	data = protowire.AppendVarint(data, uint64(part.Level))
	data = protowire.AppendTag(data, 2, protowire.VarintType)
	data = protowire.AppendVarint(data, uint64(part.PartsStatusMainId))
	for index := int32(1); index <= 4; index++ {
		sub, ok := user.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: partsUUID, StatusIndex: index}]
		if !ok {
			continue
		}
		var status []byte
		// PartsStatusSub fields in client wire order.
		for i, value := range []int32{sub.StatusIndex, sub.PartsStatusSubLotteryId, sub.Level, sub.StatusKindType, sub.StatusCalculationType, sub.StatusChangeValue} {
			status = protowire.AppendTag(status, protowire.Number(i+1), protowire.VarintType)
			status = protowire.AppendVarint(status, uint64(value))
		}
		data = protowire.AppendTag(data, 3, protowire.BytesType)
		data = protowire.AppendBytes(data, status)
	}
	return data
}

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
	// Campaign drop rate improves quality. Important-item rate bonuses retain
	// their extra draws; count bonuses still multiply each draw's quantity.
	rate, _ := h.battleDropMultipliers(user, reward, target, nowMillis, campaign.DropRateMul{}, dropCount)
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
		// Item-specific count bonuses follow the newly drawn item.
		_, count := h.battleDropMultipliers(user, reward, target, nowMillis, dropRate, dropCount)
		drops = append(drops, RewardGrant{
			PossessionType: model.PossessionTypeParts,
			PossessionId:   reward.PossessionId,
			Count:          count.Apply(reward.Count),
			RewardEffectId: planned.BattleDropEffectId,
			partsDropRate:  dropRate,
		})
	}
	return drops
}

// Reweight only the parts lottery, preserving material drops and the number
// of battle drops. The separate stream keeps the plan repeatable for reveals.
func (h *QuestHandler) applyPartsDropRate(plan []masterdata.BattleDropInfo, quest masterdata.EntityMQuest, random *rand.Rand, rate campaign.DropRateMul) {
	if rate.Apply(1000) == 1000 || h.PartsCatalog == nil {
		return
	}
	pool, configured := h.DropRewardsByQuestID[quest.QuestId]
	if !configured {
		for _, id := range h.PickupRewardIdsByGroupId[quest.QuestPickupRewardGroupId] {
			pool = append(pool, questdrop.Reward{BattleDropRewardID: id, Weight: 1})
		}
	}
	var highestRarity int32
	for _, entry := range pool {
		reward := h.BattleDropRewardById[entry.BattleDropRewardID]
		if reward.PossessionType == int32(model.PossessionTypeParts) {
			highestRarity = max(highestRarity, h.PartsById[reward.PossessionId].RarityType)
		}
	}
	weights := make([]int64, len(pool))
	var total int64
	for i, entry := range pool {
		reward := h.BattleDropRewardById[entry.BattleDropRewardID]
		if reward.PossessionType != int32(model.PossessionTypeParts) {
			continue
		}
		multiplier := int32(1000)
		if h.PartsById[reward.PossessionId].RarityType == highestRarity {
			multiplier = rate.Apply(multiplier)
		}
		weights[i] = int64(entry.Weight) * int64(multiplier)
		total += weights[i]
	}
	if total == 0 {
		return
	}
	for i, drop := range plan {
		if h.BattleDropRewardById[drop.BattleDropRewardId].PossessionType != int32(model.PossessionTypeParts) {
			continue
		}
		roll := random.Int63n(total)
		for j, entry := range pool {
			roll -= weights[j]
			if roll < 0 {
				plan[i].BattleDropRewardId = entry.BattleDropRewardID
				plan[i].BattleDropEffectId = h.BattleDropEffectIdByRewardId[entry.BattleDropRewardID]
				break
			}
		}
	}
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
		if reward.PossessionType == int32(model.PossessionTypeParts) &&
			h.BattleDropEffectIdByRewardId[entry.BattleDropRewardID] == planned.BattleDropEffectId &&
			h.PartsById[reward.PossessionId].RarityType == rarity {
			parts = append(parts, entry)
		}
	}
	return parts
}

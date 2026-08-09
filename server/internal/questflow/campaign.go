package questflow

import (
	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/model"
)

func (h *QuestHandler) targetForMain(questId int32) campaign.QuestTarget {
	return campaign.QuestTarget{
		QuestId:   questId,
		QuestType: campaign.QuestTypeMainQuest,
		ChapterId: h.MainQuestChapterIdByQuestId[questId],
	}
}

func (h *QuestHandler) targetForEvent(eventChapterId, questId int32) campaign.QuestTarget {
	return campaign.QuestTarget{
		QuestId:        questId,
		QuestType:      campaign.QuestTypeEventQuest,
		EventQuestType: h.EventQuestTypeByChapterId[eventChapterId],
		ChapterId:      eventChapterId,
	}
}

func (h *QuestHandler) targetForExtra(questId int32) campaign.QuestTarget {
	return campaign.QuestTarget{QuestId: questId, QuestType: campaign.QuestTypeExtraQuest}
}

func (h *QuestHandler) targetForBigHunt(questId int32) campaign.QuestTarget {
	return campaign.QuestTarget{QuestId: questId, QuestType: campaign.QuestTypeBigHunt}
}

func (h *QuestHandler) campaignFilter(nowMillis int64) campaign.Filter {
	return campaign.Filter{NowMillis: nowMillis, UserStatus: campaign.TargetUserStatusAll}
}

func (h *QuestHandler) staminaWithCampaign(baseStamina int32, t campaign.QuestTarget, nowMillis int64) int32 {
	if h.Campaigns == nil {
		return baseStamina
	}
	return h.Campaigns.QuestStamina(t, h.campaignFilter(nowMillis)).Apply(baseStamina)
}

func (h *QuestHandler) goldWithCampaign(baseGold int32, t campaign.QuestTarget, nowMillis int64) int32 {
	if h.Campaigns == nil {
		return baseGold
	}
	return h.Campaigns.QuestGold(t, h.campaignFilter(nowMillis)).Apply(baseGold)
}

func (h *QuestHandler) appendBonusDrops(drops []RewardGrant, t campaign.QuestTarget, nowMillis int64) []RewardGrant {
	if h.Campaigns == nil {
		return drops
	}
	for _, bd := range h.Campaigns.QuestBonusDrops(t, h.campaignFilter(nowMillis)) {
		drops = append(drops, RewardGrant{
			PossessionType: model.PossessionType(bd.PossessionType),
			PossessionId:   bd.PossessionId,
			Count:          bd.Count,
		})
	}
	return drops
}

package questflow

import (
	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
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

func (h *QuestHandler) campaignFilter(user *store.UserState, nowMillis int64) campaign.Filter {
	return h.Campaigns.FilterForUser(campaign.UserStatusContext{
		NowMillis:                 nowMillis,
		RegisterDatetime:          user.RegisterDatetime,
		LastComebackLoginDatetime: user.Login.LastComebackLoginDatetime,
		IsCampaignUnlockQuestCleared: func(questId int32) bool {
			return user.Quests[questId].QuestStateType == model.UserQuestStateTypeCleared
		},
	})
}

func (h *QuestHandler) staminaWithCampaign(user *store.UserState, baseStamina int32, t campaign.QuestTarget, nowMillis int64) int32 {
	if h.Campaigns == nil {
		return baseStamina
	}
	return h.Campaigns.QuestStamina(t, h.campaignFilter(user, nowMillis)).Apply(baseStamina)
}

func (h *QuestHandler) goldWithCampaign(user *store.UserState, baseGold int32, t campaign.QuestTarget, nowMillis int64) int32 {
	if h.Campaigns == nil {
		return baseGold
	}
	return h.Campaigns.QuestGold(t, h.campaignFilter(user, nowMillis)).Apply(baseGold)
}

func (h *QuestHandler) appendBonusDrops(user *store.UserState, drops []RewardGrant, t campaign.QuestTarget, nowMillis int64) []RewardGrant {
	if h.Campaigns == nil {
		return drops
	}
	for _, bd := range h.Campaigns.QuestBonusDrops(t, h.campaignFilter(user, nowMillis)) {
		drops = append(drops, RewardGrant{
			PossessionType: model.PossessionType(bd.PossessionType),
			PossessionId:   bd.PossessionId,
			Count:          bd.Count,
		})
	}
	return drops
}

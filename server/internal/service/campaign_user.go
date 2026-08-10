package service

import (
	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func campaignQuestCleared(user *store.UserState) func(int32) bool {
	return func(questId int32) bool {
		return user.Quests[questId].QuestStateType == model.UserQuestStateTypeCleared
	}
}

func ensureBeginnerCampaign(catalog *campaign.Catalog, user *store.UserState, nowMillis int64) {
	if catalog == nil || user.BeginnerCampaign.BeginnerCampaignId != 0 {
		return
	}
	enrollment, ok := catalog.BeginnerEnrollmentForRegistration(user.RegisterDatetime)
	if !ok {
		return
	}
	user.BeginnerCampaign = store.UserBeginnerCampaignState{
		BeginnerCampaignId:       enrollment.CampaignId,
		CampaignRegisterDatetime: enrollment.CampaignRegisterDatetime,
		LatestVersion:            nowMillis,
	}
}

func ensureComebackCampaign(catalog *campaign.Catalog, user *store.UserState, nowMillis int64) {
	if catalog == nil || user.ComebackCampaign.ComebackCampaignId != 0 || user.Login.LastComebackLoginDatetime == 0 {
		return
	}
	enrollment, ok := catalog.ComebackEnrollmentForRecordedLogin(user.Login.LastComebackLoginDatetime)
	if !ok {
		return
	}
	user.ComebackCampaign = store.UserComebackCampaignState{
		ComebackCampaignId: enrollment.CampaignId,
		ComebackDatetime:   enrollment.ComebackDatetime,
		LatestVersion:      nowMillis,
	}
}

func activateComebackCampaign(catalog *campaign.Catalog, user *store.UserState, nowMillis, lastLoginMillis int64) bool {
	if catalog == nil {
		return false
	}
	enrollment, ok := catalog.ComebackEnrollmentForLogin(nowMillis, lastLoginMillis, campaignQuestCleared(user))
	if !ok {
		return false
	}
	user.ComebackCampaign = store.UserComebackCampaignState{
		ComebackCampaignId: enrollment.CampaignId,
		ComebackDatetime:   enrollment.ComebackDatetime,
		LatestVersion:      nowMillis,
	}
	user.Login.LastComebackLoginDatetime = nowMillis
	return true
}

func userCampaignStatusContext(user *store.UserState, nowMillis int64) campaign.UserStatusContext {
	return campaign.UserStatusContext{
		NowMillis:                        nowMillis,
		RegisterDatetime:                 user.RegisterDatetime,
		LastComebackLoginDatetime:        user.Login.LastComebackLoginDatetime,
		BeginnerCampaignId:               user.BeginnerCampaign.BeginnerCampaignId,
		BeginnerCampaignRegisterDatetime: user.BeginnerCampaign.CampaignRegisterDatetime,
		ComebackCampaignId:               user.ComebackCampaign.ComebackCampaignId,
		ComebackDatetime:                 user.ComebackCampaign.ComebackDatetime,
		IsCampaignUnlockQuestCleared:     campaignQuestCleared(user),
	}
}

package userdata

import (
	pb "lunar-tear/server/gen/proto"

	"lunar-tear/server/internal/store"
)

func FullClientTableMap(user store.UserState) map[string]string {
	return map[string]string{
		"IUser":                                   projectTable("IUser", user),
		"IUserSetting":                            projectTable("IUserSetting", user),
		"IUserStatus":                             projectTable("IUserStatus", user),
		"IUserGem":                                projectTable("IUserGem", user),
		"IUserProfile":                            projectTable("IUserProfile", user),
		"IUserCharacter":                          projectTable("IUserCharacter", user),
		"IUserCharacterViewerField":               projectTable("IUserCharacterViewerField", user),
		"IUserCostume":                            projectTable("IUserCostume", user),
		"IUserWeapon":                             projectTable("IUserWeapon", user),
		"IUserWeaponStory":                        projectTable("IUserWeaponStory", user),
		"IUserCompanion":                          projectTable("IUserCompanion", user),
		"IUserThought":                            projectTable("IUserThought", user),
		"IUserDeckCharacter":                      projectTable("IUserDeckCharacter", user),
		"IUserDeck":                               projectTable("IUserDeck", user),
		"IUserTripleDeck":                         projectTable("IUserTripleDeck", user),
		"IUserLogin":                              projectTable("IUserLogin", user),
		"IUserBeginnerCampaign":                   projectTable("IUserBeginnerCampaign", user),
		"IUserComebackCampaign":                   projectTable("IUserComebackCampaign", user),
		"IUserLoginBonus":                         projectTable("IUserLoginBonus", user),
		"IUserMission":                            projectTable("IUserMission", user),
		"IUserMissionCompletionProgress":          projectTable("IUserMissionCompletionProgress", user),
		"IUserMissionPassPoint":                   projectTable("IUserMissionPassPoint", user),
		"IUserMainQuestFlowStatus":                projectTable("IUserMainQuestFlowStatus", user),
		"IUserMainQuestMainFlowStatus":            projectTable("IUserMainQuestMainFlowStatus", user),
		"IUserMainQuestProgressStatus":            projectTable("IUserMainQuestProgressStatus", user),
		"IUserMainQuestSeasonRoute":               projectTable("IUserMainQuestSeasonRoute", user),
		"IUserQuest":                              projectTable("IUserQuest", user),
		"IUserQuestMission":                       projectTable("IUserQuestMission", user),
		"IUserTutorialProgress":                   projectTable("IUserTutorialProgress", user),
		"IUserGimmick":                            projectTable("IUserGimmick", user),
		"IUserGimmickOrnamentProgress":            projectTable("IUserGimmickOrnamentProgress", user),
		"IUserGimmickSequence":                    projectTable("IUserGimmickSequence", user),
		"IUserGimmickUnlock":                      projectTable("IUserGimmickUnlock", user),
		"IUserMaterial":                           projectTable("IUserMaterial", user),
		"IUserConsumableItem":                     projectTable("IUserConsumableItem", user),
		"IUserParts":                              projectTable("IUserParts", user),
		"IUserImportantItem":                      projectTable("IUserImportantItem", user),
		"IUserPremiumItem":                        projectTable("IUserPremiumItem", user),
		"IUserDeckPartsGroup":                     projectTable("IUserDeckPartsGroup", user),
		"IUserDeckSubWeaponGroup":                 projectTable("IUserDeckSubWeaponGroup", user),
		"IUserDeckCharacterDressupCostume":        projectTable("IUserDeckCharacterDressupCostume", user),
		"IUserDeckTypeNote":                       projectTable("IUserDeckTypeNote", user),
		"IUserDeckLimitContentRestricted":         projectTable("IUserDeckLimitContentRestricted", user),
		"IUserCostumeActiveSkill":                 projectTable("IUserCostumeActiveSkill", user),
		"IUserCostumeAwakenStatusUp":              projectTable("IUserCostumeAwakenStatusUp", user),
		"IUserCostumeLevelBonusReleaseStatus":     projectTable("IUserCostumeLevelBonusReleaseStatus", user),
		"IUserCostumeLotteryEffect":               projectTable("IUserCostumeLotteryEffect", user),
		"IUserCostumeLotteryEffectAbility":        projectTable("IUserCostumeLotteryEffectAbility", user),
		"IUserCostumeLotteryEffectStatusUp":       projectTable("IUserCostumeLotteryEffectStatusUp", user),
		"IUserCostumeLotteryEffectPending":        projectTable("IUserCostumeLotteryEffectPending", user),
		"IUserWeaponNote":                         projectTable("IUserWeaponNote", user),
		"IUserWeaponAbility":                      projectTable("IUserWeaponAbility", user),
		"IUserWeaponSkill":                        projectTable("IUserWeaponSkill", user),
		"IUserWeaponAwaken":                       projectTable("IUserWeaponAwaken", user),
		"IUserPartsGroupNote":                     projectTable("IUserPartsGroupNote", user),
		"IUserPartsPreset":                        projectTable("IUserPartsPreset", user),
		"IUserPartsPresetTag":                     projectTable("IUserPartsPresetTag", user),
		"IUserPartsStatusSub":                     projectTable("IUserPartsStatusSub", user),
		"IUserNaviCutIn":                          projectTable("IUserNaviCutIn", user),
		"IUserMovie":                              projectTable("IUserMovie", user),
		"IUserContentsStory":                      projectTable("IUserContentsStory", user),
		"IUserOmikuji":                            projectTable("IUserOmikuji", user),
		"IUserDokan":                              projectTable("IUserDokan", user),
		"IUserPortalCageStatus":                   projectTable("IUserPortalCageStatus", user),
		"IUserEventQuestGuerrillaFreeOpen":        projectTable("IUserEventQuestGuerrillaFreeOpen", user),
		"IUserEventQuestProgressStatus":           projectTable("IUserEventQuestProgressStatus", user),
		"IUserExtraQuestProgressStatus":           projectTable("IUserExtraQuestProgressStatus", user),
		"IUserEventQuestDailyGroupCompleteReward": projectTable("IUserEventQuestDailyGroupCompleteReward", user),
		"IUserEventQuestLabyrinthSeason":          projectTable("IUserEventQuestLabyrinthSeason", user),
		"IUserEventQuestLabyrinthStage":           projectTable("IUserEventQuestLabyrinthStage", user),
		"IUserEventQuestTowerAccumulationReward":  projectTable("IUserEventQuestTowerAccumulationReward", user),
		"IUserMainQuestReplayFlowStatus":          projectTable("IUserMainQuestReplayFlowStatus", user),
		"IUserSideStoryQuest":                     projectTable("IUserSideStoryQuest", user),
		"IUserSideStoryQuestSceneProgressStatus":  projectTable("IUserSideStoryQuestSceneProgressStatus", user),
		"IUserQuestLimitContentStatus":            projectTable("IUserQuestLimitContentStatus", user),
		"IUserQuestReplayFlowRewardGroup":         projectTable("IUserQuestReplayFlowRewardGroup", user),
		"IUserQuestAutoOrbit":                     projectTable("IUserQuestAutoOrbit", user),
		"IUserQuestSceneChoice":                   projectTable("IUserQuestSceneChoice", user),
		"IUserQuestSceneChoiceHistory":            projectTable("IUserQuestSceneChoiceHistory", user),
		"IUserShopItem":                           projectTable("IUserShopItem", user),
		"IUserShopReplaceable":                    projectTable("IUserShopReplaceable", user),
		"IUserShopReplaceableLineup":              projectTable("IUserShopReplaceableLineup", user),
		"IUserExplore":                            projectTable("IUserExplore", user),
		"IUserExploreScore":                       projectTable("IUserExploreScore", user),
		"IUserCharacterBoard":                     projectTable("IUserCharacterBoard", user),
		"IUserCharacterBoardAbility":              projectTable("IUserCharacterBoardAbility", user),
		"IUserCharacterBoardStatusUp":             projectTable("IUserCharacterBoardStatusUp", user),
		"IUserCharacterBoardCompleteReward":       projectTable("IUserCharacterBoardCompleteReward", user),
		"IUserAutoSaleSettingDetail":              projectTable("IUserAutoSaleSettingDetail", user),
		"IUserCharacterRebirth":                   projectTable("IUserCharacterRebirth", user),
		"IUserCageOrnamentReward":                 projectTable("IUserCageOrnamentReward", user),
		"IUserBigHuntProgressStatus":              projectTable("IUserBigHuntProgressStatus", user),
		"IUserBigHuntMaxScore":                    projectTable("IUserBigHuntMaxScore", user),
		"IUserBigHuntStatus":                      projectTable("IUserBigHuntStatus", user),
		"IUserBigHuntScheduleMaxScore":            projectTable("IUserBigHuntScheduleMaxScore", user),
		"IUserBigHuntWeeklyMaxScore":              projectTable("IUserBigHuntWeeklyMaxScore", user),
		"IUserBigHuntWeeklyStatus":                projectTable("IUserBigHuntWeeklyStatus", user),
		"IUserFacebook":                           projectTable("IUserFacebook", user),
		"IUserApple":                              projectTable("IUserApple", user),
		"IUserWebviewPanelMission":                projectTable("IUserWebviewPanelMission", user),
	}
}

func FirstEntranceClientTableMap(user store.UserState) map[string]string {
	tables := FullClientTableMap(user)
	for _, table := range []string{
		"IUserCharacter",
		"IUserCostume",
		"IUserWeapon",
		"IUserCompanion",
		"IUserDeckCharacter",
		"IUserDeck",
		"IUserTutorialProgress",
		"IUserParts",
		"IUserWeaponNote",
		"IUserWeaponStory",
		"IUserWeaponSkill",
		"IUserWeaponAbility",
		"IUserWeaponAwaken",
		"IUserCostumeActiveSkill",
		"IUserDeckTypeNote",
	} {
		tables[table] = "[]"
	}
	return tables
}

func SelectTables(all map[string]string, requested []string) map[string]string {
	selected := make(map[string]string, len(requested))
	for _, table := range requested {
		if payload, ok := all[table]; ok && payload != "" {
			selected[table] = payload
			continue
		}
		selected[table] = "[]"
	}
	return selected
}

func ProjectTables(user store.UserState, requested []string) map[string]string {
	result := make(map[string]string, len(requested))
	for _, table := range requested {
		result[table] = projectTable(table, user)
	}
	return result
}

func BuildDiffFromTables(tables map[string]string) map[string]*pb.DiffData {
	diff := make(map[string]*pb.DiffData, len(tables))
	for table, payload := range tables {
		if payload == "" {
			payload = "[]"
		}
		diff[table] = &pb.DiffData{
			UpdateRecordsJson: payload,
			DeleteKeysJson:    "[]",
		}
	}
	return diff
}

// BuildDiffFromTablesOrdered builds a diff map with tables in the given order.
// Use when client applies tables in received order and order matters (e.g. IUserWeapon before IUserWeaponStory).
// Protobuf map serialization order is implementation-defined; this at least ensures we only include
// the requested tables in the specified sequence when building the map.
func BuildDiffFromTablesOrdered(tables map[string]string, order []string) map[string]*pb.DiffData {
	diff := make(map[string]*pb.DiffData, len(order))
	for _, table := range order {
		payload, ok := tables[table]
		if !ok {
			payload = "[]"
		}
		if payload == "" {
			payload = "[]"
		}
		diff[table] = &pb.DiffData{
			UpdateRecordsJson: payload,
			DeleteKeysJson:    "[]",
		}
	}
	return diff
}

func AddWeaponStoryDiff(diff map[string]*pb.DiffData, user store.UserState, weaponIds []int32) {
	if len(weaponIds) == 0 {
		return
	}
	diff["IUserWeaponStory"] = &pb.DiffData{
		UpdateRecordsJson: WeaponStoryRecordsForIds(user, weaponIds),
		DeleteKeysJson:    "[]",
	}
}

package userdata

import (
	"encoding/json"
	"maps"
	"slices"
	"sort"
	"strings"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/store"
)

func mapsEqualSimple[K comparable, V comparable](a, b map[K]V) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		if vb, ok := b[k]; !ok || va != vb {
			return false
		}
	}
	return true
}

func mapsEqualStruct[K comparable, V comparable](a, b map[K]V) bool {
	return mapsEqualSimple(a, b)
}

func mapsEqualSliceValues[K comparable, V comparable](a, b map[K][]V) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || !slices.Equal(va, vb) {
			return false
		}
	}
	return true
}

func gimmickStateEqual(a, b store.GimmickState) bool {
	return mapsEqualStruct(a.Progress, b.Progress) &&
		mapsEqualStruct(a.OrnamentProgress, b.OrnamentProgress) &&
		mapsEqualStruct(a.Sequences, b.Sequences) &&
		mapsEqualStruct(a.Unlocks, b.Unlocks)
}

func ChangedTables(before, after *store.UserState) []string {
	var changed []string
	add := func(name string) { changed = append(changed, name) }

	if before.UserId != after.UserId || before.PlayerId != after.PlayerId ||
		before.OsType != after.OsType || before.PlatformType != after.PlatformType ||
		before.UserRestrictionType != after.UserRestrictionType ||
		before.RegisterDatetime != after.RegisterDatetime ||
		before.GameStartDatetime != after.GameStartDatetime ||
		before.LatestVersion != after.LatestVersion {
		add("IUser")
	}
	if before.Setting != after.Setting {
		add("IUserSetting")
	}
	if before.Status != after.Status {
		add("IUserStatus")
	}
	if before.Gem != after.Gem {
		add("IUserGem")
	}
	if before.Profile != after.Profile {
		add("IUserProfile")
	}
	if before.Login != after.Login {
		add("IUserLogin")
	}
	if before.BeginnerCampaign != after.BeginnerCampaign {
		add("IUserBeginnerCampaign")
	}
	if before.ComebackCampaign != after.ComebackCampaign {
		add("IUserComebackCampaign")
	}
	if !mapsEqualStruct(before.LoginBonuses, after.LoginBonuses) {
		add("IUserLoginBonus")
	}
	if before.PortalCageStatus != after.PortalCageStatus {
		add("IUserPortalCageStatus")
	}
	if before.GuerrillaFreeOpen != after.GuerrillaFreeOpen {
		add("IUserEventQuestGuerrillaFreeOpen")
	}
	if before.ShopReplaceable != after.ShopReplaceable {
		add("IUserShopReplaceable")
	}
	if before.Explore != after.Explore {
		add("IUserExplore")
	}
	if before.BigHuntProgress != after.BigHuntProgress {
		add("IUserBigHuntProgressStatus")
	}
	if before.FacebookId != after.FacebookId {
		add("IUserFacebook")
	}

	if before.MainQuest != after.MainQuest {
		add("IUserMainQuestFlowStatus")
		add("IUserMainQuestMainFlowStatus")
		add("IUserMainQuestProgressStatus")
		add("IUserMainQuestReplayFlowStatus")
		// IUserMainQuestSeasonRoute is derived from MainQuest + Quests at projection
		// time (see proj_quest.go / questflow.QuestHandler.SeasonRoutesFor) — flag it
		// whenever either of those upstream inputs changes.
		add("IUserMainQuestSeasonRoute")
	}
	if before.EventQuest != after.EventQuest {
		add("IUserEventQuestProgressStatus")
	}
	if before.ExtraQuest != after.ExtraQuest {
		add("IUserExtraQuestProgressStatus")
	}
	if before.SideStoryActiveProgress != after.SideStoryActiveProgress {
		add("IUserSideStoryQuestSceneProgressStatus")
	}

	if !mapsEqualStruct(before.Tutorials, after.Tutorials) {
		add("IUserTutorialProgress")
	}
	if !mapsEqualStruct(before.Missions, after.Missions) {
		add("IUserMission")
		add("IUserMissionCompletionProgress")
	}
	if !mapsEqualStruct(before.MissionPassPoints, after.MissionPassPoints) {
		add("IUserMissionPassPoint")
	}
	if !mapsEqualStruct(before.QuestReplayFlowRewards, after.QuestReplayFlowRewards) {
		add("IUserQuestReplayFlowRewardGroup")
	}
	if !mapsEqualStruct(before.QuestSceneChoices, after.QuestSceneChoices) {
		add("IUserQuestSceneChoice")
	}
	if !mapsEqualStruct(before.QuestSceneChoiceHistory, after.QuestSceneChoiceHistory) {
		add("IUserQuestSceneChoiceHistory")
	}
	if !mapsEqualStruct(before.EventQuestDailyRewards, after.EventQuestDailyRewards) {
		add("IUserEventQuestDailyGroupCompleteReward")
	}
	if !mapsEqualStruct(before.WebviewPanelMissions, after.WebviewPanelMissions) {
		add("IUserWebviewPanelMission")
	}
	if !mapsEqualStruct(before.Characters, after.Characters) {
		add("IUserCharacter")
	}
	if !mapsEqualStruct(before.CharacterViewerFields, after.CharacterViewerFields) {
		add("IUserCharacterViewerField")
	}
	if !mapsEqualStruct(before.Costumes, after.Costumes) {
		add("IUserCostume")
	}
	if !mapsEqualStruct(before.Weapons, after.Weapons) {
		add("IUserWeapon")
	}
	if !mapsEqualStruct(before.WeaponStories, after.WeaponStories) {
		add("IUserWeaponStory")
	}
	if !mapsEqualStruct(before.WeaponNotes, after.WeaponNotes) {
		add("IUserWeaponNote")
	}
	if !mapsEqualStruct(before.Companions, after.Companions) {
		add("IUserCompanion")
	}
	if !mapsEqualStruct(before.Thoughts, after.Thoughts) {
		add("IUserThought")
	}
	if !mapsEqualSimple(before.ConsumableItems, after.ConsumableItems) {
		add("IUserConsumableItem")
	}
	if !mapsEqualSimple(before.Materials, after.Materials) {
		add("IUserMaterial")
	}
	if !mapsEqualSimple(before.ImportantItems, after.ImportantItems) {
		add("IUserImportantItem")
	}
	if !mapsEqualSimple(before.PremiumItems, after.PremiumItems) {
		add("IUserPremiumItem")
	}
	if !mapsEqualStruct(before.Parts, after.Parts) {
		add("IUserParts")
	}
	if !mapsEqualStruct(before.PartsGroupNotes, after.PartsGroupNotes) {
		add("IUserPartsGroupNote")
	}
	if !mapsEqualStruct(before.PartsPresets, after.PartsPresets) {
		add("IUserPartsPreset")
	}
	if !mapsEqualStruct(before.PartsPresetTags, after.PartsPresetTags) {
		add("IUserPartsPresetTag")
	}
	if !mapsEqualStruct(before.PartsStatusSubs, after.PartsStatusSubs) {
		add("IUserPartsStatusSub")
	}
	if !mapsEqualStruct(before.CostumeActiveSkills, after.CostumeActiveSkills) {
		add("IUserCostumeActiveSkill")
	}
	if !mapsEqualSliceValues(before.WeaponSkills, after.WeaponSkills) {
		add("IUserWeaponSkill")
	}
	if !mapsEqualSliceValues(before.WeaponAbilities, after.WeaponAbilities) {
		add("IUserWeaponAbility")
	}
	if !mapsEqualStruct(before.WeaponAwakens, after.WeaponAwakens) {
		add("IUserWeaponAwaken")
	}
	if !mapsEqualStruct(before.DeckTypeNotes, after.DeckTypeNotes) {
		add("IUserDeckTypeNote")
	}
	if !mapsEqualStruct(before.DeckCharacters, after.DeckCharacters) {
		add("IUserDeckCharacter")
		add("IUserDeckCharacterDressupCostume")
	}
	if !mapsEqualStruct(before.Decks, after.Decks) {
		add("IUserDeck")
	}
	if !mapsEqualStruct(before.TripleDecks, after.TripleDecks) {
		add("IUserTripleDeck")
	}
	if !mapsEqualSliceValues(before.DeckSubWeapons, after.DeckSubWeapons) {
		add("IUserDeckSubWeaponGroup")
	}
	if !mapsEqualSliceValues(before.DeckParts, after.DeckParts) {
		add("IUserDeckPartsGroup")
	}
	if !mapsEqualStruct(before.Quests, after.Quests) {
		add("IUserQuest")
		add("IUserMainQuestSeasonRoute")
	}
	if !mapsEqualStruct(before.QuestMissions, after.QuestMissions) {
		add("IUserQuestMission")
	}
	if !mapsEqualStruct(before.SideStoryQuests, after.SideStoryQuests) {
		add("IUserSideStoryQuest")
	}
	if !mapsEqualStruct(before.QuestLimitContentStatus, after.QuestLimitContentStatus) {
		add("IUserQuestLimitContentStatus")
	}
	if !mapsEqualSimple(before.NaviCutInPlayed, after.NaviCutInPlayed) {
		add("IUserNaviCutIn")
	}
	if !mapsEqualSimple(before.ViewedMovies, after.ViewedMovies) {
		add("IUserMovie")
	}
	if !mapsEqualSimple(before.ContentsStories, after.ContentsStories) {
		add("IUserContentsStory")
	}
	if !mapsEqualSimple(before.DrawnOmikuji, after.DrawnOmikuji) {
		add("IUserOmikuji")
	}
	if !mapsEqualSimple(before.DokanConfirmed, after.DokanConfirmed) {
		add("IUserDokan")
	}
	if !mapsEqualStruct(before.ShopItems, after.ShopItems) {
		add("IUserShopItem")
	}
	if !mapsEqualStruct(before.ShopReplaceableLineup, after.ShopReplaceableLineup) {
		add("IUserShopReplaceableLineup")
	}
	if !mapsEqualStruct(before.ExploreScores, after.ExploreScores) {
		add("IUserExploreScore")
	}
	if !mapsEqualStruct(before.CharacterBoards, after.CharacterBoards) {
		add("IUserCharacterBoard")
	}
	if !mapsEqualStruct(before.DeckLimitContentRestricted, after.DeckLimitContentRestricted) {
		add("IUserDeckLimitContentRestricted")
	}
	if !mapsEqualStruct(before.CharacterBoardAbilities, after.CharacterBoardAbilities) {
		add("IUserCharacterBoardAbility")
	}
	if !mapsEqualStruct(before.CharacterBoardStatusUps, after.CharacterBoardStatusUps) {
		add("IUserCharacterBoardStatusUp")
	}
	if !mapsEqualStruct(before.CostumeAwakenStatusUps, after.CostumeAwakenStatusUps) {
		add("IUserCostumeAwakenStatusUp")
	}
	if !mapsEqualStruct(before.CostumeLotteryEffects, after.CostumeLotteryEffects) {
		add("IUserCostumeLotteryEffect")
	}
	if !mapsEqualStruct(before.CostumeLotteryEffectAbilities, after.CostumeLotteryEffectAbilities) {
		add("IUserCostumeLotteryEffectAbility")
	}
	if !mapsEqualStruct(before.CostumeLotteryEffectStatusUps, after.CostumeLotteryEffectStatusUps) {
		add("IUserCostumeLotteryEffectStatusUp")
	}
	if !mapsEqualStruct(before.CostumeLevelBonusReleaseStatuses, after.CostumeLevelBonusReleaseStatuses) {
		add("IUserCostumeLevelBonusReleaseStatus")
	}
	if !mapsEqualStruct(before.CostumeLotteryEffectPending, after.CostumeLotteryEffectPending) {
		add("IUserCostumeLotteryEffectPending")
	}
	if !mapsEqualStruct(before.AutoSaleSettings, after.AutoSaleSettings) {
		add("IUserAutoSaleSettingDetail")
	}
	if !mapsEqualStruct(before.CharacterRebirths, after.CharacterRebirths) {
		add("IUserCharacterRebirth")
	}
	if !mapsEqualStruct(before.CageOrnamentRewards, after.CageOrnamentRewards) {
		add("IUserCageOrnamentReward")
	}
	if !mapsEqualStruct(before.TowerAccumulationRewards, after.TowerAccumulationRewards) {
		add("IUserEventQuestTowerAccumulationReward")
	}
	if !before.QuestAutoOrbit.Equal(after.QuestAutoOrbit) {
		add("IUserQuestAutoOrbit")
	}
	if !mapsEqualStruct(before.LabyrinthStages, after.LabyrinthStages) {
		add("IUserEventQuestLabyrinthStage")
	}
	if !mapsEqualStruct(before.LabyrinthSeasons, after.LabyrinthSeasons) {
		add("IUserEventQuestLabyrinthSeason")
	}

	if !mapsEqualStruct(before.BigHuntMaxScores, after.BigHuntMaxScores) {
		add("IUserBigHuntMaxScore")
	}
	if !mapsEqualStruct(before.BigHuntStatuses, after.BigHuntStatuses) {
		add("IUserBigHuntStatus")
	}
	if !mapsEqualStruct(before.BigHuntScheduleMaxScores, after.BigHuntScheduleMaxScores) {
		add("IUserBigHuntScheduleMaxScore")
	}
	if !mapsEqualStruct(before.BigHuntWeeklyMaxScores, after.BigHuntWeeklyMaxScores) {
		add("IUserBigHuntWeeklyMaxScore")
	}
	if !mapsEqualStruct(before.BigHuntWeeklyStatuses, after.BigHuntWeeklyStatuses) {
		add("IUserBigHuntWeeklyStatus")
	}

	if !gimmickStateEqual(before.Gimmick, after.Gimmick) {
		if !mapsEqualStruct(before.Gimmick.Progress, after.Gimmick.Progress) ||
			!mapsEqualStruct(before.Gimmick.Sequences, after.Gimmick.Sequences) {
			add("IUserGimmick")
		}
		if !mapsEqualStruct(before.Gimmick.OrnamentProgress, after.Gimmick.OrnamentProgress) ||
			!mapsEqualStruct(before.Gimmick.Sequences, after.Gimmick.Sequences) {
			add("IUserGimmickOrnamentProgress")
		}
		if !mapsEqualStruct(before.Gimmick.Sequences, after.Gimmick.Sequences) {
			add("IUserGimmickSequence")
		}
		if !mapsEqualStruct(before.Gimmick.Unlocks, after.Gimmick.Unlocks) {
			add("IUserGimmickUnlock")
		}
	}

	return changed
}

func ComputeDelta(before, after *store.UserState, changedTables []string) map[string]*pb.DiffData {
	diff := make(map[string]*pb.DiffData, len(changedTables))
	for _, table := range changedTables {
		afterJSON := projectTable(table, *after)
		updates := afterJSON
		deleteKeys := "[]"
		if kf := keyFieldsForTable(table); len(kf) > 0 {
			beforeRecs := parseJSONRecords(projectTable(table, *before))
			afterRecs := parseJSONRecords(afterJSON)
			updates = ComputeUpdateRecords(beforeRecs, afterRecs, kf)
			deleteKeys = ComputeDeleteKeys(beforeRecs, afterRecs, kf)
		}
		diff[table] = &pb.DiffData{
			UpdateRecordsJson: updates,
			DeleteKeysJson:    deleteKeys,
		}
	}
	return diff
}

func AllTableNames() []string {
	return slices.Sorted(maps.Keys(projectors))
}

func SortedChangedNames(tables []string) string {
	sorted := make([]string, len(tables))
	copy(sorted, tables)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

func parseJSONRecords(jsonStr string) []map[string]any {
	if jsonStr == "" || jsonStr == "[]" {
		return nil
	}
	var records []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &records); err != nil {
		return nil
	}
	return records
}

func keyFieldsForTable(table string) []string {
	switch table {
	case "IUserBeginnerCampaign", "IUserComebackCampaign":
		return []string{"userId"}
	case "IUserLoginBonus":
		return []string{"userId", "loginBonusId"}
	case "IUserWeapon":
		return []string{"userId", "userWeaponUuid"}
	case "IUserWeaponSkill":
		return []string{"userId", "userWeaponUuid", "slotNumber"}
	case "IUserWeaponAbility":
		return []string{"userId", "userWeaponUuid", "slotNumber"}
	case "IUserWeaponAwaken":
		return []string{"userId", "userWeaponUuid"}
	case "IUserCostume":
		return []string{"userId", "userCostumeUuid"}
	case "IUserCompanion":
		return []string{"userId", "userCompanionUuid"}
	case "IUserThought":
		return []string{"userId", "userThoughtUuid"}
	case "IUserParts":
		return []string{"userId", "userPartsUuid"}
	case "IUserPartsStatusSub":
		return []string{"userId", "userPartsUuid", "statusIndex"}
	case "IUserDeckCharacter":
		return []string{"userId", "userDeckCharacterUuid"}
	case "IUserDeck":
		return []string{"userId", "deckType", "userDeckNumber"}
	case "IUserTripleDeck":
		return []string{"userId", "deckType", "userDeckNumber"}
	case "IUserDeckSubWeaponGroup":
		return []string{"userId", "userDeckCharacterUuid", "sortOrder"}
	case "IUserDeckPartsGroup":
		return []string{"userId", "userDeckCharacterUuid", "sortOrder"}
	case "IUserDeckCharacterDressupCostume":
		return []string{"userId", "userDeckCharacterUuid"}
	case "IUserDeckLimitContentRestricted":
		return []string{"userId", "deckRestrictedUuid"}
	case "IUserCharacter":
		return []string{"userId", "characterId"}
	case "IUserCharacterViewerField":
		return []string{"userId", "characterViewerFieldId"}
	case "IUserConsumableItem":
		return []string{"userId", "consumableItemId"}
	case "IUserMaterial":
		return []string{"userId", "materialId"}
	case "IUserImportantItem":
		return []string{"userId", "importantItemId"}
	case "IUserPremiumItem":
		return []string{"userId", "premiumItemId"}
	case "IUserQuest":
		return []string{"userId", "questId"}
	case "IUserQuestMission":
		return []string{"userId", "questId", "questMissionId"}
	case "IUserMission":
		return []string{"userId", "missionId"}
	case "IUserMissionPassPoint":
		return []string{"userId", "missionPassId"}
	case "IUserMissionCompletionProgress":
		return []string{"userId", "missionId"}
	case "IUserQuestReplayFlowRewardGroup":
		return []string{"userId", "questReplayFlowRewardGroupId"}
	case "IUserQuestSceneChoice":
		return []string{"userId", "questSceneId", "questFlowType"}
	case "IUserQuestSceneChoiceHistory":
		return []string{"userId", "questSceneId", "questFlowType", "choiceNumber"}
	case "IUserEventQuestDailyGroupCompleteReward":
		return []string{"userId", "eventQuestDailyGroupId"}
	case "IUserWebviewPanelMission":
		return []string{"userId", "webviewPanelMissionPageId"}
	case "IUserWeaponStory":
		return []string{"userId", "weaponId"}
	case "IUserWeaponNote":
		return []string{"userId", "weaponId"}
	case "IUserTutorialProgress":
		return []string{"userId", "tutorialType"}
	case "IUserGimmick":
		return []string{"userId", "gimmickSequenceScheduleId", "gimmickSequenceId", "gimmickId"}
	case "IUserGimmickOrnamentProgress":
		return []string{"userId", "gimmickSequenceScheduleId", "gimmickSequenceId", "gimmickId", "gimmickOrnamentIndex"}
	case "IUserGimmickSequence":
		return []string{"userId", "gimmickSequenceScheduleId", "gimmickSequenceId"}
	case "IUserGimmickUnlock":
		return []string{"userId", "gimmickSequenceScheduleId", "gimmickSequenceId", "gimmickId"}
	case "IUserCostumeActiveSkill":
		return []string{"userId", "userCostumeUuid"}
	case "IUserCostumeAwakenStatusUp":
		return []string{"userId", "userCostumeUuid", "statusCalculationType"}
	case "IUserCostumeLotteryEffect":
		return []string{"userId", "userCostumeUuid", "slotNumber"}
	case "IUserCostumeLotteryEffectPending":
		return []string{"userId", "userCostumeUuid"}
	case "IUserCostumeLevelBonusReleaseStatus":
		return []string{"userId", "costumeId"}
	case "IUserCostumeLotteryEffectAbility":
		return []string{"userId", "userCostumeUuid", "slotNumber"}
	case "IUserCostumeLotteryEffectStatusUp":
		return []string{"userId", "userCostumeUuid", "statusCalculationType"}
	case "IUserCharacterBoard":
		return []string{"userId", "characterBoardId"}
	case "IUserCharacterBoardAbility":
		return []string{"userId", "characterId", "abilityId"}
	case "IUserCharacterBoardStatusUp":
		return []string{"userId", "characterId", "statusCalculationType"}
	case "IUserExploreScore":
		return []string{"userId", "exploreId"}
	case "IUserPartsGroupNote":
		return []string{"userId", "partsGroupId"}
	case "IUserPartsPreset":
		return []string{"userId", "userPartsPresetNumber"}
	case "IUserPartsPresetTag":
		return []string{"userId", "userPartsPresetTagNumber"}
	case "IUserCageOrnamentReward":
		return []string{"userId", "cageOrnamentId"}
	case "IUserEventQuestTowerAccumulationReward":
		return []string{"userId", "eventQuestChapterId"}
	case "IUserEventQuestLabyrinthStage":
		return []string{"userId", "eventQuestChapterId", "stageOrder"}
	case "IUserEventQuestLabyrinthSeason":
		return []string{"userId", "eventQuestChapterId"}
	case "IUserAutoSaleSettingDetail":
		return []string{"userId", "possessionAutoSaleItemType"}
	case "IUserCharacterRebirth":
		return []string{"userId", "characterId"}
	case "IUserShopItem":
		return []string{"userId", "shopItemId"}
	case "IUserShopReplaceableLineup":
		return []string{"userId", "slotNumber"}
	case "IUserNaviCutIn":
		return []string{"userId", "naviCutInId"}
	case "IUserMovie":
		return []string{"userId", "movieId"}
	case "IUserContentsStory":
		return []string{"userId", "contentsStoryId"}
	case "IUserOmikuji":
		return []string{"userId", "omikujiId"}
	case "IUserDokan":
		return []string{"userId", "dokanId"}
	case "IUserSideStoryQuest":
		return []string{"userId", "sideStoryQuestId"}
	case "IUserMainQuestSeasonRoute":
		return []string{"userId", "mainQuestSeasonId", "mainQuestRouteId"}
	case "IUserQuestLimitContentStatus":
		return []string{"userId", "questId"}
	case "IUserBigHuntMaxScore":
		return []string{"userId", "bigHuntBossId"}
	case "IUserBigHuntStatus":
		return []string{"userId", "bigHuntBossQuestId"}
	case "IUserBigHuntScheduleMaxScore":
		return []string{"userId", "bigHuntScheduleId", "bigHuntBossId"}
	case "IUserBigHuntWeeklyMaxScore":
		return []string{"userId", "bigHuntWeeklyVersion", "attributeType"}
	case "IUserBigHuntWeeklyStatus":
		return []string{"userId", "bigHuntWeeklyVersion"}
	case "IUserDeckTypeNote":
		return []string{"userId", "deckType"}
	case "IUserQuestAutoOrbit":
		return []string{"userId"}
	default:
		return nil
	}
}

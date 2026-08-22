package missionprogress

import (
	"math"
	"math/bits"
	"sort"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

const (
	missionCategoryDaily                    = int32(1)
	missionCategoryMissionPassDaily         = int32(9)
	maxCascadePasses                        = 16
	questClearOptionMainQuest               = int32(4)
	questClearOptionSubquest                = int32(5)
	questClearOptionMainQuestHard           = int32(8)
	questClearOptionMainQuestHardOrVeryHard = int32(9)
	questClearOptionSubquestAlt             = int32(70)
	questClearOptionDarkMemory              = int32(71)
	questClearOptionDailyChallenge          = int32(84)
	questClearOptionMainQuestHardAlt        = int32(100004)
	questClearOptionMainQuestVeryHard       = int32(100021)
	questClearOptionAbyssTower              = int32(100024)
	questClearOptionFateBoard               = int32(100025)
	questClearOptionDailyQuest              = int32(30029)
	questClearOptionGuerrilla               = int32(30030)
	questClearOptionDungeon                 = int32(101030801)
	mainQuestDifficultyHard                 = int32(2)
	mainQuestDifficultyVeryHard             = int32(3)
	eventQuestTypeMarathon                  = int32(1)
	eventQuestTypeHunt                      = int32(2)
	eventQuestTypeDungeon                   = int32(3)
	eventQuestTypeDayOfTheWeek              = int32(4)
	eventQuestTypeGuerrilla                 = int32(5)
	eventQuestTypeCharacter                 = int32(6)
	eventQuestTypeTower                     = int32(10)
	eventQuestTypeLimitContent              = int32(11)
	eventQuestTypeLabyrinth                 = int32(12)
	eventQuestTypeCharacterQuest            = int32(7)
	gachaOptionChapterSummon                = int32(100001)
	gachaOptionDailySummon                  = int32(100026)
	shopOptionItemShop                      = int32(2)
	mainFunctionTypeExploration             = int32(4)
	missionLinkDestinationQuest             = int32(4)
)

// Apply reconciles state-derived conditions and applies transaction-local
// events. Newly unlocked FROM_UNLOCK missions never inherit earlier account
// totals; that distinction is the reason those enum values exist.
func Apply(catalogs *runtime.Catalogs, before *store.UserState, user *store.UserState, events []store.MissionEvent, nowMillis int64) {
	if catalogs == nil || catalogs.Mission == nil || user == nil {
		return
	}
	user.EnsureMaps()
	removeMissionsForLockedFunctions(catalogs, user)
	resetDailyMissions(catalogs.Mission, user, nowMillis)

	eligibleForEvent := make(map[int32]bool)
	for _, mission := range orderedMissions(catalogs.Mission) {
		conditionType := model.MissionClearConditionType(mission.MissionClearConditionType)
		if !conditionType.IsFromUnlock() || before != nil && unlocked(catalogs, before, mission, nowMillis) {
			eligibleForEvent[mission.MissionId] = true
		}
	}

	reconcile(catalogs, user, nowMillis)
	for _, event := range append(deriveEvents(catalogs, before, user), events...) {
		applyEvent(catalogs, user, event, eligibleForEvent, nowMillis)
	}
	reconcile(catalogs, user, nowMillis)
}

// Sync performs the non-event half of Apply. It is useful for first login and
// for mission RPCs that only need unlock/cascade refresh.
func Sync(catalogs *runtime.Catalogs, user *store.UserState, nowMillis int64) {
	Apply(catalogs, nil, user, nil, nowMillis)
}

func reconcile(catalogs *runtime.Catalogs, user *store.UserState, nowMillis int64) {
	for range maxCascadePasses {
		changed := false
		for _, mission := range orderedMissions(catalogs.Mission) {
			if !model.MissionClearConditionType(mission.MissionClearConditionType).IsKnown() {
				continue
			}
			if !missionActive(catalogs.Mission, mission, nowMillis) || !unlocked(catalogs, user, mission, nowMillis) {
				continue
			}
			state, exists := user.Missions[mission.MissionId]
			if !exists {
				state = store.UserMissionState{
					MissionId: mission.MissionId, StartDatetime: nowMillis,
					MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress), LatestVersion: nowMillis,
				}
				user.Missions[mission.MissionId] = state
				changed = true
			}
			if state.MissionProgressStatusType >= int32(model.MissionProgressStatusTypeClear) {
				continue
			}
			if value, ok := absoluteProgress(catalogs, user, mission, state, nowMillis); ok && absoluteProgressImproved(mission, state.ProgressValue, value) {
				state.ProgressValue = value
				state.LatestVersion = nowMillis
				user.Missions[mission.MissionId] = state
				changed = true
			}
			state = user.Missions[mission.MissionId]
			if conditionSatisfied(catalogs, user, mission, state.ProgressValue, nowMillis) {
				state.MissionProgressStatusType = int32(model.MissionProgressStatusTypeClear)
				state.ClearDatetime = nowMillis
				state.LatestVersion = nowMillis
				user.Missions[mission.MissionId] = state
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}

func absoluteProgressImproved(mission masterdata.EntityMMission, current, candidate int32) bool {
	if model.MissionClearConditionType(mission.MissionClearConditionType) == model.MissionClearConditionTypePvpRank {
		return candidate > 0 && (current == 0 || candidate < current)
	}
	return candidate > current
}

func missionActive(catalog *masterdata.MissionCatalog, mission masterdata.EntityMMission, nowMillis int64) bool {
	if mission.MissionTermId == 0 {
		return true
	}
	term, ok := catalog.TermById[mission.MissionTermId]
	return ok && nowMillis >= term.StartDatetime && (term.EndDatetime == 0 || nowMillis < term.EndDatetime)
}

func unlocked(catalogs *runtime.Catalogs, user *store.UserState, mission masterdata.EntityMMission, nowMillis int64) bool {
	if !relatedMainFunctionUnlocked(catalogs, user, mission) {
		return false
	}
	if mission.MissionUnlockConditionId == 0 {
		return true
	}
	condition, ok := catalogs.Mission.UnlockById[mission.MissionUnlockConditionId]
	if !ok {
		return false
	}
	switch model.MissionUnlockConditionType(condition.MissionUnlockConditionType) {
	case model.MissionUnlockConditionTypeGrant:
		return true
	case model.MissionUnlockConditionTypeQuestClear:
		return user.Quests[condition.ConditionValue].QuestStateType == model.UserQuestStateTypeCleared
	case model.MissionUnlockConditionTypeMissionClearById:
		return missionCleared(user, condition.ConditionValue)
	case model.MissionUnlockConditionTypeMissionClearForAllDaily:
		return allDailyCleared(catalogs, user, 0, mission.MissionId, nowMillis)
	case model.MissionUnlockConditionTypeWebviewPanelMissionClearByPageNumber:
		for pageId, state := range user.WebviewPanelMissions {
			if state.RewardReceiveDatetime > 0 && catalogs.Mission.WebviewPageNumberByPageId[pageId] == condition.ConditionValue {
				return true
			}
		}
		return false
	case model.MissionUnlockConditionTypeEvaluate:
		return catalogs.ConditionResolver != nil && catalogs.ConditionResolver.Satisfied(condition.ConditionValue, user)
	default:
		return false
	}
}

func removeMissionsForLockedFunctions(catalogs *runtime.Catalogs, user *store.UserState) {
	for missionId := range user.Missions {
		mission, ok := catalogs.Mission.MissionById[missionId]
		if ok && !relatedMainFunctionUnlocked(catalogs, user, mission) {
			delete(user.Missions, missionId)
		}
	}
}

func relatedMainFunctionUnlocked(catalogs *runtime.Catalogs, user *store.UserState, mission masterdata.EntityMMission) bool {
	if mission.RelatedMainFunctionType != mainFunctionTypeExploration {
		return true
	}
	if catalogs.Explore == nil || catalogs.Explore.FirstExploreId == 0 {
		return false
	}
	explore, ok := catalogs.Explore.Explores[catalogs.Explore.FirstExploreId]
	if !ok || explore.ExploreUnlockConditionId == 0 {
		return ok
	}
	condition, ok := catalogs.Explore.UnlockConditions[explore.ExploreUnlockConditionId]
	if !ok {
		return false
	}
	switch condition.ExploreUnlockConditionType {
	case 1:
		return user.Quests[condition.ConditionValue].QuestStateType == model.UserQuestStateTypeCleared
	case 2:
		lowerExploreId := catalogs.Explore.LowerDifficulty[explore.ExploreId]
		return lowerExploreId != 0 && user.ExploreScores[lowerExploreId].MaxScore >= condition.ConditionValue
	default:
		return false
	}
}

func resetDailyMissions(catalog *masterdata.MissionCatalog, user *store.UserState, nowMillis int64) {
	today := gametime.StartOfBusinessDayAtMillis(nowMillis)
	for id, state := range user.Missions {
		mission, ok := catalog.MissionById[id]
		if !ok || !isDaily(catalog, mission) || state.StartDatetime >= today {
			continue
		}
		user.Missions[id] = store.UserMissionState{
			MissionId: id, StartDatetime: today,
			MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress), LatestVersion: nowMillis,
		}
	}
}

func isDaily(catalog *masterdata.MissionCatalog, mission masterdata.EntityMMission) bool {
	category := catalog.GroupById[mission.MissionGroupId].MissionCategoryType
	return category == missionCategoryDaily || category == missionCategoryMissionPassDaily
}

func isDailyAggregateMember(catalog *masterdata.MissionCatalog, mission masterdata.EntityMMission) bool {
	return catalog.GroupById[mission.MissionGroupId].MissionCategoryType == missionCategoryDaily
}

func absoluteProgress(catalogs *runtime.Catalogs, user *store.UserState, mission masterdata.EntityMMission, state store.UserMissionState, nowMillis int64) (int32, bool) {
	t := model.MissionClearConditionType(mission.MissionClearConditionType)
	switch t {
	case model.MissionClearConditionTypeQuestClearByCount:
		return questClearCount(catalogs, user, mission, state), true
	case model.MissionClearConditionTypeQuestClearById:
		if questId := missionTargetId(mission); questId != 0 && questClearedForMission(catalogs, user.Quests[questId], mission, state) {
			return 1, true
		}
		return 0, true
	case model.MissionClearConditionTypeUserLevel:
		return user.Status.Level, true
	case model.MissionClearConditionTypeUserLoginByCount:
		if isDaily(catalogs.Mission, mission) {
			if user.Login.LastLoginDatetime >= state.StartDatetime {
				return 1, true
			}
			return 0, true
		}
		return user.Login.TotalLoginCount, true
	case model.MissionClearConditionTypeMissionClearByCount:
		var count int32
		for id, s := range user.Missions {
			if id != mission.MissionId && s.MissionProgressStatusType >= int32(model.MissionProgressStatusTypeClear) &&
				(!isDaily(catalogs.Mission, mission) || s.ClearDatetime >= state.StartDatetime) {
				count++
			}
		}
		return count, true
	case model.MissionClearConditionTypeSetFavoriteCharacter:
		return boolValue(user.Profile.FavoriteCostumeId != 0), true
	case model.MissionClearConditionTypeMaxDeckPower:
		var value int32
		for _, deck := range user.Decks {
			value = max(value, deck.Power)
		}
		for _, note := range user.DeckTypeNotes {
			value = max(value, note.MaxDeckPower)
		}
		return value, true
	case model.MissionClearConditionTypeExploreHighScore:
		return exploreHighScore(user, mission), true
	case model.MissionClearConditionTypeMissionClearForAllDaily:
		return boolValue(allDailyCleared(catalogs, user, 0, mission.MissionId, nowMillis)), true
	case model.MissionClearConditionTypeCageOrnamentRewardAcquiredByCount:
		return clampInt(len(user.CageOrnamentRewards)), true
	case model.MissionClearConditionTypeCompleteTransferSettings:
		return boolValue(user.FacebookId != 0 || user.BackupToken != "" && user.BackupToken != "mock-backup-token"), true
	case model.MissionClearConditionTypeLibraryElementCount:
		return libraryElementCount(user), true
	case model.MissionClearConditionTypeCostumeMaxLevel:
		return maxCostumeLevel(user, mission), true
	case model.MissionClearConditionTypeCostumeSkillMaxLevel:
		return maxCostumeSkillLevel(user), true
	case model.MissionClearConditionTypeCostumeAbilityMaxLevel:
		return maxCostumeAbilityLevel(catalogs, user), true
	case model.MissionClearConditionTypeWeaponMaxLevel:
		return maxWeaponLevel(user, mission), true
	case model.MissionClearConditionTypeWeaponSkillMaxLevel:
		return maxWeaponSkillLevel(user), true
	case model.MissionClearConditionTypeWeaponAbilityMaxLevel:
		return maxWeaponAbilityLevel(user), true
	case model.MissionClearConditionTypeCompanionMaxLevel:
		return maxCompanionLevel(user), true
	case model.MissionClearConditionTypePartsMaxLevel:
		return maxPartsLevel(user), true
	case model.MissionClearConditionTypePossessionComplete:
		return completePossessionCount(catalogs.Mission, user, mission.MissionId), true
	case model.MissionClearConditionTypeBigHuntHighScore:
		return bigHuntHighScore(catalogs, user, mission), true
	case model.MissionClearConditionTypeCharacterBoardPanelReleaseByCount:
		return releasedPanelCount(catalogs, user, mission), true
	case model.MissionClearConditionTypeReportCount:
		return reportCount(catalogs, user), true
	case model.MissionClearConditionTypeCharacterBoardFullOpen:
		return fullOpenBoardCount(catalogs, user, mission), true
	case model.MissionClearConditionTypeWeaponCountWithLevelGE:
		return weaponCountWithLevelGE(user, mission), true
	case model.MissionClearConditionTypeCostumeAwakenCount:
		return costumeAwakenCount(user, mission), true
	case model.MissionClearConditionTypeMissionClearForAllDailyBySubCategoryId:
		return boolValue(allDailyCleared(catalogs, user, mission.ClearConditionValue, mission.MissionId, nowMillis)), true
	case model.MissionClearConditionTypePvpRank:
		return user.Profile.CurrentPvpRank, true
	case model.MissionClearConditionTypeWeaponAwakenCount:
		return clampInt(len(user.WeaponAwakens)), true
	case model.MissionClearConditionTypeCharacterRebirthCount:
		return characterRebirthCount(catalogs, user, mission), true
	case model.MissionClearConditionTypeCostumeLotteryEffectSlotUnlockCount:
		return costumeLotterySlotCount(user), true
	default:
		return 0, false
	}
}

func conditionSatisfied(catalogs *runtime.Catalogs, user *store.UserState, mission masterdata.EntityMMission, progress int32, nowMillis int64) bool {
	t := model.MissionClearConditionType(mission.MissionClearConditionType)
	switch t {
	case model.MissionClearConditionTypeUnknown:
		return false
	case model.MissionClearConditionTypeQuestClearById:
		return progress >= 1
	case model.MissionClearConditionTypePvpRank:
		return progress > 0 && progress <= mission.ClearConditionValue
	case model.MissionClearConditionTypeMissionClearForAllDaily,
		model.MissionClearConditionTypeMissionClearForAllDailyBySubCategoryId,
		model.MissionClearConditionTypeCompleteTransferSettings,
		model.MissionClearConditionTypeSetFavoriteCharacter:
		return progress >= 1
	default:
		return mission.ClearConditionValue > 0 && progress >= mission.ClearConditionValue
	}
}

func applyEvent(catalogs *runtime.Catalogs, user *store.UserState, event store.MissionEvent, eligible map[int32]bool, nowMillis int64) {
	if event.ConditionType == 0 || !event.Reset && (event.IsValue && event.Value < 0 || !event.IsValue && event.Count <= 0) {
		return
	}
	for _, id := range missionIdsByType(catalogs.Mission, event.ConditionType) {
		if !eligible[id] {
			continue
		}
		mission := catalogs.Mission.MissionById[id]
		if !missionActive(catalogs.Mission, mission, nowMillis) || !unlocked(catalogs, user, mission, nowMillis) || !eventMatches(catalogs, mission, event) {
			continue
		}
		state, ok := user.Missions[id]
		if !ok || state.MissionProgressStatusType >= int32(model.MissionProgressStatusTypeClear) {
			continue
		}
		if event.Reset {
			state.ProgressValue = 0
		} else if event.IsValue {
			state.ProgressValue = max(state.ProgressValue, event.Value)
		} else {
			state.ProgressValue = saturatingAdd(state.ProgressValue, event.Count)
		}
		if conditionSatisfied(catalogs, user, mission, state.ProgressValue, nowMillis) {
			state.MissionProgressStatusType = int32(model.MissionProgressStatusTypeClear)
			state.ClearDatetime = nowMillis
		}
		state.LatestVersion = nowMillis
		user.Missions[id] = state
	}
}

func eventMatches(catalogs *runtime.Catalogs, mission masterdata.EntityMMission, event store.MissionEvent) bool {
	conditionType := model.MissionClearConditionType(mission.MissionClearConditionType)
	if conditionType == model.MissionClearConditionTypeQuestClearByCount ||
		conditionType == model.MissionClearConditionTypeQuestClearByCountWithoutSkip {
		if !questMissionMatches(catalogs, mission, event.TargetId) {
			return false
		}
		if conditionType == model.MissionClearConditionTypeQuestClearByCount {
			requiredCharacterId := requiredDeckCharacterByOption[mission.MissionClearConditionOptionGroupId]
			if requiredCharacterId != 0 {
				if !event.QuestClearWithDeck || !containsTarget(event.DeckCharacterIds, requiredCharacterId) {
					return false
				}
			}
			if requiredCostumeId := requiredDeckCostumeByOption[mission.MissionClearConditionOptionGroupId]; requiredCostumeId != 0 {
				if !event.QuestClearWithDeck || !containsTarget(event.DeckCostumeIds, requiredCostumeId) {
					return false
				}
			}
			// Deck-context clear events exist only for conditions that snapshots
			// cannot reconstruct. Ordinary clear-count missions are reconciled
			// from UserQuestState and must not be incremented a second time.
			if event.QuestClearWithDeck && requiredCharacterId == 0 && requiredDeckCostumeByOption[mission.MissionClearConditionOptionGroupId] == 0 {
				return false
			}
		}
		if requiredCharacterId := requiredSoloCharacterByDetail[mission.MissionClearConditionOptionDetailGroupId]; requiredCharacterId != 0 {
			return len(event.DeckCharacterIds) == 1 && event.DeckCharacterIds[0] == requiredCharacterId
		}
		return true
	}
	if conditionType == model.MissionClearConditionTypeBigHuntHighScore {
		if requiredCharacterIds := requiredBigHuntCharactersByDetail[mission.MissionClearConditionOptionDetailGroupId]; len(requiredCharacterIds) != 0 {
			return event.BigHuntWithDeck && bigHuntBossId(catalogs, mission) == event.TargetId && containsAllTargets(event.DeckCharacterIds, requiredCharacterIds)
		}
		if mission.MissionClearConditionOptionGroupId >= 300001 && mission.MissionClearConditionOptionGroupId <= 300005 {
			return bigHuntBossId(catalogs, mission) == event.TargetId
		}
	}
	if detail := mission.MissionClearConditionOptionDetailGroupId; detail != 0 {
		return detail == event.OptionDetailGroupId || detail == event.TargetId
	}
	option := mission.MissionClearConditionOptionGroupId
	if option == 0 {
		return true
	}
	if conditionType == model.MissionClearConditionTypeGachaDrawByCount && option == gachaOptionChapterSummon {
		entry := gachaEntry(catalogs, event.TargetId)
		return entry != nil && entry.GachaLabelType == model.GachaLabelChapter
	}
	if conditionType == model.MissionClearConditionTypeGachaDrawByCount {
		if targetIds, ok := gachaTargetsByOption[option]; ok {
			return containsTarget(targetIds, event.TargetId)
		}
	}
	if conditionType == model.MissionClearConditionTypeShopBuyByCount && option == shopOptionItemShop {
		return catalogs.Shop != nil && containsTarget(catalogs.Shop.ItemShopPool, event.TargetId)
	}
	if option == event.OptionGroupId || option == event.TargetId {
		return true
	}
	if targetIds, ok := knownOptionTargets(conditionType, option); ok {
		return containsTarget(targetIds, event.TargetId)
	}
	if conditionType == model.MissionClearConditionTypeExploreScore && option == 391 {
		return true
	}
	return false
}

func deriveEvents(catalogs *runtime.Catalogs, before *store.UserState, after *store.UserState) []store.MissionEvent {
	if before == nil {
		return nil
	}
	var events []store.MissionEvent
	add := func(t model.MissionClearConditionType, count, target int32) {
		if count > 0 {
			events = append(events, store.MissionEvent{ConditionType: int32(t), Count: count, TargetId: target})
		}
	}
	set := func(t model.MissionClearConditionType, value, target int32) {
		if value >= 0 {
			events = append(events, store.MissionEvent{ConditionType: int32(t), Value: value, IsValue: true, TargetId: target})
		}
	}

	for uuid, current := range after.Weapons {
		old, existed := before.Weapons[uuid]
		if !existed {
			continue
		}
		if current.Level > old.Level || current.Exp > old.Exp {
			add(model.MissionClearConditionTypeWeaponEnhanceByCount, 1, current.WeaponId)
		}
		if current.WeaponId != old.WeaponId {
			add(model.MissionClearConditionTypeWeaponEvolveByCount, 1, current.WeaponId)
		}
		add(model.MissionClearConditionTypeWeaponLimitBreakByCount, current.LimitBreakCount-old.LimitBreakCount, current.WeaponId)
		if current.IsProtected && !old.IsProtected {
			add(model.MissionClearConditionTypeWeaponProtectByCount, 1, current.WeaponId)
		}
	}
	for uuid, current := range after.WeaponSkills {
		old, existed := before.WeaponSkills[uuid]
		if !existed {
			continue
		}
		add(model.MissionClearConditionTypeWeaponEnhanceSkillByCount, positiveLevelDelta(old, current), after.Weapons[uuid].WeaponId)
	}
	for uuid, current := range after.Costumes {
		old, existed := before.Costumes[uuid]
		if !existed {
			continue
		}
		if current.Level > old.Level || current.Exp > old.Exp {
			add(model.MissionClearConditionTypeCostumeEnhanceByCount, 1, current.CostumeId)
		}
		add(model.MissionClearConditionTypeCostumeLimitBreakByCount, current.LimitBreakCount-old.LimitBreakCount, current.CostumeId)
	}
	for uuid, current := range after.CostumeActiveSkills {
		if old, existed := before.CostumeActiveSkills[uuid]; existed && current.Level > old.Level {
			add(model.MissionClearConditionTypeCostumeActiveSkillEnhanceByCount, current.Level-old.Level, after.Costumes[uuid].CostumeId)
		}
	}
	for uuid, current := range after.Companions {
		old, existed := before.Companions[uuid]
		if !existed {
		} else if current.Level > old.Level {
			add(model.MissionClearConditionTypeCompanionEnhanceByCount, 1, current.CompanionId)
		}
	}
	for uuid, current := range after.Parts {
		old, existed := before.Parts[uuid]
		if !existed {
			add(model.MissionClearConditionTypePartsAddByCount, 1, current.PartsId)
		} else if current.Level > old.Level {
			add(model.MissionClearConditionTypePartsEnhanceByCount, 1, current.PartsId)
		}
	}
	for id, current := range after.Gacha.BannerStates {
		if delta := current.DrawCount - before.Gacha.BannerStates[id].DrawCount; delta > 0 {
			add(model.MissionClearConditionTypeGachaDrawByCount, delta, id)
			if entry := gachaEntry(catalogs, id); entry != nil {
				events = append(events, store.MissionEvent{ConditionType: int32(model.MissionClearConditionTypeGachaDrawByGachaLabelType), Count: delta, TargetId: entry.GachaLabelType, OptionGroupId: entry.GachaLabelType})
			}
		}
	}
	for id, current := range after.ShopItems {
		add(model.MissionClearConditionTypeShopBuyByCount, current.BoughtCount-before.ShopItems[id].BoughtCount, id)
	}
	for id, current := range after.Friends {
		if current.CheerSentDatetime > before.Friends[id].CheerSentDatetime {
			add(model.MissionClearConditionTypeCheerFriendByCount, 1, 0)
		}
	}
	if delta := after.Login.TotalLoginCount - before.Login.TotalLoginCount; delta > 0 {
		add(model.MissionClearConditionTypeUserLoginByCountFromUnlock, delta, 0)
	}
	for key, current := range after.CostumeLotteryEffects {
		if old, existed := before.CostumeLotteryEffects[key]; !existed || old.OddsNumber != current.OddsNumber {
			add(model.MissionClearConditionTypeCostumeLotteryEffectDrawCount, 1, after.Costumes[key.UserCostumeUuid].CostumeId)
		}
	}
	for id, current := range after.ExploreScores {
		if current.MaxScore > before.ExploreScores[id].MaxScore {
			set(model.MissionClearConditionTypeExploreHighScore, current.MaxScore, id)
		}
	}
	return events
}

func allDailyCleared(catalogs *runtime.Catalogs, user *store.UserState, subCategoryId, excludeMissionId int32, nowMillis int64) bool {
	found := false
	for _, mission := range orderedMissions(catalogs.Mission) {
		if mission.MissionId == excludeMissionId || !isDailyAggregateMember(catalogs.Mission, mission) {
			continue
		}
		group := catalogs.Mission.GroupById[mission.MissionGroupId]
		if subCategoryId != 0 && group.MissionSubCategoryId != subCategoryId {
			continue
		}
		t := model.MissionClearConditionType(mission.MissionClearConditionType)
		if t == model.MissionClearConditionTypeMissionClearForAllDaily || t == model.MissionClearConditionTypeMissionClearForAllDailyBySubCategoryId {
			continue
		}
		if nowMillis != 0 && (!missionActive(catalogs.Mission, mission, nowMillis) || !unlocked(catalogs, user, mission, nowMillis)) {
			continue
		}
		found = true
		if !missionCleared(user, mission.MissionId) {
			return false
		}
	}
	return found
}

func missionCleared(user *store.UserState, id int32) bool {
	return user.Missions[id].MissionProgressStatusType >= int32(model.MissionProgressStatusTypeClear)
}

func missionTargetId(mission masterdata.EntityMMission) int32 {
	if mission.MissionClearConditionOptionDetailGroupId != 0 {
		return mission.MissionClearConditionOptionDetailGroupId
	}
	if mission.MissionClearConditionOptionGroupId != 0 {
		return mission.MissionClearConditionOptionGroupId
	}
	return mission.ClearConditionValue
}

func questClearCount(catalogs *runtime.Catalogs, user *store.UserState, mission masterdata.EntityMMission, missionState store.UserMissionState) int32 {
	if requiredDeckCharacterByOption[mission.MissionClearConditionOptionGroupId] != 0 ||
		requiredDeckCostumeByOption[mission.MissionClearConditionOptionGroupId] != 0 {
		// Historical deck composition is not persisted per clear. These
		// missions are advanced only by transaction-local clear events.
		return 0
	}
	var count int32
	for questId, state := range user.Quests {
		if !questMissionMatches(catalogs, mission, questId) {
			continue
		}
		clearCount := state.ClearCount
		if isDaily(catalogs.Mission, mission) {
			if state.LastClearDatetime < missionState.StartDatetime {
				continue
			}
			clearCount = state.DailyClearCount
		}
		if clearCount > 0 {
			count = saturatingAdd(count, clearCount)
		}
	}
	return count
}

func questClearedForMission(catalogs *runtime.Catalogs, quest store.UserQuestState, mission masterdata.EntityMMission, missionState store.UserMissionState) bool {
	return quest.QuestStateType == model.UserQuestStateTypeCleared &&
		(!isDaily(catalogs.Mission, mission) || quest.LastClearDatetime >= missionState.StartDatetime)
}

func questMissionMatches(catalogs *runtime.Catalogs, mission masterdata.EntityMMission, questId int32) bool {
	if mission.MissionClearConditionOptionGroupId == 0 && mission.MissionClearConditionOptionDetailGroupId == 0 {
		return true
	}
	if catalogs == nil || catalogs.Quest == nil {
		return false
	}
	if detail := mission.MissionClearConditionOptionDetailGroupId; detail != 0 {
		if targetIds, ok := mainQuestTargetsByDetail[detail]; ok {
			return containsTarget(targetIds, questId)
		}
		if detail == 500013 {
			return eventQuestSelectorMatchesAnyChapter(catalogs.Quest, eventQuestSelector{
				difficulty: eventQuestDifficultyExHard, ordinal: 3,
			}, questId)
		}
		return false
	}
	option := mission.MissionClearConditionOptionGroupId
	if targetIds, ok := specificEventQuestTargetsByOption[option]; ok {
		return containsTarget(targetIds, questId)
	}
	if directQuestOptions[option] {
		return option == questId
	}
	if requiredDeckCharacterByOption[option] != 0 || requiredDeckCostumeByOption[option] != 0 {
		return true
	}
	if targetIds, ok := mainQuestTargetsByOption[option]; ok {
		return containsTarget(targetIds, questId)
	}
	if selector, ok := eventSelectorForOption(option); ok {
		chapterIds, anyChapter := eventQuestChapterIds(catalogs, mission)
		if anyChapter {
			return eventQuestSelectorMatchesAnyChapter(catalogs.Quest, selector, questId)
		}
		for _, chapterId := range chapterIds {
			if eventQuestSelectorMatches(catalogs.Quest, chapterId, selector, questId) {
				return true
			}
		}
		return false
	}
	return option == 0 || questOptionMatches(catalogs, option, questId)
}

func questOptionMatches(catalogs *runtime.Catalogs, option, questId int32) bool {
	if catalogs.Quest == nil {
		return false
	}
	switch option {
	case questClearOptionMainQuest:
		return catalogs.Quest.RouteIdByQuestId[questId] != 0
	case questClearOptionSubquest, questClearOptionSubquestAlt:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeMarathon, questId) ||
			eventQuestTypeMatches(catalogs.Quest, eventQuestTypeHunt, questId)
	case questClearOptionMainQuestHard, questClearOptionMainQuestHardAlt:
		return catalogs.Quest.MainQuestDifficultyTypeByQuestId[questId] == mainQuestDifficultyHard
	case questClearOptionMainQuestHardOrVeryHard:
		difficultyType := catalogs.Quest.MainQuestDifficultyTypeByQuestId[questId]
		return difficultyType == mainQuestDifficultyHard || difficultyType == mainQuestDifficultyVeryHard
	case questClearOptionMainQuestVeryHard:
		return catalogs.Quest.MainQuestDifficultyTypeByQuestId[questId] == mainQuestDifficultyVeryHard
	case eventQuestTypeCharacter, questClearOptionDarkMemory:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeCharacter, questId)
	case 421, 540:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeCharacter, questId)
	case eventQuestTypeTower, questClearOptionAbyssTower:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeTower, questId)
	case eventQuestTypeLimitContent:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeLimitContent, questId)
	case eventQuestTypeLabyrinth, questClearOptionFateBoard:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeLabyrinth, questId)
	case questClearOptionDailyQuest:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeDayOfTheWeek, questId)
	case questClearOptionGuerrilla:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeGuerrilla, questId)
	case questClearOptionDungeon, 500072:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeDungeon, questId)
	case questClearOptionDailyChallenge:
		return eventDailyQuestMatches(catalogs.Quest, questId)
	case 85:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeCharacterQuest, questId)
	case 86:
		return eventQuestTypeMatches(catalogs.Quest, eventQuestTypeLimitContent, questId)
	}
	if characterId := limitContentCharacterByOption[option]; characterId != 0 {
		return eventQuestCharacterMatches(catalogs.Quest, eventQuestTypeLimitContent, characterId, questId)
	}
	if characterId := darkMemoryCharacterByOption[option]; characterId != 0 {
		return eventQuestCharacterMatches(catalogs.Quest, eventQuestTypeCharacter, characterId, questId)
	}
	if ids, isEventChapter := catalogs.Quest.EventQuestIdsByChapterId[option]; isEventChapter {
		return containsTarget(ids, questId)
	}
	return false
}

func eventQuestChapterIds(catalogs *runtime.Catalogs, mission masterdata.EntityMMission) ([]int32, bool) {
	if catalogs.Mission == nil || catalogs.Quest == nil {
		return nil, false
	}
	option := mission.MissionClearConditionOptionGroupId
	if link, ok := catalogs.Mission.LinkById[mission.MissionLinkId]; ok && link.DestinationDomainType == missionLinkDestinationQuest {
		if link.DestinationDomainId == 0 && genericEventOptionsWithoutChapter[option] {
			return nil, true
		}
		if len(catalogs.Quest.EventQuestIdsByChapterId[link.DestinationDomainId]) != 0 {
			return []int32{link.DestinationDomainId}, false
		}
	}

	seen := make(map[int32]bool)
	var chapterIds []int32
	add := func(chapterId int32) {
		if chapterId == 0 || seen[chapterId] || len(catalogs.Quest.EventQuestIdsByChapterId[chapterId]) == 0 {
			return
		}
		seen[chapterId] = true
		chapterIds = append(chapterIds, chapterId)
	}
	for _, chapterId := range catalogs.Mission.QuestChapterIdsByClearOption[option] {
		add(chapterId)
	}
	for _, chapterId := range fallbackEventChapterIdsByOption[option] {
		add(chapterId)
	}
	if len(chapterIds) == 0 && genericEventOptionsWithoutChapter[option] {
		return nil, true
	}
	return chapterIds, false
}

func eventQuestSelectorMatchesAnyChapter(catalog *masterdata.QuestCatalog, selector eventQuestSelector, questId int32) bool {
	for chapterId := range catalog.EventQuestIdsByChapterId {
		if eventQuestSelectorMatches(catalog, chapterId, selector, questId) {
			return true
		}
	}
	return false
}

func eventQuestSelectorMatches(catalog *masterdata.QuestCatalog, chapterId int32, selector eventQuestSelector, questId int32) bool {
	if selector.all {
		return catalog.EventQuestBelongsToChapter(chapterId, questId)
	}
	if selector.sortOrder != 0 {
		return containsTarget(catalog.EventQuestIdsByChapterSortOrder[chapterId][selector.sortOrder], questId)
	}
	questIds := catalog.EventQuestIdsByChapterDifficulty[chapterId][selector.difficulty]
	if selector.last {
		return len(questIds) != 0 && questIds[len(questIds)-1] == questId
	}
	if selector.ordinal <= 0 || int(selector.ordinal) > len(questIds) {
		return false
	}
	return questIds[selector.ordinal-1] == questId
}

func eventQuestTypeMatches(catalog *masterdata.QuestCatalog, eventQuestType, questId int32) bool {
	for chapterId, candidateType := range catalog.EventQuestTypeByChapterId {
		if candidateType == eventQuestType && catalog.EventQuestBelongsToChapter(chapterId, questId) {
			return true
		}
	}
	return false
}

func eventQuestCharacterMatches(catalog *masterdata.QuestCatalog, eventQuestType, characterId, questId int32) bool {
	for chapterId, candidateType := range catalog.EventQuestTypeByChapterId {
		if candidateType == eventQuestType && catalog.EventCharacterIdsByChapterId[chapterId][characterId] &&
			catalog.EventQuestBelongsToChapter(chapterId, questId) {
			return true
		}
	}
	return false
}

func eventDailyQuestMatches(catalog *masterdata.QuestCatalog, questId int32) bool {
	for _, group := range catalog.EventDailyGroups {
		for _, chapterId := range group.ChapterIds {
			if catalog.EventQuestBelongsToChapter(chapterId, questId) {
				return true
			}
		}
	}
	return false
}

func exploreHighScore(user *store.UserState, mission masterdata.EntityMMission) int32 {
	if id := mission.MissionClearConditionOptionGroupId; id != 0 {
		return user.ExploreScores[id].MaxScore
	}
	var value int32
	for _, score := range user.ExploreScores {
		value = max(value, score.MaxScore)
	}
	return value
}

func libraryElementCount(user *store.UserState) int32 {
	costumes := make(map[int32]bool)
	for _, row := range user.Costumes {
		costumes[row.CostumeId] = true
	}
	weapons := make(map[int32]bool)
	for _, row := range user.Weapons {
		weapons[row.WeaponId] = true
	}
	for id, row := range user.WeaponNotes {
		if row.WeaponId != 0 {
			id = row.WeaponId
		}
		weapons[id] = true
	}
	companions := make(map[int32]bool)
	for _, row := range user.Companions {
		companions[row.CompanionId] = true
	}
	thoughts := make(map[int32]bool)
	for _, row := range user.Thoughts {
		thoughts[row.ThoughtId] = true
	}
	memories := make(map[int32]bool)
	for id, row := range user.PartsGroupNotes {
		if row.PartsGroupId != 0 {
			id = row.PartsGroupId
		}
		memories[id] = true
	}
	return clampInt(len(costumes) + len(weapons) + len(companions) + len(memories) + len(thoughts))
}

func maxCostumeLevel(user *store.UserState, mission masterdata.EntityMMission) int32 {
	var value int32
	targetIds, hasTarget := knownOptionTargets(model.MissionClearConditionTypeCostumeMaxLevel, mission.MissionClearConditionOptionGroupId)
	for _, row := range user.Costumes {
		if hasTarget && !containsTarget(targetIds, row.CostumeId) {
			continue
		}
		value = max(value, row.Level)
	}
	return value
}

func maxCostumeSkillLevel(user *store.UserState) int32 {
	var value int32
	for _, row := range user.CostumeActiveSkills {
		value = max(value, row.Level)
	}
	return value
}

func maxCostumeAbilityLevel(catalogs *runtime.Catalogs, user *store.UserState) int32 {
	var value int32
	for _, row := range user.Costumes {
		if catalogs.Costume != nil {
			value = max(value, catalogs.Costume.AbilityLevel(row.CostumeId, row.LimitBreakCount))
		}
	}
	return value
}

func maxWeaponLevel(user *store.UserState, mission masterdata.EntityMMission) int32 {
	var value int32
	targetIds, hasTarget := knownOptionTargets(model.MissionClearConditionTypeWeaponMaxLevel, mission.MissionClearConditionOptionGroupId)
	for _, row := range user.Weapons {
		if hasTarget && !containsTarget(targetIds, row.WeaponId) {
			continue
		}
		value = max(value, row.Level)
	}
	return value
}

func maxWeaponSkillLevel(user *store.UserState) int32 {
	var value int32
	for _, rows := range user.WeaponSkills {
		for _, row := range rows {
			value = max(value, row.Level)
		}
	}
	return value
}

func maxWeaponAbilityLevel(user *store.UserState) int32 {
	var value int32
	for _, rows := range user.WeaponAbilities {
		for _, row := range rows {
			value = max(value, row.Level)
		}
	}
	return value
}

func maxCompanionLevel(user *store.UserState) int32 {
	var value int32
	for _, row := range user.Companions {
		value = max(value, row.Level)
	}
	return value
}

func maxPartsLevel(user *store.UserState) int32 {
	var value int32
	for _, row := range user.Parts {
		value = max(value, row.Level)
	}
	return value
}

func completePossessionCount(catalog *masterdata.MissionCatalog, user *store.UserState, missionId int32) int32 {
	var count int32
	for _, requirement := range catalog.CompletePossessionsByMissionId[missionId] {
		if hasPossession(user, model.PossessionType(requirement.PossessionType), requirement.PossessionId) {
			count++
		}
	}
	return count
}

func hasPossession(user *store.UserState, possessionType model.PossessionType, id int32) bool {
	switch possessionType {
	case model.PossessionTypeCostume, model.PossessionTypeCostumeEnhanced:
		for _, row := range user.Costumes {
			if row.CostumeId == id {
				return true
			}
		}
	case model.PossessionTypeWeapon, model.PossessionTypeWeaponEnhanced:
		for _, row := range user.Weapons {
			if row.WeaponId == id {
				return true
			}
		}
	case model.PossessionTypeCompanion, model.PossessionTypeCompanionEnhanced:
		for _, row := range user.Companions {
			if row.CompanionId == id {
				return true
			}
		}
	case model.PossessionTypeParts, model.PossessionTypePartsEnhanced:
		for _, row := range user.Parts {
			if row.PartsId == id {
				return true
			}
		}
	case model.PossessionTypeThought:
		for _, row := range user.Thoughts {
			if row.ThoughtId == id {
				return true
			}
		}
	case model.PossessionTypeMaterial:
		return user.Materials[id] > 0
	case model.PossessionTypeConsumableItem:
		return user.ConsumableItems[id] > 0
	case model.PossessionTypeImportantItem:
		return user.ImportantItems[id] > 0
	case model.PossessionTypePremiumItem:
		return user.PremiumItems[id] > 0
	case model.PossessionTypePaidGem:
		return user.Gem.PaidGem > 0
	case model.PossessionTypeFreeGem:
		return user.Gem.FreeGem > 0
	}
	return false
}

func bigHuntHighScore(catalogs *runtime.Catalogs, user *store.UserState, mission masterdata.EntityMMission) int32 {
	if len(requiredBigHuntCharactersByDetail[mission.MissionClearConditionOptionDetailGroupId]) != 0 {
		// Historical score state does not retain the decks used for that score.
		return 0
	}
	var value int64
	if bossId := bigHuntBossId(catalogs, mission); bossId != 0 {
		value = user.BigHuntMaxScores[bossId].MaxScore
	} else {
		for _, row := range user.BigHuntMaxScores {
			value = max(value, row.MaxScore)
		}
		for _, row := range user.BigHuntScheduleMaxScores {
			value = max(value, row.MaxScore)
		}
	}
	return clampInt64(value)
}

func bigHuntBossId(catalogs *runtime.Catalogs, mission masterdata.EntityMMission) int32 {
	if catalogs.BigHunt == nil {
		return 0
	}
	if questId := mission.MissionClearConditionOptionDetailGroupId; questId != 0 {
		if bossId := bigHuntBossIdByMissionDetail[questId]; bossId != 0 {
			return bossId
		}
		if bossId := catalogs.BigHunt.BossIdByQuestId[questId]; bossId != 0 {
			return bossId
		}
		return 0
	}
	option := mission.MissionClearConditionOptionGroupId
	// Localized score missions use option 300001..300005 for the five
	// Big Hunt bosses. Validate the decoded ID against master data.
	if bossId := option - 300000; option >= 300001 && catalogs.BigHunt.BossByBossId[bossId].BigHuntBossId != 0 {
		return bossId
	}
	if catalogs.BigHunt.BossByBossId[option].BigHuntBossId != 0 {
		return option
	}
	return 0
}

// Historical score missions reference option-detail IDs whose corresponding
// Big Hunt quest rows are no longer present in the final master data. Their
// localized mission descriptions identify the target Cursed God explicitly.
var bigHuntBossIdByMissionDetail = map[int32]int32{
	500045: 3, // Windy
	500067: 1, // Fiery
	500068: 2, // Soggy
	500078: 1, // Fiery
	500089: 1, // Fiery
	500102: 4, // Bright
	500108: 2, // Soggy
	500109: 4, // Bright
}

func containsAllTargets(actual, required []int32) bool {
	for _, target := range required {
		if !containsTarget(actual, target) {
			return false
		}
	}
	return true
}

func releasedPanelCount(catalogs *runtime.Catalogs, user *store.UserState, mission masterdata.EntityMMission) int32 {
	var count int32
	for boardId, board := range user.CharacterBoards {
		if !characterBoardMatches(catalogs, mission.MissionClearConditionOptionGroupId, boardId) {
			continue
		}
		count = saturatingAdd(count, int32(bits.OnesCount32(uint32(board.PanelReleaseBit1))+bits.OnesCount32(uint32(board.PanelReleaseBit2))+bits.OnesCount32(uint32(board.PanelReleaseBit3))+bits.OnesCount32(uint32(board.PanelReleaseBit4))))
	}
	return count
}

func fullOpenBoardCount(catalogs *runtime.Catalogs, user *store.UserState, mission masterdata.EntityMMission) int32 {
	if catalogs.CharacterBoard == nil {
		return 0
	}
	var count int32
	for boardId, board := range user.CharacterBoards {
		if !characterBoardMatches(catalogs, mission.MissionClearConditionOptionGroupId, boardId) {
			continue
		}
		released := int(releasedPanelCount(catalogs, &store.UserState{CharacterBoards: map[int32]store.CharacterBoardState{boardId: board}}, masterdata.EntityMMission{MissionClearConditionOptionGroupId: boardId}))
		if total := len(catalogs.CharacterBoard.PanelsByBoardId[boardId]); total > 0 && released >= total {
			count++
		}
	}
	return count
}

func characterBoardMatches(catalogs *runtime.Catalogs, optionGroupId, boardId int32) bool {
	if optionGroupId == 0 || optionGroupId == boardId {
		return true
	}
	return catalogs.CharacterBoard != nil && catalogs.CharacterBoard.MissionOptionByBoardId[boardId] == optionGroupId
}

func weaponCountWithLevelGE(user *store.UserState, mission masterdata.EntityMMission) int32 {
	level := mission.MissionClearConditionOptionGroupId
	if level <= 0 {
		level = 1
	}
	var count int32
	for _, row := range user.Weapons {
		if row.Level >= level {
			count++
		}
	}
	return count
}

func costumeAwakenCount(user *store.UserState, mission masterdata.EntityMMission) int32 {
	var count int32
	targetIds, hasTarget := knownOptionTargets(model.MissionClearConditionTypeCostumeAwakenCount, mission.MissionClearConditionOptionGroupId)
	for _, row := range user.Costumes {
		if hasTarget && !containsTarget(targetIds, row.CostumeId) {
			continue
		}
		count = saturatingAdd(count, row.AwakenCount)
	}
	return count
}

func characterRebirthCount(catalogs *runtime.Catalogs, user *store.UserState, mission masterdata.EntityMMission) int32 {
	if id := characterRebirthMissionTarget(catalogs, mission); id != 0 {
		return user.CharacterRebirths[id].RebirthCount
	}
	var count int32
	for _, row := range user.CharacterRebirths {
		count = saturatingAdd(count, row.RebirthCount)
	}
	return count
}

func characterRebirthMissionTarget(catalogs *runtime.Catalogs, mission masterdata.EntityMMission) int32 {
	if mission.MissionClearConditionOptionGroupId == 0 {
		return 0
	}
	if catalogs.CharacterRebirth != nil {
		// Rebirth mission IDs 520xx1..5 and 521xx1..5 encode the
		// m_character_rebirth SortOrder (regular and crossover characters).
		group := mission.MissionId / 10
		var sortOrder int32
		switch {
		case group > 52000 && group < 52100:
			sortOrder = group - 52000
		case group > 52100 && group < 52200:
			sortOrder = 100 + group - 52100
		}
		if characterId := catalogs.CharacterRebirth.CharacterIdBySortOrder[sortOrder]; characterId != 0 {
			return characterId
		}
		if catalogs.CharacterRebirth.StepGroupByCharacterId[mission.MissionClearConditionOptionGroupId] != 0 {
			return mission.MissionClearConditionOptionGroupId
		}
	}
	return 0
}

func costumeLotterySlotCount(user *store.UserState) int32 {
	var count int32
	for _, row := range user.Costumes {
		count = saturatingAdd(count, row.CostumeLotteryEffectUnlockedSlotCount)
	}
	return count
}

func reportCount(catalogs *runtime.Catalogs, user *store.UserState) int32 {
	if catalogs.Gimmick == nil {
		return 0
	}
	return catalogs.Gimmick.ClearedCountByType(user, model.GimmickTypeReport)
}

func gachaEntry(catalogs *runtime.Catalogs, id int32) *store.GachaCatalogEntry {
	for i := range catalogs.GachaEntries {
		if catalogs.GachaEntries[i].GachaId == id {
			return &catalogs.GachaEntries[i]
		}
	}
	return nil
}

func orderedMissions(catalog *masterdata.MissionCatalog) []masterdata.EntityMMission {
	if len(catalog.OrderedMissions) != 0 {
		return catalog.OrderedMissions
	}
	missions := make([]masterdata.EntityMMission, 0, len(catalog.MissionById))
	for _, mission := range catalog.MissionById {
		missions = append(missions, mission)
	}
	sort.Slice(missions, func(i, j int) bool { return missions[i].MissionId < missions[j].MissionId })
	return missions
}

func missionIdsByType(catalog *masterdata.MissionCatalog, conditionType int32) []int32 {
	if ids := catalog.MissionIdsByType[conditionType]; len(ids) != 0 {
		return ids
	}
	var ids []int32
	for id, mission := range catalog.MissionById {
		if mission.MissionClearConditionType == conditionType {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func positiveLevelDelta(before, after []store.WeaponSkillState) int32 {
	old := make(map[int32]int32, len(before))
	for _, row := range before {
		old[row.SlotNumber] = row.Level
	}
	var count int32
	for _, row := range after {
		if row.Level > old[row.SlotNumber] {
			count = saturatingAdd(count, row.Level-old[row.SlotNumber])
		}
	}
	return count
}

func boolValue(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func saturatingAdd(left, right int32) int32 {
	value := int64(left) + int64(right)
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < 0 {
		return 0
	}
	return int32(value)
}

func clampInt(value int) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(value)
}

func clampInt64(value int64) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < 0 {
		return 0
	}
	return int32(value)
}

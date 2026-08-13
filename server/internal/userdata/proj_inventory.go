package userdata

import (
	"log"
	"sort"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

func init() {
	register("IUserCharacter", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCharacterRecords(user)...)
		return s
	})
	register("IUserCharacterViewerField", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCharacterViewerFieldRecords(user)...)
		return s
	})
	register("IUserCostume", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCostumeRecords(user)...)
		return s
	})
	register("IUserWeapon", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(SortedWeaponRecords(user)...)
		return s
	})
	register("IUserWeaponStory", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedWeaponStoryRecords(user)...)
		return s
	})
	register("IUserWeaponNote", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedWeaponNoteRecords(user)...)
		return s
	})
	register("IUserCompanion", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCompanionRecords(user)...)
		return s
	})
	register("IUserThought", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedThoughtRecords(user)...)
		return s
	})
	register("IUserConsumableItem", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(SortedConsumableItemRecords(user)...)
		return s
	})
	register("IUserMaterial", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(SortedMaterialRecords(user)...)
		return s
	})
	register("IUserImportantItem", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedImportantItemRecords(user)...)
		return s
	})
	register("IUserPremiumItem", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedPremiumItemRecords(user)...)
		return s
	})
	register("IUserParts", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(SortedPartsRecords(user)...)
		return s
	})
	register("IUserCostumeActiveSkill", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCostumeActiveSkillRecords(user)...)
		return s
	})
	register("IUserWeaponSkill", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(SortedWeaponSkillRecords(user)...)
		return s
	})
	register("IUserWeaponAbility", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(SortedWeaponAbilityRecords(user)...)
		return s
	})
	register("IUserExplore", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(exploreRecord(user))
		return s
	})
	register("IUserExploreScore", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedExploreScoreRecords(user)...)
		return s
	})
	register("IUserPartsGroupNote", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedPartsGroupNoteRecords(user)...)
		return s
	})
	register("IUserPartsPreset", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedPartsPresetRecords(user)...)
		return s
	})
	register("IUserPartsPresetTag", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedPartsPresetTagRecords(user)...)
		return s
	})
	register("IUserCostumeAwakenStatusUp", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCostumeAwakenStatusUpRecords(user)...)
		return s
	})
	register("IUserAutoSaleSettingDetail", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedAutoSaleSettingRecords(user)...)
		return s
	})
	register("IUserCharacterRebirth", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCharacterRebirthRecords(user)...)
		return s
	})
	register("IUserCageOrnamentReward", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCageOrnamentRewardRecords(user)...)
		return s
	})
	register("IUserWeaponAwaken", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(SortedWeaponAwakenRecords(user)...)
		return s
	})
	register("IUserCostumeLotteryEffect", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCostumeLotteryEffectRecords(user)...)
		return s
	})
	register("IUserCostumeLotteryEffectPending", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(SortedCostumeLotteryEffectPendingRecords(user)...)
		return s
	})
	register("IUserPartsStatusSub", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedPartsStatusSubRecords(user)...)
		return s
	})
	register("IUserCostumeLevelBonusReleaseStatus", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCostumeLevelBonusRecords(user)...)
		return s
	})
	register("IUserCostumeLotteryEffectAbility", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCostumeLotteryAbilityRecords(user)...)
		return s
	})
	register("IUserCostumeLotteryEffectStatusUp", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedCostumeLotteryStatusRecords(user)...)
		return s
	})
}

func sortedCostumeLevelBonusRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.CostumeLevelBonusReleaseStatuses))
	for id := range user.CostumeLevelBonusReleaseStatuses {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row := user.CostumeLevelBonusReleaseStatuses[int32(id)]
		records = append(records, map[string]any{"userId": user.UserId, "costumeId": row.CostumeId, "lastReleasedBonusLevel": row.LastReleasedBonusLevel, "confirmedBonusLevel": row.ConfirmedBonusLevel, "latestVersion": row.LatestVersion})
	}
	return records
}

func sortedCostumeLotteryAbilityRecords(user store.UserState) []map[string]any {
	keys := make([]store.CostumeLotteryEffectKey, 0, len(user.CostumeLotteryEffectAbilities))
	for key := range user.CostumeLotteryEffectAbilities {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].UserCostumeUuid != keys[j].UserCostumeUuid {
			return keys[i].UserCostumeUuid < keys[j].UserCostumeUuid
		}
		return keys[i].SlotNumber < keys[j].SlotNumber
	})
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.CostumeLotteryEffectAbilities[key]
		records = append(records, map[string]any{"userId": user.UserId, "userCostumeUuid": row.UserCostumeUuid, "slotNumber": row.SlotNumber, "abilityId": row.AbilityId, "abilityLevel": row.AbilityLevel, "latestVersion": row.LatestVersion})
	}
	return records
}

func sortedCostumeLotteryStatusRecords(user store.UserState) []map[string]any {
	keys := make([]store.CostumeLotteryEffectStatusKey, 0, len(user.CostumeLotteryEffectStatusUps))
	for key := range user.CostumeLotteryEffectStatusUps {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].UserCostumeUuid != keys[j].UserCostumeUuid {
			return keys[i].UserCostumeUuid < keys[j].UserCostumeUuid
		}
		return keys[i].StatusCalculationType < keys[j].StatusCalculationType
	})
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.CostumeLotteryEffectStatusUps[key]
		records = append(records, map[string]any{"userId": user.UserId, "userCostumeUuid": row.UserCostumeUuid, "statusCalculationType": int32(row.StatusCalculationType), "hp": row.Hp, "attack": row.Attack, "vitality": row.Vitality, "agility": row.Agility, "criticalRatio": row.CriticalRatio, "criticalAttack": row.CriticalAttack, "latestVersion": row.LatestVersion})
	}
	return records
}

func sortedCharacterRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.Characters))
	for id := range user.Characters {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row := user.Characters[int32(id)]
		records = append(records, map[string]any{
			"userId":        user.UserId,
			"characterId":   row.CharacterId,
			"level":         row.Level,
			"exp":           row.Exp,
			"latestVersion": row.LatestVersion,
		})
	}
	return records
}

func sortedCharacterViewerFieldRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.CharacterViewerFields))
	for id := range user.CharacterViewerFields {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		records = append(records, characterViewerFieldRecord(user.UserId, user.CharacterViewerFields[int32(id)]))
	}
	return records
}

func CharacterViewerFieldRecordsForIds(user store.UserState, fieldIds []int32) string {
	records := make([]map[string]any, 0, len(fieldIds))
	for _, fieldId := range fieldIds {
		row, ok := user.CharacterViewerFields[fieldId]
		if !ok {
			continue
		}
		records = append(records, characterViewerFieldRecord(user.UserId, row))
	}
	s, _ := utils.EncodeJSONMaps(records...)
	return s
}

func characterViewerFieldRecord(userId int64, row store.CharacterViewerFieldState) map[string]any {
	return map[string]any{
		"userId":                 userId,
		"characterViewerFieldId": row.CharacterViewerFieldId,
		"releaseDatetime":        row.ReleaseDatetime,
		"latestVersion":          row.LatestVersion,
	}
}

func sortedCostumeRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.Costumes)
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Costumes[key]
		records = append(records, map[string]any{
			"userId":                                user.UserId,
			"userCostumeUuid":                       row.UserCostumeUuid,
			"costumeId":                             row.CostumeId,
			"limitBreakCount":                       row.LimitBreakCount,
			"level":                                 row.Level,
			"exp":                                   row.Exp,
			"headupDisplayViewId":                   row.HeadupDisplayViewId,
			"acquisitionDatetime":                   row.AcquisitionDatetime,
			"awakenCount":                           row.AwakenCount,
			"costumeLotteryEffectUnlockedSlotCount": row.CostumeLotteryEffectUnlockedSlotCount,
			"latestVersion":                         row.LatestVersion,
		})
	}
	return records
}

func sortedAutoSaleSettingRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.AutoSaleSettings))
	for id := range user.AutoSaleSettings {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row := user.AutoSaleSettings[int32(id)]
		records = append(records, map[string]any{
			"userId":                      user.UserId,
			"possessionAutoSaleItemType":  row.PossessionAutoSaleItemType,
			"possessionAutoSaleItemValue": row.PossessionAutoSaleItemValue,
			"latestVersion":               gametime.NowMillis(),
		})
	}
	return records
}

func sortedCostumeAwakenStatusUpRecords(user store.UserState) []map[string]any {
	keys := make([]store.CostumeAwakenStatusKey, 0, len(user.CostumeAwakenStatusUps))
	for k := range user.CostumeAwakenStatusUps {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].UserCostumeUuid != keys[j].UserCostumeUuid {
			return keys[i].UserCostumeUuid < keys[j].UserCostumeUuid
		}
		return keys[i].StatusCalculationType < keys[j].StatusCalculationType
	})
	records := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		row := user.CostumeAwakenStatusUps[k]
		records = append(records, map[string]any{
			"userId":                user.UserId,
			"userCostumeUuid":       row.UserCostumeUuid,
			"statusCalculationType": int32(row.StatusCalculationType),
			"hp":                    row.Hp,
			"attack":                row.Attack,
			"vitality":              row.Vitality,
			"agility":               row.Agility,
			"criticalRatio":         row.CriticalRatio,
			"criticalAttack":        row.CriticalAttack,
			"latestVersion":         row.LatestVersion,
		})
	}
	return records
}

func SortedWeaponRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.Weapons)
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Weapons[key]
		uuid := row.UserWeaponUuid
		if uuid == "" {
			log.Printf("[userdata] sortedWeaponRecords: using key as fallback for weapon key=%q (empty userWeaponUuid)", key)
			uuid = key
		}
		records = append(records, map[string]any{
			"userId":              user.UserId,
			"userWeaponUuid":      uuid,
			"weaponId":            row.WeaponId,
			"level":               row.Level,
			"exp":                 row.Exp,
			"limitBreakCount":     row.LimitBreakCount,
			"isProtected":         row.IsProtected,
			"acquisitionDatetime": row.AcquisitionDatetime,
			"latestVersion":       row.LatestVersion,
		})
	}
	return records
}

func sortedWeaponStoryRecords(user store.UserState) []map[string]any {
	if user.WeaponStories == nil {
		return []map[string]any{}
	}
	weaponIdsInWeapons := make(map[int32]bool)
	for _, row := range user.Weapons {
		weaponIdsInWeapons[row.WeaponId] = true
	}
	weaponIds := make([]int32, 0, len(user.WeaponStories))
	for weaponId := range user.WeaponStories {
		if weaponIdsInWeapons[weaponId] {
			weaponIds = append(weaponIds, weaponId)
		}
	}
	sort.Slice(weaponIds, func(i, j int) bool { return weaponIds[i] < weaponIds[j] })
	records := make([]map[string]any, 0, len(weaponIds))
	for _, weaponId := range weaponIds {
		row := user.WeaponStories[weaponId]
		records = append(records, map[string]any{
			"userId":                user.UserId,
			"weaponId":              row.WeaponId,
			"releasedMaxStoryIndex": row.ReleasedMaxStoryIndex,
			"latestVersion":         row.LatestVersion,
		})
	}
	return records
}

func WeaponStoryRecordsForIds(user store.UserState, weaponIds []int32) string {
	if len(weaponIds) == 0 {
		return "[]"
	}
	records := make([]map[string]any, 0, len(weaponIds))
	for _, weaponId := range weaponIds {
		row, ok := user.WeaponStories[weaponId]
		if !ok {
			continue
		}
		records = append(records, map[string]any{
			"userId":                user.UserId,
			"weaponId":              row.WeaponId,
			"releasedMaxStoryIndex": row.ReleasedMaxStoryIndex,
			"latestVersion":         row.LatestVersion,
		})
	}
	s, _ := utils.EncodeJSONMaps(records...)
	return s
}

func sortedWeaponNoteRecords(user store.UserState) []map[string]any {
	weaponIds := make([]int32, 0, len(user.WeaponNotes))
	for id := range user.WeaponNotes {
		weaponIds = append(weaponIds, id)
	}
	sort.Slice(weaponIds, func(i, j int) bool { return weaponIds[i] < weaponIds[j] })
	records := make([]map[string]any, 0, len(weaponIds))
	for _, id := range weaponIds {
		row := user.WeaponNotes[id]
		records = append(records, map[string]any{
			"userId":                   user.UserId,
			"weaponId":                 row.WeaponId,
			"maxLevel":                 row.MaxLevel,
			"maxLimitBreakCount":       row.MaxLimitBreakCount,
			"firstAcquisitionDatetime": row.FirstAcquisitionDatetime,
			"latestVersion":            row.LatestVersion,
		})
	}
	return records
}

func sortedCompanionRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.Companions)
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Companions[key]
		records = append(records, map[string]any{
			"userId":              user.UserId,
			"userCompanionUuid":   row.UserCompanionUuid,
			"companionId":         row.CompanionId,
			"headupDisplayViewId": row.HeadupDisplayViewId,
			"level":               row.Level,
			"acquisitionDatetime": row.AcquisitionDatetime,
			"latestVersion":       row.LatestVersion,
		})
	}
	return records
}

func sortedThoughtRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.Thoughts)
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Thoughts[key]
		records = append(records, map[string]any{
			"userId":              user.UserId,
			"userThoughtUuid":     row.UserThoughtUuid,
			"thoughtId":           row.ThoughtId,
			"acquisitionDatetime": row.AcquisitionDatetime,
			"latestVersion":       row.LatestVersion,
		})
	}
	return records
}

func SortedConsumableItemRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.ConsumableItems))
	for id := range user.ConsumableItems {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	nowMillis := gametime.NowMillis()
	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		records = append(records, map[string]any{
			"userId":                   user.UserId,
			"consumableItemId":         int32(id),
			"count":                    user.ConsumableItems[int32(id)],
			"firstAcquisitionDatetime": nowMillis,
			"latestVersion":            nowMillis,
		})
	}
	return records
}

func SortedMaterialRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.Materials))
	for id := range user.Materials {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	nowMillis := gametime.NowMillis()
	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		records = append(records, map[string]any{
			"userId":                   user.UserId,
			"materialId":               int32(id),
			"count":                    user.Materials[int32(id)],
			"firstAcquisitionDatetime": nowMillis,
			"latestVersion":            nowMillis,
		})
	}
	return records
}

func sortedImportantItemRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.ImportantItems))
	for id := range user.ImportantItems {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	nowMillis := gametime.NowMillis()
	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		records = append(records, map[string]any{
			"userId":                   user.UserId,
			"importantItemId":          int32(id),
			"count":                    user.ImportantItems[int32(id)],
			"firstAcquisitionDatetime": nowMillis,
			"latestVersion":            nowMillis,
		})
	}
	return records
}

func sortedPremiumItemRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.PremiumItems))
	for id := range user.PremiumItems {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	nowMillis := gametime.NowMillis()
	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		acqTime := user.PremiumItems[int32(id)]
		if acqTime == 0 {
			acqTime = nowMillis
		}
		records = append(records, map[string]any{
			"userId":              user.UserId,
			"premiumItemId":       int32(id),
			"acquisitionDatetime": acqTime,
			"latestVersion":       nowMillis,
		})
	}
	return records
}

func SortedPartsRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.Parts)
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Parts[key]
		records = append(records, map[string]any{
			"userId":              user.UserId,
			"userPartsUuid":       row.UserPartsUuid,
			"partsId":             row.PartsId,
			"level":               row.Level,
			"partsStatusMainId":   row.PartsStatusMainId,
			"isProtected":         row.IsProtected,
			"acquisitionDatetime": row.AcquisitionDatetime,
			"latestVersion":       row.LatestVersion,
		})
	}
	return records
}

func sortedPartsGroupNoteRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.PartsGroupNotes))
	for id := range user.PartsGroupNotes {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row := user.PartsGroupNotes[int32(id)]
		records = append(records, map[string]any{
			"userId":                   user.UserId,
			"partsGroupId":             row.PartsGroupId,
			"firstAcquisitionDatetime": row.FirstAcquisitionDatetime,
			"latestVersion":            row.LatestVersion,
		})
	}
	return records
}

func sortedPartsPresetRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.PartsPresets))
	for id := range user.PartsPresets {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row := user.PartsPresets[int32(id)]
		records = append(records, map[string]any{
			"userId":                   user.UserId,
			"userPartsPresetNumber":    row.UserPartsPresetNumber,
			"userPartsUuid01":          row.UserPartsUuid01,
			"userPartsUuid02":          row.UserPartsUuid02,
			"userPartsUuid03":          row.UserPartsUuid03,
			"name":                     row.Name,
			"userPartsPresetTagNumber": row.UserPartsPresetTagNumber,
			"latestVersion":            row.LatestVersion,
		})
	}
	return records
}

func sortedPartsPresetTagRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.PartsPresetTags))
	for id := range user.PartsPresetTags {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row := user.PartsPresetTags[int32(id)]
		records = append(records, map[string]any{
			"userId":                   user.UserId,
			"userPartsPresetTagNumber": row.UserPartsPresetTagNumber,
			"name":                     row.Name,
			"latestVersion":            row.LatestVersion,
		})
	}
	return records
}

func sortedPartsStatusSubRecords(user store.UserState) []map[string]any {
	keys := make([]store.PartsStatusSubKey, 0, len(user.PartsStatusSubs))
	for k := range user.PartsStatusSubs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].UserPartsUuid != keys[j].UserPartsUuid {
			return keys[i].UserPartsUuid < keys[j].UserPartsUuid
		}
		return keys[i].StatusIndex < keys[j].StatusIndex
	})
	records := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		row := user.PartsStatusSubs[k]
		records = append(records, map[string]any{
			"userId":                  user.UserId,
			"userPartsUuid":           row.UserPartsUuid,
			"statusIndex":             row.StatusIndex,
			"partsStatusSubLotteryId": row.PartsStatusSubLotteryId,
			"level":                   row.Level,
			"statusKindType":          row.StatusKindType,
			"statusCalculationType":   row.StatusCalculationType,
			"statusChangeValue":       row.StatusChangeValue,
			"latestVersion":           row.LatestVersion,
		})
	}
	return records
}

func sortedCostumeActiveSkillRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.CostumeActiveSkills)
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.CostumeActiveSkills[key]
		records = append(records, map[string]any{
			"userId":              user.UserId,
			"userCostumeUuid":     row.UserCostumeUuid,
			"level":               row.Level,
			"acquisitionDatetime": row.AcquisitionDatetime,
			"latestVersion":       row.LatestVersion,
		})
	}
	return records
}

func SortedWeaponSkillRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.WeaponSkills)
	records := make([]map[string]any, 0)
	for _, key := range keys {
		for _, row := range user.WeaponSkills[key] {
			records = append(records, map[string]any{
				"userId":         user.UserId,
				"userWeaponUuid": row.UserWeaponUuid,
				"slotNumber":     row.SlotNumber,
				"level":          row.Level,
				"latestVersion":  int64(0),
			})
		}
	}
	return records
}

func SortedWeaponAbilityRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.WeaponAbilities)
	records := make([]map[string]any, 0)
	for _, key := range keys {
		for _, row := range user.WeaponAbilities[key] {
			records = append(records, map[string]any{
				"userId":         user.UserId,
				"userWeaponUuid": row.UserWeaponUuid,
				"slotNumber":     row.SlotNumber,
				"level":          row.Level,
				"latestVersion":  int64(0),
			})
		}
	}
	return records
}

func SortedWeaponAwakenRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.WeaponAwakens)
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.WeaponAwakens[key]
		records = append(records, map[string]any{
			"userId":         user.UserId,
			"userWeaponUuid": row.UserWeaponUuid,
			"latestVersion":  row.LatestVersion,
		})
	}
	return records
}

func exploreRecord(user store.UserState) map[string]any {
	return map[string]any{
		"userId":             user.UserId,
		"isUseExploreTicket": user.Explore.IsUseExploreTicket,
		"playingExploreId":   user.Explore.PlayingExploreId,
		"latestPlayDatetime": user.Explore.LatestPlayDatetime,
		"latestVersion":      user.Explore.LatestVersion,
	}
}

func sortedCharacterRebirthRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.CharacterRebirths))
	for id := range user.CharacterRebirths {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row := user.CharacterRebirths[int32(id)]
		records = append(records, map[string]any{
			"userId":        user.UserId,
			"characterId":   row.CharacterId,
			"rebirthCount":  row.RebirthCount,
			"latestVersion": row.LatestVersion,
		})
	}
	return records
}

func sortedExploreScoreRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.ExploreScores))
	for id := range user.ExploreScores {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row := user.ExploreScores[int32(id)]
		records = append(records, map[string]any{
			"userId":                 user.UserId,
			"exploreId":              row.ExploreId,
			"maxScore":               row.MaxScore,
			"maxScoreUpdateDatetime": row.MaxScoreUpdateDatetime,
			"latestVersion":          row.LatestVersion,
		})
	}
	return records
}

func sortedCageOrnamentRewardRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.CageOrnamentRewards))
	for id := range user.CageOrnamentRewards {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row := user.CageOrnamentRewards[int32(id)]
		records = append(records, map[string]any{
			"userId":              user.UserId,
			"cageOrnamentId":      row.CageOrnamentId,
			"acquisitionDatetime": row.AcquisitionDatetime,
			"latestVersion":       row.LatestVersion,
		})
	}
	return records
}

func SortedCostumeLotteryEffectPendingRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.CostumeLotteryEffectPending)
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.CostumeLotteryEffectPending[key]
		records = append(records, map[string]any{
			"userId":          user.UserId,
			"userCostumeUuid": row.UserCostumeUuid,
			"slotNumber":      row.SlotNumber,
			"oddsNumber":      row.OddsNumber,
			"latestVersion":   row.LatestVersion,
		})
	}
	return records
}

func sortedCostumeLotteryEffectRecords(user store.UserState) []map[string]any {
	keys := make([]store.CostumeLotteryEffectKey, 0, len(user.CostumeLotteryEffects))
	for k := range user.CostumeLotteryEffects {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].UserCostumeUuid != keys[j].UserCostumeUuid {
			return keys[i].UserCostumeUuid < keys[j].UserCostumeUuid
		}
		return keys[i].SlotNumber < keys[j].SlotNumber
	})
	records := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		row := user.CostumeLotteryEffects[k]
		records = append(records, map[string]any{
			"userId":          user.UserId,
			"userCostumeUuid": row.UserCostumeUuid,
			"slotNumber":      row.SlotNumber,
			"oddsNumber":      row.OddsNumber,
			"latestVersion":   row.LatestVersion,
		})
	}
	return records
}

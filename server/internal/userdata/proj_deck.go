package userdata

import (
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

func init() {
	register("IUserDeck", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedDeckRecords(user)...)
		return s
	})
	register("IUserTripleDeck", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedTripleDeckRecords(user)...)
		return s
	})
	register("IUserDeckCharacter", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedDeckCharacterRecords(user)...)
		return s
	})
	register("IUserDeckSubWeaponGroup", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedDeckSubWeaponGroupRecords(user)...)
		return s
	})
	register("IUserDeckTypeNote", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedDeckTypeNoteRecords(user)...)
		return s
	})
	register("IUserDeckPartsGroup", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedDeckPartsGroupRecords(user)...)
		return s
	})
	register("IUserDeckCharacterDressupCostume", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedDeckDressupCostumeRecords(user)...)
		return s
	})
	register("IUserDeckLimitContentRestricted", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedDeckLimitContentRestrictedRecords(user)...)
		return s
	})
}

func sortedDeckLimitContentRestrictedRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.DeckLimitContentRestricted)
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.DeckLimitContentRestricted[key]
		records = append(records, map[string]any{"userId": user.UserId, "eventQuestChapterId": row.EventQuestChapterId, "questId": row.QuestId, "deckRestrictedUuid": row.DeckRestrictedUuid, "possessionType": row.PossessionType, "targetUuid": row.TargetUuid, "latestVersion": row.LatestVersion})
	}
	return records
}

func sortedDeckRecords(user store.UserState) []map[string]any {
	keys := make([]store.DeckKey, 0, len(user.Decks))
	for key := range user.Decks {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].DeckType != keys[j].DeckType {
			return keys[i].DeckType < keys[j].DeckType
		}
		return keys[i].UserDeckNumber < keys[j].UserDeckNumber
	})

	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Decks[key]
		records = append(records, map[string]any{
			"userId":                  user.UserId,
			"deckType":                row.DeckType,
			"userDeckNumber":          row.UserDeckNumber,
			"userDeckCharacterUuid01": row.UserDeckCharacterUuid01,
			"userDeckCharacterUuid02": row.UserDeckCharacterUuid02,
			"userDeckCharacterUuid03": row.UserDeckCharacterUuid03,
			"name":                    row.Name,
			"power":                   row.Power,
			"latestVersion":           row.LatestVersion,
		})
	}
	return records
}

func sortedTripleDeckRecords(user store.UserState) []map[string]any {
	keys := make([]store.DeckKey, 0, len(user.TripleDecks))
	for key := range user.TripleDecks {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].DeckType != keys[j].DeckType {
			return keys[i].DeckType < keys[j].DeckType
		}
		return keys[i].UserDeckNumber < keys[j].UserDeckNumber
	})

	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.TripleDecks[key]
		records = append(records, map[string]any{
			"userId":         user.UserId,
			"deckType":       int32(row.DeckType),
			"userDeckNumber": row.UserDeckNumber,
			"name":           row.Name,
			"deckNumber01":   row.DeckNumber01,
			"deckNumber02":   row.DeckNumber02,
			"deckNumber03":   row.DeckNumber03,
			"latestVersion":  row.LatestVersion,
		})
	}
	return records
}

func sortedDeckCharacterRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.DeckCharacters)
	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.DeckCharacters[key]
		records = append(records, map[string]any{
			"userId":                user.UserId,
			"userDeckCharacterUuid": row.UserDeckCharacterUuid,
			"userCostumeUuid":       row.UserCostumeUuid,
			"mainUserWeaponUuid":    row.MainUserWeaponUuid,
			"userCompanionUuid":     row.UserCompanionUuid,
			"power":                 row.Power,
			"userThoughtUuid":       row.UserThoughtUuid,
			"latestVersion":         row.LatestVersion,
		})
	}
	return records
}

func sortedDeckSubWeaponGroupRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.DeckSubWeapons)
	records := make([]map[string]any, 0)
	for _, dcUuid := range keys {
		weapons := user.DeckSubWeapons[dcUuid]
		var lv int64
		if dc, ok := user.DeckCharacters[dcUuid]; ok {
			lv = dc.LatestVersion
		}
		for idx, weaponUuid := range weapons {
			records = append(records, map[string]any{
				"userId":                user.UserId,
				"userDeckCharacterUuid": dcUuid,
				"userWeaponUuid":        weaponUuid,
				"sortOrder":             int32(idx + 1),
				"latestVersion":         lv,
			})
		}
	}
	return records
}

func sortedDeckTypeNoteRecords(user store.UserState) []map[string]any {
	ids := make([]int, 0, len(user.DeckTypeNotes))
	for id := range user.DeckTypeNotes {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	records := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row := user.DeckTypeNotes[model.DeckType(id)]
		records = append(records, map[string]any{
			"userId":        user.UserId,
			"deckType":      row.DeckType,
			"maxDeckPower":  row.MaxDeckPower,
			"latestVersion": row.LatestVersion,
		})
	}
	return records
}

func sortedDeckPartsGroupRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.DeckParts)
	records := make([]map[string]any, 0)
	for _, dcUuid := range keys {
		parts := user.DeckParts[dcUuid]
		var lv int64
		if dc, ok := user.DeckCharacters[dcUuid]; ok {
			lv = dc.LatestVersion
		}
		for idx, partsUuid := range parts {
			records = append(records, map[string]any{
				"userId":                user.UserId,
				"userDeckCharacterUuid": dcUuid,
				"userPartsUuid":         partsUuid,
				"sortOrder":             int32(idx + 1),
				"latestVersion":         lv,
			})
		}
	}
	return records
}

func DeckSubWeaponRecords(user store.UserState) []map[string]any {
	return sortedDeckSubWeaponGroupRecords(user)
}

func DeckPartsGroupRecords(user store.UserState) []map[string]any {
	return sortedDeckPartsGroupRecords(user)
}

func sortedDeckDressupCostumeRecords(user store.UserState) []map[string]any {
	keys := sortedStringKeys(user.DeckCharacters)
	records := make([]map[string]any, 0)
	for _, key := range keys {
		row := user.DeckCharacters[key]
		if row.DressupCostumeId == 0 {
			continue
		}
		records = append(records, map[string]any{
			"userId":                user.UserId,
			"userDeckCharacterUuid": row.UserDeckCharacterUuid,
			"dressupCostumeId":      row.DressupCostumeId,
			"latestVersion":         row.LatestVersion,
		})
	}
	return records
}

func DeckDressupCostumeRecords(user store.UserState) []map[string]any {
	return sortedDeckDressupCostumeRecords(user)
}

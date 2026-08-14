package questflow

import (
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

const (
	questBonusTypeExp        int32 = 2
	questBonusTypeDropReward int32 = 3
	questBonusExpCharacter   int32 = 1
	questBonusExpCostume     int32 = 2
	questBonusRewardEffectId int32 = 1
)

type questBonusDeckUnit struct {
	costumeUuid string
	characterId int32
	weaponUuids []string
}

func (h *QuestHandler) questDeck(user *store.UserState, questId int32) (store.DeckState, bool) {
	deckNumber := user.Quests[questId].UserDeckNumber
	if deckNumber == 0 {
		return store.DeckState{}, false
	}

	quest := h.QuestById[questId]
	deckTypes := []model.DeckType{model.DeckTypeQuest}
	switch {
	case h.LimitContentQuestIds[questId]:
		deckTypes = []model.DeckType{model.DeckTypeRestrictedLimitContentQuest, model.DeckTypeQuest}
	case quest.QuestDeckRestrictionGroupId != 0:
		deckTypes = []model.DeckType{model.DeckTypeRestrictedQuest, model.DeckTypeQuest}
	}
	for _, deckType := range deckTypes {
		if deck, ok := user.Decks[store.DeckKey{DeckType: deckType, UserDeckNumber: deckNumber}]; ok {
			return deck, true
		}
	}
	return store.DeckState{}, false
}

func (h *QuestHandler) questBonusDeckUnits(user *store.UserState, questId int32) []questBonusDeckUnit {
	deck, ok := h.questDeck(user, questId)
	if !ok {
		return nil
	}

	deckCharacterUuids := []string{
		deck.UserDeckCharacterUuid01,
		deck.UserDeckCharacterUuid02,
		deck.UserDeckCharacterUuid03,
	}
	units := make([]questBonusDeckUnit, 0, len(deckCharacterUuids))
	for _, deckCharacterUuid := range deckCharacterUuids {
		if deckCharacterUuid == "" {
			continue
		}
		deckCharacter, ok := user.DeckCharacters[deckCharacterUuid]
		if !ok {
			continue
		}
		unit := questBonusDeckUnit{costumeUuid: deckCharacter.UserCostumeUuid}
		if costume, ok := user.Costumes[deckCharacter.UserCostumeUuid]; ok {
			unit.characterId = h.CostumeById[costume.CostumeId].CharacterId
		}
		if deckCharacter.MainUserWeaponUuid != "" {
			unit.weaponUuids = append(unit.weaponUuids, deckCharacter.MainUserWeaponUuid)
		}
		unit.weaponUuids = append(unit.weaponUuids, user.DeckSubWeapons[deckCharacterUuid]...)
		units = append(units, unit)
	}
	return units
}

func (h *QuestHandler) questBonusTermActive(groupId int32, nowMillis int64) bool {
	if groupId == 0 {
		return true
	}
	for _, term := range h.QuestBonusTermsByGroupId[groupId] {
		if nowMillis >= term.StartDatetime && nowMillis <= term.EndDatetime {
			return true
		}
	}
	return false
}

func (h *QuestHandler) questBonusWeaponMatches(configuredWeaponId, equippedWeaponId int32) bool {
	if configuredWeaponId == equippedWeaponId {
		return true
	}
	configured, configuredOK := h.WeaponEvolutionByWeaponId[configuredWeaponId]
	equipped, equippedOK := h.WeaponEvolutionByWeaponId[equippedWeaponId]
	return configuredOK && equippedOK && configured.GroupId != 0 &&
		configured.GroupId == equipped.GroupId && equipped.Order >= configured.Order
}

func (h *QuestHandler) bestQuestBonusWeaponRow(
	rows []masterdata.EntityMQuestBonusWeaponGroup,
	weapon store.WeaponState,
	nowMillis int64,
) (masterdata.EntityMQuestBonusWeaponGroup, bool) {
	var best masterdata.EntityMQuestBonusWeaponGroup
	found := false
	bestEvolutionOrder := int32(-1)
	for _, row := range rows {
		if row.LimitBreakCountLowerLimit > weapon.LimitBreakCount ||
			!h.questBonusTermActive(row.QuestBonusTermGroupId, nowMillis) ||
			!h.questBonusWeaponMatches(row.WeaponId, weapon.WeaponId) {
			continue
		}
		evolutionOrder := h.WeaponEvolutionByWeaponId[row.WeaponId].Order
		if !found || row.LimitBreakCountLowerLimit > best.LimitBreakCountLowerLimit ||
			(row.LimitBreakCountLowerLimit == best.LimitBreakCountLowerLimit && evolutionOrder > bestEvolutionOrder) {
			best = row
			bestEvolutionOrder = evolutionOrder
			found = true
		}
	}
	return best, found
}

func (h *QuestHandler) questBonusDropRewards(user *store.UserState, quest masterdata.EntityMQuest, nowMillis int64) []RewardGrant {
	bonus, ok := h.QuestBonusById[quest.QuestBonusId]
	if !ok || bonus.QuestBonusWeaponGroupId == 0 {
		return nil
	}
	rows := h.QuestBonusWeaponRowsByGroupId[bonus.QuestBonusWeaponGroupId]
	if len(rows) == 0 {
		return nil
	}

	seenWeaponUuids := make(map[string]bool)
	grantIndex := make(map[[2]int32]int)
	var grants []RewardGrant
	for _, unit := range h.questBonusDeckUnits(user, quest.QuestId) {
		for _, weaponUuid := range unit.weaponUuids {
			if weaponUuid == "" || seenWeaponUuids[weaponUuid] {
				continue
			}
			seenWeaponUuids[weaponUuid] = true
			weapon, ok := user.Weapons[weaponUuid]
			if !ok {
				continue
			}
			row, ok := h.bestQuestBonusWeaponRow(rows, weapon, nowMillis)
			if !ok {
				continue
			}
			for _, effect := range h.QuestBonusEffectsByGroupId[row.QuestBonusEffectGroupId] {
				if effect.QuestBonusType != questBonusTypeDropReward {
					continue
				}
				reward, ok := h.QuestBonusDropByEffectId[effect.QuestBonusEffectId]
				if !ok || reward.AdditionalCount <= 0 {
					continue
				}
				key := [2]int32{reward.PossessionType, reward.PossessionId}
				if index, exists := grantIndex[key]; exists {
					grants[index].Count += reward.AdditionalCount
					continue
				}
				grantIndex[key] = len(grants)
				grants = append(grants, RewardGrant{
					PossessionType: model.PossessionType(reward.PossessionType),
					PossessionId:   reward.PossessionId,
					Count:          reward.AdditionalCount,
					RewardEffectId: questBonusRewardEffectId,
				})
			}
		}
	}
	return grants
}

func (h *QuestHandler) questBonusExpPermilByCostume(
	user *store.UserState,
	quest masterdata.EntityMQuest,
	nowMillis int64,
) (character, costume map[string]int32) {
	character = make(map[string]int32)
	costume = make(map[string]int32)
	bonus, ok := h.QuestBonusById[quest.QuestBonusId]
	if !ok || bonus.QuestBonusCharacterGroupId == 0 {
		return character, costume
	}
	rows := h.QuestBonusCharacterRowsByGroupId[bonus.QuestBonusCharacterGroupId]
	for _, unit := range h.questBonusDeckUnits(user, quest.QuestId) {
		if unit.costumeUuid == "" || unit.characterId == 0 {
			continue
		}
		for _, row := range rows {
			if row.CharacterId != unit.characterId || !h.questBonusTermActive(row.QuestBonusTermGroupId, nowMillis) {
				continue
			}
			for _, effect := range h.QuestBonusEffectsByGroupId[row.QuestBonusEffectGroupId] {
				if effect.QuestBonusType != questBonusTypeExp {
					continue
				}
				exp, ok := h.QuestBonusExpByEffectId[effect.QuestBonusEffectId]
				if !ok {
					continue
				}
				switch exp.ExpType {
				case questBonusExpCharacter:
					character[unit.costumeUuid] += exp.BonusValuePermil
				case questBonusExpCostume:
					costume[unit.costumeUuid] += exp.BonusValuePermil
				}
			}
		}
	}
	return character, costume
}

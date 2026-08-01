package questflow

import (
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type questDeckUnit struct {
	costume masterdata.EntityMCostume
	weapon  masterdata.EntityMWeapon
}

func (h *QuestHandler) questDeckUnits(user *store.UserState, questId int32) []questDeckUnit {
	deck, ok := user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: user.Quests[questId].UserDeckNumber}]
	if !ok {
		return nil
	}
	var units []questDeckUnit
	for _, id := range []string{deck.UserDeckCharacterUuid01, deck.UserDeckCharacterUuid02, deck.UserDeckCharacterUuid03} {
		dc, ok := user.DeckCharacters[id]
		if !ok {
			continue
		}
		costumeState, ok := user.Costumes[dc.UserCostumeUuid]
		if !ok {
			continue
		}
		costume, ok := h.CostumeById[costumeState.CostumeId]
		if !ok {
			continue
		}
		weaponState, ok := user.Weapons[dc.MainUserWeaponUuid]
		if !ok {
			continue
		}
		weapon, ok := h.WeaponById[weaponState.WeaponId]
		if !ok {
			continue
		}
		units = append(units, questDeckUnit{costume: costume, weapon: weapon})
	}
	return units
}

func (h *QuestHandler) questMissionSatisfied(user *store.UserState, questId int32, mission masterdata.EntityMQuestMission) bool {
	t := model.QuestMissionConditionType(mission.QuestMissionConditionType)
	value := mission.ConditionValue
	values := h.MissionConditionValuesByGroupId[mission.QuestMissionConditionValueGroupId]
	units := h.questDeckUnits(user, questId)
	detail := user.Battle.MissionDetail
	freshBattle := detail.IsValid && user.Battle.LastFinishedAt >= user.Quests[questId].LatestStartDatetime
	switch t {
	case model.QuestMissionConditionTypeLessThanOrEqualXPeopleNotAlive:
		return freshBattle && detail.CharacterDeathCount <= value
	case model.QuestMissionConditionTypeMaxDamage:
		return freshBattle && detail.MaxDamage >= int64(value)
	case model.QuestMissionConditionTypeSpecifiedCharacterIsInDeck:
		for _, unit := range units {
			if unit.costume.CharacterId == value {
				return true
			}
		}
		return false
	case model.QuestMissionConditionTypeSpecifiedAttributeMainWeaponIsInDeck:
		for _, unit := range units {
			if unit.weapon.AttributeType == value {
				return true
			}
		}
		return false
	case model.QuestMissionConditionTypeGreaterThanOrEqualXCostumeSkillUseCount:
		return freshBattle && detail.CostumeSkillUseCount >= value
	case model.QuestMissionConditionTypeGreaterThanOrEqualXWeaponSkillUseCount:
		return freshBattle && detail.WeaponSkillUseCount >= value
	case model.QuestMissionConditionTypeGreaterThanOrEqualXCompanionSkillUseCount:
		return freshBattle && detail.CompanionSkillUseCount >= value
	case model.QuestMissionConditionTypeCostumeSkillfulWeaponAnyCharacter:
		for _, unit := range units {
			if unit.costume.SkillfulWeaponType == unit.weapon.WeaponType {
				return true
			}
		}
		return false
	case model.QuestMissionConditionTypeSpecifiedAttributeMainWeaponAllCharacter:
		if len(units) == 0 {
			return false
		}
		for _, unit := range units {
			if unit.weapon.AttributeType != value {
				return false
			}
		}
		return true
	case model.QuestMissionConditionTypeDeckCostumeNumLe:
		return int32(len(units)) <= value
	case model.QuestMissionConditionTypeCriticalCountGe:
		return freshBattle && detail.CriticalCount >= value
	case model.QuestMissionConditionTypeMinHpPercentageGe:
		if !freshBattle || detail.CostumeResultCount == 0 || detail.CostumeResultCount < user.Battle.LastUserPartyCount {
			return false
		}
		threshold := int64(value)
		if threshold <= 100 {
			threshold *= 10
		}
		for _, result := range detail.CostumeResults {
			if result.MaxHp > 0 && result.RemainingHp*1000/result.MaxHp < threshold {
				return false
			}
		}
		return true
	case model.QuestMissionConditionTypeComboCountGe:
		return freshBattle && detail.ComboCount >= value
	case model.QuestMissionConditionTypeLessThanOrEqualXCostumeSkillUseCount:
		return freshBattle && detail.CostumeSkillUseCount <= value
	case model.QuestMissionConditionTypeLessThanOrEqualXWeaponSkillUseCount:
		return freshBattle && detail.WeaponSkillUseCount <= value
	case model.QuestMissionConditionTypeLessThanOrEqualXCompanionSkillUseCount:
		return freshBattle && detail.CompanionSkillUseCount <= value
	case model.QuestMissionConditionTypeWithoutRecoverySkill:
		return freshBattle && detail.TotalRecoverPoint == 0
	case model.QuestMissionConditionTypeCharacterContainAll:
		for _, required := range values {
			found := false
			for _, unit := range units {
				if unit.costume.CharacterId == required {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return len(values) > 0
	case model.QuestMissionConditionTypeCostumeSkillfulWeaponContainAll:
		for _, required := range values {
			found := false
			for _, unit := range units {
				if unit.costume.SkillfulWeaponType == required && unit.weapon.WeaponType == required {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return len(values) > 0
	case model.QuestMissionConditionTypeComplete:
		return false
	default:
		return false
	}
}

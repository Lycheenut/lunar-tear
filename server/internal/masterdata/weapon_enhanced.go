package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/utils"
)

type WeaponEnhancedReward struct {
	EntityMWeaponEnhanced
	Exp           int32
	SkillLevels   map[int32]int32 // slot -> level
	AbilityLevels map[int32]int32 // slot -> level
}

func loadWeaponEnhancedRewards(weapons map[int32]EntityMWeapon, skills []EntityMWeaponSkillGroup, abilities []EntityMWeaponAbilityGroup, parameters []EntityMNumericalParameterMap) (map[int32]WeaponEnhancedReward, error) {
	rows, err := utils.ReadTable[EntityMWeaponEnhanced]("m_weapon_enhanced")
	if err != nil {
		return nil, fmt.Errorf("load enhanced weapons: %w", err)
	}
	skillRows, err := utils.ReadTable[EntityMWeaponEnhancedSkill]("m_weapon_enhanced_skill")
	if err != nil {
		return nil, fmt.Errorf("load enhanced weapon skills: %w", err)
	}
	abilityRows, err := utils.ReadTable[EntityMWeaponEnhancedAbility]("m_weapon_enhanced_ability")
	if err != nil {
		return nil, fmt.Errorf("load enhanced weapon abilities: %w", err)
	}
	enhanceRows, err := utils.ReadTable[EntityMWeaponSpecificEnhance]("m_weapon_specific_enhance")
	if err != nil {
		return nil, fmt.Errorf("load weapon enhancement curves: %w", err)
	}
	rarityRows, err := utils.ReadTable[EntityMWeaponRarity]("m_weapon_rarity")
	if err != nil {
		return nil, fmt.Errorf("load weapon rarity curves: %w", err)
	}
	expMapByEnhance, expMapByRarity := map[int32]int32{}, map[int32]int32{}
	for _, row := range enhanceRows {
		expMapByEnhance[row.WeaponSpecificEnhanceId] = row.RequiredExpForLevelUpNumericalParameterMapId
	}
	for _, row := range rarityRows {
		expMapByRarity[row.RarityType] = row.RequiredExpForLevelUpNumericalParameterMapId
	}
	result := make(map[int32]WeaponEnhancedReward, len(rows))
	for _, row := range rows {
		weapon, ok := weapons[row.WeaponId]
		if !ok || row.Level < 1 || row.LimitBreakCount < 0 {
			return nil, fmt.Errorf("invalid enhanced weapon %d", row.WeaponEnhancedId)
		}
		expMap := expMapByEnhance[weapon.WeaponSpecificEnhanceId]
		if weapon.WeaponSpecificEnhanceId == 0 {
			expMap = expMapByRarity[weapon.RarityType]
		}
		thresholds := BuildExpThresholds(parameters, expMap)
		if int(row.Level) >= len(thresholds) {
			return nil, fmt.Errorf("enhanced weapon %d has no experience threshold for level %d", row.WeaponEnhancedId, row.Level)
		}
		result[row.WeaponEnhancedId] = WeaponEnhancedReward{
			EntityMWeaponEnhanced: row, Exp: thresholds[row.Level], SkillLevels: map[int32]int32{}, AbilityLevels: map[int32]int32{},
		}
	}
	for _, row := range skillRows {
		reward, ok := result[row.WeaponEnhancedId]
		if !ok || row.Level < 1 {
			return nil, fmt.Errorf("invalid skill for enhanced weapon %d", row.WeaponEnhancedId)
		}
		matched := false
		for _, skill := range skills {
			if skill.WeaponSkillGroupId == weapons[reward.WeaponId].WeaponSkillGroupId && skill.SkillId == row.SkillId {
				reward.SkillLevels[skill.SlotNumber] = row.Level
				matched = true
			}
		}
		if !matched {
			return nil, fmt.Errorf("enhanced weapon %d has unknown skill %d", row.WeaponEnhancedId, row.SkillId)
		}
	}
	for _, row := range abilityRows {
		reward, ok := result[row.WeaponEnhancedId]
		if !ok || row.Level < 1 {
			return nil, fmt.Errorf("invalid ability for enhanced weapon %d", row.WeaponEnhancedId)
		}
		matched := false
		for _, ability := range abilities {
			if ability.WeaponAbilityGroupId == weapons[reward.WeaponId].WeaponAbilityGroupId && ability.AbilityId == row.AbilityId {
				reward.AbilityLevels[ability.SlotNumber] = row.Level
				matched = true
			}
		}
		if !matched {
			return nil, fmt.Errorf("enhanced weapon %d has unknown ability %d", row.WeaponEnhancedId, row.AbilityId)
		}
	}
	return result, nil
}

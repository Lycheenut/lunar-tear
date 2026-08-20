package masterdata

import (
	"fmt"
	"log"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/utils"
)

type WeaponAwakenEffectGroupRow struct {
	WeaponAwakenEffectGroupId int32 `json:"WeaponAwakenEffectGroupId"`
	WeaponAwakenEffectType    int32 `json:"WeaponAwakenEffectType"`
	WeaponAwakenEffectId      int32 `json:"WeaponAwakenEffectId"`
}

type WeaponCatalog struct {
	Weapons                             map[int32]EntityMWeapon
	Materials                           map[int32]EntityMMaterial
	ExpByEnhanceId                      map[int32][]int32
	GoldCostByEnhanceId                 map[int32]NumericalFunc
	MaxLevelByEnhanceId                 map[int32]NumericalFunc
	SellPriceByEnhanceId                map[int32]NumericalFunc
	MedalsByWeaponId                    map[int32]map[int32]int32 // WeaponId -> ConsumableItemId -> Count
	EvolutionNextWeaponId               map[int32]int32
	EvolutionGroupByWeaponId            map[int32]int32
	EvolutionOrder                      map[int32]int32                                 // WeaponId -> 0-based position in evolution chain
	EvolutionMaterials                  map[int32][]EntityMWeaponEvolutionMaterialGroup // WeaponEvolutionMaterialGroupId -> materials
	EvolutionCostByEnhanceId            map[int32]NumericalFunc
	AbilitySlots                        map[int32][]int32 // WeaponAbilityGroupId -> slot numbers
	SkillGroupsByGroupId                map[int32][]EntityMWeaponSkillGroup
	SkillEnhanceMats                    map[[2]int32][]EntityMWeaponSkillEnhancementMaterial // key: [enhancementMaterialId, skillLevel]
	SkillMaxLevelByEnhanceId            map[int32]NumericalFunc
	SkillCostByEnhanceId                map[int32]NumericalFunc
	AbilityGroupsByGroupId              map[int32][]EntityMWeaponAbilityGroup
	AbilityEnhanceMats                  map[[2]int32][]EntityMWeaponAbilityEnhancementMaterial // key: [enhancementMaterialId, abilityLevel]
	AbilityMaxLevelByEnhanceId          map[int32]NumericalFunc
	AbilityCostByEnhanceId              map[int32]NumericalFunc
	EnhanceCostByWeaponByEnhanceId      map[int32]NumericalFunc
	LimitBreakCostByWeaponByEnhanceId   map[int32]NumericalFunc
	LimitBreakCostByMaterialByEnhanceId map[int32]NumericalFunc
	SpecificLimitBreakByGroup           map[int32][]EntityMWeaponSpecificLimitBreakMaterialGroup
	RarityLimitBreakByRarity            map[int32]MaterialOption
	BaseExpByEnhanceId                  map[int32]int32
	ReleaseConditionsByGroupId          map[int32][]EntityMWeaponStoryReleaseConditionGroup

	LevelingEnhanceIdByWeaponId map[int32]int32

	AwakenByWeaponId        map[int32]EntityMWeaponAwaken
	AwakenMaterialByGroupId map[int32]EntityMWeaponAwakenMaterialGroup
}

func (c *WeaponCatalog) LimitBreakMaterialOptions(weaponId, currentLimitBreakCount int32) []MaterialOption {
	weapon, ok := c.Weapons[weaponId]
	if !ok {
		return nil
	}
	rows := c.SpecificLimitBreakByGroup[weapon.WeaponSpecificLimitBreakMaterialGroupId]
	threshold := int32(-1)
	for _, row := range rows {
		if row.LimitBreakCountLowerLimit <= currentLimitBreakCount && row.LimitBreakCountLowerLimit > threshold {
			threshold = row.LimitBreakCountLowerLimit
		}
	}
	options := make([]MaterialOption, 0)
	for _, row := range rows {
		if row.LimitBreakCountLowerLimit == threshold {
			options = append(options, MaterialOption{MaterialId: row.MaterialId, Count: row.Count})
		}
	}
	if option, ok := c.RarityLimitBreakByRarity[weapon.RarityType]; ok {
		options = append(options, option)
	}
	return uniqueMaterialOptions(options)
}

func (c *WeaponCatalog) IsSameWeapon(firstWeaponId, secondWeaponId int32) bool {
	if firstWeaponId == secondWeaponId {
		return true
	}
	firstGroupId, firstFound := c.EvolutionGroupByWeaponId[firstWeaponId]
	secondGroupId, secondFound := c.EvolutionGroupByWeaponId[secondWeaponId]
	return firstFound && secondFound && firstGroupId == secondGroupId
}

func LoadWeaponCatalog(matCatalog *MaterialCatalog) (*WeaponCatalog, error) {
	weapons, err := utils.ReadTable[EntityMWeapon]("m_weapon")
	if err != nil {
		return nil, fmt.Errorf("load weapon table: %w", err)
	}

	enhanceRows, err := utils.ReadTable[EntityMWeaponSpecificEnhance]("m_weapon_specific_enhance")
	if err != nil {
		return nil, fmt.Errorf("load weapon specific enhance table: %w", err)
	}

	rarityEnhanceRows, err := utils.ReadTable[EntityMWeaponRarity]("m_weapon_rarity")
	if err != nil {
		return nil, fmt.Errorf("load weapon rarity table: %w", err)
	}

	paramMapRows, err := LoadParameterMap()
	if err != nil {
		return nil, err
	}

	funcResolver, err := LoadFunctionResolver()
	if err != nil {
		return nil, fmt.Errorf("load function resolver: %w", err)
	}

	exchangeRows, err := utils.ReadTable[EntityMWeaponConsumeExchangeConsumableItemGroup]("m_weapon_consume_exchange_consumable_item_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon consume exchange table: %w", err)
	}

	evoGroupRows, err := utils.ReadTable[EntityMWeaponEvolutionGroup]("m_weapon_evolution_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon evolution group table: %w", err)
	}
	evoMatRows, err := utils.ReadTable[EntityMWeaponEvolutionMaterialGroup]("m_weapon_evolution_material_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon evolution material group table: %w", err)
	}
	abilityGroupRows, err := utils.ReadTable[EntityMWeaponAbilityGroup]("m_weapon_ability_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon ability group table: %w", err)
	}
	skillGroupRows, err := utils.ReadTable[EntityMWeaponSkillGroup]("m_weapon_skill_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon skill group table: %w", err)
	}
	skillMatRows, err := utils.ReadTable[EntityMWeaponSkillEnhancementMaterial]("m_weapon_skill_enhancement_material")
	if err != nil {
		return nil, fmt.Errorf("load weapon skill enhancement material table: %w", err)
	}
	abilityMatRows, err := utils.ReadTable[EntityMWeaponAbilityEnhancementMaterial]("m_weapon_ability_enhancement_material")
	if err != nil {
		return nil, fmt.Errorf("load weapon ability enhancement material table: %w", err)
	}
	specificLimitBreakRows, err := utils.ReadTable[EntityMWeaponSpecificLimitBreakMaterialGroup]("m_weapon_specific_limit_break_material_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon specific limit break material table: %w", err)
	}
	rarityLimitBreakRows, err := utils.ReadTable[EntityMWeaponRarityLimitBreakMaterialGroup]("m_weapon_rarity_limit_break_material_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon rarity limit break material table: %w", err)
	}
	releaseConditions, err := utils.ReadTable[EntityMWeaponStoryReleaseConditionGroup]("m_weapon_story_release_condition_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon story release condition table: %w", err)
	}

	awakenRows, err := utils.ReadTable[EntityMWeaponAwaken]("m_weapon_awaken")
	if err != nil {
		return nil, fmt.Errorf("load weapon awaken table: %w", err)
	}
	awakenMatRows, err := utils.ReadTable[EntityMWeaponAwakenMaterialGroup]("m_weapon_awaken_material_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon awaken material group table: %w", err)
	}

	catalog := &WeaponCatalog{
		Weapons:                             make(map[int32]EntityMWeapon, len(weapons)),
		Materials:                           matCatalog.ByType[model.MaterialTypeWeaponEnhancement],
		ExpByEnhanceId:                      make(map[int32][]int32, len(enhanceRows)),
		GoldCostByEnhanceId:                 make(map[int32]NumericalFunc, len(enhanceRows)),
		MaxLevelByEnhanceId:                 make(map[int32]NumericalFunc, len(enhanceRows)),
		SellPriceByEnhanceId:                make(map[int32]NumericalFunc, len(enhanceRows)),
		MedalsByWeaponId:                    make(map[int32]map[int32]int32),
		EvolutionNextWeaponId:               make(map[int32]int32),
		EvolutionGroupByWeaponId:            make(map[int32]int32),
		EvolutionOrder:                      make(map[int32]int32),
		EvolutionMaterials:                  make(map[int32][]EntityMWeaponEvolutionMaterialGroup),
		EvolutionCostByEnhanceId:            make(map[int32]NumericalFunc, len(enhanceRows)),
		AbilitySlots:                        make(map[int32][]int32),
		SkillGroupsByGroupId:                make(map[int32][]EntityMWeaponSkillGroup),
		SkillEnhanceMats:                    make(map[[2]int32][]EntityMWeaponSkillEnhancementMaterial),
		SkillMaxLevelByEnhanceId:            make(map[int32]NumericalFunc, len(enhanceRows)),
		SkillCostByEnhanceId:                make(map[int32]NumericalFunc, len(enhanceRows)),
		AbilityGroupsByGroupId:              make(map[int32][]EntityMWeaponAbilityGroup),
		AbilityEnhanceMats:                  make(map[[2]int32][]EntityMWeaponAbilityEnhancementMaterial),
		AbilityMaxLevelByEnhanceId:          make(map[int32]NumericalFunc, len(enhanceRows)),
		AbilityCostByEnhanceId:              make(map[int32]NumericalFunc, len(enhanceRows)),
		EnhanceCostByWeaponByEnhanceId:      make(map[int32]NumericalFunc, len(enhanceRows)),
		LimitBreakCostByWeaponByEnhanceId:   make(map[int32]NumericalFunc, len(enhanceRows)),
		LimitBreakCostByMaterialByEnhanceId: make(map[int32]NumericalFunc, len(enhanceRows)),
		SpecificLimitBreakByGroup:           make(map[int32][]EntityMWeaponSpecificLimitBreakMaterialGroup),
		RarityLimitBreakByRarity:            make(map[int32]MaterialOption),
		BaseExpByEnhanceId:                  make(map[int32]int32, len(enhanceRows)),
		ReleaseConditionsByGroupId:          make(map[int32][]EntityMWeaponStoryReleaseConditionGroup),

		LevelingEnhanceIdByWeaponId: make(map[int32]int32, len(weapons)),

		AwakenByWeaponId:        make(map[int32]EntityMWeaponAwaken, len(awakenRows)),
		AwakenMaterialByGroupId: make(map[int32]EntityMWeaponAwakenMaterialGroup),
	}

	for _, w := range weapons {
		catalog.Weapons[w.WeaponId] = w
	}

	for _, r := range enhanceRows {
		if _, ok := catalog.ExpByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			catalog.ExpByEnhanceId[r.WeaponSpecificEnhanceId] = BuildExpThresholds(paramMapRows, r.RequiredExpForLevelUpNumericalParameterMapId)
		}
		if _, ok := catalog.GoldCostByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.EnhancementCostByMaterialNumericalFunctionId); found {
				catalog.GoldCostByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.MaxLevelByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.MaxLevelNumericalFunctionId); found {
				catalog.MaxLevelByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.SellPriceByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.SellPriceNumericalFunctionId); found {
				catalog.SellPriceByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.EvolutionCostByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.EvolutionCostNumericalFunctionId); found {
				catalog.EvolutionCostByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.SkillMaxLevelByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.MaxSkillLevelNumericalFunctionId); found {
				catalog.SkillMaxLevelByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.SkillCostByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.SkillEnhancementCostNumericalFunctionId); found {
				catalog.SkillCostByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.AbilityMaxLevelByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.MaxAbilityLevelNumericalFunctionId); found {
				catalog.AbilityMaxLevelByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.AbilityCostByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.AbilityEnhancementCostNumericalFunctionId); found {
				catalog.AbilityCostByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.EnhanceCostByWeaponByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.EnhancementCostByWeaponNumericalFunctionId); found {
				catalog.EnhanceCostByWeaponByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.LimitBreakCostByWeaponByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.LimitBreakCostByWeaponNumericalFunctionId); found {
				catalog.LimitBreakCostByWeaponByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.LimitBreakCostByMaterialByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			if f, found := funcResolver.Resolve(r.LimitBreakCostByMaterialNumericalFunctionId); found {
				catalog.LimitBreakCostByMaterialByEnhanceId[r.WeaponSpecificEnhanceId] = f
			}
		}
		if _, ok := catalog.BaseExpByEnhanceId[r.WeaponSpecificEnhanceId]; !ok {
			catalog.BaseExpByEnhanceId[r.WeaponSpecificEnhanceId] = r.BaseEnhancementObtainedExp
		}
	}

	for _, ex := range exchangeRows {
		if catalog.MedalsByWeaponId[ex.WeaponId] == nil {
			catalog.MedalsByWeaponId[ex.WeaponId] = make(map[int32]int32)
		}
		catalog.MedalsByWeaponId[ex.WeaponId][ex.ConsumableItemId] = ex.Count
	}

	grouped := make(map[int32][]EntityMWeaponEvolutionGroup)
	for _, row := range evoGroupRows {
		grouped[row.WeaponEvolutionGroupId] = append(grouped[row.WeaponEvolutionGroupId], row)
		catalog.EvolutionGroupByWeaponId[row.WeaponId] = row.WeaponEvolutionGroupId
	}
	for _, rows := range grouped {
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].EvolutionOrder < rows[j].EvolutionOrder
		})
		for i, row := range rows {
			catalog.EvolutionOrder[row.WeaponId] = int32(i)
			if i < len(rows)-1 {
				catalog.EvolutionNextWeaponId[row.WeaponId] = rows[i+1].WeaponId
			}
		}
	}

	for _, row := range evoMatRows {
		catalog.EvolutionMaterials[row.WeaponEvolutionMaterialGroupId] = append(
			catalog.EvolutionMaterials[row.WeaponEvolutionMaterialGroupId], row)
	}

	for _, row := range abilityGroupRows {
		catalog.AbilitySlots[row.WeaponAbilityGroupId] = append(
			catalog.AbilitySlots[row.WeaponAbilityGroupId], row.SlotNumber)
	}

	for _, row := range skillGroupRows {
		catalog.SkillGroupsByGroupId[row.WeaponSkillGroupId] = append(
			catalog.SkillGroupsByGroupId[row.WeaponSkillGroupId], row)
	}

	for _, row := range skillMatRows {
		key := [2]int32{row.WeaponSkillEnhancementMaterialId, row.SkillLevel}
		catalog.SkillEnhanceMats[key] = append(catalog.SkillEnhanceMats[key], row)
	}

	for _, row := range abilityGroupRows {
		catalog.AbilityGroupsByGroupId[row.WeaponAbilityGroupId] = append(
			catalog.AbilityGroupsByGroupId[row.WeaponAbilityGroupId], row)
	}

	for _, row := range abilityMatRows {
		key := [2]int32{row.WeaponAbilityEnhancementMaterialId, row.AbilityLevel}
		catalog.AbilityEnhanceMats[key] = append(catalog.AbilityEnhanceMats[key], row)
	}
	for _, row := range specificLimitBreakRows {
		catalog.SpecificLimitBreakByGroup[row.WeaponSpecificLimitBreakMaterialGroupId] = append(
			catalog.SpecificLimitBreakByGroup[row.WeaponSpecificLimitBreakMaterialGroupId], row)
	}
	for _, row := range rarityLimitBreakRows {
		catalog.RarityLimitBreakByRarity[row.RarityType] = MaterialOption{
			MaterialId: row.MaterialId,
			Count:      row.Count,
		}
	}

	for _, c := range releaseConditions {
		catalog.ReleaseConditionsByGroupId[c.WeaponStoryReleaseConditionGroupId] = append(
			catalog.ReleaseConditionsByGroupId[c.WeaponStoryReleaseConditionGroupId], c)
	}

	for _, row := range awakenRows {
		catalog.AwakenByWeaponId[row.WeaponId] = row
	}
	for _, row := range awakenMatRows {
		catalog.AwakenMaterialByGroupId[row.WeaponAwakenMaterialGroupId] = row
	}

	// Rarity-based enhancement fallback: for weapons with WeaponSpecificEnhanceId == 0,
	// use EntityMWeaponRarityTable curves via synthetic enhance IDs (-RarityType).
	rarityByType := make(map[int32]EntityMWeaponRarity, len(rarityEnhanceRows))
	for _, r := range rarityEnhanceRows {
		rarityByType[r.RarityType] = r
	}

	registeredRarity := make(map[int32]bool, len(rarityEnhanceRows))
	fallbackCount := 0
	for wid, w := range catalog.Weapons {
		if w.WeaponSpecificEnhanceId != 0 {
			continue
		}
		syntheticId := -w.RarityType
		if !registeredRarity[w.RarityType] {
			r, ok := rarityByType[w.RarityType]
			if !ok {
				continue
			}
			catalog.ExpByEnhanceId[syntheticId] = BuildExpThresholds(paramMapRows, r.RequiredExpForLevelUpNumericalParameterMapId)
			if f, found := funcResolver.Resolve(r.EnhancementCostByMaterialNumericalFunctionId); found {
				catalog.GoldCostByEnhanceId[syntheticId] = f
			}
			if f, found := funcResolver.Resolve(r.MaxLevelNumericalFunctionId); found {
				catalog.MaxLevelByEnhanceId[syntheticId] = f
			}
			if f, found := funcResolver.Resolve(r.SellPriceNumericalFunctionId); found {
				catalog.SellPriceByEnhanceId[syntheticId] = f
			}
			if f, found := funcResolver.Resolve(r.EvolutionCostNumericalFunctionId); found {
				catalog.EvolutionCostByEnhanceId[syntheticId] = f
			}
			if f, found := funcResolver.Resolve(r.MaxSkillLevelNumericalFunctionId); found {
				catalog.SkillMaxLevelByEnhanceId[syntheticId] = f
			}
			if f, found := funcResolver.Resolve(r.SkillEnhancementCostNumericalFunctionId); found {
				catalog.SkillCostByEnhanceId[syntheticId] = f
			}
			if f, found := funcResolver.Resolve(r.MaxAbilityLevelNumericalFunctionId); found {
				catalog.AbilityMaxLevelByEnhanceId[syntheticId] = f
			}
			if f, found := funcResolver.Resolve(r.AbilityEnhancementCostNumericalFunctionId); found {
				catalog.AbilityCostByEnhanceId[syntheticId] = f
			}
			if f, found := funcResolver.Resolve(r.EnhancementCostByWeaponNumericalFunctionId); found {
				catalog.EnhanceCostByWeaponByEnhanceId[syntheticId] = f
			}
			if f, found := funcResolver.Resolve(r.LimitBreakCostByWeaponNumericalFunctionId); found {
				catalog.LimitBreakCostByWeaponByEnhanceId[syntheticId] = f
			}
			if f, found := funcResolver.Resolve(r.LimitBreakCostByMaterialNumericalFunctionId); found {
				catalog.LimitBreakCostByMaterialByEnhanceId[syntheticId] = f
			}
			catalog.BaseExpByEnhanceId[syntheticId] = r.BaseEnhancementObtainedExp
			registeredRarity[w.RarityType] = true
		}
		w.WeaponSpecificEnhanceId = syntheticId
		catalog.Weapons[wid] = w
		fallbackCount++
	}
	log.Printf("[WeaponCatalog] rarity fallback: assigned synthetic enhance IDs to %d weapons", fallbackCount)

	// Build LevelingEnhanceIdByWeaponId: every member of an evolution chain
	// inherits the chain root's (post-rarity-fallback) WeaponSpecificEnhanceId
	// so cumulative exp stays consistent across Evolve. Weapons not in any
	// chain map to their own enhance id.
	for _, rows := range grouped {
		if len(rows) == 0 {
			continue
		}
		rootMaster, ok := catalog.Weapons[rows[0].WeaponId]
		if !ok {
			continue
		}
		rootEnhanceId := rootMaster.WeaponSpecificEnhanceId
		for _, row := range rows {
			catalog.LevelingEnhanceIdByWeaponId[row.WeaponId] = rootEnhanceId
		}
	}
	for wid, w := range catalog.Weapons {
		if _, exists := catalog.LevelingEnhanceIdByWeaponId[wid]; !exists {
			catalog.LevelingEnhanceIdByWeaponId[wid] = w.WeaponSpecificEnhanceId
		}
	}

	return catalog, nil
}

package masterdata

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/utils"
)

type CostumeCatalog struct {
	Costumes               map[int32]EntityMCostume
	Materials              map[int32]EntityMMaterial
	ExpByRarity            map[int32][]int32
	EnhanceCostByRarity    map[int32]NumericalFunc
	MaxLevelByRarity       map[int32]NumericalFunc
	LimitBreakCostByRarity map[int32]NumericalFunc

	AwakenByCostumeId           map[int32]EntityMCostumeAwaken
	AwakenPriceByGroup          map[int32]EntityMCostumeAwakenPriceGroup
	AwakenStepByGroup           map[int32]EntityMCostumeAwakenStepMaterialGroup
	AwakenMaterialsByGroup      map[int32][]MaterialOption
	AwakenEffectsByGroupAndStep map[int32]map[int32]EntityMCostumeAwakenEffectGroup
	AwakenStatusUpByGroup       map[int32][]EntityMCostumeAwakenStatusUpGroup
	AwakenItemAcquireById       map[int32]EntityMCostumeAwakenItemAcquire

	ActiveSkillGroupsByGroupId  map[int32][]EntityMCostumeActiveSkillGroup                  // sorted by CostumeLimitBreakCountLowerLimit desc
	ActiveSkillEnhanceMats      map[[2]int32][]EntityMCostumeActiveSkillEnhancementMaterial // key: [enhancementMaterialId, skillLevel]
	ActiveSkillMaxLevelByRarity map[int32]NumericalFunc
	ActiveSkillCostByRarity     map[int32]NumericalFunc
	AbilityGroupsByGroupId      map[int32][]EntityMCostumeAbilityGroup
	AbilityLevelsByGroupId      map[int32][]EntityMCostumeAbilityLevelGroup

	LotteryEffects               map[[2]int32]EntityMCostumeLotteryEffect             // key: [costumeId, slotNumber]
	LotteryEffectMats            map[int32][]EntityMCostumeLotteryEffectMaterialGroup // key: materialGroupId (both unlock and draw)
	LotteryEffectOdds            map[int32][]EntityMCostumeLotteryEffectOddsGroup     // key: oddsGroupId
	LotteryEffectOddsByNumber    map[[2]int32]EntityMCostumeLotteryEffectOddsGroup
	LotteryEffectTargetAbilities map[int32]EntityMCostumeLotteryEffectTargetAbility
	LotteryEffectTargetStatusUps map[int32][]EntityMCostumeLotteryEffectTargetStatusUp
	LevelBonusLevelsByCostume    map[int32][]int32
	LimitBreakMaterialsByCostume map[int32][]MaterialOption
}

func (c *CostumeCatalog) AbilityLevel(costumeId, limitBreakCount int32) int32 {
	costume, ok := c.Costumes[costumeId]
	if !ok {
		return 0
	}
	var maximum int32
	for _, ability := range c.AbilityGroupsByGroupId[costume.CostumeAbilityGroupId] {
		var level int32
		for _, threshold := range c.AbilityLevelsByGroupId[ability.CostumeAbilityLevelGroupId] {
			if limitBreakCount < threshold.CostumeLimitBreakCountLowerLimit {
				break
			}
			level = threshold.AbilityLevel
		}
		maximum = max(maximum, level)
	}
	return maximum
}

type MaterialOption struct {
	MaterialId int32
	Count      int32
}

func (c *CostumeCatalog) AwakenGold(priceGroupId, awakenStep int32) (int32, bool) {
	row, ok := c.AwakenPriceByGroup[priceGroupId]
	if !ok || awakenStep < row.AwakenStepLowerLimit {
		return 0, false
	}
	return row.Gold, true
}

func (c *CostumeCatalog) AwakenMaterialOptions(stepGroupId, awakenStep int32) []MaterialOption {
	row, ok := c.AwakenStepByGroup[stepGroupId]
	if !ok || awakenStep < row.AwakenStepLowerLimit {
		return nil
	}
	return c.AwakenMaterialsByGroup[row.CostumeAwakenMaterialGroupId]
}

func LoadCostumeCatalog(matCatalog *MaterialCatalog) (*CostumeCatalog, error) {
	costumes, err := utils.ReadTable[EntityMCostume]("m_costume")
	if err != nil {
		return nil, fmt.Errorf("load costume table: %w", err)
	}

	rarities, err := utils.ReadTable[EntityMCostumeRarity]("m_costume_rarity")
	if err != nil {
		return nil, fmt.Errorf("load costume rarity table: %w", err)
	}

	paramMapRows, err := LoadParameterMap()
	if err != nil {
		return nil, err
	}

	funcResolver, err := LoadFunctionResolver()
	if err != nil {
		return nil, fmt.Errorf("load function resolver: %w", err)
	}

	awakenRows, err := utils.ReadTable[EntityMCostumeAwaken]("m_costume_awaken")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken table: %w", err)
	}
	awakenPriceRows, err := utils.ReadTable[EntityMCostumeAwakenPriceGroup]("m_costume_awaken_price_group")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken price table: %w", err)
	}
	awakenStepRows, err := utils.ReadTable[EntityMCostumeAwakenStepMaterialGroup]("m_costume_awaken_step_material_group")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken step material table: %w", err)
	}
	awakenMaterialRows, err := utils.ReadTable[EntityMCostumeAwakenMaterialGroup]("m_costume_awaken_material_group")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken material table: %w", err)
	}
	awakenEffectRows, err := utils.ReadTable[EntityMCostumeAwakenEffectGroup]("m_costume_awaken_effect_group")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken effect table: %w", err)
	}
	awakenStatusUpRows, err := utils.ReadTable[EntityMCostumeAwakenStatusUpGroup]("m_costume_awaken_status_up_group")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken status up table: %w", err)
	}
	awakenItemAcquireRows, err := utils.ReadTable[EntityMCostumeAwakenItemAcquire]("m_costume_awaken_item_acquire")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken item acquire table: %w", err)
	}

	activeSkillGroupRows, err := utils.ReadTable[EntityMCostumeActiveSkillGroup]("m_costume_active_skill_group")
	if err != nil {
		return nil, fmt.Errorf("load costume active skill group table: %w", err)
	}
	activeSkillMatRows, err := utils.ReadTable[EntityMCostumeActiveSkillEnhancementMaterial]("m_costume_active_skill_enhancement_material")
	if err != nil {
		return nil, fmt.Errorf("load costume active skill enhancement material table: %w", err)
	}
	abilityGroupRows, err := utils.ReadTable[EntityMCostumeAbilityGroup]("m_costume_ability_group")
	if err != nil {
		return nil, fmt.Errorf("load costume ability group table: %w", err)
	}
	abilityLevelRows, err := utils.ReadTable[EntityMCostumeAbilityLevelGroup]("m_costume_ability_level_group")
	if err != nil {
		return nil, fmt.Errorf("load costume ability level table: %w", err)
	}

	lotteryEffectRows, err := utils.ReadTable[EntityMCostumeLotteryEffect]("m_costume_lottery_effect")
	if err != nil {
		return nil, fmt.Errorf("load costume lottery effect table: %w", err)
	}
	lotteryEffectMatRows, err := utils.ReadTable[EntityMCostumeLotteryEffectMaterialGroup]("m_costume_lottery_effect_material_group")
	if err != nil {
		return nil, fmt.Errorf("load costume lottery effect material group table: %w", err)
	}
	lotteryEffectOddsRows, err := utils.ReadTable[EntityMCostumeLotteryEffectOddsGroup]("m_costume_lottery_effect_odds_group")
	if err != nil {
		return nil, fmt.Errorf("load costume lottery effect odds group table: %w", err)
	}
	lotteryAbilityRows, err := utils.ReadTable[EntityMCostumeLotteryEffectTargetAbility]("m_costume_lottery_effect_target_ability")
	if err != nil {
		return nil, fmt.Errorf("load costume lottery effect target ability table: %w", err)
	}
	lotteryStatusRows, err := utils.ReadTable[EntityMCostumeLotteryEffectTargetStatusUp]("m_costume_lottery_effect_target_status_up")
	if err != nil {
		return nil, fmt.Errorf("load costume lottery effect target status table: %w", err)
	}
	levelBonusRows, err := utils.ReadTable[EntityMCostumeLevelBonus]("m_costume_level_bonus")
	if err != nil {
		return nil, fmt.Errorf("load costume level bonus table: %w", err)
	}
	limitBreakRows, err := utils.ReadTable[EntityMCostumeLimitBreakMaterialGroup]("m_costume_limit_break_material_group")
	if err != nil {
		return nil, fmt.Errorf("load costume limit break material table: %w", err)
	}
	limitBreakRarityRows, err := utils.ReadTable[EntityMCostumeLimitBreakMaterialRarityGroup]("m_costume_limit_break_material_rarity_group")
	if err != nil {
		return nil, fmt.Errorf("load costume rarity limit break material table: %w", err)
	}

	catalog := &CostumeCatalog{
		Costumes:               make(map[int32]EntityMCostume, len(costumes)),
		Materials:              matCatalog.ByType[model.MaterialTypeCostumeEnhancement],
		ExpByRarity:            make(map[int32][]int32, len(rarities)),
		EnhanceCostByRarity:    make(map[int32]NumericalFunc, len(rarities)),
		MaxLevelByRarity:       make(map[int32]NumericalFunc, len(rarities)),
		LimitBreakCostByRarity: make(map[int32]NumericalFunc, len(rarities)),

		AwakenByCostumeId:           make(map[int32]EntityMCostumeAwaken, len(awakenRows)),
		AwakenPriceByGroup:          make(map[int32]EntityMCostumeAwakenPriceGroup),
		AwakenStepByGroup:           make(map[int32]EntityMCostumeAwakenStepMaterialGroup),
		AwakenMaterialsByGroup:      make(map[int32][]MaterialOption),
		AwakenEffectsByGroupAndStep: make(map[int32]map[int32]EntityMCostumeAwakenEffectGroup),
		AwakenStatusUpByGroup:       make(map[int32][]EntityMCostumeAwakenStatusUpGroup),
		AwakenItemAcquireById:       make(map[int32]EntityMCostumeAwakenItemAcquire, len(awakenItemAcquireRows)),

		ActiveSkillGroupsByGroupId:  make(map[int32][]EntityMCostumeActiveSkillGroup),
		ActiveSkillEnhanceMats:      make(map[[2]int32][]EntityMCostumeActiveSkillEnhancementMaterial),
		ActiveSkillMaxLevelByRarity: make(map[int32]NumericalFunc, len(rarities)),
		ActiveSkillCostByRarity:     make(map[int32]NumericalFunc, len(rarities)),
		AbilityGroupsByGroupId:      make(map[int32][]EntityMCostumeAbilityGroup),
		AbilityLevelsByGroupId:      make(map[int32][]EntityMCostumeAbilityLevelGroup),

		LotteryEffects:               make(map[[2]int32]EntityMCostumeLotteryEffect, len(lotteryEffectRows)),
		LotteryEffectMats:            make(map[int32][]EntityMCostumeLotteryEffectMaterialGroup),
		LotteryEffectOdds:            make(map[int32][]EntityMCostumeLotteryEffectOddsGroup),
		LotteryEffectOddsByNumber:    make(map[[2]int32]EntityMCostumeLotteryEffectOddsGroup),
		LotteryEffectTargetAbilities: make(map[int32]EntityMCostumeLotteryEffectTargetAbility),
		LotteryEffectTargetStatusUps: make(map[int32][]EntityMCostumeLotteryEffectTargetStatusUp),
		LevelBonusLevelsByCostume:    make(map[int32][]int32),
		LimitBreakMaterialsByCostume: make(map[int32][]MaterialOption),
	}

	for _, row := range costumes {
		catalog.Costumes[row.CostumeId] = row
	}

	for _, r := range rarities {
		if _, ok := catalog.ExpByRarity[r.RarityType]; !ok {
			catalog.ExpByRarity[r.RarityType] = BuildExpThresholds(paramMapRows, r.RequiredExpForLevelUpNumericalParameterMapId)
		}
		if _, ok := catalog.EnhanceCostByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.EnhancementCostByMaterialNumericalFunctionId); found {
				catalog.EnhanceCostByRarity[r.RarityType] = f
			}
		}
		if _, ok := catalog.MaxLevelByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.MaxLevelNumericalFunctionId); found {
				catalog.MaxLevelByRarity[r.RarityType] = f
			}
		}
		if _, ok := catalog.LimitBreakCostByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.LimitBreakCostNumericalFunctionId); found {
				catalog.LimitBreakCostByRarity[r.RarityType] = f
			}
		}
		if _, ok := catalog.ActiveSkillMaxLevelByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.ActiveSkillMaxLevelNumericalFunctionId); found {
				catalog.ActiveSkillMaxLevelByRarity[r.RarityType] = f
			}
		}
		if _, ok := catalog.ActiveSkillCostByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.ActiveSkillEnhancementCostNumericalFunctionId); found {
				catalog.ActiveSkillCostByRarity[r.RarityType] = f
			}
		}
	}

	for _, row := range awakenRows {
		catalog.AwakenByCostumeId[row.CostumeId] = row
	}
	for _, row := range awakenPriceRows {
		catalog.AwakenPriceByGroup[row.CostumeAwakenPriceGroupId] = row
	}
	for _, row := range awakenStepRows {
		catalog.AwakenStepByGroup[row.CostumeAwakenStepMaterialGroupId] = row
	}
	for _, row := range awakenMaterialRows {
		catalog.AwakenMaterialsByGroup[row.CostumeAwakenMaterialGroupId] = append(catalog.AwakenMaterialsByGroup[row.CostumeAwakenMaterialGroupId], MaterialOption{
			MaterialId: row.MaterialId,
			Count:      row.Count,
		})
	}
	for _, row := range awakenEffectRows {
		m, ok := catalog.AwakenEffectsByGroupAndStep[row.CostumeAwakenEffectGroupId]
		if !ok {
			m = make(map[int32]EntityMCostumeAwakenEffectGroup)
			catalog.AwakenEffectsByGroupAndStep[row.CostumeAwakenEffectGroupId] = m
		}
		m[row.AwakenStep] = row
	}
	for _, row := range awakenStatusUpRows {
		catalog.AwakenStatusUpByGroup[row.CostumeAwakenStatusUpGroupId] = append(
			catalog.AwakenStatusUpByGroup[row.CostumeAwakenStatusUpGroupId], row)
	}
	for _, row := range awakenItemAcquireRows {
		catalog.AwakenItemAcquireById[row.CostumeAwakenItemAcquireId] = row
	}

	for _, row := range activeSkillGroupRows {
		gid := row.CostumeActiveSkillGroupId
		catalog.ActiveSkillGroupsByGroupId[gid] = append(catalog.ActiveSkillGroupsByGroupId[gid], row)
	}
	for gid, rows := range catalog.ActiveSkillGroupsByGroupId {
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].CostumeLimitBreakCountLowerLimit > rows[j].CostumeLimitBreakCountLowerLimit
		})
		catalog.ActiveSkillGroupsByGroupId[gid] = rows
	}
	for _, row := range abilityGroupRows {
		catalog.AbilityGroupsByGroupId[row.CostumeAbilityGroupId] = append(catalog.AbilityGroupsByGroupId[row.CostumeAbilityGroupId], row)
	}
	for _, row := range abilityLevelRows {
		catalog.AbilityLevelsByGroupId[row.CostumeAbilityLevelGroupId] = append(catalog.AbilityLevelsByGroupId[row.CostumeAbilityLevelGroupId], row)
	}
	for groupId := range catalog.AbilityLevelsByGroupId {
		sort.Slice(catalog.AbilityLevelsByGroupId[groupId], func(i, j int) bool {
			return catalog.AbilityLevelsByGroupId[groupId][i].CostumeLimitBreakCountLowerLimit < catalog.AbilityLevelsByGroupId[groupId][j].CostumeLimitBreakCountLowerLimit
		})
	}

	for _, row := range activeSkillMatRows {
		key := [2]int32{row.CostumeActiveSkillEnhancementMaterialId, row.SkillLevel}
		catalog.ActiveSkillEnhanceMats[key] = append(catalog.ActiveSkillEnhanceMats[key], row)
	}

	for _, row := range lotteryEffectRows {
		key := [2]int32{row.CostumeId, row.SlotNumber}
		catalog.LotteryEffects[key] = row
	}
	for _, row := range lotteryEffectMatRows {
		gid := row.CostumeLotteryEffectMaterialGroupId
		catalog.LotteryEffectMats[gid] = append(catalog.LotteryEffectMats[gid], row)
	}
	for _, row := range lotteryEffectOddsRows {
		gid := row.CostumeLotteryEffectOddsGroupId
		catalog.LotteryEffectOdds[gid] = append(catalog.LotteryEffectOdds[gid], row)
		catalog.LotteryEffectOddsByNumber[[2]int32{gid, row.OddsNumber}] = row
	}
	for _, row := range lotteryAbilityRows {
		catalog.LotteryEffectTargetAbilities[row.CostumeLotteryEffectTargetAbilityId] = row
	}
	for _, row := range lotteryStatusRows {
		catalog.LotteryEffectTargetStatusUps[row.CostumeLotteryEffectTargetStatusUpId] = append(catalog.LotteryEffectTargetStatusUps[row.CostumeLotteryEffectTargetStatusUpId], row)
	}
	bonusLevelsById := make(map[int32][]int32)
	for _, row := range levelBonusRows {
		bonusLevelsById[row.CostumeLevelBonusId] = append(bonusLevelsById[row.CostumeLevelBonusId], row.Level)
	}
	for costumeId, costume := range catalog.Costumes {
		levels := bonusLevelsById[costume.CostumeLevelBonusId]
		sort.Slice(levels, func(i, j int) bool { return levels[i] < levels[j] })
		catalog.LevelBonusLevelsByCostume[costumeId] = levels
	}

	limitBreakByGroup := make(map[int32]MaterialOption, len(limitBreakRows))
	for _, row := range limitBreakRows {
		limitBreakByGroup[row.CostumeLimitBreakMaterialGroupId] = MaterialOption{
			MaterialId: row.MaterialId,
			Count:      row.Count,
		}
	}
	rarityLimitBreakByGroup := make(map[int32]MaterialOption, len(limitBreakRarityRows))
	for _, row := range limitBreakRarityRows {
		rarityLimitBreakByGroup[row.CostumeLimitBreakMaterialRarityGroupId] = MaterialOption{
			MaterialId: row.MaterialId,
			Count:      row.Count,
		}
	}
	rarityLimitBreakGroup := make(map[int32]int32, len(rarities))
	for _, row := range rarities {
		rarityLimitBreakGroup[row.RarityType] = row.CostumeLimitBreakMaterialRarityGroupId
	}
	for costumeId, costume := range catalog.Costumes {
		options := make([]MaterialOption, 0, 2)
		if option, ok := limitBreakByGroup[costume.CostumeLimitBreakMaterialGroupId]; ok {
			options = append(options, option)
		}
		if option, ok := rarityLimitBreakByGroup[rarityLimitBreakGroup[costume.RarityType]]; ok {
			options = append(options, option)
		}
		catalog.LimitBreakMaterialsByCostume[costumeId] = uniqueMaterialOptions(options)
	}

	return catalog, nil
}

func uniqueMaterialOptions(options []MaterialOption) []MaterialOption {
	result := make([]MaterialOption, 0, len(options))
	indexByMaterial := make(map[int32]int, len(options))
	for _, option := range options {
		if index, exists := indexByMaterial[option.MaterialId]; exists {
			if option.Count > 0 && (result[index].Count <= 0 || option.Count < result[index].Count) {
				result[index].Count = option.Count
			}
			continue
		}
		indexByMaterial[option.MaterialId] = len(result)
		result = append(result, option)
	}
	return result
}

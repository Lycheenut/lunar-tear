package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/utils"
)

type PartsStatusMainDef struct {
	StatusKindType            int32
	StatusCalculationType     int32
	StatusChangeInitialValue  int32
	StatusNumericalFunctionId int32
}

type PartsCatalog struct {
	PartsById                            map[int32]EntityMParts
	EnhancedById                         map[int32]EntityMPartsEnhanced
	EnhancedSubStatuses                  map[int32][]EntityMPartsEnhancedSubStatus
	DefaultPartsStatusMainByLotteryGroup map[int32]int32
	RarityByRarityType                   map[model.RarityType]EntityMPartsRarity
	RateByGroupAndLevel                  map[int32]map[int32]int32
	PriceByGroupAndLevel                 map[int32]map[int32]int32
	SellPriceByRarity                    map[model.RarityType]NumericalFunc

	PartsStatusMainById map[int32]PartsStatusMainDef
	SubStatusPool       map[int32][]int32            // lotteryGroupId -> eligible PartsStatusMainIds
	SubStatusUnlockLvls map[model.RarityType][]int32 // rarity -> levels where sub-slots unlock
	FuncResolver        *FunctionResolver
}

func LoadPartsCatalog() (*PartsCatalog, error) {
	partsRows, err := utils.ReadTable[EntityMParts]("m_parts")
	if err != nil {
		return nil, fmt.Errorf("load parts table: %w", err)
	}

	rarityRows, err := utils.ReadTable[EntityMPartsRarity]("m_parts_rarity")
	if err != nil {
		return nil, fmt.Errorf("load parts rarity table: %w", err)
	}

	rateRows, err := utils.ReadTable[EntityMPartsLevelUpRateGroup]("m_parts_level_up_rate_group")
	if err != nil {
		return nil, fmt.Errorf("load parts level up rate table: %w", err)
	}

	priceRows, err := utils.ReadTable[EntityMPartsLevelUpPriceGroup]("m_parts_level_up_price_group")
	if err != nil {
		return nil, fmt.Errorf("load parts level up price table: %w", err)
	}

	partsById := make(map[int32]EntityMParts, len(partsRows))
	for _, p := range partsRows {
		partsById[p.PartsId] = p
	}
	enhancedRows, err := utils.ReadTable[EntityMPartsEnhanced]("m_parts_enhanced")
	if err != nil {
		return nil, fmt.Errorf("load enhanced parts: %w", err)
	}
	subRows, err := utils.ReadTable[EntityMPartsEnhancedSubStatus]("m_parts_enhanced_sub_status")
	if err != nil {
		return nil, fmt.Errorf("load enhanced parts sub statuses: %w", err)
	}
	enhancedById := make(map[int32]EntityMPartsEnhanced, len(enhancedRows))
	for _, row := range enhancedRows {
		if _, ok := partsById[row.PartsId]; !ok {
			return nil, fmt.Errorf("enhanced parts %d references missing parts %d", row.PartsEnhancedId, row.PartsId)
		}
		enhancedById[row.PartsEnhancedId] = row
	}
	enhancedSubStatuses := make(map[int32][]EntityMPartsEnhancedSubStatus)
	for _, row := range subRows {
		if _, ok := enhancedById[row.PartsEnhancedId]; !ok {
			return nil, fmt.Errorf("sub status references missing enhanced parts %d", row.PartsEnhancedId)
		}
		enhancedSubStatuses[row.PartsEnhancedId] = append(enhancedSubStatuses[row.PartsEnhancedId], row)
	}

	// Lottery group ID encodes tier (first digit 1-4) and stat category
	// (second digit 1-6). Formula: mainStatId = (category - 1) * 4 + tier.
	defaultPartsStatusMainByLotteryGroup := make(map[int32]int32, 24)
	for tier := int32(1); tier <= 4; tier++ {
		for cat := int32(1); cat <= 6; cat++ {
			groupId := tier*10 + cat
			mainStatId := (cat-1)*4 + tier
			defaultPartsStatusMainByLotteryGroup[groupId] = mainStatId
		}
	}

	funcResolver, err := LoadFunctionResolver()
	if err != nil {
		return nil, fmt.Errorf("load function resolver: %w", err)
	}

	rarityByRarityType := make(map[model.RarityType]EntityMPartsRarity, len(rarityRows))
	sellPriceByRarity := make(map[model.RarityType]NumericalFunc, len(rarityRows))
	for _, r := range rarityRows {
		rarityByRarityType[r.RarityType] = r
		if f, ok := funcResolver.Resolve(r.SellPriceNumericalFunctionId); ok {
			sellPriceByRarity[r.RarityType] = f
		}
	}

	rateByGroupAndLevel := make(map[int32]map[int32]int32)
	for _, r := range rateRows {
		if rateByGroupAndLevel[r.PartsLevelUpRateGroupId] == nil {
			rateByGroupAndLevel[r.PartsLevelUpRateGroupId] = make(map[int32]int32)
		}
		rateByGroupAndLevel[r.PartsLevelUpRateGroupId][r.LevelLowerLimit] = r.SuccessRatePermil
	}

	priceByGroupAndLevel := make(map[int32]map[int32]int32)
	for _, p := range priceRows {
		if priceByGroupAndLevel[p.PartsLevelUpPriceGroupId] == nil {
			priceByGroupAndLevel[p.PartsLevelUpPriceGroupId] = make(map[int32]int32)
		}
		priceByGroupAndLevel[p.PartsLevelUpPriceGroupId][p.LevelLowerLimit] = p.Gold
	}

	partsStatusMainById, subStatusPool := buildPartsStatusMain()

	unlockLvls := []int32{3, 6, 9, 12}
	subStatusUnlockLvls := map[model.RarityType][]int32{
		model.RarityNormal: unlockLvls,
		model.RarityRare:   unlockLvls,
		model.RaritySRare:  unlockLvls,
		model.RaritySSRare: unlockLvls,
	}

	return &PartsCatalog{
		PartsById:                            partsById,
		EnhancedById:                         enhancedById,
		EnhancedSubStatuses:                  enhancedSubStatuses,
		DefaultPartsStatusMainByLotteryGroup: defaultPartsStatusMainByLotteryGroup,
		RarityByRarityType:                   rarityByRarityType,
		RateByGroupAndLevel:                  rateByGroupAndLevel,
		PriceByGroupAndLevel:                 priceByGroupAndLevel,
		SellPriceByRarity:                    sellPriceByRarity,
		PartsStatusMainById:                  partsStatusMainById,
		SubStatusPool:                        subStatusPool,
		SubStatusUnlockLvls:                  subStatusUnlockLvls,
		FuncResolver:                         funcResolver,
	}, nil
}

// buildPartsStatusMain constructs the 36 PartsStatusMain definitions and
// groups them into sub-status lottery pools by tier (1-4).
// The data mirrors EntityMPartsStatusMainTable.json which is structured as
// 9 stat categories x 4 tiers. Tier within each category maps to the
// PartsStatusSubLotteryGroupId on the part definition.
func buildPartsStatusMain() (map[int32]PartsStatusMainDef, map[int32][]int32) {
	type statCat struct {
		kindType  int32
		calcType  int32
		initVals  [4]int32
		funcStart int32
	}
	cats := []statCat{
		{2, 1, [4]int32{50, 100, 150, 250}, 101},     // Attack flat
		{7, 1, [4]int32{50, 100, 150, 250}, 101},     // Vitality flat
		{2, 2, [4]int32{10, 30, 70, 120}, 105},       // Attack %
		{7, 2, [4]int32{10, 30, 70, 120}, 105},       // Vitality %
		{6, 2, [4]int32{10, 30, 70, 120}, 105},       // HP %
		{6, 1, [4]int32{600, 1200, 1800, 3000}, 109}, // HP flat
		{4, 1, [4]int32{10, 30, 70, 120}, 113},       // CritRatio
		{3, 1, [4]int32{20, 50, 80, 100}, 117},       // CritAttack
		{1, 1, [4]int32{10, 20, 30, 40}, 121},        // Agility
	}

	defs := make(map[int32]PartsStatusMainDef, 36)
	pool := map[int32][]int32{1: {}, 2: {}, 3: {}, 4: {}}
	id := int32(1)
	for _, c := range cats {
		for tier := 0; tier < 4; tier++ {
			defs[id] = PartsStatusMainDef{
				StatusKindType:            c.kindType,
				StatusCalculationType:     c.calcType,
				StatusChangeInitialValue:  c.initVals[tier],
				StatusNumericalFunctionId: c.funcStart + int32(tier),
			}
			pool[int32(tier+1)] = append(pool[int32(tier+1)], id)
			id++
		}
	}
	// Newer parts groups (PartsGroupId 401-490) use PartsStatusSubLotteryGroupId
	// 11/12 for rarities 10/20 instead of 1/2. Same stat pools — alias them.
	pool[11] = pool[1]
	pool[12] = pool[2]
	return defs, pool
}

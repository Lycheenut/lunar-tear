package masterdata

import (
	"fmt"
	"log"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

type GachaPoolItem struct {
	PossessionType int32
	PossessionId   int32
	RarityType     model.RarityType
	CharacterId    int32
}

type FeaturedSet struct {
	Costumes []GachaPoolItem
	Weapons  []GachaPoolItem
}

type BannerPool struct {
	CostumesByRarity map[int32][]GachaPoolItem
	WeaponsByRarity  map[int32][]GachaPoolItem
	Featured         []GachaPoolItem
}

type ShopFeaturedEntry struct {
	CostumeId int32
	WeaponId  int32
}

type CatalogTerm struct {
	TermId        int32
	StartDatetime int64
	Costumes      []GachaPoolItem
	Weapons       []GachaPoolItem
}

// StandardPoolTermId is the catalog term whose items form the cross-banner
// standard pool (term 1 holds the launch starter set).
const StandardPoolTermId int32 = 1

const (
	maxFeaturedDisplayItems = 13
	maxDisplayedCostumes    = 3
	maxDisplayedWeapons     = 2
)

type GachaCatalog struct {
	CostumesByRarity         map[int32][]GachaPoolItem
	WeaponsByRarity          map[int32][]GachaPoolItem
	StandardCostumesByRarity map[int32][]GachaPoolItem
	StandardWeaponsByRarity  map[int32][]GachaPoolItem
	Materials                []GachaPoolItem
	CostumeById              map[int32]GachaPoolItem
	WeaponById               map[int32]GachaPoolItem
	ConfigurableWeaponById   map[int32]GachaPoolItem // catalog root weapons with a supported Gacha rarity
	EligibleWeaponById       map[int32]GachaPoolItem // configurable weapons allowed to enter a configured pool
	CostumeWeaponMap         map[int32]int32         // costumeId -> paired weaponId
	CostumeByWeaponId        map[int32]GachaPoolItem // paired catalog costume selected by weaponId
	FeaturedByGacha          map[int32]FeaturedSet
	BannerPools              map[int32]*BannerPool
	ShopFeaturedByMedal      map[int32][]ShopFeaturedEntry // consumableId -> paired entries
	TermById                 map[int32]*CatalogTerm
	TermsByStartDatetime     map[int64][]*CatalogTerm
}

func LoadGachaBoxRewardKeys() (map[[2]int32]bool, error) {
	keys := map[[2]int32]bool{{int32(model.PossessionTypeFreeGem), 0}: true}
	materials, err := utils.ReadTable[EntityMMaterial]("m_material")
	if err != nil {
		return nil, fmt.Errorf("load material rewards: %w", err)
	}
	for _, item := range materials {
		keys[[2]int32{int32(model.PossessionTypeMaterial), item.MaterialId}] = true
	}
	weapons, err := utils.ReadTable[EntityMWeapon]("m_weapon")
	if err != nil {
		return nil, fmt.Errorf("load weapon rewards: %w", err)
	}
	for _, item := range weapons {
		keys[[2]int32{int32(model.PossessionTypeWeapon), item.WeaponId}] = true
	}
	companions, err := utils.ReadTable[EntityMCompanion]("m_companion")
	if err != nil {
		return nil, fmt.Errorf("load companion rewards: %w", err)
	}
	for _, item := range companions {
		keys[[2]int32{int32(model.PossessionTypeCompanion), item.CompanionId}] = true
	}
	consumables, err := utils.ReadTable[EntityMConsumableItem]("m_consumable_item")
	if err != nil {
		return nil, fmt.Errorf("load consumable rewards: %w", err)
	}
	for _, item := range consumables {
		keys[[2]int32{int32(model.PossessionTypeConsumableItem), item.ConsumableItemId}] = true
	}
	return keys, nil
}

func LoadGachaPool() (*GachaCatalog, error) {
	costumes, err := utils.ReadTable[EntityMCostume]("m_costume")
	if err != nil {
		return nil, fmt.Errorf("load costume table: %w", err)
	}
	weapons, err := utils.ReadTable[EntityMWeapon]("m_weapon")
	if err != nil {
		return nil, fmt.Errorf("load weapon table: %w", err)
	}
	catalogCostumes, err := utils.ReadTable[EntityMCatalogCostume]("m_catalog_costume")
	if err != nil {
		return nil, fmt.Errorf("load catalog costume table: %w", err)
	}
	catalogWeapons, err := utils.ReadTable[EntityMCatalogWeapon]("m_catalog_weapon")
	if err != nil {
		return nil, fmt.Errorf("load catalog weapon table: %w", err)
	}
	materials, err := utils.ReadTable[EntityMMaterial]("m_material")
	if err != nil {
		return nil, fmt.Errorf("load material table: %w", err)
	}
	evoGroupRows, err := utils.ReadTable[EntityMWeaponEvolutionGroup]("m_weapon_evolution_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon evolution group table: %w", err)
	}
	evolvedWeapons := buildEvolvedWeaponSet(evoGroupRows)

	terms, err := utils.ReadTable[EntityMCatalogTerm]("m_catalog_term")
	if err != nil {
		return nil, fmt.Errorf("load catalog term table: %w", err)
	}
	firstClearRewards, err := utils.ReadTable[EntityMQuestFirstClearRewardGroup]("m_quest_first_clear_reward_group")
	if err != nil {
		return nil, fmt.Errorf("load quest first clear reward group table: %w", err)
	}
	sceneGrants, err := utils.ReadTable[EntityMUserQuestSceneGrantPossession]("m_user_quest_scene_grant_possession")
	if err != nil {
		return nil, fmt.Errorf("load user quest scene grant possession table: %w", err)
	}
	missionRewardRows, err := utils.ReadTable[EntityMMissionReward]("m_mission_reward")
	if err != nil {
		return nil, fmt.Errorf("load mission reward table: %w", err)
	}

	questGrantedCostumes := make(map[int32]bool)
	questGrantedWeapons := make(map[int32]bool)
	collectGrant := func(possType, possId int32) {
		switch possType {
		case int32(model.PossessionTypeCostume):
			questGrantedCostumes[possId] = true
		case int32(model.PossessionTypeWeapon):
			questGrantedWeapons[possId] = true
		}
	}
	for _, r := range firstClearRewards {
		collectGrant(r.PossessionType, r.PossessionId)
	}
	for _, r := range sceneGrants {
		collectGrant(r.PossessionType, r.PossessionId)
	}
	for _, r := range missionRewardRows {
		collectGrant(r.PossessionType, r.PossessionId)
	}

	catalogCostumeSet := make(map[int32]bool, len(catalogCostumes))
	for _, c := range catalogCostumes {
		catalogCostumeSet[c.CostumeId] = true
	}
	catalogWeaponSet := make(map[int32]bool, len(catalogWeapons))
	for _, w := range catalogWeapons {
		catalogWeaponSet[w.WeaponId] = true
	}

	restrictedWeapons := make(map[int32]bool)
	for _, w := range weapons {
		if w.IsRestrictDiscard {
			restrictedWeapons[w.WeaponId] = true
		}
	}

	pool := &GachaCatalog{
		CostumesByRarity:       make(map[int32][]GachaPoolItem),
		WeaponsByRarity:        make(map[int32][]GachaPoolItem),
		CostumeById:            make(map[int32]GachaPoolItem),
		WeaponById:             make(map[int32]GachaPoolItem),
		ConfigurableWeaponById: make(map[int32]GachaPoolItem),
		EligibleWeaponById:     make(map[int32]GachaPoolItem),
		CostumeWeaponMap:       make(map[int32]int32),
		CostumeByWeaponId:      make(map[int32]GachaPoolItem),
		FeaturedByGacha:        make(map[int32]FeaturedSet),
		TermById:               make(map[int32]*CatalogTerm),
		TermsByStartDatetime:   make(map[int64][]*CatalogTerm),
	}
	for _, t := range terms {
		ct := &CatalogTerm{TermId: t.CatalogTermId, StartDatetime: t.StartDatetime}
		pool.TermById[t.CatalogTermId] = ct
		pool.TermsByStartDatetime[t.StartDatetime] = append(pool.TermsByStartDatetime[t.StartDatetime], ct)
	}

	questGrantedCostumeCount := 0
	for _, c := range costumes {
		if !catalogCostumeSet[c.CostumeId] {
			continue
		}
		if c.RarityType < model.RaritySRare {
			continue
		}
		if questGrantedCostumes[c.CostumeId] {
			questGrantedCostumeCount++
			continue
		}
		item := GachaPoolItem{
			PossessionType: int32(model.PossessionTypeCostume),
			PossessionId:   c.CostumeId,
			RarityType:     c.RarityType,
			CharacterId:    c.CharacterId,
		}
		pool.CostumesByRarity[c.RarityType] = append(pool.CostumesByRarity[c.RarityType], item)
		pool.CostumeById[c.CostumeId] = item
	}

	restrictedCount := 0
	questGrantedWeaponCount := 0
	evolvedFilteredCount := 0
	for _, w := range weapons {
		if !catalogWeaponSet[w.WeaponId] {
			continue
		}
		if evolvedWeapons[w.WeaponId] {
			evolvedFilteredCount++
			continue
		}
		item := GachaPoolItem{
			PossessionType: int32(model.PossessionTypeWeapon),
			PossessionId:   w.WeaponId,
			RarityType:     w.RarityType,
		}
		if w.RarityType == model.RarityRare || w.RarityType == model.RaritySRare || w.RarityType == model.RaritySSRare {
			pool.ConfigurableWeaponById[w.WeaponId] = item
		}
		if questGrantedWeapons[w.WeaponId] {
			questGrantedWeaponCount++
			continue
		}
		pool.WeaponById[w.WeaponId] = item
		if w.IsRestrictDiscard {
			restrictedCount++
			continue
		}
		pool.WeaponsByRarity[w.RarityType] = append(pool.WeaponsByRarity[w.RarityType], item)
		pool.EligibleWeaponById[w.WeaponId] = item
	}

	// Bucket catalog items into their terms (uses the post-filter CostumeById/WeaponById).
	for _, cc := range catalogCostumes {
		ct := pool.TermById[cc.CatalogTermId]
		if ct == nil {
			continue
		}
		if item, ok := pool.CostumeById[cc.CostumeId]; ok {
			ct.Costumes = append(ct.Costumes, item)
		}
	}
	for _, cw := range catalogWeapons {
		ct := pool.TermById[cw.CatalogTermId]
		if ct == nil || restrictedWeapons[cw.WeaponId] {
			continue
		}
		if item, ok := pool.WeaponById[cw.WeaponId]; ok {
			ct.Weapons = append(ct.Weapons, item)
		}
	}

	// Standard pool: items in term 1 (the launch starter set, same on every banner).
	pool.StandardCostumesByRarity = make(map[int32][]GachaPoolItem)
	pool.StandardWeaponsByRarity = make(map[int32][]GachaPoolItem)
	if std := pool.TermById[StandardPoolTermId]; std != nil {
		for _, c := range std.Costumes {
			pool.StandardCostumesByRarity[c.RarityType] = append(pool.StandardCostumesByRarity[c.RarityType], c)
		}
		for _, w := range std.Weapons {
			pool.StandardWeaponsByRarity[w.RarityType] = append(pool.StandardWeaponsByRarity[w.RarityType], w)
		}
	}
	stdCos, stdWea := 0, 0
	for _, items := range pool.StandardCostumesByRarity {
		stdCos += len(items)
	}
	for _, items := range pool.StandardWeaponsByRarity {
		stdWea += len(items)
	}

	log.Printf("[GachaPool] catalog terms: %d, standard pool: %d costumes + %d weapons (term %d)",
		len(pool.TermById), stdCos, stdWea, StandardPoolTermId)
	log.Printf("[GachaPool] pool excludes %d evolved, %d quest-granted costumes, %d quest-granted weapons, %d restricted weapons",
		evolvedFilteredCount, questGrantedCostumeCount, questGrantedWeaponCount, restrictedCount)

	for costumeId := range pool.CostumeById {
		if wid, ok := costumeWeaponPairings[costumeId]; ok {
			pool.CostumeWeaponMap[costumeId] = wid
			if _, eligible := pool.EligibleWeaponById[wid]; eligible {
				if previous, exists := pool.CostumeByWeaponId[wid]; !exists || costumeId < previous.PossessionId {
					pool.CostumeByWeaponId[wid] = pool.CostumeById[costumeId]
				}
			}
		}
	}
	log.Printf("[GachaPool] costume-weapon pairing: %d entries from lookup table", len(pool.CostumeWeaponMap))

	for _, m := range materials {
		pool.Materials = append(pool.Materials, GachaPoolItem{
			PossessionType: int32(model.PossessionTypeMaterial),
			PossessionId:   m.MaterialId,
			RarityType:     m.RarityType,
		})
	}

	return pool, nil
}

func (pool *GachaCatalog) BuildShopFeatured(shop *ShopCatalog) {
	pool.ShopFeaturedByMedal = make(map[int32][]ShopFeaturedEntry)
	for _, cells := range shop.ExchangeShopCells {
		consumableId := shop.Items[cells[0].ShopItemId].PriceId

		var entries []ShopFeaturedEntry
		for _, cell := range cells {
			contents := shop.Contents[cell.ShopItemId]
			var costumeId, weaponId int32
			for _, c := range contents {
				switch c.PossessionType {
				case int32(model.PossessionTypeCostume):
					costumeId = c.PossessionId
				case int32(model.PossessionTypeWeapon):
					weaponId = c.PossessionId
				}
			}
			if costumeId == 0 && weaponId == 0 {
				continue
			}
			entries = append(entries, ShopFeaturedEntry{CostumeId: costumeId, WeaponId: weaponId})
		}
		if len(entries) > 0 {
			pool.ShopFeaturedByMedal[consumableId] = entries
		}
	}
	log.Printf("[GachaPool] shop featured: %d consumables", len(pool.ShopFeaturedByMedal))
}

func (pool *GachaCatalog) PruneUnpairedCostumes() {
	pruned := 0
	for costumeId := range pool.CostumeById {
		if _, ok := pool.CostumeWeaponMap[costumeId]; !ok {
			delete(pool.CostumeById, costumeId)
			pruned++
		}
	}
	for rarity, items := range pool.CostumesByRarity {
		filtered := items[:0]
		for _, item := range items {
			if _, ok := pool.CostumeWeaponMap[item.PossessionId]; ok {
				filtered = append(filtered, item)
			}
		}
		pool.CostumesByRarity[rarity] = filtered
	}
	log.Printf("[GachaPool] pruned %d unpaired costumes", pruned)
}

// BuildFeaturedFromTerms derives a featured set for each non-chapter banner.
// Medal exchange contents are the authoritative per-banner targets. Catalog
// terms are only a fallback for banners without a matching exchange shop.
func (pool *GachaCatalog) BuildFeaturedFromTerms(entries []store.GachaCatalogEntry) {
	matched := 0
	fromShop := 0
	fromTerms := 0
	gachaEligible := 0
	for _, entry := range entries {
		if entry.GachaLabelType == model.GachaLabelChapter {
			continue
		}
		gachaEligible++

		var costumes, weapons []GachaPoolItem
		usedShop := false
		if entry.MedalConsumableItemId != 0 {
			if shopEntries, ok := pool.ShopFeaturedByMedal[entry.MedalConsumableItemId]; ok {
				costumes, weapons = pool.featuredFromShop(shopEntries)
				if len(costumes) > 0 || len(weapons) > 0 {
					usedShop = true
					fromShop++
				}
			}
		}
		if len(costumes) == 0 && len(weapons) == 0 {
			costumes, weapons = pool.unionTermFeatured(entry.StartDatetime)
			if len(costumes) > 0 || len(weapons) > 0 {
				fromTerms++
			}
		}
		if len(costumes) == 0 && len(weapons) == 0 {
			continue
		}
		// Exchange shop cells already carry the authoritative display order.
		if !usedShop {
			sort.Slice(costumes, func(i, j int) bool { return costumes[i].PossessionId < costumes[j].PossessionId })
			sort.Slice(weapons, func(i, j int) bool { return weapons[i].PossessionId < weapons[j].PossessionId })
		}
		costumes, weapons = pool.selectFeaturedTargets(entry, costumes, weapons)

		pool.FeaturedByGacha[entry.GachaId] = FeaturedSet{Costumes: costumes, Weapons: weapons}
		matched++
	}
	log.Printf("[GachaPool] featured per banner: %d/%d (%d shop + %d term fallback)",
		matched, gachaEligible, fromShop, fromTerms)
}

func (pool *GachaCatalog) selectFeaturedTargets(entry store.GachaCatalogEntry, costumes, weapons []GachaPoolItem) ([]GachaPoolItem, []GachaPoolItem) {
	maxRarity := int32(0)
	for _, item := range costumes {
		maxRarity = max(maxRarity, item.RarityType)
	}
	for _, item := range weapons {
		maxRarity = max(maxRarity, item.RarityType)
	}
	filterRarity := func(items []GachaPoolItem) []GachaPoolItem {
		filtered := make([]GachaPoolItem, 0, len(items))
		for _, item := range items {
			if item.RarityType == maxRarity {
				filtered = append(filtered, item)
			}
		}
		return filtered
	}
	costumes = filterRarity(costumes)
	weapons = filterRarity(weapons)

	if entry.GachaModeType == model.GachaModeStepup && len(costumes) > 0 {
		costume := costumes[0]
		weapons = nil
		if weapon, ok := pool.WeaponById[pool.CostumeWeaponMap[costume.PossessionId]]; ok {
			weapons = append(weapons, weapon)
		}
		return []GachaPoolItem{costume}, weapons
	}
	if len(costumes)+len(weapons) > maxFeaturedDisplayItems {
		costumes = costumes[:min(maxDisplayedCostumes, len(costumes))]
		weapons = weapons[:min(maxDisplayedWeapons, len(weapons))]
	}
	return costumes, weapons
}

func (pool *GachaCatalog) unionTermFeatured(startDatetime int64) (costumes, weapons []GachaPoolItem) {
	coTerms := pool.TermsByStartDatetime[startDatetime]
	if len(coTerms) == 0 {
		return nil, nil
	}
	seenCostume := make(map[int32]bool)
	seenWeapon := make(map[int32]bool)
	for _, t := range coTerms {
		if t.TermId == StandardPoolTermId {
			continue
		}
		for _, c := range t.Costumes {
			if c.RarityType < model.RaritySRare || seenCostume[c.PossessionId] {
				continue
			}
			costumes = append(costumes, c)
			seenCostume[c.PossessionId] = true
		}
		for _, w := range t.Weapons {
			if w.RarityType < model.RaritySRare || seenWeapon[w.PossessionId] {
				continue
			}
			weapons = append(weapons, w)
			seenWeapon[w.PossessionId] = true
		}
	}
	return costumes, pool.excludeCostumeBonusWeapons(costumes, weapons)
}

func (pool *GachaCatalog) excludeCostumeBonusWeapons(costumes, weapons []GachaPoolItem) []GachaPoolItem {
	linked := make(map[int32]bool, len(costumes))
	for _, costume := range costumes {
		if weaponId := pool.CostumeWeaponMap[costume.PossessionId]; weaponId != 0 {
			linked[weaponId] = true
		}
	}
	filtered := make([]GachaPoolItem, 0, len(weapons))
	for _, weapon := range weapons {
		if !linked[weapon.PossessionId] {
			filtered = append(filtered, weapon)
		}
	}
	return filtered
}

func (pool *GachaCatalog) featuredFromShop(shopEntries []ShopFeaturedEntry) (costumes, weapons []GachaPoolItem) {
	seenCostume := make(map[int32]bool)
	seenWeapon := make(map[int32]bool)
	linkedWeapons := make(map[int32]bool)
	for _, se := range shopEntries {
		if se.CostumeId == 0 || seenCostume[se.CostumeId] {
			continue
		}
		if item, ok := pool.CostumeById[se.CostumeId]; ok && item.RarityType >= model.RaritySRare {
			costumes = append(costumes, item)
			seenCostume[se.CostumeId] = true
			linkedWeapons[se.WeaponId] = true
		}
	}
	for _, se := range shopEntries {
		if se.WeaponId == 0 || linkedWeapons[se.WeaponId] || seenWeapon[se.WeaponId] {
			continue
		}
		if item, ok := pool.WeaponById[se.WeaponId]; ok && item.RarityType >= model.RaritySRare {
			weapons = append(weapons, item)
			seenWeapon[se.WeaponId] = true
		}
	}
	return costumes, weapons
}

func (pool *GachaCatalog) BuildBannerPools(entries []store.GachaCatalogEntry) {
	pool.BannerPools = make(map[int32]*BannerPool)
	for _, entry := range entries {
		fs, hasFeatured := pool.FeaturedByGacha[entry.GachaId]
		var allFeatured []GachaPoolItem
		if hasFeatured {
			allFeatured = uniqueGachaItems(fs.Costumes, fs.Weapons)
			normalized := FeaturedSet{}
			for _, item := range allFeatured {
				if item.PossessionType == int32(model.PossessionTypeCostume) {
					normalized.Costumes = append(normalized.Costumes, item)
				} else if item.PossessionType == int32(model.PossessionTypeWeapon) {
					normalized.Weapons = append(normalized.Weapons, item)
				}
			}
			pool.FeaturedByGacha[entry.GachaId] = normalized
		}
		featuredCostumes := make(map[int32]bool)
		featuredWeapons := make(map[int32]bool)
		for _, item := range allFeatured {
			if item.PossessionType == int32(model.PossessionTypeCostume) {
				featuredCostumes[item.PossessionId] = true
			} else if item.PossessionType == int32(model.PossessionTypeWeapon) {
				featuredWeapons[item.PossessionId] = true
			}
		}
		pool.BannerPools[entry.GachaId] = &BannerPool{
			CostumesByRarity: cloneRarityMapExcluding(pool.StandardCostumesByRarity, featuredCostumes),
			WeaponsByRarity:  cloneRarityMapExcluding(pool.StandardWeaponsByRarity, featuredWeapons),
			Featured:         allFeatured,
		}
	}
	log.Printf("[GachaPool] banner pools: %d banners built from standard fallback + separate per-banner featured", len(pool.BannerPools))
}

func uniqueGachaItems(groups ...[]GachaPoolItem) []GachaPoolItem {
	seen := make(map[[2]int32]bool)
	var result []GachaPoolItem
	for _, items := range groups {
		for _, item := range items {
			key := [2]int32{item.PossessionType, item.PossessionId}
			if seen[key] {
				continue
			}
			seen[key] = true
			result = append(result, item)
		}
	}
	return result
}

func cloneRarityMapExcluding(src map[int32][]GachaPoolItem, excluded map[int32]bool) map[int32][]GachaPoolItem {
	dst := make(map[int32][]GachaPoolItem, len(src))
	seen := make(map[int32]bool)
	for k, v := range src {
		for _, item := range v {
			if excluded[item.PossessionId] || seen[item.PossessionId] {
				continue
			}
			seen[item.PossessionId] = true
			dst[k] = append(dst[k], item)
		}
	}
	return dst
}

func buildEvolvedWeaponSet(rows []EntityMWeaponEvolutionGroup) map[int32]bool {
	grouped := make(map[int32][]EntityMWeaponEvolutionGroup)
	for _, r := range rows {
		grouped[r.WeaponEvolutionGroupId] = append(grouped[r.WeaponEvolutionGroupId], r)
	}
	evolved := make(map[int32]bool)
	for _, chain := range grouped {
		sort.Slice(chain, func(i, j int) bool {
			return chain[i].EvolutionOrder < chain[j].EvolutionOrder
		})
		for i := 1; i < len(chain); i++ {
			evolved[chain[i].WeaponId] = true
		}
	}
	return evolved
}

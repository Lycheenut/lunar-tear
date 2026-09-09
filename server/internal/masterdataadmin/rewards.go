package masterdataadmin

import (
	"fmt"
	"path"
	"sort"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
)

type RewardReference struct {
	PossessionType  int32             `json:"possessionType"`
	PossessionId    int32             `json:"possessionId"`
	Names           map[string]string `json:"names,omitempty"`
	IconPath        string            `json:"iconPath"`
	MaterialType    int32             `json:"materialType,omitempty"`
	RarityType      int32             `json:"rarityType,omitempty"`
	WeaponType      int32             `json:"weaponType,omitempty"`
	AttributeType   int32             `json:"attributeType,omitempty"`
	ConsumableType  int32             `json:"consumableType,omitempty"`
	GrantsCharacter bool              `json:"grantsCharacter,omitempty"`
	CostumeNames    map[string]string `json:"costumeNames,omitempty"`
	CostumeIconPath string            `json:"costumeIconPath,omitempty"`
}

type RewardReferenceCatalog struct {
	DefaultType        string            `json:"defaultType"`
	Materials          []RewardReference `json:"materials"`
	Weapons            []RewardReference `json:"weapons"`
	Costumes           []RewardReference `json:"costumes"`
	Companions         []RewardReference `json:"companions"`
	Parts              []RewardReference `json:"parts"`
	EnhancedCostumes   []RewardReference `json:"enhancedCostumes"`
	EnhancedWeapons    []RewardReference `json:"enhancedWeapons"`
	EnhancedCompanions []RewardReference `json:"enhancedCompanions"`
	EnhancedParts      []RewardReference `json:"enhancedParts"`
	ConsumableItems    []RewardReference `json:"consumableItems"`
	ImportantItems     []RewardReference `json:"importantItems"`
	Thoughts           []RewardReference `json:"thoughts"`
	MissionPassPoints  []RewardReference `json:"missionPassPoints"`
	PremiumItems       []RewardReference `json:"premiumItems"`
	PaidGems           []RewardReference `json:"paidGems"`
	FreeGems           []RewardReference `json:"freeGems"`
}

func LoadRewardReferenceCatalog(
	masterDataPath string,
	pool *masterdata.GachaCatalog,
	costumeCatalog *masterdata.CostumeCatalog,
) (*RewardReferenceCatalog, error) {
	file, err := memorydb.OpenFile(masterDataPath)
	if err != nil {
		return nil, err
	}
	resolver := newTitleResolver(file, loadLocalizationIndex(masterDataPath))
	result := &RewardReferenceCatalog{DefaultType: "material"}

	for _, row := range readRows(file, "m_material") {
		if reference, ok := materialRewardReference(row, resolver); ok {
			result.Materials = append(result.Materials, reference)
		}
	}

	characterCostumes := make(map[int32]masterdata.EntityMCostume)
	if pool != nil && costumeCatalog != nil {
		for costumeID, weaponID := range pool.CostumeWeaponMap {
			costume, ok := costumeCatalog.Costumes[costumeID]
			if !ok {
				continue
			}
			if previous, exists := characterCostumes[weaponID]; !exists || costume.CostumeId < previous.CostumeId {
				characterCostumes[weaponID] = costume
			}
		}
	}
	for _, row := range readRows(file, "m_weapon") {
		if reference, ok := weaponRewardReference(row, resolver, characterCostumes); ok {
			result.Weapons = append(result.Weapons, reference)
		}
	}
	for _, row := range readRows(file, "m_costume") {
		if reference, ok := costumeRewardReference(row, resolver); ok {
			result.Costumes = append(result.Costumes, reference)
		}
	}
	for _, row := range readRows(file, "m_companion") {
		if reference, ok := companionRewardReference(row, resolver); ok {
			result.Companions = append(result.Companions, reference)
		}
	}
	partsGroupAssets := make(map[int64]int64)
	for _, row := range readRows(file, "m_parts_group") {
		groupID, groupOK := integerAt(row, 0)
		assetID, assetOK := integerAt(row, 3)
		if groupOK && assetOK {
			partsGroupAssets[groupID] = assetID
		}
	}
	for _, row := range readRows(file, "m_parts") {
		if reference, ok := partsRewardReference(row, resolver, partsGroupAssets); ok {
			result.Parts = append(result.Parts, reference)
		}
	}
	for _, row := range readRows(file, "m_consumable_item") {
		if reference, ok := consumableRewardReference(row, resolver); ok {
			result.ConsumableItems = append(result.ConsumableItems, reference)
		}
	}
	for _, row := range readRows(file, "m_important_item") {
		if reference, ok := importantItemRewardReference(row, resolver); ok {
			result.ImportantItems = append(result.ImportantItems, reference)
		}
	}
	for _, row := range readRows(file, "m_thought") {
		if reference, ok := thoughtRewardReference(row, resolver); ok {
			result.Thoughts = append(result.Thoughts, reference)
		}
	}
	result.EnhancedCostumes = enhancedRewardReferences(readRows(file, "m_costume_enhanced"), result.Costumes, model.PossessionTypeCostumeEnhanced)
	result.EnhancedWeapons = enhancedRewardReferences(readRows(file, "m_weapon_enhanced"), result.Weapons, model.PossessionTypeWeaponEnhanced)
	result.EnhancedCompanions = enhancedRewardReferences(readRows(file, "m_companion_enhanced"), result.Companions, model.PossessionTypeCompanionEnhanced)
	result.EnhancedParts = enhancedRewardReferences(readRows(file, "m_parts_enhanced"), result.Parts, model.PossessionTypePartsEnhanced)
	// These tables identify the pass or entitlement, without localized names or icon assets.
	for _, row := range readRows(file, "m_mission_pass") {
		if id, ok := integerAt(row, 0); ok {
			result.MissionPassPoints = append(result.MissionPassPoints, RewardReference{
				PossessionType: int32(model.PossessionTypeMissionPassPoint), PossessionId: int32(id),
			})
		}
	}
	for _, row := range readRows(file, "m_premium_item") {
		if id, ok := integerAt(row, 0); ok {
			result.PremiumItems = append(result.PremiumItems, RewardReference{
				PossessionType: int32(model.PossessionTypePremiumItem), PossessionId: int32(id),
			})
		}
	}
	result.PaidGems = []RewardReference{{
		PossessionType: int32(model.PossessionTypePaidGem),
		Names:          resolver.byKey("gem.name"),
		IconPath:       path.Join("gem", "gem", "gem_standard.png"),
	}}
	result.FreeGems = []RewardReference{{
		PossessionType: int32(model.PossessionTypeFreeGem),
		Names:          resolver.byKey("gem.name"),
		IconPath:       path.Join("gem", "gem", "gem_standard.png"),
	}}

	for _, references := range [][]RewardReference{
		result.Materials, result.Weapons, result.Costumes, result.Companions, result.Parts,
		result.EnhancedCostumes, result.EnhancedWeapons, result.EnhancedCompanions, result.EnhancedParts,
		result.ConsumableItems, result.ImportantItems, result.Thoughts, result.MissionPassPoints, result.PremiumItems,
	} {
		sort.Slice(references, func(i, j int) bool {
			return references[i].PossessionId < references[j].PossessionId
		})
	}
	return result, nil
}

func costumeRewardReference(row []interface{}, resolver *titleResolver) (RewardReference, bool) {
	id, idOK := integerAt(row, 0)
	skeletonID, skeletonOK := integerAt(row, 4)
	variationID, variationOK := integerAt(row, 5)
	weaponType, weaponOK := integerAt(row, 6)
	rarityType, rarityOK := integerAt(row, 7)
	if !idOK || !skeletonOK || !variationOK || !weaponOK || !rarityOK {
		return RewardReference{}, false
	}
	costume := masterdata.EntityMCostume{
		CostumeId: int32(id), ActorSkeletonId: int32(skeletonID), AssetVariationId: int32(variationID),
	}
	return RewardReference{
		PossessionType: int32(model.PossessionTypeCostume), PossessionId: int32(id),
		Names: costumeTitles(resolver, costume), IconPath: costumeIconPath(costume),
		WeaponType: int32(weaponType), RarityType: int32(rarityType),
	}, true
}

func enhancedRewardReferences(rows [][]interface{}, base []RewardReference, possessionType model.PossessionType) []RewardReference {
	byID := make(map[int32]RewardReference, len(base))
	for _, reference := range base {
		byID[reference.PossessionId] = reference
	}
	var result []RewardReference
	for _, row := range rows {
		id, idOK := integerAt(row, 0)
		baseID, baseOK := integerAt(row, 1)
		reference, exists := byID[int32(baseID)]
		if !idOK || !baseOK || !exists {
			continue
		}
		// Enhanced possession IDs belong to their own table, even when they match the base ID.
		reference.PossessionType = int32(possessionType)
		reference.PossessionId = int32(id)
		result = append(result, reference)
	}
	return result
}

func thoughtRewardReference(row []interface{}, resolver *titleResolver) (RewardReference, bool) {
	id, idOK := integerAt(row, 0)
	rarityType, rarityOK := integerAt(row, 1)
	assetID, assetOK := integerAt(row, 4)
	if !idOK || !rarityOK || !assetOK {
		return RewardReference{}, false
	}
	assetName := fmt.Sprintf("thought%06d", assetID)
	return RewardReference{
		PossessionType: int32(model.PossessionTypeThought), PossessionId: int32(id),
		Names:    resolver.byKey(fmt.Sprintf("thought.name.%06d", assetID)),
		IconPath: path.Join("thought", assetName, assetName+"_standard.png"), RarityType: int32(rarityType),
	}, true
}

func materialRewardReference(row []interface{}, resolver *titleResolver) (RewardReference, bool) {
	id, idOK := integerAt(row, 0)
	materialType, typeOK := integerAt(row, 1)
	rarityType, rarityOK := integerAt(row, 2)
	categoryID, categoryOK := integerAt(row, 8)
	variationID, variationOK := integerAt(row, 9)
	if !idOK || !typeOK || !rarityOK || !categoryOK || !variationOK {
		return RewardReference{}, false
	}
	assetName, _ := stringAt(row, 7)
	if assetName == "" {
		assetName = fmt.Sprintf("material%03d%03d", categoryID, variationID)
	}
	return RewardReference{
		PossessionType: int32(model.PossessionTypeMaterial),
		PossessionId:   int32(id),
		Names:          resolver.byKey(fmt.Sprintf("material.name.%03d%03d", categoryID, variationID)),
		IconPath:       path.Join("material", assetName, assetName+"_standard.png"),
		MaterialType:   int32(materialType),
		RarityType:     int32(rarityType),
	}, true
}

func weaponRewardReference(
	row []interface{},
	resolver *titleResolver,
	characterCostumes map[int32]masterdata.EntityMCostume,
) (RewardReference, bool) {
	id, idOK := integerAt(row, 0)
	categoryType, categoryOK := integerAt(row, 1)
	weaponType, weaponTypeOK := integerAt(row, 2)
	variationID, variationOK := integerAt(row, 3)
	rarityType, rarityOK := integerAt(row, 4)
	attributeType, attributeOK := integerAt(row, 5)
	if !idOK || !categoryOK || !weaponTypeOK || !variationOK || !rarityOK || !attributeOK {
		return RewardReference{}, false
	}
	weapon := masterdata.EntityMWeapon{
		WeaponId:           int32(id),
		WeaponCategoryType: int32(categoryType),
		WeaponType:         int32(weaponType),
		AssetVariationId:   int32(variationID),
		RarityType:         int32(rarityType),
		AttributeType:      int32(attributeType),
	}
	reference := RewardReference{
		PossessionType: int32(model.PossessionTypeWeapon),
		PossessionId:   int32(id),
		Names:          weaponTitles(resolver, weapon),
		IconPath:       rewardWeaponIconPath(weapon),
		RarityType:     int32(rarityType),
		WeaponType:     int32(weaponType),
		AttributeType:  int32(attributeType),
	}
	if costume, ok := characterCostumes[int32(id)]; ok {
		reference.GrantsCharacter = true
		reference.CostumeNames = costumeTitles(resolver, costume)
		reference.CostumeIconPath = costumeIconPath(costume)
	}
	return reference, true
}

func companionRewardReference(row []interface{}, resolver *titleResolver) (RewardReference, bool) {
	id, idOK := integerAt(row, 0)
	attributeType, attributeOK := integerAt(row, 1)
	actorSkeletonID, skeletonOK := integerAt(row, 8)
	variationID, variationOK := integerAt(row, 9)
	if !idOK || !attributeOK || !skeletonOK || !variationOK {
		return RewardReference{}, false
	}
	assetName := fmt.Sprintf("cm%03d%03d", actorSkeletonID, variationID)
	return RewardReference{
		PossessionType: int32(model.PossessionTypeCompanion),
		PossessionId:   int32(id),
		Names:          resolver.byKey("companion.name." + assetName),
		IconPath:       path.Join("companion", assetName, assetName+"_standard.png"),
		AttributeType:  int32(attributeType),
	}, true
}

func partsRewardReference(row []interface{}, resolver *titleResolver, groupAssets map[int64]int64) (RewardReference, bool) {
	id, idOK := integerAt(row, 0)
	rarityType, rarityOK := integerAt(row, 1)
	groupID, groupOK := integerAt(row, 2)
	assetID, assetOK := groupAssets[groupID]
	if !idOK || !rarityOK || !groupOK || !assetOK || assetID <= 0 {
		return RewardReference{}, false
	}
	assetName := fmt.Sprintf("memory%03d", assetID)
	return RewardReference{
		PossessionType: int32(model.PossessionTypeParts),
		PossessionId:   int32(id),
		Names:          resolver.byKey(fmt.Sprintf("parts.group.name.%d", groupID)),
		IconPath:       path.Join("memory", assetName, assetName+"_standard.png"),
		RarityType:     int32(rarityType),
	}, true
}

func consumableRewardReference(row []interface{}, resolver *titleResolver) (RewardReference, bool) {
	id, idOK := integerAt(row, 0)
	consumableType, typeOK := integerAt(row, 1)
	categoryID, categoryOK := integerAt(row, 6)
	variationID, variationOK := integerAt(row, 7)
	if !idOK || !typeOK || !categoryOK || !variationOK {
		return RewardReference{}, false
	}
	assetName, _ := stringAt(row, 5)
	if assetName == "" {
		assetName = fmt.Sprintf("consumable%03d%03d", categoryID, variationID)
	}
	return RewardReference{
		PossessionType: int32(model.PossessionTypeConsumableItem),
		PossessionId:   int32(id),
		Names:          resolver.byKey(fmt.Sprintf("consumable_item.name.%03d%03d", categoryID, variationID)),
		IconPath:       path.Join("consumable_item", assetName, assetName+"_standard.png"),
		ConsumableType: int32(consumableType),
	}, true
}

func importantItemRewardReference(row []interface{}, resolver *titleResolver) (RewardReference, bool) {
	id, idOK := integerAt(row, 0)
	nameTextID, nameOK := integerAt(row, 1)
	categoryID, categoryOK := integerAt(row, 4)
	variationID, variationOK := integerAt(row, 5)
	if !idOK || !nameOK || !categoryOK || !variationOK {
		return RewardReference{}, false
	}
	assetName := fmt.Sprintf("important%03d%03d", categoryID, variationID)
	return RewardReference{
		PossessionType: int32(model.PossessionTypeImportantItem),
		PossessionId:   int32(id),
		Names:          resolver.byKey(fmt.Sprintf("important_item.name.%d", nameTextID)),
		IconPath:       path.Join("important_item", assetName, assetName+"_standard.png"),
	}, true
}

func rewardWeaponAssetName(weapon masterdata.EntityMWeapon) string {
	prefix := "wp"
	if weapon.WeaponCategoryType == 2 {
		prefix = "mw"
	}
	return fmt.Sprintf("%s%03d%03d", prefix, weapon.WeaponType, weapon.AssetVariationId)
}

func rewardWeaponIconPath(weapon masterdata.EntityMWeapon) string {
	assetName := rewardWeaponAssetName(weapon)
	return path.Join("weapon", assetName, assetName+"_standard.png")
}

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
}

type RewardReferenceCatalog struct {
	DefaultType     string            `json:"defaultType"`
	Materials       []RewardReference `json:"materials"`
	Weapons         []RewardReference `json:"weapons"`
	Companions      []RewardReference `json:"companions"`
	ConsumableItems []RewardReference `json:"consumableItems"`
	FreeGems        []RewardReference `json:"freeGems"`
}

func LoadRewardReferenceCatalog(masterDataPath string, pool *masterdata.GachaCatalog) (*RewardReferenceCatalog, error) {
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

	characterWeapons := make(map[int32]bool)
	if pool != nil {
		for _, weaponID := range pool.CostumeWeaponMap {
			characterWeapons[weaponID] = true
		}
	}
	for _, row := range readRows(file, "m_weapon") {
		if reference, ok := weaponRewardReference(row, resolver, characterWeapons); ok {
			result.Weapons = append(result.Weapons, reference)
		}
	}
	for _, row := range readRows(file, "m_companion") {
		if reference, ok := companionRewardReference(row, resolver); ok {
			result.Companions = append(result.Companions, reference)
		}
	}
	for _, row := range readRows(file, "m_consumable_item") {
		if reference, ok := consumableRewardReference(row, resolver); ok {
			result.ConsumableItems = append(result.ConsumableItems, reference)
		}
	}
	result.FreeGems = []RewardReference{{
		PossessionType: int32(model.PossessionTypeFreeGem),
		Names:          resolver.byKey("gem.name"),
		IconPath:       path.Join("gem", "gem", "gem_standard.png"),
	}}

	sort.Slice(result.Materials, func(i, j int) bool {
		return result.Materials[i].PossessionId < result.Materials[j].PossessionId
	})
	sort.Slice(result.Weapons, func(i, j int) bool {
		return result.Weapons[i].PossessionId < result.Weapons[j].PossessionId
	})
	sort.Slice(result.Companions, func(i, j int) bool {
		return result.Companions[i].PossessionId < result.Companions[j].PossessionId
	})
	sort.Slice(result.ConsumableItems, func(i, j int) bool {
		return result.ConsumableItems[i].PossessionId < result.ConsumableItems[j].PossessionId
	})
	return result, nil
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

func weaponRewardReference(row []interface{}, resolver *titleResolver, characterWeapons map[int32]bool) (RewardReference, bool) {
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
	assetName := rewardWeaponAssetName(weapon)
	return RewardReference{
		PossessionType:  int32(model.PossessionTypeWeapon),
		PossessionId:    int32(id),
		Names:           weaponTitles(resolver, weapon),
		IconPath:        path.Join("weapon", assetName, assetName+"_standard.png"),
		RarityType:      int32(rarityType),
		WeaponType:      int32(weaponType),
		AttributeType:   int32(attributeType),
		GrantsCharacter: characterWeapons[int32(id)],
	}, true
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

func rewardWeaponAssetName(weapon masterdata.EntityMWeapon) string {
	prefix := "wp"
	if weapon.WeaponCategoryType == 2 {
		prefix = "mw"
	}
	return fmt.Sprintf("%s%03d%03d", prefix, weapon.WeaponType, weapon.AssetVariationId)
}

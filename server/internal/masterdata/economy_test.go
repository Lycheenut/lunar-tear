package masterdata

import "testing"

func TestCostumeAwakenCostsUseConfiguredStep(t *testing.T) {
	catalog := &CostumeCatalog{
		AwakenPriceByGroup: map[int32]EntityMCostumeAwakenPriceGroup{
			1: {AwakenStepLowerLimit: 1, Gold: 100},
		},
		AwakenStepByGroup: map[int32]EntityMCostumeAwakenStepMaterialGroup{
			2: {AwakenStepLowerLimit: 1, CostumeAwakenMaterialGroupId: 10},
		},
		AwakenMaterialsByGroup: map[int32][]MaterialOption{
			10: {{MaterialId: 1000, Count: 1}},
		},
	}

	if gold, ok := catalog.AwakenGold(1, 3); !ok || gold != 100 {
		t.Fatalf("step 3 gold = (%d,%v), want (100,true)", gold, ok)
	}
	options := catalog.AwakenMaterialOptions(2, 3)
	if len(options) != 1 || options[0].MaterialId != 1000 || options[0].Count != 1 {
		t.Fatalf("step 3 material options = %+v", options)
	}
}

func TestWeaponLimitBreakMaterialOptionsUseSpecificAndRarityAlternatives(t *testing.T) {
	catalog := &WeaponCatalog{
		Weapons: map[int32]EntityMWeapon{
			1: {WeaponId: 1, RarityType: 40, WeaponSpecificLimitBreakMaterialGroupId: 7},
		},
		SpecificLimitBreakByGroup: map[int32][]EntityMWeaponSpecificLimitBreakMaterialGroup{
			7: {
				{LimitBreakCountLowerLimit: 0, MaterialId: 100, Count: 10},
				{LimitBreakCountLowerLimit: 2, MaterialId: 200, Count: 5},
			},
		},
		RarityLimitBreakByRarity: map[int32]MaterialOption{
			40: {MaterialId: 300, Count: 1},
		},
	}

	options := catalog.LimitBreakMaterialOptions(1, 2)
	if len(options) != 2 || options[0] != (MaterialOption{MaterialId: 200, Count: 5}) || options[1] != (MaterialOption{MaterialId: 300, Count: 1}) {
		t.Fatalf("limit break options = %+v", options)
	}
}

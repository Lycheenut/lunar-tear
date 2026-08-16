package masterdataadmin

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
)

func TestRewardReferencesResolveNamesIconsAndFilters(t *testing.T) {
	resolver := &titleResolver{texts: localizationIndex{
		"en": {
			"material.name.100001":        "Small Weapon Enhancement",
			"weapon.name.wp001002.1":      "Nameless Blade",
			"costume.name.ch003004":       "Abstract Hunter",
			"companion.name.cm001002":     "Bear: Precious",
			"consumable_item.name.110004": "Gold Automata Medal",
		},
		"ja": {},
		"ko": {},
	}}

	material, ok := materialRewardReference([]interface{}{
		int32(5001), int32(10), int32(2), int32(0), int32(0), int32(100), int32(5),
		"material100001", int32(100), int32(1), int32(0),
	}, resolver)
	if !ok {
		t.Fatal("material reference was not built")
	}
	if material.Names["en"] != "Small Weapon Enhancement" || material.IconPath != "material/material100001/material100001_standard.png" || material.MaterialType != 10 {
		t.Fatalf("unexpected material reference: %+v", material)
	}

	weapon, ok := weaponRewardReference([]interface{}{
		int32(6001), int32(1), int32(1), int32(2), int32(4), int32(3), false,
	}, resolver, map[int32]masterdata.EntityMCostume{
		6001: {CostumeId: 9001, ActorSkeletonId: 3, AssetVariationId: 4},
	})
	if !ok {
		t.Fatal("weapon reference was not built")
	}
	if weapon.Names["en"] != "Nameless Blade" || weapon.IconPath != "weapon/wp001002/wp001002_standard.png" ||
		!weapon.GrantsCharacter || weapon.CostumeNames["en"] != "Abstract Hunter" ||
		weapon.CostumeIconPath != "costume/ch003004/ch003004_standard.png" {
		t.Fatalf("unexpected weapon reference: %+v", weapon)
	}

	companion, ok := companionRewardReference([]interface{}{
		int32(7001), int32(2), int32(1), int32(0), int32(0), int32(0), int32(0), int32(0), int32(1), int32(2),
	}, resolver)
	if !ok || companion.Names["en"] != "Bear: Precious" || companion.IconPath != "companion/cm001002/cm001002_standard.png" {
		t.Fatalf("unexpected companion reference: %+v", companion)
	}

	consumable, ok := consumableRewardReference([]interface{}{
		int32(8001), int32(11), int32(0), int32(0), int32(0), "consumable110004", int32(110), int32(4),
	}, resolver)
	if !ok || consumable.Names["en"] != "Gold Automata Medal" || consumable.IconPath != "consumable_item/consumable110004/consumable110004_standard.png" {
		t.Fatalf("unexpected consumable reference: %+v", consumable)
	}
}

func TestRewardWeaponIconPathUsesWeaponAssetNaming(t *testing.T) {
	tests := []struct {
		name   string
		weapon masterdata.EntityMWeapon
		want   string
	}{
		{
			name:   "standard weapon",
			weapon: masterdata.EntityMWeapon{WeaponCategoryType: 1, WeaponType: 3, AssetVariationId: 12},
			want:   "weapon/wp003012/wp003012_standard.png",
		},
		{
			name:   "special weapon",
			weapon: masterdata.EntityMWeapon{WeaponCategoryType: 2, WeaponType: 5, AssetVariationId: 7},
			want:   "weapon/mw005007/mw005007_standard.png",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := rewardWeaponIconPath(test.weapon); got != test.want {
				t.Fatalf("rewardWeaponIconPath() = %q, want %q", got, test.want)
			}
		})
	}
}

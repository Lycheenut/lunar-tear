package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata"
)

func TestInstalledRewardCatalogResolvesCommemorativeRewards(t *testing.T) {
	masterDataPath := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(masterDataPath); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master data is not installed")
	} else if err != nil {
		t.Fatal(err)
	}

	catalog, err := LoadRewardReferenceCatalog(masterDataPath, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	foundImportantItem := false
	for _, item := range catalog.ImportantItems {
		if item.PossessionId != 310010 {
			continue
		}
		if item.Names["en"] == "" || item.Names["ja"] == "" {
			t.Fatalf("important item 310010 has incomplete names: %+v", item.Names)
		}
		if item.IconPath != "important_item/important310010/important310010_standard.png" {
			t.Fatalf("important item 310010 icon = %q", item.IconPath)
		}
		foundImportantItem = true
		break
	}
	if !foundImportantItem {
		t.Fatal("reward catalog is missing important item 310010")
	}

	for _, item := range catalog.Parts {
		if item.PossessionId != 728 {
			continue
		}
		if item.Names["en"] == "" || item.Names["ja"] == "" {
			t.Fatalf("parts 728 has incomplete names: %+v", item.Names)
		}
		if item.IconPath != "memory/memory037/memory037_standard.png" {
			t.Fatalf("parts 728 icon = %q", item.IconPath)
		}
		return
	}
	t.Fatal("reward catalog is missing parts 728")
}

func TestRewardReferencesResolveNamesIconsAndFilters(t *testing.T) {
	resolver := &titleResolver{texts: localizationIndex{
		"en": {
			"material.name.100001":        "Small Weapon Enhancement",
			"weapon.name.wp001002.1":      "Nameless Blade",
			"costume.name.ch003004":       "Abstract Hunter",
			"companion.name.cm001002":     "Bear: Precious",
			"parts.group.name.37":         "Beauty Potion",
			"consumable_item.name.110004": "Gold Automata Medal",
			"important_item.name.310010":  "Record: The Girl and the Monster",
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

	parts, ok := partsRewardReference([]interface{}{
		int32(728), int32(20), int32(37), int32(1), int32(2), int32(3),
	}, resolver, map[int64]int64{37: 37})
	if !ok || parts.Names["en"] != "Beauty Potion" || parts.IconPath != "memory/memory037/memory037_standard.png" ||
		parts.RarityType != 20 || parts.PossessionType != 4 {
		t.Fatalf("unexpected parts reference: %+v", parts)
	}

	consumable, ok := consumableRewardReference([]interface{}{
		int32(8001), int32(11), int32(0), int32(0), int32(0), "consumable110004", int32(110), int32(4),
	}, resolver)
	if !ok || consumable.Names["en"] != "Gold Automata Medal" || consumable.IconPath != "consumable_item/consumable110004/consumable110004_standard.png" {
		t.Fatalf("unexpected consumable reference: %+v", consumable)
	}

	importantItem, ok := importantItemRewardReference([]interface{}{
		int32(310010), int32(310010), int32(310010), int32(1), int32(310), int32(10),
		int32(0), int32(0), int32(0), int32(1), int32(0),
	}, resolver)
	if !ok || importantItem.Names["en"] != "Record: The Girl and the Monster" ||
		importantItem.IconPath != "important_item/important310010/important310010_standard.png" {
		t.Fatalf("unexpected important-item reference: %+v", importantItem)
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

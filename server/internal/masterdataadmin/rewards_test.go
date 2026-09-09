package masterdataadmin

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
)

func TestInstalledRewardCatalogCoversPossessionTables(t *testing.T) {
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
	encoded, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	file, err := memorydb.OpenFile(masterDataPath)
	if err != nil {
		t.Fatal(err)
	}
	var samples []RewardReference
	for _, test := range []struct {
		key, table     string
		possessionType int32
	}{
		{"costumes", "m_costume", 1}, {"weapons", "m_weapon", 2},
		{"companions", "m_companion", 3}, {"parts", "m_parts", 4},
		{"materials", "m_material", 5}, {"consumableItems", "m_consumable_item", 6},
		{"enhancedCostumes", "m_costume_enhanced", 7}, {"enhancedWeapons", "m_weapon_enhanced", 8},
		{"enhancedCompanions", "m_companion_enhanced", 9}, {"enhancedParts", "m_parts_enhanced", 10},
		{"paidGems", "", 11}, {"freeGems", "", 12},
		{"importantItems", "m_important_item", 13}, {"thoughts", "m_thought", 14},
		{"missionPassPoints", "m_mission_pass", 15}, {"premiumItems", "m_premium_item", 16},
	} {
		t.Run(test.key, func(t *testing.T) {
			raw, ok := fields[test.key]
			if !ok {
				t.Fatalf("reward catalog is missing %s (type %d)", test.key, test.possessionType)
			}
			var references []RewardReference
			if err := json.Unmarshal(raw, &references); err != nil {
				t.Fatal(err)
			}
			rows := readRows(file, test.table)
			want := len(rows)
			if test.table == "" {
				want = 1
			}
			if len(references) != want {
				t.Fatalf("got %d references, want %d", len(references), want)
			}
			ids := make(map[int32]bool)
			for i, reference := range references {
				if reference.PossessionType != test.possessionType || ids[reference.PossessionId] {
					t.Fatalf("invalid or duplicate reference: %+v", reference)
				}
				if i > 0 && references[i-1].PossessionId >= reference.PossessionId {
					t.Fatal("references are not sorted by ID")
				}
				ids[reference.PossessionId] = true
			}
			for _, row := range rows {
				id, _ := integerAt(row, 0)
				if !ids[int32(id)] {
					t.Errorf("missing possession ID %d", id)
				}
			}
			if test.table == "" && !ids[0] {
				t.Fatal("gems must use possession ID 0")
			}
			if len(references) > 0 {
				samples = append(samples, references[0])
			}
		})
	}
	// Exercise the actual delivery writer, then read the rebuilt data back.
	activityCatalog, err := Load(masterDataPath)
	if err != nil {
		t.Fatal(err)
	}
	var changes []Change
	for index, sample := range samples {
		changes = append(changes,
			Change{Table: missionRewardTable, Row: index, Field: "PossessionType", Value: sample.PossessionType},
			Change{Table: missionRewardTable, Row: index, Field: "PossessionId", Value: sample.PossessionId},
		)
	}
	candidate, _, err := BuildUpdate(masterDataPath, UpdateRequest{ExpectedVersion: activityCatalog.Version, Changes: changes})
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rows := readRows(rebuilt, missionRewardTable)
	for index, sample := range samples {
		possessionType, _ := integerAt(rows[index], 1)
		possessionID, _ := integerAt(rows[index], 2)
		if int32(possessionType) != sample.PossessionType || int32(possessionID) != sample.PossessionId {
			t.Errorf("reward %d round trip = %d:%d, want %d:%d", index, possessionType, possessionID, sample.PossessionType, sample.PossessionId)
		}
	}
}

func TestCostumeAndThoughtRewardReferencesUseAssetIDs(t *testing.T) {
	resolver := &titleResolver{texts: localizationIndex{"en": {
		"costume.name.ch003004":         "Old title",
		"costume.name.replace.ch003004": "Abstract Hunter",
		"thought.name.001008":           "Debris: Vigor Fragment",
	}}}
	costume, ok := costumeRewardReference([]interface{}{9001, 3, 0, 1, 3, 4, 2, 40}, resolver)
	if !ok || costume.PossessionType != 1 || costume.PossessionId != 9001 || costume.Names["en"] != "Abstract Hunter" ||
		costume.IconPath != "costume/ch003004/ch003004_standard.png" || costume.WeaponType != 2 || costume.RarityType != 40 {
		t.Fatalf("unexpected costume reference: %+v", costume)
	}
	thought, ok := thoughtRewardReference([]interface{}{10100834, 40, 1, 1, 1008}, resolver)
	if !ok || thought.PossessionType != 14 || thought.PossessionId != 10100834 || thought.Names["en"] != "Debris: Vigor Fragment" ||
		thought.IconPath != "thought/thought001008/thought001008_standard.png" || thought.RarityType != 40 {
		t.Fatalf("unexpected thought reference: %+v", thought)
	}
	if _, ok := costumeRewardReference([]interface{}{9001}, resolver); ok {
		t.Fatal("accepted incomplete costume")
	}
	if _, ok := thoughtRewardReference([]interface{}{10100834, 40, 1, 1, "invalid"}, resolver); ok {
		t.Fatal("accepted invalid thought asset ID")
	}
}

func TestEnhancedRewardReferencesPreserveEnhancedIDs(t *testing.T) {
	for _, possessionType := range []model.PossessionType{
		model.PossessionTypeCostumeEnhanced, model.PossessionTypeWeaponEnhanced,
		model.PossessionTypeCompanionEnhanced, model.PossessionTypePartsEnhanced,
	} {
		base := []RewardReference{{PossessionId: 101, Names: map[string]string{"en": "Base item"}, IconPath: "base.png", RarityType: 40}}
		references := enhancedRewardReferences([][]interface{}{{9001, 101, 50}, {9002, 999}, {9003}}, base, possessionType)
		if len(references) != 1 {
			t.Fatalf("type %d: unexpected references: %+v", possessionType, references)
		}
		reference := references[0]
		if reference.PossessionType != int32(possessionType) || reference.PossessionId != 9001 || reference.Names["en"] != "Base item" ||
			reference.IconPath != "base.png" || reference.RarityType != 40 || base[0].PossessionId != 101 {
			t.Fatalf("type %d: unexpected reference: %+v", possessionType, reference)
		}
	}
}

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

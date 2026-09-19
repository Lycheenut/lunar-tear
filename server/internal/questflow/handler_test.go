package questflow

import (
	"fmt"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestCompanionGrantsUseMasterDataLevels(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := masterdata.LoadConditionResolver()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := masterdata.LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	granter := BuildGranter(catalog, nil)
	for _, test := range []struct {
		possessionType model.PossessionType
		id, level      int32
	}{
		{model.PossessionTypeCompanionEnhanced, 49, 50},
		{model.PossessionTypeCompanionEnhanced, 50, 50},
		{model.PossessionTypeCompanionEnhanced, 51, 50},
		{model.PossessionTypeCompanionEnhanced, 53, 50},
		{model.PossessionTypeCompanion, 1, 1},
		{model.PossessionTypeCompanion, 35, 1},
		{model.PossessionTypeCompanion, 53, 1},
	} {
		t.Run(fmt.Sprintf("type=%d/id=%d", test.possessionType, test.id), func(t *testing.T) {
			user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
			result := granter.GrantFull(user, test.possessionType, test.id, 1, 1000)
			if result.Status != store.GrantStatusGranted || len(user.Companions) != 1 {
				t.Fatalf("grant = %+v, companions = %d", result, len(user.Companions))
			}
			for _, companion := range user.Companions {
				if companion.CompanionId != test.id || companion.Level != test.level {
					t.Fatalf("companion = %+v, want id=%d level=%d", companion, test.id, test.level)
				}
			}
		})
	}
}

func TestBuildGranterBuildsEnhancedCostumeGrant(t *testing.T) {
	thresholds := make([]int32, 16)
	thresholds[15] = 4321
	catalog := &masterdata.QuestCatalog{
		CostumeById: map[int32]masterdata.EntityMCostume{
			10103: {CostumeId: 10103, CharacterId: 101, RarityType: 20},
		},
		CostumeEnhancedById: map[int32]masterdata.EntityMCostumeEnhanced{
			9001: {CostumeEnhancedId: 9001, CostumeId: 10103, Level: 15},
		},
		CostumeExpByRarity: map[int32][]int32{20: thresholds},
		PartsCatalog:       &masterdata.PartsCatalog{},
	}

	granter := BuildGranter(catalog, nil)
	got := granter.CostumeEnhancedById[9001]
	if got.CostumeId != 10103 || got.Level != 15 || got.Exp != 4321 {
		t.Fatalf("enhanced costume grant = %+v, want id=10103 level=15 exp=4321", got)
	}
}

func TestBuildGranterRandomizesPartsMainStatus(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	g := BuildGranter(&masterdata.QuestCatalog{PartsCatalog: parts}, nil)
	for _, test := range []struct {
		group   int32
		allowed []int32
	}{
		{1, []int32{4, 8, 12, 16, 20, 24}},
		{2, []int32{4, 8, 12, 16, 20, 24, 28, 32}},
		{3, []int32{4, 8, 12, 16, 20, 24, 36}},
		{401, []int32{4, 8, 12, 16, 20, 24}},
		{402, []int32{4, 8, 12, 16, 20, 24}},
		{403, []int32{4, 8, 12, 16, 20, 24}},
		{455, []int32{4, 8, 12, 16, 20, 24}},
	} {
		t.Run(fmt.Sprintf("group=%d", test.group), func(t *testing.T) {
			var partID int32
			for id, part := range parts.PartsById {
				if part.PartsGroupId == test.group && part.RarityType == 40 {
					partID = id
					break
				}
			}
			if partID == 0 {
				t.Fatal("missing test part")
			}
			user := store.SeedUserState(1, "parts", 1, model.ClientPlatform{})
			for range 256 {
				g.GrantParts(user, partID, 1000)
			}
			seen := make(map[int32]int)
			for _, part := range user.Parts {
				seen[part.PartsStatusMainId]++
			}
			for _, id := range test.allowed {
				if seen[id] == 0 {
					t.Errorf("main status %d was never rolled: %v", id, seen)
				}
			}
			if len(seen) != len(test.allowed) {
				t.Errorf("rolled main statuses=%v, want only %v", seen, test.allowed)
			}
		})
	}
}

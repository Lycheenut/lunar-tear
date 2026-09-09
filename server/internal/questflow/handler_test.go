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

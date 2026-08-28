package questflow

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
)

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

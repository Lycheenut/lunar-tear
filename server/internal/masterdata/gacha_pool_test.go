package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestGacha614PromotionFollowsExchangeShopCellOrder(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}

	entries, _, err := LoadGachaCatalog()
	if err != nil {
		t.Fatal(err)
	}
	pool, err := LoadGachaPool()
	if err != nil {
		t.Fatal(err)
	}
	shop, err := LoadShopCatalog()
	if err != nil {
		t.Fatal(err)
	}

	pool.BuildShopFeatured(shop)
	pool.BuildFeaturedFromTerms(entries)
	EnrichCatalogPromotions(entries, pool)

	for _, entry := range entries {
		if entry.GachaId != 614 {
			continue
		}

		wantCostumes := []int32{32006, 33005, 34009}
		wantWeapons := []int32{320151, 330161, 340231}
		if got := len(entry.PromotionItems); got != len(wantCostumes) {
			t.Fatalf("promotion item count = %d, want %d", got, len(wantCostumes))
		}
		for i := range wantCostumes {
			got := entry.PromotionItems[i]
			if got.PossessionId != wantCostumes[i] {
				t.Errorf("promotion item %d costume = %d, want %d", i, got.PossessionId, wantCostumes[i])
			}
			if got.BonusPossessionId != wantWeapons[i] {
				t.Errorf("promotion item %d bonus weapon = %d, want %d", i, got.BonusPossessionId, wantWeapons[i])
			}
		}
		return
	}

	t.Fatal("gacha 614 not found")
}

package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestGacha614PromotionFollowsExchangeShopCellOrder(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}

	entries, medals, err := LoadGachaCatalog(61)
	if err != nil {
		t.Fatal(err)
	}
	medal := medals[614]
	entries = append(entries, store.GachaCatalogEntry{
		GachaId:               614,
		GachaLabelType:        model.GachaLabelPremium,
		GachaMedalId:          medal.GachaMedalId,
		MedalConsumableItemId: medal.ConsumableItemId,
	})
	pool, err := LoadGachaPool()
	if err != nil {
		t.Fatal(err)
	}
	shop, err := LoadShopCatalog()
	if err != nil {
		t.Fatal(err)
	}

	pool.BuildShopFeatured(shop)
	pool.PruneUnpairedCostumes()
	pool.BuildFeaturedFromTerms(entries)
	pool.BuildBannerPools(entries)
	EnrichCatalogPromotions(entries, pool)

	for _, entry := range entries {
		if entry.GachaId != 614 {
			continue
		}

		wantCostumes := []int32{32006, 33005, 34009}
		wantWeapons := []int32{320151, 330161, 340231}
		featured := pool.FeaturedByGacha[entry.GachaId]
		if got := len(featured.Costumes); got != len(wantCostumes) {
			t.Fatalf("featured costume count = %d, want %d", got, len(wantCostumes))
		}
		if got := len(featured.Weapons); got != 0 {
			t.Fatalf("featured weapon count = %d, want 0 paired weapons", got)
		}
		if got := len(pool.BannerPools[entry.GachaId].Featured); got != len(wantCostumes) {
			t.Fatalf("drawable featured count = %d, want %d", got, len(wantCostumes))
		}
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

func TestBuildFeaturedPrefersExchangeShopTargetsOverSameStartTerms(t *testing.T) {
	shopCostume := GachaPoolItem{PossessionType: int32(model.PossessionTypeCostume), PossessionId: 10, RarityType: model.RaritySSRare}
	shopWeapon := GachaPoolItem{PossessionType: int32(model.PossessionTypeWeapon), PossessionId: 100, RarityType: model.RaritySSRare}
	unrelated := GachaPoolItem{PossessionType: int32(model.PossessionTypeCostume), PossessionId: 20, RarityType: model.RaritySSRare}
	pool := &GachaCatalog{
		CostumeById:          map[int32]GachaPoolItem{10: shopCostume, 20: unrelated},
		WeaponById:           map[int32]GachaPoolItem{100: shopWeapon},
		CostumeWeaponMap:     map[int32]int32{10: 100},
		ShopFeaturedByMedal:  map[int32][]ShopFeaturedEntry{900: {{CostumeId: 10, WeaponId: 100}}},
		TermsByStartDatetime: map[int64][]*CatalogTerm{123: {{TermId: 2, StartDatetime: 123, Costumes: []GachaPoolItem{unrelated}}}},
		FeaturedByGacha:      make(map[int32]FeaturedSet),
	}
	entries := []store.GachaCatalogEntry{{
		GachaId: 1, GachaLabelType: model.GachaLabelPremium, StartDatetime: 123, MedalConsumableItemId: 900,
	}}

	pool.BuildFeaturedFromTerms(entries)
	featured := pool.FeaturedByGacha[1]
	if len(featured.Costumes) != 1 || featured.Costumes[0].PossessionId != 10 {
		t.Fatalf("featured costumes = %+v, want only shop costume 10", featured.Costumes)
	}
	if len(featured.Weapons) != 0 {
		t.Fatalf("featured weapons = %+v, want paired weapon represented only as costume bonus", featured.Weapons)
	}
}

func TestBuildBannerPoolsSeparatesAndDeduplicatesFeaturedItems(t *testing.T) {
	standardTarget := GachaPoolItem{PossessionType: int32(model.PossessionTypeCostume), PossessionId: 1, RarityType: model.RaritySSRare}
	standardOther := GachaPoolItem{PossessionType: int32(model.PossessionTypeCostume), PossessionId: 2, RarityType: model.RaritySSRare}
	featuredOther := GachaPoolItem{PossessionType: int32(model.PossessionTypeCostume), PossessionId: 3, RarityType: model.RaritySSRare}
	featuredWeapon := GachaPoolItem{PossessionType: int32(model.PossessionTypeWeapon), PossessionId: 4, RarityType: model.RaritySSRare}
	pool := &GachaCatalog{
		StandardCostumesByRarity: map[int32][]GachaPoolItem{model.RaritySSRare: {standardTarget, standardOther}},
		StandardWeaponsByRarity:  map[int32][]GachaPoolItem{model.RaritySSRare: {{PossessionType: int32(model.PossessionTypeWeapon), PossessionId: 5, RarityType: model.RaritySSRare}}},
		FeaturedByGacha: map[int32]FeaturedSet{1: {
			Costumes: []GachaPoolItem{standardTarget, featuredOther, featuredOther},
			Weapons:  []GachaPoolItem{featuredWeapon, featuredWeapon},
		}},
	}

	pool.BuildBannerPools([]store.GachaCatalogEntry{{GachaId: 1}})
	banner := pool.BannerPools[1]
	if got := len(banner.Featured); got != 3 {
		t.Fatalf("featured count = %d, want 3 unique items", got)
	}
	standard := banner.CostumesByRarity[model.RaritySSRare]
	if len(standard) != 1 || standard[0].PossessionId != standardOther.PossessionId {
		t.Fatalf("standard costume fallback = %+v, want only non-featured costume", standard)
	}
}

func TestEnrichCatalogPromotionsDisplaysEveryFeaturedTarget(t *testing.T) {
	costume1 := GachaPoolItem{PossessionType: int32(model.PossessionTypeCostume), PossessionId: 10, RarityType: model.RaritySSRare}
	costume2 := GachaPoolItem{PossessionType: int32(model.PossessionTypeCostume), PossessionId: 20, RarityType: model.RaritySRare}
	weapon := GachaPoolItem{PossessionType: int32(model.PossessionTypeWeapon), PossessionId: 30, RarityType: model.RaritySSRare}
	pool := &GachaCatalog{
		FeaturedByGacha:  map[int32]FeaturedSet{1: {Costumes: []GachaPoolItem{costume1, costume2}, Weapons: []GachaPoolItem{weapon}}},
		CostumeWeaponMap: map[int32]int32{},
	}
	entries := []store.GachaCatalogEntry{{GachaId: 1, GachaModeType: model.GachaModeStepup}}

	EnrichCatalogPromotions(entries, pool)
	if got := len(entries[0].PromotionItems); got != 3 {
		t.Fatalf("promotion count = %d, want all 3 featured targets", got)
	}
	for i, want := range []int32{10, 20, 30} {
		if got := entries[0].PromotionItems[i].PossessionId; got != want {
			t.Fatalf("promotion %d = %d, want %d", i, got, want)
		}
	}
}

func TestPremiumBannerPromotionsMatchUniqueDrawableFeaturedPool(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog(61)
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
	pool.PruneUnpairedCostumes()
	pool.BuildFeaturedFromTerms(entries)
	pool.BuildBannerPools(entries)
	EnrichCatalogPromotions(entries, pool)

	requiredFallbacks := []struct {
		possessionType int32
		rarity         int32
	}{
		{int32(model.PossessionTypeCostume), model.RaritySSRare},
		{int32(model.PossessionTypeCostume), model.RaritySRare},
		{int32(model.PossessionTypeWeapon), model.RaritySSRare},
		{int32(model.PossessionTypeWeapon), model.RaritySRare},
		{int32(model.PossessionTypeWeapon), model.RarityRare},
	}
	for _, entry := range entries {
		if entry.GachaLabelType != model.GachaLabelPremium {
			continue
		}
		banner := pool.BannerPools[entry.GachaId]
		if banner == nil {
			t.Fatalf("premium gacha %d has no banner pool", entry.GachaId)
		}
		featured := make(map[[2]int32]bool)
		for _, item := range banner.Featured {
			key := [2]int32{item.PossessionType, item.PossessionId}
			if featured[key] {
				t.Fatalf("premium gacha %d has duplicate featured item %v", entry.GachaId, key)
			}
			featured[key] = true
		}
		promoted := make(map[[2]int32]bool)
		for _, item := range entry.PromotionItems {
			key := [2]int32{item.PossessionType, item.PossessionId}
			if promoted[key] {
				t.Fatalf("premium gacha %d has duplicate promotion item %v", entry.GachaId, key)
			}
			promoted[key] = true
		}
		if len(promoted) != len(featured) {
			t.Fatalf("premium gacha %d promotion count %d != featured count %d", entry.GachaId, len(promoted), len(featured))
		}
		for key := range featured {
			if !promoted[key] {
				t.Fatalf("premium gacha %d hides featured item %v", entry.GachaId, key)
			}
		}
		for _, fallback := range requiredFallbacks {
			var items []GachaPoolItem
			if fallback.possessionType == int32(model.PossessionTypeCostume) {
				items = banner.CostumesByRarity[fallback.rarity]
			} else {
				items = banner.WeaponsByRarity[fallback.rarity]
			}
			if len(items) == 0 {
				t.Fatalf("premium gacha %d has no non-featured fallback for type=%d rarity=%d", entry.GachaId, fallback.possessionType, fallback.rarity)
			}
			seen := make(map[int32]bool)
			for _, item := range items {
				if seen[item.PossessionId] {
					t.Fatalf("premium gacha %d has duplicate fallback item %d", entry.GachaId, item.PossessionId)
				}
				seen[item.PossessionId] = true
				if featured[[2]int32{item.PossessionType, item.PossessionId}] {
					t.Fatalf("premium gacha %d includes featured item %d in non-featured fallback", entry.GachaId, item.PossessionId)
				}
			}
		}
	}
}

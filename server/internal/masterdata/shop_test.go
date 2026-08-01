package masterdata

import "testing"

func TestShopCatalogAvailabilityUsesShopMembershipAndTerms(t *testing.T) {
	catalog := &ShopCatalog{
		Shops: map[int32]EntityMShop{
			1: {ShopId: 1, StartDatetime: 100, EndDatetime: 300},
			2: {ShopId: 2, StartDatetime: 100, EndDatetime: 300},
		},
		CellsByShop: map[int32]map[int32][]ShopCell{
			1: {
				10: {{StartDatetime: 150, EndDatetime: 250}},
			},
		},
	}

	if !catalog.IsItemAvailable(1, 10, 200) {
		t.Fatal("open item was unavailable")
	}
	if catalog.IsItemAvailable(1, 10, 125) {
		t.Fatal("item was available before its cell term")
	}
	if catalog.IsItemAvailable(2, 10, 200) {
		t.Fatal("item was available in an unrelated shop")
	}
	if catalog.IsItemAvailable(1, 10, 350) {
		t.Fatal("item was available after its shop term")
	}
}

func TestShopCatalogAvailabilityRejectsLimitedOpenCell(t *testing.T) {
	catalog := &ShopCatalog{
		Shops: map[int32]EntityMShop{1: {ShopId: 1}},
		CellsByShop: map[int32]map[int32][]ShopCell{
			1: {10: {{LimitedOpenId: 99}}},
		},
	}

	if catalog.IsItemAvailable(1, 10, 100) {
		t.Fatal("limited-open item was available without an unlock source")
	}
}

func TestShopCatalogResolvesLevelAdditionalContent(t *testing.T) {
	catalog := &ShopCatalog{
		UserLevelsByItem: map[int32][]EntityMShopItemUserLevelCondition{
			10: {
				{UserLevelLowerLimit: 1, UserLevelUpperLimit: 10, ShopItemAdditionalContentId: 100},
				{UserLevelLowerLimit: 11, UserLevelUpperLimit: 20, ShopItemAdditionalContentId: 200},
			},
		},
		AdditionalContent: map[int32][]EntityMShopItemAdditionalContent{
			100: {{ShopItemAdditionalContentId: 100, PossessionId: 1000, Count: 1}},
			200: {{ShopItemAdditionalContentId: 200, PossessionId: 2000, Count: 2}},
		},
	}

	contents, available := catalog.AdditionalContentsForLevel(10, 15)
	if !available || len(contents) != 1 || contents[0].PossessionId != 2000 {
		t.Fatalf("level 15 contents = %+v, available=%v", contents, available)
	}
	if _, available := catalog.AdditionalContentsForLevel(10, 21); available {
		t.Fatal("item was available outside every configured level range")
	}
	if contents, available := catalog.AdditionalContentsForLevel(20, 99); !available || len(contents) != 0 {
		t.Fatalf("unconditional item contents = %+v, available=%v", contents, available)
	}
}

func TestShopCatalogReplaceableGemPriceUsesHighestReachedTier(t *testing.T) {
	catalog := &ShopCatalog{ReplaceableGems: []EntityMShopReplaceableGem{
		{LineupUpdateCountLowerLimit: 1, NecessaryGem: 10},
		{LineupUpdateCountLowerLimit: 2, NecessaryGem: 30},
		{LineupUpdateCountLowerLimit: 3, NecessaryGem: 50},
	}}

	for _, tc := range []struct {
		count int32
		price int32
		ok    bool
	}{
		{count: 0, ok: false},
		{count: 1, price: 10, ok: true},
		{count: 2, price: 30, ok: true},
		{count: 20, price: 50, ok: true},
	} {
		price, ok := catalog.ReplaceableGemPrice(tc.count)
		if price != tc.price || ok != tc.ok {
			t.Fatalf("price for refresh %d = (%d,%v), want (%d,%v)", tc.count, price, ok, tc.price, tc.ok)
		}
	}
}

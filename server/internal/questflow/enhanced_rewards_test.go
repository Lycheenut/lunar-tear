package questflow

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestBuildGranterAndDropPreserveEnhancedRewards(t *testing.T) {
	catalog := &masterdata.QuestCatalog{
		WeaponById:       map[int32]masterdata.EntityMWeapon{101: {WeaponId: 101, WeaponSkillGroupId: 10, WeaponAbilityGroupId: 20}},
		WeaponSkillSlots: map[int32][]int32{10: {1}}, WeaponAbilitySlots: map[int32][]int32{20: {2}},
		WeaponEnhancedById: map[int32]masterdata.WeaponEnhancedReward{9001: {
			EntityMWeaponEnhanced: masterdata.EntityMWeaponEnhanced{WeaponId: 101, Level: 70, LimitBreakCount: 3},
			Exp:                   5000, SkillLevels: map[int32]int32{1: 7}, AbilityLevels: map[int32]int32{2: 8},
		}},
		PartsCatalog: &masterdata.PartsCatalog{
			PartsById:           map[int32]masterdata.EntityMParts{201: {PartsId: 201, PartsGroupId: 10, RarityType: 40, PartsInitialLotteryId: 3}},
			EnhancedById:        map[int32]masterdata.EntityMPartsEnhanced{9002: {PartsEnhancedId: 9002, PartsId: 201, Level: 15, PartsStatusMainId: 8, SubStatusCount: 1}},
			EnhancedSubStatuses: map[int32][]masterdata.EntityMPartsEnhancedSubStatus{9002: {{StatusIndex: 1, PartsStatusSubLotteryId: 1, Level: 7, StatusKindType: 6, StatusCalculationType: 2, FixedStatusChangeValue: 777}}},
			SellPriceByRarity:   map[model.RarityType]masterdata.NumericalFunc{40: {Type: model.NumericalFunctionTypeLinear, Params: []int32{10, 5}}},
		},
	}
	g := BuildGranter(catalog, &masterdata.GameConfig{ConsumableItemIdForGold: 99})
	h := &QuestHandler{QuestCatalog: catalog, Granter: g}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	drops := []RewardGrant{
		{PossessionType: model.PossessionTypeWeaponEnhanced, PossessionId: 9001, Count: 2},
		{PossessionType: model.PossessionTypePartsEnhanced, PossessionId: 9002, Count: 3},
	}
	drops = h.grantDropRewards(user, drops, nil, nil, 1000)
	if len(user.Weapons) != 2 || len(user.Parts) != 3 || len(user.PartsStatusSubs) != 3 {
		t.Fatalf("drop inventory: weapons=%v parts=%v subs=%v", user.Weapons, user.Parts, user.PartsStatusSubs)
	}
	for id, weapon := range user.Weapons {
		if weapon.WeaponId != 101 || weapon.Level != 70 || weapon.Exp != 5000 || weapon.LimitBreakCount != 3 || user.WeaponSkills[id][0].Level != 7 || user.WeaponAbilities[id][0].Level != 8 {
			t.Fatalf("weapon=%+v", weapon)
		}
	}
	for id, part := range user.Parts {
		sub := user.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: id, StatusIndex: 1}]
		if part.PartsId != 201 || part.Level != 15 || part.PartsStatusMainId != 8 || sub.Level != 7 || sub.StatusChangeValue != 777 {
			t.Fatalf("part=%+v sub=%+v", part, sub)
		}
	}
	if drops[1].PossessionType != model.PossessionTypePartsEnhanced || drops[1].PossessionId != 9002 || drops[1].IsAutoSale {
		t.Fatalf("kept drop=%+v", drops[1])
	}
	user = store.SeedUserState(2, "test", 1, model.ClientPlatform{})
	soldDrops := h.grantDropRewards(user, drops[1:], map[int32]bool{40: true}, map[int32]bool{2: true}, 2000)
	if len(user.Parts)+len(user.PartsStatusSubs) != 0 || user.ConsumableItems[99] != 465 {
		t.Fatalf("enhanced auto sale: parts=%v gold=%d", user.Parts, user.ConsumableItems[99])
	}
	if len(soldDrops) != 3 {
		t.Fatalf("sold rewards=%d, want 3 independently resolved copies", len(soldDrops))
	}
	for _, drop := range soldDrops {
		if drop.PossessionType != model.PossessionTypeParts || drop.PossessionId != 201 || !drop.IsAutoSale || drop.Count != 1 {
			t.Fatalf("sold popup=%+v", drop)
		}
	}
}

func TestRandomEnhancedPartsReportEachSaleSeparately(t *testing.T) {
	catalog := &masterdata.QuestCatalog{PartsCatalog: &masterdata.PartsCatalog{
		PartsById: map[int32]masterdata.EntityMParts{},
		EnhancedById: map[int32]masterdata.EntityMPartsEnhanced{9002: {
			PartsEnhancedId: 9002, PartsId: 101, Level: 1, PartsStatusMainId: 8, IsRandomSubStatusCount: true,
		}},
		SubStatusPool:      map[int32][]int32{1: {1, 2, 3, 4}},
		PartsStatusSubById: map[int32]model.PartsStatusSubDef{},
		SellPriceByRarity:  map[model.RarityType]masterdata.NumericalFunc{40: {Type: model.NumericalFunctionTypeLinear, Params: []int32{0, 100}}},
	}}
	for rank := int32(1); rank <= 5; rank++ {
		catalog.PartsById[100+rank] = masterdata.EntityMParts{
			PartsId: 100 + rank, PartsGroupId: 1, RarityType: 40, PartsInitialLotteryId: rank, PartsStatusSubLotteryGroupId: 1,
		}
	}
	for id := int32(1); id <= 4; id++ {
		catalog.PartsStatusSubById[id] = model.PartsStatusSubDef{Initial: model.PartsSubStatusRange{Min: 1, Max: 1, Step: 1}}
	}
	h := &QuestHandler{QuestCatalog: catalog, Granter: BuildGranter(catalog, &masterdata.GameConfig{ConsumableItemIdForGold: 99})}
	user := store.SeedUserState(1, "parts", 1, model.ClientPlatform{})
	drops := h.grantDropRewards(user, []RewardGrant{{PossessionType: model.PossessionTypePartsEnhanced, PossessionId: 9002, Count: 200}},
		map[int32]bool{40: true}, map[int32]bool{5: true}, 1000)
	sold := 0
	for _, drop := range drops {
		if drop.Count != 1 {
			t.Fatalf("combined independently sold copies: %+v", drop)
		}
		if drop.IsAutoSale {
			sold++
		}
	}
	if len(drops) != 200 || sold == 0 || sold == 200 || len(user.Parts)+sold != 200 || user.ConsumableItems[99] != int32(sold)*100 {
		t.Fatalf("drops=%d sold=%d kept=%d gold=%d", len(drops), sold, len(user.Parts), user.ConsumableItems[99])
	}
}

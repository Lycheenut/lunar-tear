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
	h.grantDropRewards(user, drops, nil, nil, 1000)
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
	h.grantDropRewards(user, drops[1:], map[int32]bool{40: true}, map[int32]bool{3: true}, 2000)
	if len(user.Parts)+len(user.PartsStatusSubs) != 0 || user.ConsumableItems[99] != 465 {
		t.Fatalf("enhanced auto sale: parts=%v gold=%d", user.Parts, user.ConsumableItems[99])
	}
	if drops[1].PossessionType != model.PossessionTypeParts || drops[1].PossessionId != 201 || !drops[1].IsAutoSale || drops[1].Count != 3 {
		t.Fatalf("sold popup=%+v", drops[1])
	}
}

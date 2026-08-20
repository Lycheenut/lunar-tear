package service

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestPlatformPurchaseEndpointsAreDisabled(t *testing.T) {
	server := &ShopServiceServer{}
	if _, err := server.CreatePurchaseTransaction(context.Background(), &pb.CreatePurchaseTransactionRequest{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("CreatePurchaseTransaction status = %v, want FailedPrecondition", status.Code(err))
	}
	if _, err := server.PurchaseGooglePlayStoreProduct(context.Background(), &pb.PurchaseGooglePlayStoreProductRequest{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("PurchaseGooglePlayStoreProduct status = %v, want FailedPrecondition", status.Code(err))
	}
}

func TestBuildReplaceableLineupPreservesCatalogOrder(t *testing.T) {
	pool := []int32{10, 20, 30}

	lineup := buildReplaceableLineup(pool, 1234)
	for i, itemId := range pool {
		slot := int32(i + 1)
		row := lineup[slot]
		if row.SlotNumber != slot || row.LatestVersion != 1234 {
			t.Fatalf("invalid lineup row for slot %d: %+v", slot, row)
		}
		if row.ShopItemId != itemId {
			t.Fatalf("lineup item for slot %d = %d, want %d", slot, row.ShopItemId, itemId)
		}
	}
	if len(lineup) != len(pool) {
		t.Fatalf("lineup length = %d, want %d", len(lineup), len(pool))
	}
}

func TestReplaceableRefreshCountResetsBeforeFirstPaidRefreshOfDay(t *testing.T) {
	if got := nextReplaceableRefreshCount(12, true); got != 1 {
		t.Fatalf("first paid refresh of day count = %d, want 1", got)
	}
	if got := nextReplaceableRefreshCount(2, false); got != 3 {
		t.Fatalf("same-day paid refresh count = %d, want 3", got)
	}
}

func TestResetShopItemStockIfDue(t *testing.T) {
	// The international server's Monday starts at 08:00 UTC.
	now := time.Date(2026, time.August, 3, 8, 1, 0, 0, time.UTC)
	beforeWeeklyReset := time.Date(2026, time.August, 3, 7, 59, 0, 0, time.UTC).UnixMilli()
	afterWeeklyReset := time.Date(2026, time.August, 3, 8, 0, 0, 0, time.UTC).UnixMilli()
	rule := masterdata.ShopLimitedStockRule{MaxCount: 5, AutoResetType: model.ShopItemAutoResetWeekly, AutoResetPeriod: 1}

	reset, err := resetShopItemStockIfDue(store.UserShopItemState{BoughtCount: 5, LatestBoughtCountChangedDatetime: beforeWeeklyReset}, rule, now.UnixMilli())
	if err != nil || reset.BoughtCount != 0 {
		t.Fatalf("weekly reset state = %+v, err=%v", reset, err)
	}
	unchanged, err := resetShopItemStockIfDue(store.UserShopItemState{BoughtCount: 3, LatestBoughtCountChangedDatetime: afterWeeklyReset}, rule, now.UnixMilli())
	if err != nil || unchanged.BoughtCount != 3 {
		t.Fatalf("same-week state = %+v, err=%v", unchanged, err)
	}

	monthlyRule := masterdata.ShopLimitedStockRule{MaxCount: 5, AutoResetType: model.ShopItemAutoResetMonthly, AutoResetPeriod: 1}
	monthly, err := resetShopItemStockIfDue(store.UserShopItemState{
		BoughtCount:                      2,
		LatestBoughtCountChangedDatetime: time.Date(2026, time.August, 1, 7, 59, 0, 0, time.UTC).UnixMilli(),
	}, monthlyRule, now.UnixMilli())
	if err != nil || monthly.BoughtCount != 0 {
		t.Fatalf("monthly reset state = %+v, err=%v", monthly, err)
	}
}

func TestResetShopItemStockRejectsInvalidRule(t *testing.T) {
	_, err := resetShopItemStockIfDue(store.UserShopItemState{}, masterdata.ShopLimitedStockRule{
		AutoResetType:   model.ShopItemAutoResetWeekly,
		AutoResetPeriod: 8,
	}, time.Now().UnixMilli())
	if err == nil {
		t.Fatal("invalid weekly reset rule was accepted")
	}
}

func TestGrantShopPossessionSendsCapacityOverflowToGiftBox(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Weapons["existing"] = store.WeaponState{UserWeaponUuid: "existing", WeaponId: 1}
	config := &masterdata.GameConfig{PossessionCountLimitWeapon: 1}
	granter := &store.PossessionGranter{}

	overflow, err := grantShopPossession(granter, user, int32(model.PossessionTypeWeapon), 2, 1, 1, config, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Weapons) != 1 {
		t.Fatalf("weapon inventory count = %d, want 1", len(user.Weapons))
	}
	if overflow == nil || overflow.PossessionType != int32(model.PossessionTypeWeapon) || overflow.PossessionId != 2 || overflow.Count != 1 {
		t.Fatalf("overflow possession = %+v", overflow)
	}
	if len(user.Gifts.NotReceived) != 1 {
		t.Fatalf("not-received gifts = %d, want 1", len(user.Gifts.NotReceived))
	}
	gift := user.Gifts.NotReceived[0]
	if gift.UserGiftUuid == "" || gift.GiftCommon.PossessionType != int32(model.PossessionTypeWeapon) || gift.GiftCommon.PossessionId != 2 || gift.GiftCommon.Count != 1 {
		t.Fatalf("overflow gift = %+v", gift)
	}
	if user.Notifications.GiftNotReceiveCount != 1 {
		t.Fatalf("gift notification count = %d, want 1", user.Notifications.GiftNotReceiveCount)
	}
}

func TestGrantShopPossessionSplitsStackableOverflowByPossessionLimit(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	config := &masterdata.GameConfig{PossessionCountLimitMaterial: 99_999}
	granter := &store.PossessionGranter{}

	overflow, err := grantShopPossession(granter, user, int32(model.PossessionTypeMaterial), 77, 200_001, 1, config, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if overflow == nil || overflow.Count != 200_001 {
		t.Fatalf("overflow possession = %+v, want count 200001", overflow)
	}
	if user.Materials[77] != 0 {
		t.Fatalf("material inventory count = %d, want 0", user.Materials[77])
	}
	if len(user.Gifts.NotReceived) != 3 {
		t.Fatalf("not-received gifts = %d, want 3", len(user.Gifts.NotReceived))
	}
	for i, want := range []int32{99_999, 99_999, 3} {
		if got := user.Gifts.NotReceived[i].GiftCommon.Count; got != want {
			t.Fatalf("gift %d count = %d, want %d", i, got, want)
		}
	}
	if user.Notifications.GiftNotReceiveCount != 3 {
		t.Fatalf("gift notification count = %d, want 3", user.Notifications.GiftNotReceiveCount)
	}
}

func TestGrantShopPossessionKeepsWeaponWhenCapacityAllows(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Weapons["existing"] = store.WeaponState{UserWeaponUuid: "existing", WeaponId: 1}
	config := &masterdata.GameConfig{PossessionCountLimitWeapon: 2}
	granter := &store.PossessionGranter{}

	overflow, err := grantShopPossession(granter, user, int32(model.PossessionTypeWeapon), 2, 1, 1, config, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if overflow != nil {
		t.Fatalf("unexpected overflow possession = %+v", overflow)
	}
	if len(user.Weapons) != 2 || len(user.Gifts.NotReceived) != 0 {
		t.Fatalf("weapons=%d gifts=%d, want 2 and 0", len(user.Weapons), len(user.Gifts.NotReceived))
	}
}

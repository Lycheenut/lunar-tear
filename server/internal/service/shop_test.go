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

func TestRollReplaceableLineupChangesLineupWithoutChangingPool(t *testing.T) {
	pool := []int32{10, 20, 30}
	previous := map[int32]store.UserShopReplaceableLineupState{
		1: {SlotNumber: 1, ShopItemId: 10},
		2: {SlotNumber: 2, ShopItemId: 20},
		3: {SlotNumber: 3, ShopItemId: 30},
	}

	lineup := rollReplaceableLineup(pool, previous, 1234)
	seen := make(map[int32]bool, len(lineup))
	unchanged := true
	for slot, row := range lineup {
		seen[row.ShopItemId] = true
		if row.SlotNumber != slot || row.LatestVersion != 1234 {
			t.Fatalf("invalid lineup row for slot %d: %+v", slot, row)
		}
		if previous[slot].ShopItemId != row.ShopItemId {
			unchanged = false
		}
	}
	if unchanged {
		t.Fatal("refresh left the complete lineup unchanged")
	}
	for _, itemId := range pool {
		if !seen[itemId] {
			t.Fatalf("refreshed lineup omitted item %d", itemId)
		}
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

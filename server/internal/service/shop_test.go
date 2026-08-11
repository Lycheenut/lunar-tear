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

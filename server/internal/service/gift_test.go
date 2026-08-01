package service

import (
	"math"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestGrantGiftGrantsAssetsAndRejectsOverflow(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Materials[10] = 8
	granter := &store.PossessionGranter{}
	config := &masterdata.GameConfig{PossessionCountLimitMaterial: 10}

	if result := grantGift(user, store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 10, Count: 3,
	}, granter, config, 1000); result.Status != store.GrantStatusOverflow {
		t.Fatalf("overflowing gift status = %v, want overflow", result.Status)
	}
	if got := user.Materials[10]; got != 8 {
		t.Fatalf("material changed after overflow: got %d, want 8", got)
	}
	if result := grantGift(user, store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 10, Count: 2,
	}, granter, config, 1000); result.Status != store.GrantStatusGranted {
		t.Fatalf("gift at inventory limit status = %v, want granted", result.Status)
	}
	if got := user.Materials[10]; got != 10 {
		t.Fatalf("material count = %d, want 10", got)
	}
}

func TestGrantGiftDoesNotBlockOnUnrelatedExistingOverflow(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Materials[10] = 11
	config := &masterdata.GameConfig{PossessionCountLimitMaterial: 10}
	if result := grantGift(user, store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeFreeGem), Count: 5,
	}, &store.PossessionGranter{}, config, 1000); result.Status != store.GrantStatusGranted {
		t.Fatalf("unrelated existing overflow status = %v, want granted", result.Status)
	}
	if got := user.Gem.FreeGem; got != 5 {
		t.Fatalf("free gems = %d, want 5", got)
	}
}

func TestGiftRejectsUnsupportedPossessionWithoutCallingItOverflow(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	result := grantGift(user, store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeMissionPassPoint),
		PossessionId:   1,
		Count:          10,
	}, &store.PossessionGranter{}, &masterdata.GameConfig{}, 1000)
	if result.Status != store.GrantStatusUnsupported {
		t.Fatalf("mission pass gift status = %v, want unsupported", result.Status)
	}
}

func TestGiftPageRangeRejectsUntrustedCursor(t *testing.T) {
	if _, _, _, err := giftPageRange(0, 2, math.MaxInt64); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("large cursor status = %v, want InvalidArgument", status.Code(err))
	}
	start, end, pages, err := giftPageRange(5, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if start != 4 || end != 5 || pages != 3 {
		t.Fatalf("page range = (%d,%d,%d), want (4,5,3)", start, end, pages)
	}
}

func TestClaimableGiftCountExcludesExpiredGifts(t *testing.T) {
	gifts := []store.NotReceivedGiftState{
		{ExpirationDatetime: 999},
		{ExpirationDatetime: 1001},
		{},
	}
	if got := claimableGiftCount(gifts, 1000); got != 2 {
		t.Fatalf("claimable gift count = %d, want 2", got)
	}
}

func TestGiftFiltersMatchClientEnums(t *testing.T) {
	config := &masterdata.GameConfig{ConsumableItemIdForGold: 99}
	cases := []struct {
		possessionType model.PossessionType
		possessionId   int32
		want           int32
	}{
		{model.PossessionTypeFreeGem, 0, giftRewardKindGem},
		{model.PossessionTypeConsumableItem, 99, giftRewardKindGold},
		{model.PossessionTypeWeapon, 1, giftRewardKindWeapon},
		{model.PossessionTypeCompanion, 1, giftRewardKindCompanion},
		{model.PossessionTypeParts, 1, giftRewardKindParts},
		{model.PossessionTypeMaterial, 1, giftRewardKindMaterial},
		{model.PossessionTypeImportantItem, 1, giftRewardKindOther},
		{model.PossessionTypeCostume, 1, giftRewardKindCostume},
	}
	for _, tc := range cases {
		gift := store.GiftCommonState{PossessionType: int32(tc.possessionType), PossessionId: tc.possessionId}
		if got := giftRewardKind(gift, config); got != tc.want {
			t.Errorf("type %d id %d kind = %d, want %d", tc.possessionType, tc.possessionId, got, tc.want)
		}
	}

	expiring := store.NotReceivedGiftState{ExpirationDatetime: 100}
	permanent := store.NotReceivedGiftState{}
	if !matchesGiftExpirationFilter(expiring, giftExpirationFilterOnlyExpire) ||
		matchesGiftExpirationFilter(permanent, giftExpirationFilterOnlyExpire) ||
		!matchesGiftExpirationFilter(permanent, giftExpirationFilterOnlyNotExpire) {
		t.Fatal("expiration filters did not match client enum semantics")
	}
}

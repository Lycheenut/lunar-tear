package service

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestGetGiftListPreservesPermanentExpirationOnWire(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	repo := sqlite.New(db, nil)
	userID, err := repo.CreateUser("permanent-gift", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateUser(userID, func(user *store.UserState) {
		user.Gifts.NotReceived = append(user.Gifts.NotReceived, store.NotReceivedGiftState{
			GiftCommon: store.GiftCommonState{
				PossessionType: int32(model.PossessionTypeWeapon),
				PossessionId:   1,
				Count:          1,
				GrantDatetime:  1,
			},
			UserGiftUuid: "permanent-gift-uuid",
		})
	}); err != nil {
		t.Fatal(err)
	}

	masterData, err := os.ReadFile(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if err != nil {
		t.Fatal(err)
	}
	masterDataPath := filepath.Join(t.TempDir(), "master-data.bin.e")
	if err := os.WriteFile(masterDataPath, masterData, 0o600); err != nil {
		t.Fatal(err)
	}
	holder, err := runtime.NewHolder(masterDataPath)
	if err != nil {
		t.Fatal(err)
	}

	response, err := NewGiftServiceServer(repo, nil, holder).GetGiftList(context.Background(), &pb.GetGiftListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	wire, err := proto.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &pb.GetGiftListResponse{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Gift) != 1 {
		t.Fatalf("gift count = %d, want 1", len(decoded.Gift))
	}
	expiration := decoded.Gift[0].ExpirationDatetime
	if expiration == nil {
		t.Fatal("permanent gift expiration was omitted from the protobuf payload")
	}
	if expiration.Seconds != 0 || expiration.Nanos != 0 {
		t.Fatalf("permanent gift expiration = (%d,%d), want (0,0)", expiration.Seconds, expiration.Nanos)
	}
}

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

func TestGrantGiftPreservesEnhancedTemplatesAndRollsBackOverflow(t *testing.T) {
	g := &store.PossessionGranter{
		WeaponById:            map[int32]store.WeaponRef{101: {}},
		WeaponEnhancedById:    map[int32]store.WeaponEnhancedRef{9001: {WeaponId: 101, Level: 70, Exp: 5000, LimitBreakCount: 3}},
		CompanionEnhancedById: map[int32]store.CompanionEnhancedRef{9003: {CompanionId: 49, Level: 50}},
		PartsById:             map[int32]store.PartsRef{201: {PartsGroupId: 10}},
		PartsEnhancedById: map[int32]store.PartsEnhancedRef{9002: {
			PartsId: 201, Level: 15, PartsStatusMainId: 8, SubStatusCount: 1,
			SubStatuses: []store.PartsStatusSubState{{StatusIndex: 1, PartsStatusSubLotteryId: 1, Level: 7, StatusChangeValue: 777}},
		}},
	}
	config := &masterdata.GameConfig{PossessionCountLimitWeapon: 1, PossessionCountLimitParts: 1}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	for _, reward := range []store.GiftCommonState{
		{PossessionType: int32(model.PossessionTypeWeaponEnhanced), PossessionId: 9001, Count: 2},
		{PossessionType: int32(model.PossessionTypePartsEnhanced), PossessionId: 9002, Count: 2},
	} {
		if result := grantGift(user, reward, g, config, 1000); result.Status != store.GrantStatusOverflow {
			t.Fatalf("overflow result=%+v", result)
		}
		if len(user.Weapons)+len(user.Parts)+len(user.PartsStatusSubs)+len(user.WeaponNotes)+len(user.PartsGroupNotes) != 0 {
			t.Fatal("overflow left partial inventory or notes")
		}
	}
	for _, reward := range []store.GiftCommonState{
		{PossessionType: int32(model.PossessionTypeWeaponEnhanced), PossessionId: 9001, Count: 1},
		{PossessionType: int32(model.PossessionTypePartsEnhanced), PossessionId: 9002, Count: 1},
		{PossessionType: int32(model.PossessionTypeCompanionEnhanced), PossessionId: 9003, Count: 1},
	} {
		if result := grantGift(user, reward, g, config, 1000); result.Status != store.GrantStatusGranted {
			t.Fatalf("grant result=%+v", result)
		}
	}
	for _, weapon := range user.Weapons {
		if weapon.WeaponId != 101 || weapon.Level != 70 || weapon.Exp != 5000 || weapon.LimitBreakCount != 3 {
			t.Fatalf("weapon=%+v", weapon)
		}
	}
	for _, part := range user.Parts {
		if part.PartsId != 201 || part.Level != 15 || part.PartsStatusMainId != 8 {
			t.Fatalf("part=%+v", part)
		}
	}
	for _, companion := range user.Companions {
		if companion.CompanionId != 49 || companion.Level != 50 {
			t.Fatalf("companion=%+v", companion)
		}
	}
	if result := grantGift(user, store.GiftCommonState{PossessionType: 8, PossessionId: 9999, Count: 1}, g, config, 2000); result.Status != store.GrantStatusInvalid {
		t.Fatalf("missing template result=%+v", result)
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

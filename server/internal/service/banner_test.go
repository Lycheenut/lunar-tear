package service

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestMamaBannerExcludesTicketOnlyGachas(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("banner-visibility", model.ClientPlatform{})
	if err != nil {
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

	holder.Get().GachaEntries = []store.GachaCatalogEntry{
		visibleBannerEntry(45, 0, true),
		visibleBannerEntry(model.GachaIdGuaranteedThreeStarOrHigher, model.ConsumableIdGuaranteedThreeStarOrHigherTicket, false),
		visibleBannerEntry(model.GachaIdGuaranteedFourStar, model.ConsumableIdGuaranteedFourStarTicket, false),
	}
	bannerServer := NewBannerServiceServer(repo, repo, holder)
	gachaServer := NewGachaServiceServer(repo, repo, holder)

	assertBannerAndGachaIds(t, bannerServer, gachaServer, []int32{45}, []int32{45})
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.ConsumableItems[model.ConsumableIdGuaranteedFourStarTicket] = 1
	}); err != nil {
		t.Fatal(err)
	}
	assertBannerAndGachaIds(t, bannerServer, gachaServer, []int32{45}, []int32{45, model.GachaIdGuaranteedFourStar})

	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.ConsumableItems[model.ConsumableIdGuaranteedThreeStarOrHigherTicket] = 1
	}); err != nil {
		t.Fatal(err)
	}
	assertBannerAndGachaIds(t, bannerServer, gachaServer, []int32{45}, []int32{45, model.GachaIdGuaranteedThreeStarOrHigher, model.GachaIdGuaranteedFourStar})
}

func visibleBannerEntry(gachaId, requiredItemId int32, isMamaBanner bool) store.GachaCatalogEntry {
	return store.GachaCatalogEntry{
		GachaId:                  gachaId,
		IsMamaBanner:             isMamaBanner,
		GachaLabelType:           model.GachaLabelPremium,
		GachaModeType:            model.GachaModeBasic,
		IsUserGachaUnlock:        true,
		RequiredConsumableItemId: requiredItemId,
		UnlockConditions: []store.GachaUnlockConditionEntry{{
			GachaUnlockConditionType: model.GachaUnlockNone,
		}},
	}
}

func assertBannerAndGachaIds(t *testing.T, bannerServer *BannerServiceServer, gachaServer *GachaServiceServer, wantBanners, wantGachas []int32) {
	t.Helper()
	banners, err := bannerServer.GetMamaBanner(context.Background(), &pb.GetMamaBannerRequest{})
	if err != nil {
		t.Fatal(err)
	}
	gotBanners := make([]int32, 0, len(banners.TermLimitedGacha))
	for _, banner := range banners.TermLimitedGacha {
		gotBanners = append(gotBanners, banner.GachaId)
	}

	gachas, err := gachaServer.GetGachaList(context.Background(), &pb.GetGachaListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	gotGachas := make([]int32, 0, len(gachas.Gacha))
	for _, gacha := range gachas.Gacha {
		gotGachas = append(gotGachas, gacha.GachaId)
	}

	if !slices.Equal(gotBanners, wantBanners) {
		t.Fatalf("GetMamaBanner ids = %v, want %v", gotBanners, wantBanners)
	}
	if !slices.Equal(gotGachas, wantGachas) {
		t.Fatalf("GetGachaList ids = %v, want %v", gotGachas, wantGachas)
	}
}

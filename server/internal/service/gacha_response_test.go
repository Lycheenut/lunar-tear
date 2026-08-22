package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"

	"google.golang.org/protobuf/proto"
)

func TestChapterPromotionReportsConfiguredQuantityAndMonthlyProgress(t *testing.T) {
	entry := store.GachaCatalogEntry{
		GachaLabelType: model.GachaLabelChapter,
		PromotionItems: []store.GachaPromotionItem{{
			PossessionType:   int32(model.PossessionTypeMaterial),
			PossessionId:     100004,
			Count:            5,
			MaxDrawableCount: 20,
			CounterId:        1,
			IsTarget:         true,
		}},
	}
	bs := &store.GachaBannerState{BoxDrewCounts: map[int32]int32{
		model.ChapterGachaMonthCounterId: gametime.BusinessMonthKey(gametime.NowMillis()),
		1:                                3,
	}}
	items := buildProtoPromotionItems(entry, bs)
	if len(items) != 1 || items[0].GachaItem.Count != 5 || items[0].MaxDrawableCount != 20 || items[0].DrewCount != 3 {
		t.Fatalf("chapter promotion = %+v", items)
	}
}

func TestPremiumPickupPromotionPairsCostumeAndWeaponOrder(t *testing.T) {
	entry := store.GachaCatalogEntry{
		GachaLabelType: model.GachaLabelPremium,
		PromotionItems: []store.GachaPromotionItem{
			{
				PossessionType:      int32(model.PossessionTypeCostume),
				PossessionId:        32000,
				BonusPossessionType: int32(model.PossessionTypeWeapon),
				BonusPossessionId:   320081,
			},
			{
				PossessionType: int32(model.PossessionTypeWeapon),
				PossessionId:   320082,
			},
		},
	}

	items := buildProtoPromotionItems(entry, nil)
	if len(items) != 2 {
		t.Fatalf("promotion count = %d, want 2", len(items))
	}
	if items[0].GachaItem.PromotionOrder != 1 || items[0].GachaItemBonus.PromotionOrder != 1 {
		t.Fatalf("paired promotion orders = %d/%d, want 1/1", items[0].GachaItem.PromotionOrder, items[0].GachaItemBonus.PromotionOrder)
	}
	if items[1].GachaItem.PromotionOrder != 2 || items[1].GachaItemBonus.PromotionOrder != 0 {
		t.Fatalf("weapon-only promotion orders = %d/%d, want 2/0", items[1].GachaItem.PromotionOrder, items[1].GachaItemBonus.PromotionOrder)
	}
}

func TestEventBoxResponseReportsProgressionAndResetState(t *testing.T) {
	entry := store.GachaCatalogEntry{
		GachaId:                  300001,
		GachaLabelType:           model.GachaLabelEvent,
		GachaModeType:            model.GachaModeBox,
		BoxCount:                 3,
		IsCurrentBoxResettable:   true,
		IsResettableByAllTargets: true,
		IsInvalidReset:           false,
		GroupId:                  300001,
		PricePhases:              []store.GachaPricePhaseEntry{{PhaseId: 3000011}},
	}
	state := &store.GachaBannerState{BoxNumber: 2}
	mode := toProtoGacha(entry, state).GetGachaModeBoxComposition()
	if mode == nil || mode.BoxNumber != 3 || mode.CurrentBoxNumber != 2 || !mode.IsCurrentBoxResettable || !mode.IsResettableByDrawingAllTargets || mode.IsInvalidReset {
		t.Fatalf("event box response = %+v", mode)
	}
}

func TestGuaranteedFourStarGachaResponseMatchesConfirmBanner(t *testing.T) {
	holder := newGachaResponseTestHolder(t)

	var entry *store.GachaCatalogEntry
	for i := range holder.Get().GachaEntries {
		if holder.Get().GachaEntries[i].GachaId == model.GachaIdGuaranteedFourStar {
			entry = &holder.Get().GachaEntries[i]
			break
		}
	}
	if entry == nil {
		t.Fatal("four-star guaranteed Gacha is missing")
	}
	gacha := toProtoGacha(*entry, nil)
	if gacha.StartDatetime.AsTime().UnixMilli() == 0 || gacha.EndDatetime.AsTime().UnixMilli() == 0 {
		t.Fatalf("guaranteed Gacha response uses Unix epoch: %v..%v", gacha.StartDatetime, gacha.EndDatetime)
	}
	if len(gacha.GachaPricePhase) != 1 {
		t.Fatalf("price phase count = %d, want 1", len(gacha.GachaPricePhase))
	}
	phase := gacha.GachaPricePhase[0]
	if phase.PriceType != model.PriceTypeConsumableItem ||
		phase.PriceId != model.ConsumableIdGuaranteedFourStarTicket ||
		phase.Price != 1 || phase.RegularPrice != 0 {
		t.Fatalf("unexpected guaranteed Gacha price: %+v", phase)
	}

	mode := gacha.GetGachaModeBasic()
	if mode == nil {
		t.Fatal("four-star guaranteed Gacha is not basic mode")
	}
	want := []struct {
		costumeId int32
		weaponId  int32
	}{
		{32000, 320081},
		{35001, 350161},
		{33000, 330001},
	}
	if len(mode.PromotionGachaOddsItem) != len(want) {
		t.Fatalf("promotion count = %d, want %d", len(mode.PromotionGachaOddsItem), len(want))
	}
	for i, item := range mode.PromotionGachaOddsItem {
		if item.GachaItem.PossessionType != int32(model.PossessionTypeCostume) ||
			item.GachaItem.PossessionId != want[i].costumeId ||
			item.GachaItemBonus.PossessionType != int32(model.PossessionTypeWeapon) ||
			item.GachaItemBonus.PossessionId != want[i].weaponId {
			t.Fatalf("promotion %d = %+v, want costume %d with weapon %d", i, item, want[i].costumeId, want[i].weaponId)
		}
	}
}

func TestGuaranteedThreeStarGachaResponseUsesRequestedPromotions(t *testing.T) {
	holder := newGachaResponseTestHolder(t)

	var entry *store.GachaCatalogEntry
	for i := range holder.Get().GachaEntries {
		if holder.Get().GachaEntries[i].GachaId == model.GachaIdGuaranteedThreeStarOrHigher {
			entry = &holder.Get().GachaEntries[i]
			break
		}
	}
	if entry == nil {
		t.Fatal("three-star guaranteed Gacha is missing")
	}
	mode := toProtoGacha(*entry, nil).GetGachaModeBasic()
	if mode == nil {
		t.Fatal("three-star guaranteed Gacha is not basic mode")
	}
	want := []struct {
		costumeId int32
		weaponId  int32
	}{
		{22003, 220061},
		{21003, 210181},
	}
	if len(mode.PromotionGachaOddsItem) != len(want) {
		t.Fatalf("promotion count = %d, want %d", len(mode.PromotionGachaOddsItem), len(want))
	}
	for i, item := range mode.PromotionGachaOddsItem {
		if item.GachaItem.PossessionType != int32(model.PossessionTypeCostume) ||
			item.GachaItem.PossessionId != want[i].costumeId ||
			item.GachaItemBonus.PossessionType != int32(model.PossessionTypeWeapon) ||
			item.GachaItemBonus.PossessionId != want[i].weaponId {
			t.Fatalf("promotion %d = %+v, want costume %d with weapon %d", i, item, want[i].costumeId, want[i].weaponId)
		}
	}
}

func TestGuaranteedTicketDrawKeepsNextGachaUnlockedAfterLastTicket(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("ticket-draw-next-gacha", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.ConsumableItems[model.ConsumableIdGuaranteedFourStarTicket] = 1
	}); err != nil {
		t.Fatal(err)
	}

	holder := newGachaResponseTestHolder(t)
	var entry *store.GachaCatalogEntry
	for i := range holder.Get().GachaEntries {
		if holder.Get().GachaEntries[i].GachaId == model.GachaIdGuaranteedFourStar {
			entry = &holder.Get().GachaEntries[i]
			break
		}
	}
	if entry == nil || len(entry.PricePhases) != 1 {
		t.Fatal("four-star guaranteed Gacha price phase is missing")
	}
	entry.StartDatetime = 0
	entry.EndDatetime = 0

	server := NewGachaServiceServer(repo, repo, holder)
	response, err := server.Draw(context.Background(), &pb.DrawRequest{
		GachaId:           entry.GachaId,
		GachaPricePhaseId: entry.PricePhases[0].PhaseId,
		ExecCount:         1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.NextGacha == nil || !response.NextGacha.IsUserGachaUnlock {
		t.Fatalf("NextGacha must stay unlocked for result production: %+v", response.NextGacha)
	}
	wire, err := proto.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &pb.DrawResponse{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.GachaResult) != 1 || decoded.GachaResult[0].MedalBonus == nil {
		t.Fatalf("guaranteed Gacha result must include an empty medal bonus message: %+v", decoded.GachaResult)
	}

	user, err := repo.LoadUser(userId)
	if err != nil {
		t.Fatal(err)
	}
	if got := user.ConsumableItems[model.ConsumableIdGuaranteedFourStarTicket]; got != 0 {
		t.Fatalf("ticket balance = %d, want 0", got)
	}
	list, err := server.GetGachaList(context.Background(), &pb.GetGachaListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, gacha := range list.Gacha {
		if gacha.GachaId == model.GachaIdGuaranteedFourStar {
			t.Fatal("consumed ticket Gacha remained in a later list response")
		}
	}
}

func newGachaResponseTestHolder(t *testing.T) *runtime.Holder {
	t.Helper()
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
	return holder
}

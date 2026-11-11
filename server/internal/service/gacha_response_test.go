package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/gacha"
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

func TestChapterPromotionPreservesUnlimitedCount(t *testing.T) {
	entry := store.GachaCatalogEntry{
		GachaLabelType: model.GachaLabelChapter,
		GachaModeType:  model.GachaModeBox,
		PromotionItems: []store.GachaPromotionItem{{
			PossessionType: int32(model.PossessionTypeMaterial),
			PossessionId:   100003,
			Count:          6,
			CounterId:      1,
			IsTarget:       true,
		}},
	}
	wire, err := proto.Marshal(toProtoGacha(entry, nil))
	if err != nil {
		t.Fatal(err)
	}
	decoded := &pb.Gacha{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatal(err)
	}
	items := decoded.GetGachaModeBoxComposition().GetPromotionGachaOddsItem()
	if len(items) != 1 {
		t.Fatalf("promotion count = %d, want 1", len(items))
	}
	if items[0].MaxDrawableCount != 0 {
		t.Fatalf("unlimited promotion max drawable count = %d, want 0 for the client's infinity display", items[0].MaxDrawableCount)
	}
	if items[0].GachaItem.Count != 6 || items[0].DrewCount != 0 {
		t.Fatalf("unlimited chapter promotion = %+v", items[0])
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

func TestEventBoxResponseUsesLocalizedArtAndDefaultDescription(t *testing.T) {
	cat := newGachaResponseTestHolder(t).Get()
	for id, wantAsset := range map[int32]string{
		324001: "event_317_01", 324011: "event_317_11", 324021: "event_317_21",
		333001: "event_333_01", 334001: "event_334_01", 335001: "event_335_01",
	} {
		entry := findCatalogEntry(cat.GachaEntries, id)
		if entry == nil {
			t.Fatalf("missing event pool %d", id)
		}
		box := toProtoGacha(*entry, nil).GetGachaModeBoxComposition()
		if box == nil || box.GachaAssetName != wantAsset || box.GachaDescriptionTextId != 0 {
			t.Errorf("pool %d: got asset %q, description %d, want %q and default description (0)",
				id, box.GetGachaAssetName(), box.GetGachaDescriptionTextId(), wantAsset)
		}
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
	if mode == nil || mode.BoxNumber != 2 || mode.CurrentBoxNumber != 2 || !mode.IsCurrentBoxResettable || !mode.IsResettableByDrawingAllTargets || mode.IsInvalidReset {
		t.Fatalf("event box response = %+v", mode)
	}
	state.BoxNumber = 4 // A previous configuration had more boxes.
	mode = toProtoGacha(entry, state).GetGachaModeBoxComposition()
	if mode.BoxNumber != 3 || mode.CurrentBoxNumber != 3 {
		t.Fatalf("out-of-range event box = %+v", mode)
	}
}

func TestEventGachaListAndProgressionWithoutTickets(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	userID, err := repo.CreateUser("event-gacha-boxes", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}

	holder := newGachaResponseTestHolder(t)
	cat := holder.Get()
	const id = 329001
	entry := findCatalogEntry(cat.GachaEntries, id)
	if entry == nil || len(entry.PricePhases) == 0 {
		t.Fatal("event ticket pool is missing from the catalog")
	}
	now := gametime.NowMillis()
	entry.StartDatetime, entry.EndDatetime = now-60000, now+60000
	chapter := cat.Quest.EventChapterById[entry.RelatedEventQuestChapterId]
	chapter.StartDatetime, chapter.EndDatetime = entry.StartDatetime, entry.EndDatetime
	cat.Quest.EventChapterById[chapter.EventQuestChapterId] = chapter
	config := cat.GachaHandler.Premium.Config
	var boxes []gacha.BoxConfig
	for _, reward := range []int32{100004, 200004} {
		boxes = append(boxes, gacha.BoxConfig{
			GroupWeights: gacha.BoxGroupWeights{Limited: 10000},
			LimitedRewards: []gacha.BoxRewardConfig{{
				PossessionType: int32(model.PossessionTypeMaterial), PossessionId: reward,
				Count: 1, MaxCount: 1, Jackpot: true,
			}},
		})
	}
	config.EventBanners[id] = gacha.EventBoxConfig{Boxes: boxes}
	gacha.ApplyConfiguredBoxes(cat.GachaEntries, config)
	server := NewGachaServiceServer(repo, repo, holder)
	questIDs := cat.Quest.EventUnlockQuestIdsForChapter(chapter.EventQuestChapterId)
	if _, err := repo.UpdateUser(userID, func(user *store.UserState) {
		for _, questID := range questIDs {
			user.Quests[questID] = store.UserQuestState{QuestId: questID, QuestStateType: model.UserQuestStateTypeCleared}
		}
		delete(user.ConsumableItems, entry.RequiredConsumableItemId)
	}); err != nil {
		t.Fatal(err)
	}

	// CreateEventGacha selects the current box by number, then indexes each
	// box's pickups by that number when the player pages through the group.
	checkList := func(current int32, currentDrew int32) {
		t.Helper()
		list, err := server.GetGachaList(ctx, &pb.GetGachaListRequest{GachaLabelType: []int32{model.GachaLabelEvent}})
		if err != nil {
			t.Fatal(err)
		}
		seen := map[int32]bool{}
		for _, row := range list.Gacha {
			box := row.GetGachaModeBoxComposition()
			if box.GetGachaBoxGroupId() != entry.GroupId {
				continue
			}
			n := box.BoxNumber
			if row.GachaId != id || !row.IsUserGachaUnlock || seen[n] || n < 1 || n > int32(len(boxes)) || box.CurrentBoxNumber != current {
				t.Fatalf("invalid event box group member: %+v", row)
			}
			seen[n] = true
			wantDrew := int32(0)
			if n == current {
				wantDrew = currentDrew
			}
			items := box.PromotionGachaOddsItem
			if len(items) != 1 || items[0].GetGachaItem().GetPossessionId() != boxes[n-1].LimitedRewards[0].PossessionId || items[0].DrewCount != wantDrew {
				t.Fatalf("box %d, current %d: pickups = %v", n, current, items)
			}
		}
		if len(seen) != len(boxes) {
			t.Fatalf("received boxes %v, want both boxes including current %d", seen, current)
		}
	}
	checkList(1, 0)
	draw := &pb.DrawRequest{GachaId: id, GachaPricePhaseId: entry.PricePhases[0].PhaseId, ExecCount: 1}
	if _, err := server.Draw(ctx, draw); err == nil {
		t.Fatal("event draw succeeded without a ticket")
	}
	if _, err := repo.UpdateUser(userID, func(user *store.UserState) {
		user.ConsumableItems[entry.RequiredConsumableItemId] = 1
	}); err != nil {
		t.Fatal(err)
	}
	result, err := server.Draw(ctx, draw)
	if err != nil {
		t.Fatal(err)
	}
	if box := result.GetNextGacha().GetGachaModeBoxComposition(); box.GetBoxNumber() != 1 || box.GetCurrentBoxNumber() != 1 || !box.GetIsCurrentBoxResettable() {
		t.Fatalf("draw returned an unusable current box: %v", box)
	}
	checkList(1, 1)
	reset, err := server.ResetBoxGacha(ctx, &pb.ResetBoxGachaRequest{GachaId: id})
	if err != nil {
		t.Fatal(err)
	}
	if box := reset.GetGacha().GetGachaModeBoxComposition(); box.GetBoxNumber() != 2 || box.GetCurrentBoxNumber() != 2 {
		t.Fatalf("reset returned an unusable current box: %v", box)
	}
	checkList(2, 0)
	lookup, err := server.GetGacha(ctx, &pb.GetGachaRequest{GachaId: []int32{id}})
	if err != nil || !proto.Equal(lookup.GetGacha()[id], reset.Gacha) {
		t.Fatalf("lookup after reset differs from current box: %v, %v", lookup, err)
	}
}

func TestDailyGachaResponsePromotesAbstractMonsterAndGirl(t *testing.T) {
	holder := newGachaResponseTestHolder(t)
	var entry *store.GachaCatalogEntry
	for i := range holder.Get().GachaEntries {
		if holder.Get().GachaEntries[i].GachaId == model.GachaIdDaily {
			entry = &holder.Get().GachaEntries[i]
			break
		}
	}
	if entry == nil {
		t.Fatal("daily Gacha is missing")
	}
	wire, err := proto.Marshal(toProtoGacha(*entry, nil))
	if err != nil {
		t.Fatal(err)
	}
	decoded := &pb.Gacha{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatal(err)
	}
	items := decoded.GetGachaModeBasic().GetPromotionGachaOddsItem()
	want := []struct{ costumeId, weaponId int32 }{
		{35006, 350121}, // 形而上の怪物 / 孤高ノ拳
		{32003, 320111}, // 形而上の少女 / 血根の花瓶
	}
	if len(items) != len(want) {
		t.Fatalf("daily promotion count = %d, want %d", len(items), len(want))
	}
	for i, item := range items {
		if item.GetGachaItem().GetPossessionType() != int32(model.PossessionTypeCostume) ||
			item.GetGachaItem().GetPossessionId() != want[i].costumeId ||
			item.GetGachaItemBonus().GetPossessionType() != int32(model.PossessionTypeWeapon) ||
			item.GetGachaItemBonus().GetPossessionId() != want[i].weaponId ||
			item.GetGachaItem().GetPromotionOrder() != int32(i+1) ||
			item.GetGachaItemBonus().GetPromotionOrder() != int32(i+1) {
			t.Fatalf("daily promotion %d = %+v, want costume %d with weapon %d", i, item, want[i].costumeId, want[i].weaponId)
		}
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
	config := gacha.DefaultConfig()
	// The Aerie Warrior's weapon is event-only and must not stand in for Rion.
	config.Weapons[210031] = gacha.WeaponConfig{Availability: gacha.AvailabilityEvent}
	catalog, err := gacha.BuildPremiumCatalog(config, holder.Get().GachaPool, holder.Get().GachaEntries, gacha.BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	gacha.ApplyConfiguredPromotions(holder.Get().GachaEntries, catalog)

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
		{22001, 220021},
		{24001, 240121}, // Dissenting Exile / 異存たる亡命者
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

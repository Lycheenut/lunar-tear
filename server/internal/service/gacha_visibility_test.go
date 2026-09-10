package service

import (
	"context"
	"path/filepath"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestGuaranteedGachaIsVisibleOnlyWithMatchingTicket(t *testing.T) {
	tests := []struct {
		name          string
		gachaId       int32
		ticketId      int32
		otherTicketId int32
	}{
		{"three-star or higher", model.GachaIdGuaranteedThreeStarOrHigher, model.ConsumableIdGuaranteedThreeStarOrHigherTicket, model.ConsumableIdGuaranteedFourStarTicket},
		{"four-star", model.GachaIdGuaranteedFourStar, model.ConsumableIdGuaranteedFourStarTicket, model.ConsumableIdGuaranteedThreeStarOrHigherTicket},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := &runtime.Catalogs{}
			entry := store.GachaCatalogEntry{
				GachaId:                  tt.gachaId,
				IsUserGachaUnlock:        true,
				RequiredConsumableItemId: tt.ticketId,
				UnlockConditions: []store.GachaUnlockConditionEntry{{
					GachaUnlockConditionType: model.GachaUnlockNone,
				}},
			}
			user := &store.UserState{}
			user.EnsureMaps()

			if gachaVisibleForUser(cat, user, entry, 1) || gachaUnlocked(cat, user, entry, 1) {
				t.Fatal("guaranteed Gacha is available without its ticket")
			}
			user.ConsumableItems[tt.otherTicketId] = 1
			if gachaVisibleForUser(cat, user, entry, 1) || gachaUnlocked(cat, user, entry, 1) {
				t.Fatal("guaranteed Gacha is available with a different ticket")
			}
			user.ConsumableItems[tt.ticketId] = 1
			if !gachaVisibleForUser(cat, user, entry, 1) || !gachaUnlocked(cat, user, entry, 1) {
				t.Fatal("guaranteed Gacha is unavailable while its ticket is owned")
			}
		})
	}
}

func TestDailyGachaIsVisibleOnlyAfterItsUnlockQuest(t *testing.T) {
	cat := &runtime.Catalogs{}
	entry := store.GachaCatalogEntry{
		GachaId:            model.GachaIdDaily,
		GachaLabelType:     model.GachaLabelPremium,
		GachaAutoResetType: model.GachaAutoResetDaily,
		IsUserGachaUnlock:  true,
		PricePhases: []store.GachaPricePhaseEntry{{
			DrawCount:      model.DailyGachaDrawCount,
			LimitExecCount: model.DailyGachaExecLimit,
		}},
		UnlockConditions: []store.GachaUnlockConditionEntry{{
			GachaUnlockConditionType: model.GachaUnlockMainQuestClear,
			ConditionValue:           61,
		}},
	}
	user := &store.UserState{}
	user.EnsureMaps()

	if gachaVisibleForUser(cat, user, entry, 1) {
		t.Fatal("daily Gacha was visible before its unlock quest was cleared")
	}
	user.Quests[61] = store.UserQuestState{QuestStateType: model.UserQuestStateTypeCleared}
	if !gachaVisibleForUser(cat, user, entry, 1) {
		t.Fatal("daily Gacha remained hidden after its unlock quest was cleared")
	}
}

func TestDailyGachaVisibilityResetsAtBusinessDayBoundary(t *testing.T) {
	cat := &runtime.Catalogs{}
	entry := store.GachaCatalogEntry{
		GachaId:            model.GachaIdDaily,
		GachaLabelType:     model.GachaLabelPremium,
		GachaAutoResetType: model.GachaAutoResetDaily,
		IsUserGachaUnlock:  true,
		PricePhases: []store.GachaPricePhaseEntry{{
			DrawCount:      model.DailyGachaDrawCount,
			LimitExecCount: 2,
		}},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	reset := gametime.StartOfBusinessDayAtMillis(1788508800000)
	for _, tt := range []struct {
		name      string
		drawCount int32
		nowMillis int64
		visible   bool
	}{
		{"unused", 0, reset - 1, true},
		{"remaining execution", model.DailyGachaDrawCount, reset - 1, true},
		{"exhausted", 2 * model.DailyGachaDrawCount, reset - 1, false},
		{"over limit", 3 * model.DailyGachaDrawCount, reset - 1, false},
		{"next day", 2 * model.DailyGachaDrawCount, reset, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			user.Gacha.BannerStates[entry.GachaId] = store.GachaBannerState{
				DrawCount: tt.drawCount,
				BoxDrewCounts: map[int32]int32{
					model.DailyGachaDayCounterId: gametime.BusinessDayKey(reset - 1),
				},
			}
			if got := gachaVisibleForUser(cat, user, entry, tt.nowMillis); got != tt.visible {
				t.Fatalf("daily Gacha visible = %v, want %v", got, tt.visible)
			}
			if got := user.Gacha.BannerStates[entry.GachaId].DrawCount; got != tt.drawCount {
				t.Fatalf("visibility check changed stored draw count to %d", got)
			}
		})
	}
}

func TestDailyGachaResponseReportsCurrentExecutionAndNextReset(t *testing.T) {
	nowMillis := int64(1788508800000)
	entry := store.GachaCatalogEntry{
		GachaId:            model.GachaIdDaily,
		GachaLabelType:     model.GachaLabelPremium,
		GachaModeType:      model.GachaModeBasic,
		GachaAutoResetType: model.GachaAutoResetDaily,
		IsUserGachaUnlock:  true,
		PricePhases: []store.GachaPricePhaseEntry{{
			PhaseId:        model.DailyGachaPricePhaseId,
			DrawCount:      model.DailyGachaDrawCount,
			LimitExecCount: model.DailyGachaExecLimit,
		}},
		UnlockConditions: []store.GachaUnlockConditionEntry{{GachaUnlockConditionType: model.GachaUnlockNone}},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	entry = gachaForUser(&runtime.Catalogs{}, user, entry, nowMillis)
	wantReset := gametime.StartOfBusinessDayAtMillis(nowMillis) + 24*60*60*1000
	if entry.NextAutoResetDatetime != wantReset {
		t.Fatalf("next daily reset = %d, want %d", entry.NextAutoResetDatetime, wantReset)
	}

	state := store.GachaBannerState{
		DrawCount: model.DailyGachaDrawCount,
		BoxDrewCounts: map[int32]int32{
			model.DailyGachaDayCounterId: gametime.BusinessDayKey(nowMillis),
		},
	}
	proto := toProtoGacha(entry, &state)
	phase := proto.GachaPricePhase[0]
	if phase.UserExecCount != 1 || phase.IsEnabled || phase.EachMaxExecCount != model.DailyGachaExecLimit {
		t.Fatalf("completed daily phase = %+v, want one disabled execution", phase)
	}

	state.BoxDrewCounts[model.DailyGachaDayCounterId]--
	state = gachaBannerStateForUser(entry, state, nowMillis)
	proto = toProtoGacha(entry, &state)
	phase = proto.GachaPricePhase[0]
	if phase.UserExecCount != 0 || !phase.IsEnabled {
		t.Fatalf("reset daily phase = %+v, want enabled with zero executions", phase)
	}
}

func TestDailyGachaListAndPortalLookupFollowDrawAvailability(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := migrations.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	userId, err := repo.CreateUser("daily-gacha-visibility", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	holder := newGachaResponseTestHolder(t)
	entry := findCatalogEntry(holder.Get().GachaEntries, model.GachaIdDaily)
	if entry == nil || len(entry.PricePhases) != 1 {
		t.Fatal("daily Gacha price phase is missing")
	}
	server := NewGachaServiceServer(repo, repo, holder)
	assertVisible := func(want bool) {
		t.Helper()
		list, err := server.GetGachaList(ctx, &pb.GetGachaListRequest{GachaLabelType: []int32{model.GachaLabelPremium}})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, gacha := range list.Gacha {
			found = found || gacha.GachaId == entry.GachaId
		}
		if found != want {
			t.Fatalf("daily Gacha in list = %v, want %v", found, want)
		}
		portal, err := server.GetGacha(ctx, &pb.GetGachaRequest{GachaId: []int32{entry.GachaId}})
		if err != nil {
			t.Fatal(err)
		}
		gacha := portal.Gacha[entry.GachaId]
		if (gacha != nil) != want {
			t.Fatalf("daily Gacha at portal = %v, want visible %v", gacha, want)
		}
		if want && (gacha.GachaPricePhase[0].UserExecCount != 0 || !gacha.GachaPricePhase[0].IsEnabled) {
			t.Fatalf("available portal phase = %+v", gacha.GachaPricePhase[0])
		}
	}
	assertVisible(false)
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		for _, condition := range entry.UnlockConditions {
			if condition.GachaUnlockConditionType == model.GachaUnlockMainQuestClear {
				user.Quests[condition.ConditionValue] = store.UserQuestState{QuestStateType: model.UserQuestStateTypeCleared}
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	assertVisible(true)
	request := &pb.DrawRequest{GachaId: entry.GachaId, GachaPricePhaseId: entry.PricePhases[0].PhaseId, ExecCount: 1}
	draw, err := server.Draw(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if draw.NextGacha == nil || draw.NextGacha.GachaPricePhase[0].IsEnabled {
		t.Fatalf("completed daily Gacha result = %+v", draw.NextGacha)
	}
	assertVisible(false)
	if _, err := server.Draw(ctx, request); err == nil {
		t.Fatal("exhausted daily Gacha accepted another draw")
	}
	if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
		state := user.Gacha.BannerStates[entry.GachaId]
		state.BoxDrewCounts[model.DailyGachaDayCounterId] = gametime.BusinessDayKey(gametime.NowMillis()) - 1
		user.Gacha.BannerStates[entry.GachaId] = state
	}); err != nil {
		t.Fatal(err)
	}
	assertVisible(true)
}

func TestChapterGachaIsVisibleOnlyAfterUnlock(t *testing.T) {
	cat := &runtime.Catalogs{}
	entry := store.GachaCatalogEntry{
		GachaLabelType:    model.GachaLabelChapter,
		IsUserGachaUnlock: true,
		UnlockConditions: []store.GachaUnlockConditionEntry{{
			GachaUnlockConditionType: model.GachaUnlockMainQuestClear,
			ConditionValue:           10,
		}},
	}
	user := &store.UserState{}
	user.EnsureMaps()

	if gachaVisibleForUser(cat, user, entry, 1) {
		t.Fatal("locked chapter Gacha is visible")
	}
	user.Quests[10] = store.UserQuestState{QuestStateType: model.UserQuestStateTypeCleared}
	if !gachaVisibleForUser(cat, user, entry, 1) {
		t.Fatal("unlocked chapter Gacha is hidden")
	}
}

func TestChapterGachaIsVisibleOnlyForSelectedStoryRoute(t *testing.T) {
	quests := &masterdata.QuestCatalog{
		MainQuestRouteIdByChapterId: map[int32]int32{17: 2, 25: 3},
		SeasonIdByRouteId:           map[int32]int32{2: 2, 3: 2},
		RoutesBySeason:              map[int32][]int32{2: {2, 3}},
		RouteCompletionQuestId:      make(map[int32]int32),
	}
	cat := &runtime.Catalogs{
		Quest:        quests,
		QuestHandler: &questflow.QuestHandler{QuestCatalog: quests},
	}
	routeA := store.GachaCatalogEntry{
		GachaLabelType:            model.GachaLabelChapter,
		RelatedMainQuestChapterId: 17,
		IsUserGachaUnlock:         true,
		UnlockConditions: []store.GachaUnlockConditionEntry{{
			GachaUnlockConditionType: model.GachaUnlockNone,
		}},
	}
	routeB := routeA
	routeB.RelatedMainQuestChapterId = 25
	user := &store.UserState{}
	user.EnsureMaps()

	if gachaVisibleForUser(cat, user, routeA, 1) || gachaVisibleForUser(cat, user, routeB, 1) {
		t.Fatal("a second-season chapter Gacha was visible before choosing a route")
	}
	user.MainQuest.MainQuestSeasonId = 2
	user.MainQuest.CurrentMainQuestRouteId = 2
	if !gachaVisibleForUser(cat, user, routeA, 1) || gachaVisibleForUser(cat, user, routeB, 1) {
		t.Fatal("route 2 did not exclusively expose its Chapter Gacha group")
	}
	user.MainQuest.CurrentMainQuestRouteId = 3
	if gachaVisibleForUser(cat, user, routeA, 1) || !gachaVisibleForUser(cat, user, routeB, 1) {
		t.Fatal("route 3 did not exclusively expose its Chapter Gacha group")
	}
}

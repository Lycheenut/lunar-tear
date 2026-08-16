package service

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
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

func TestChapterGachaIsVisibleOnlyAfterUnlock(t *testing.T) {
	cat := &runtime.Catalogs{}
	entry := store.GachaCatalogEntry{
		GachaLabelType:    model.GachaLabelChapter,
		BoxCount:          1,
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
		BoxCount:                  1,
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

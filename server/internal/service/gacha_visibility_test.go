package service

import (
	"testing"

	"lunar-tear/server/internal/model"
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

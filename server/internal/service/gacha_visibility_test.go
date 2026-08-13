package service

import (
	"testing"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

func TestGuaranteedFourStarWeaponGachaIsVisibleOnlyWithTicket(t *testing.T) {
	cat := &runtime.Catalogs{}
	entry := store.GachaCatalogEntry{
		GachaId:                  model.GachaIdGuaranteedFourStarWeapon,
		IsUserGachaUnlock:        true,
		RequiredConsumableItemId: model.ConsumableIdGuaranteedFourStarWeaponTicket,
		UnlockConditions: []store.GachaUnlockConditionEntry{{
			GachaUnlockConditionType: model.GachaUnlockNone,
		}},
	}
	user := &store.UserState{}
	user.EnsureMaps()

	if gachaVisibleForUser(cat, user, entry, 1) {
		t.Fatal("guaranteed weapon Gacha is visible without its ticket")
	}
	if gachaUnlocked(cat, user, entry, 1) {
		t.Fatal("guaranteed weapon Gacha is unlocked without its ticket")
	}

	user.ConsumableItems[model.ConsumableIdGuaranteedFourStarWeaponTicket] = 1
	if !gachaVisibleForUser(cat, user, entry, 1) {
		t.Fatal("guaranteed weapon Gacha is hidden while its ticket is owned")
	}
	if !gachaUnlocked(cat, user, entry, 1) {
		t.Fatal("guaranteed weapon Gacha is locked while its ticket is owned")
	}
}

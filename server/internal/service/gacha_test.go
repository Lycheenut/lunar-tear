package service

import (
	"testing"

	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

func TestGachaForUserPreservesCalculatedLockState(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	entry := store.GachaCatalogEntry{
		IsUserGachaUnlock: true,
		UnlockConditions: []store.GachaUnlockConditionEntry{{
			GachaUnlockConditionType: model.GachaUnlockMainQuestClear,
			ConditionValue:           10,
		}},
	}
	if got := gachaForUser(&runtime.Catalogs{}, user, entry, 1); got.IsUserGachaUnlock {
		t.Fatal("uncleared quest unlocked gacha")
	}
	user.Quests[10] = store.UserQuestState{QuestStateType: model.UserQuestStateTypeCleared}
	if got := gachaForUser(&runtime.Catalogs{}, user, entry, 1); !got.IsUserGachaUnlock {
		t.Fatal("cleared quest did not unlock gacha")
	}
}

func TestAcquiredWeaponIdsIncludesSoldWeapons(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.WeaponNotes[100] = store.WeaponNoteState{
		WeaponId:                 100,
		FirstAcquisitionDatetime: 1,
	}

	weaponIds := acquiredWeaponIds(user)
	if !weaponIds[100] {
		t.Fatal("weapon acquisition history was ignored when inventory was empty")
	}
}

func TestIsNewWeaponInDrawMarksOnlyFirstDuplicateAsNew(t *testing.T) {
	acquiredWeapons := map[int32]bool{}

	if !isNewWeaponInDraw(100, acquiredWeapons) {
		t.Fatal("first copy of an unacquired weapon was not new")
	}
	if isNewWeaponInDraw(100, acquiredWeapons) {
		t.Fatal("second copy of the same weapon in one draw was new")
	}
}

func TestAutoConvertExpiredMedalsUsesMedalDeadlineAndTarget(t *testing.T) {
	const (
		gachaId     = int32(614)
		medalItemId = int32(8244)
		bookmarkId  = int32(2)
	)
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.ConsumableItems[medalItemId] = 7
	user.ConsumableItems[bookmarkId] = 3
	user.Gacha.BannerStates[gachaId] = store.GachaBannerState{GachaId: gachaId, MedalCount: 7}
	handler := &gacha.GachaHandler{
		Config: &masterdata.GameConfig{ConsumableItemIdForMedal: bookmarkId},
		MedalInfo: map[int32]masterdata.GachaMedalInfo{
			gachaId: {
				GachaMedalId:        medalItemId,
				ConsumableItemId:    medalItemId,
				AutoConvertDatetime: 200,
				ConversionRate:      2,
			},
		},
	}
	entry := store.GachaCatalogEntry{GachaId: gachaId, GachaMedalId: medalItemId, EndDatetime: 100}

	autoConvertExpiredMedals(user, []store.GachaCatalogEntry{entry}, handler, 150)
	if user.ConsumableItems[medalItemId] != 7 || user.ConsumableItems[bookmarkId] != 3 {
		t.Fatalf("medals converted before deadline: medals=%d bookmarks=%d", user.ConsumableItems[medalItemId], user.ConsumableItems[bookmarkId])
	}

	autoConvertExpiredMedals(user, []store.GachaCatalogEntry{entry}, handler, 200)
	if user.ConsumableItems[medalItemId] != 0 || user.ConsumableItems[bookmarkId] != 17 {
		t.Fatalf("converted balances = medals %d, bookmarks %d; want 0 and 17", user.ConsumableItems[medalItemId], user.ConsumableItems[bookmarkId])
	}
	if user.Gacha.BannerStates[gachaId].MedalCount != 0 {
		t.Fatal("converted banner medal count was not cleared")
	}
	converted := user.Gacha.ConvertedGachaMedal
	if len(converted.ConvertedMedalPossession) != 1 || converted.ConvertedMedalPossession[0] != (store.ConsumableItemState{ConsumableItemId: medalItemId, Count: 7}) {
		t.Fatalf("converted source record = %+v", converted.ConvertedMedalPossession)
	}
	if converted.ObtainPossession == nil || *converted.ObtainPossession != (store.ConsumableItemState{ConsumableItemId: bookmarkId, Count: 14}) {
		t.Fatalf("converted obtain record = %+v", converted.ObtainPossession)
	}
}

func TestAutoConvertDoesNotRunWhileGachaIsOpen(t *testing.T) {
	const (
		gachaId = int32(1)
		medalId = int32(8001)
	)
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.ConsumableItems[medalId] = 1
	user.Gacha.BannerStates[gachaId] = store.GachaBannerState{GachaId: gachaId, MedalCount: 1}
	handler := &gacha.GachaHandler{
		Config: &masterdata.GameConfig{ConsumableItemIdForMedal: 2},
		MedalInfo: map[int32]masterdata.GachaMedalInfo{
			gachaId: {GachaMedalId: medalId, ConsumableItemId: medalId, AutoConvertDatetime: 150, ConversionRate: 1},
		},
	}
	entry := store.GachaCatalogEntry{GachaId: gachaId, GachaMedalId: medalId, StartDatetime: 100, EndDatetime: 300}

	autoConvertExpiredMedals(user, []store.GachaCatalogEntry{entry}, handler, 200)
	if user.ConsumableItems[medalId] != 1 || user.Gacha.BannerStates[gachaId].MedalCount != 1 {
		t.Fatal("an open gacha had its medals auto-converted")
	}
}

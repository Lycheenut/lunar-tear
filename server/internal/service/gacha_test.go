package service

import (
	"testing"

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

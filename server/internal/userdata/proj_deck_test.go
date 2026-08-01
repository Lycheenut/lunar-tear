package userdata

import (
	"testing"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestDeckLimitContentRestrictedProjectionUsesState(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.DeckLimitContentRestricted["id"] = store.DeckLimitContentRestrictedState{DeckRestrictedUuid: "id", EventQuestChapterId: 30}
	if len(sortedDeckLimitContentRestrictedRecords(*user)) != 1 {
		t.Fatal("limit content projection was empty")
	}
}

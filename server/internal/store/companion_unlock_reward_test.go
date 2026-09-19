package store

import (
	"reflect"
	"testing"

	"lunar-tear/server/internal/model"
)

func TestCompanionUnlockRewardPreservesOwnedAndRepairsLevels(t *testing.T) {
	user := SeedUserState(1, "test", 1, model.ClientPlatform{})
	granter := &PossessionGranter{CompanionDupExchange: map[int32][]model.DupExchangeEntry{}}
	for id := int32(31); id <= 53; id++ {
		granter.CompanionDupExchange[id] = []model.DupExchangeEntry{{PossessionType: 6, PossessionId: 601, Count: 50}}
	}
	for id, level := range map[int32]int32{2: 10, 31: 37, 49: 1, 50: 25, 51: 50, 53: 15} {
		granter.grantCompanion(user, id, level, 100)
	}
	before := make(map[string]CompanionState)
	for key, companion := range user.Companions {
		before[key] = companion
	}
	GrantCompanionUnlockReward(user, granter, 1000)
	if len(user.Companions) != 24 {
		t.Fatalf("companions=%d, want 24", len(user.Companions))
	}
	for key, original := range before {
		want := original
		if want.CompanionId == 49 || want.CompanionId == 50 {
			want.Level, want.LatestVersion = 50, 1000
		}
		if user.Companions[key] != want {
			t.Fatalf("existing companion changed unexpectedly: got=%+v want=%+v", user.Companions[key], want)
		}
	}
	after := make(map[string]CompanionState)
	for key, companion := range user.Companions {
		after[key] = companion
	}
	GrantCompanionUnlockReward(user, granter, 2000)
	if !reflect.DeepEqual(after, user.Companions) {
		t.Fatal("repeated backfill changed companions")
	}
	if user.ConsumableItems[601] != 0 {
		t.Fatal("backfill converted owned companions into duplicate rewards")
	}
}

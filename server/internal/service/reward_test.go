package service

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/importantitem"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

func TestBigHuntDailyRewardCountUsesImportantItemEffect(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatalf("initialize master data: %v", err)
	}
	effects, err := importantitem.Load()
	if err != nil {
		t.Fatalf("load important-item effects: %v", err)
	}
	cat := &runtime.Catalogs{ImportantItems: effects}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	item := masterdata.RewardItem{
		PossessionType: int32(model.PossessionTypeMaterial),
		PossessionId:   999,
		Count:          3,
	}
	const nowMillis = int64(1787241600000)

	if got := bigHuntDailyRewardCount(cat, user, item, nowMillis); got != 3 {
		t.Fatalf("daily reward without important item = %d, want 3", got)
	}
	user.ImportantItems[200020] = 1
	if got := bigHuntDailyRewardCount(cat, user, item, nowMillis); got != 6 {
		t.Fatalf("daily reward with important item = %d, want 6", got)
	}
}

func TestClaimMissionPassRemainingRewardReturnsEmptySuccessWhenNothingPending(t *testing.T) {
	cat := &runtime.Catalogs{Mission: &masterdata.MissionCatalog{PassById: map[int32]masterdata.MissionPassCatalog{
		7: {Definition: masterdata.EntityMMissionPass{MissionPassId: 7, EndDatetime: 100}},
	}}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.MissionPassRemaining[7] = store.MissionPassRemainingState{MissionPassId: 7, RewardReceived: true}

	passId, err := claimMissionPassRemainingReward(cat, user, 200)
	if err != nil {
		t.Fatalf("empty remaining reward was rejected: %v", err)
	}
	if passId != 0 {
		t.Fatalf("received mission pass id = %d, want 0", passId)
	}
}

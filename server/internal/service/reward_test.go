package service

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

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

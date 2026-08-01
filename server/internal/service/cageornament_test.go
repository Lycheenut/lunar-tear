package service

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestCageOrnamentClaimIsOneShot(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	reward := masterdata.CageOrnamentReward{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 10, Count: 2}
	granter := &store.PossessionGranter{}
	if !claimCageOrnamentReward(user, 1, reward, granter, 100) {
		t.Fatal("first claim rejected")
	}
	if claimCageOrnamentReward(user, 1, reward, granter, 200) {
		t.Fatal("repeat claim accepted")
	}
	if user.Materials[10] != 2 || user.CageOrnamentRewards[1].AcquisitionDatetime != 100 {
		t.Fatalf("claim state = materials %d, state %+v", user.Materials[10], user.CageOrnamentRewards[1])
	}
}

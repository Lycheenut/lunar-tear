package userdata

import (
	"testing"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestCostumeLotteryEffectProjectionsUsePersistedResults(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.CostumeLotteryEffectAbilities[store.CostumeLotteryEffectKey{UserCostumeUuid: "ability", SlotNumber: 1}] = store.CostumeLotteryEffectAbilityState{UserCostumeUuid: "ability", SlotNumber: 1, AbilityId: 100, AbilityLevel: 2, LatestVersion: 10}
	user.CostumeLotteryEffectStatusUps[store.CostumeLotteryEffectStatusKey{UserCostumeUuid: "status", StatusCalculationType: model.StatusCalculationTypeAdd}] = store.CostumeLotteryEffectStatusUpState{UserCostumeUuid: "status", StatusCalculationType: model.StatusCalculationTypeAdd, Attack: 20, LatestVersion: 11}
	if records := sortedCostumeLotteryAbilityRecords(*user); len(records) != 1 || records[0]["abilityId"] != int32(100) {
		t.Fatalf("ability projection = %#v", records)
	}
	if records := sortedCostumeLotteryStatusRecords(*user); len(records) != 1 || records[0]["attack"] != int32(20) {
		t.Fatalf("status projection = %#v", records)
	}
}

func TestCostumeLevelBonusProjectionUsesState(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.CostumeLevelBonusReleaseStatuses[10] = store.CostumeLevelBonusReleaseStatusState{CostumeId: 10, ConfirmedBonusLevel: 50}
	if len(sortedCostumeLevelBonusRecords(*user)) != 1 {
		t.Fatal("costume level bonus projection was empty")
	}
}

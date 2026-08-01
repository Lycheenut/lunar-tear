package service

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestCostumeLotteryResultsArePersistedFromConfirmedEffects(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Costumes["costume"] = store.CostumeState{UserCostumeUuid: "costume", CostumeId: 100}
	user.CostumeLotteryEffects[store.CostumeLotteryEffectKey{UserCostumeUuid: "costume", SlotNumber: 1}] = store.CostumeLotteryEffectState{UserCostumeUuid: "costume", SlotNumber: 1, OddsNumber: 1}
	user.CostumeLotteryEffects[store.CostumeLotteryEffectKey{UserCostumeUuid: "costume", SlotNumber: 2}] = store.CostumeLotteryEffectState{UserCostumeUuid: "costume", SlotNumber: 2, OddsNumber: 2}
	catalog := &masterdata.CostumeCatalog{
		LotteryEffects: map[[2]int32]masterdata.EntityMCostumeLotteryEffect{
			{100, 1}: {CostumeId: 100, SlotNumber: 1, CostumeLotteryEffectOddsGroupId: 10},
			{100, 2}: {CostumeId: 100, SlotNumber: 2, CostumeLotteryEffectOddsGroupId: 20},
		},
		LotteryEffectOddsByNumber: map[[2]int32]masterdata.EntityMCostumeLotteryEffectOddsGroup{
			{10, 1}: {CostumeLotteryEffectOddsGroupId: 10, OddsNumber: 1, CostumeLotteryEffectType: int32(model.CostumeLotteryEffectTypeAbility), CostumeLotteryEffectTargetId: 30},
			{20, 2}: {CostumeLotteryEffectOddsGroupId: 20, OddsNumber: 2, CostumeLotteryEffectType: int32(model.CostumeLotteryEffectTypeStatusUp), CostumeLotteryEffectTargetId: 40},
		},
		LotteryEffectTargetAbilities: map[int32]masterdata.EntityMCostumeLotteryEffectTargetAbility{
			30: {CostumeLotteryEffectTargetAbilityId: 30, AbilityId: 300, AbilityLevel: 2},
		},
		LotteryEffectTargetStatusUps: map[int32][]masterdata.EntityMCostumeLotteryEffectTargetStatusUp{
			40: {
				{CostumeLotteryEffectTargetStatusUpId: 40, StatusKindType: int32(model.StatusKindTypeAttack), StatusCalculationType: int32(model.StatusCalculationTypeAdd), EffectValue: 25},
				{CostumeLotteryEffectTargetStatusUpId: 40, StatusKindType: int32(model.StatusKindTypeHp), StatusCalculationType: int32(model.StatusCalculationTypeAdd), EffectValue: 50},
			},
		},
	}

	recomputeCostumeLotteryEffectResults(user, catalog, "costume", 1000)
	ability := user.CostumeLotteryEffectAbilities[store.CostumeLotteryEffectKey{UserCostumeUuid: "costume", SlotNumber: 1}]
	if ability.AbilityId != 300 || ability.AbilityLevel != 2 {
		t.Fatalf("ability result = %+v", ability)
	}
	statusResult := user.CostumeLotteryEffectStatusUps[store.CostumeLotteryEffectStatusKey{UserCostumeUuid: "costume", StatusCalculationType: model.StatusCalculationTypeAdd}]
	if statusResult.Attack != 25 || statusResult.Hp != 50 {
		t.Fatalf("status result = %+v", statusResult)
	}
}

func TestCostumeLevelBonusStatusIsMonotonic(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	registerCostumeLevelBonusStatus(user, 10, 60, 60, 100)
	registerCostumeLevelBonusStatus(user, 10, 50, 50, 200)
	state := user.CostumeLevelBonusReleaseStatuses[10]
	if state.ConfirmedBonusLevel != 60 || state.LastReleasedBonusLevel != 60 {
		t.Fatalf("level bonus state = %+v", state)
	}
}

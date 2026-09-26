package service

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestCostumeLotteryCampaignBoostsFourStarProbability(t *testing.T) {
	_, campaigns := loadCampaignAndLoginBonusCatalogs(t)
	now := time.Date(2026, time.August, 10, 13, 30, 0, 0, time.UTC).UnixMilli()
	day := int64(24 * time.Hour / time.Millisecond)
	pool := []masterdata.EntityMCostumeLotteryEffectOddsGroup{
		{OddsNumber: 1, RarityType: 40, Weight: 5},
		{OddsNumber: 2, RarityType: 40, Weight: 15},
		{OddsNumber: 3, RarityType: 30, Weight: 196},
		{OddsNumber: 4, RarityType: 50, Weight: 784},
	}
	original := append([]masterdata.EntityMCostumeLotteryEffectOddsGroup(nil), pool...)
	for _, test := range []struct {
		name       string
		registered int64
		comeback   int64
		at         int64
		unlocked   bool
		wantPermil int64
	}{
		{"ordinary", now - 100*day, 0, now, true, 20},
		{"beginner", now - day, 0, now, true, 100},
		{"comeback", now - 100*day, now - day, now, true, 60},
		{"locked", now - day, 0, now, false, 20},
		{"expired", now - day, 0, 1 << 62, true, 20},
	} {
		t.Run(test.name, func(t *testing.T) {
			user := store.SeedUserState(1, "test", test.registered, model.ClientPlatform{})
			user.Login.LastComebackLoginDatetime = test.comeback
			if test.unlocked {
				user.Quests[1] = store.UserQuestState{QuestId: 1, QuestStateType: model.UserQuestStateTypeCleared}
			}
			bonus := campaigns.CostumeRateBonus(campaign.CostumeTarget{}, enhancementCampaignFilter(campaigns, user, test.at))
			for _, roll := range []int64{test.wantPermil*1000 - 1, test.wantPermil * 1000} {
				picked := drawCostumeLotteryEffect(pool, bonus, func(n int64) int64 {
					if n == 1000000 {
						return roll
					}
					return 0
				})
				if got := picked.RarityType == 40; got != (roll < test.wantPermil*1000) {
					t.Fatalf("roll=%d picked=%+v, want four-star cutoff %d", roll, picked, test.wantPermil*1000)
				}
			}
			random := rand.New(rand.NewSource(123))
			counts := map[int32]int{}
			for range 100000 {
				counts[drawCostumeLotteryEffect(pool, bonus, random.Int63n).OddsNumber]++
			}
			for _, group := range []struct {
				first, second int32
				want          float64
			}{{1, 2, 0.25}, {3, 4, 0.2}} {
				got := float64(counts[group.first]) / float64(counts[group.first]+counts[group.second])
				if math.Abs(got-group.want) > 0.03 {
					t.Fatalf("relative weights changed: %v", counts)
				}
			}
		})
	}
	if !reflect.DeepEqual(pool, original) {
		t.Fatal("campaign changed shared master-data weights")
	}
}

func TestCostumeLotteryPreservesFractionalOddsAndSingleRarityPools(t *testing.T) {
	for _, pool := range [][]masterdata.EntityMCostumeLotteryEffectOddsGroup{
		{{OddsNumber: 1, RarityType: 40, Weight: 1}, {OddsNumber: 2, RarityType: 30, Weight: 2}},
		{{OddsNumber: 1, RarityType: 40, Weight: 1}},
		{{OddsNumber: 1, RarityType: 30, Weight: 1}},
	} {
		for _, last := range []bool{false, true} {
			picked := drawCostumeLotteryEffect(pool, campaign.RateBonus{}, func(n int64) int64 {
				if n == 3000 {
					// Exactly 1/3, not the truncated 333/1000.
					return 999
				}
				if last {
					return n - 1
				}
				return 0
			})
			if picked.OddsNumber != 1 {
				t.Fatalf("unexpected draw: %+v from %+v", picked, pool)
			}
		}
	}
}

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

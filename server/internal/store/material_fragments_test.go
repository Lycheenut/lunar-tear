package store

import (
	"math"
	"testing"

	"lunar-tear/server/internal/model"
)

func TestGrantPossessionConvertsMaterialFragments(t *testing.T) {
	for _, item := range []struct {
		name       string
		fragmentId int32
		materialId int32
	}{
		{"zenith", 501001, 322002},
		{"black_pearl", 501002, 312003},
	} {
		t.Run(item.name, func(t *testing.T) {
			for _, test := range []struct {
				name      string
				initial   int32
				grants    []int32
				remaining int32
				converted int32
			}{
				{"below_threshold", 0, []int32{9}, 9, 0},
				{"exact_threshold", 0, []int32{10}, 0, 1},
				{"accumulated", 8, []int32{1, 1}, 0, 1},
				{"batch_with_remainder", 8, []int32{27}, 5, 3},
				{"multiple_batches", 0, []int32{25, 5}, 0, 3},
				{"legacy_stock", 25, []int32{1}, 6, 2},
				{"large_batch", 9, []int32{math.MaxInt32}, 6, 214748365},
			} {
				t.Run(test.name, func(t *testing.T) {
					user := SeedUserState(1, "fragments", 1, model.ClientPlatform{})
					user.Materials[item.fragmentId] = test.initial
					user.Materials[item.materialId] = 4
					for _, count := range test.grants {
						GrantPossession(user, model.PossessionTypeMaterial, item.fragmentId, count)
					}
					if got := user.Materials[item.fragmentId]; got != test.remaining {
						t.Fatalf("remaining fragments = %d, want %d", got, test.remaining)
					}
					if got := user.Materials[item.materialId]; got != 4+test.converted {
						t.Fatalf("materials = %d, want %d", got, 4+test.converted)
					}
					var obtained int32
					for _, event := range user.PendingMissionEvents {
						if event.ConditionType == int32(model.MissionClearConditionTypePossessionAddByCount) &&
							event.TargetId == item.materialId && event.OptionGroupId == int32(model.PossessionTypeMaterial) {
							obtained += event.Count
						}
					}
					if obtained != test.converted {
						t.Fatalf("obtained material mission count = %d, want %d", obtained, test.converted)
					}
				})
			}
		})
	}
}

func TestMaterialFragmentConversionDoesNotAffectOtherPossessions(t *testing.T) {
	user := SeedUserState(1, "fragments", 1, model.ClientPlatform{})
	GrantPossession(user, model.PossessionTypeMaterial, 501003, 25)
	GrantPossession(user, model.PossessionTypeMaterial, 312003, 25)
	GrantPossession(user, model.PossessionTypeConsumableItem, 501002, 25)
	if user.Materials[501003] != 25 || user.Materials[312003] != 25 || user.ConsumableItems[501002] != 25 {
		t.Fatal("conversion changed an unrelated possession")
	}
}

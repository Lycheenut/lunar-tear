package masterdata

import "testing"

func TestCostumeAbilityLevelUsesLimitBreakThresholds(t *testing.T) {
	catalog := &CostumeCatalog{
		Costumes: map[int32]EntityMCostume{1: {CostumeId: 1, CostumeAbilityGroupId: 10}},
		AbilityGroupsByGroupId: map[int32][]EntityMCostumeAbilityGroup{
			10: {{CostumeAbilityGroupId: 10, CostumeAbilityLevelGroupId: 20}},
		},
		AbilityLevelsByGroupId: map[int32][]EntityMCostumeAbilityLevelGroup{
			20: {
				{CostumeAbilityLevelGroupId: 20, CostumeLimitBreakCountLowerLimit: 0, AbilityLevel: 1},
				{CostumeAbilityLevelGroupId: 20, CostumeLimitBreakCountLowerLimit: 2, AbilityLevel: 3},
			},
		},
	}
	if got := catalog.AbilityLevel(1, 1); got != 1 {
		t.Fatalf("ability level before threshold = %d, want 1", got)
	}
	if got := catalog.AbilityLevel(1, 2); got != 3 {
		t.Fatalf("ability level at threshold = %d, want 3", got)
	}
}

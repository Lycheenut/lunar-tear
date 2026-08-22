package questdrop

import (
	"strings"
	"testing"

	"lunar-tear/server/internal/masterdata"
)

func TestBuildOverridesLeavesLegacyPoolsUntouchedAndAppliesConfiguredPool(t *testing.T) {
	const (
		questID = int32(10)
		groupID = int32(20)
	)
	catalog := &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			questID: {QuestId: questID, QuestPickupRewardGroupId: groupID},
		},
		PickupRewardIdsByGroupId: map[int32][]int32{groupID: {1001, 1001, 2001}},
		BattleDropRewardById: map[int32]masterdata.EntityMBattleDropReward{
			1001: {BattleDropRewardId: 1001},
			2001: {BattleDropRewardId: 2001},
		},
	}

	legacy, err := BuildOverrides(DefaultConfig(), catalog, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := legacy[questID]; exists {
		t.Fatalf("unconfigured quest was normalized into an override: %+v", legacy[questID])
	}

	config := DefaultConfig()
	config.Quests[questID] = QuestConfig{Rewards: []Reward{
		{BattleDropRewardID: 1001, Weight: 7},
		{BattleDropRewardID: 2001, Weight: 3},
	}}
	configured, err := BuildOverrides(config, catalog, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := configured[questID]; len(got) != 2 || got[0].Weight != 7 || got[1].Weight != 3 {
		t.Fatalf("configured pool = %+v", got)
	}
}

func TestBuildOverridesRejectsDuplicateRewardsAndInvalidWeights(t *testing.T) {
	const questID = int32(10)
	catalog := &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{questID: {QuestId: questID}},
		BattleDropRewardById: map[int32]masterdata.EntityMBattleDropReward{
			1001: {BattleDropRewardId: 1001},
		},
	}
	tests := []struct {
		name    string
		rewards []Reward
		want    string
	}{
		{"duplicate", []Reward{{BattleDropRewardID: 1001, Weight: 1}, {BattleDropRewardID: 1001, Weight: 2}}, "duplicate"},
		{"zero weight", []Reward{{BattleDropRewardID: 1001, Weight: 0}}, "weight must be"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Quests[questID] = QuestConfig{Rewards: test.rewards}
			_, err := BuildOverrides(config, catalog, BuildOptions{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("BuildOverrides error = %v, want text %q", err, test.want)
			}
		})
	}
}

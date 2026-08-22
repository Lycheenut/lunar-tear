package runtime_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/questdrop"
	"lunar-tear/server/internal/runtime"
)

func TestHolderLoadsQuestDropOverrides(t *testing.T) {
	source := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	original, err := os.ReadFile(source)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	}
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	masterDataPath := filepath.Join(directory, "current.bin.e")
	configPath := filepath.Join(directory, "quest_drops.json")
	if err := os.WriteFile(masterDataPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	baseline, err := runtime.NewHolder(masterDataPath)
	if err != nil {
		t.Fatal(err)
	}
	var questID, rewardID int32
	for candidateQuestID, quest := range baseline.Get().Quest.QuestById {
		pool := baseline.Get().Quest.PickupRewardIdsByGroupId[quest.QuestPickupRewardGroupId]
		if len(pool) > 0 {
			questID, rewardID = candidateQuestID, pool[0]
			break
		}
	}
	if questID == 0 || rewardID == 0 {
		t.Fatal("master data has no quest pickup reward")
	}
	config := questdrop.DefaultConfig()
	config.Quests[questID] = questdrop.QuestConfig{Rewards: []questdrop.Reward{{
		BattleDropRewardID: rewardID,
		Weight:             7,
	}}}
	encoded, _, err := questdrop.EncodeConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	holder, err := runtime.NewHolderWithConfigs(masterDataPath, "", configPath)
	if err != nil {
		t.Fatal(err)
	}
	pool := holder.Get().QuestHandler.DropRewardsByQuestID[questID]
	if len(pool) != 1 || pool[0].BattleDropRewardID != rewardID || pool[0].Weight != 7 {
		t.Fatalf("loaded quest drop override = %+v", pool)
	}
}

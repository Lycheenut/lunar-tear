package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestLoadGameConfigIncludesQuestMissionBigWinBonusPower(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadGameConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.QuestMissionBigWinBonusPower != 30000 {
		t.Fatalf("quest mission big win bonus power = %d, want 30000", cfg.QuestMissionBigWinBonusPower)
	}
	if cfg.FunctionUnlockQuestIdForDailyGacha != 61 {
		t.Fatalf("daily Gacha unlock quest = %d, want 61", cfg.FunctionUnlockQuestIdForDailyGacha)
	}
}

package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestGimmickOrnamentRewardsLoadFromMasterData(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	if rewards := loadGimmickOrnamentRewards(LoadCageOrnamentCatalog()); len(rewards) == 0 {
		t.Fatal("gimmick ornament reward mapping was empty")
	}
}

func TestGimmickOrnamentRewardLookupHasNoFallback(t *testing.T) {
	catalog := &GimmickCatalog{ornamentRewards: map[GimmickOrnamentRef]SequenceReward{{GimmickId: 7, OrnamentIndex: 1}: {PossessionType: 5, PossessionId: 9, Count: 2}}}
	if reward, ok := catalog.OrnamentReward(7, 1); !ok || reward.PossessionId != 9 {
		t.Fatalf("mapped reward = %+v, %v", reward, ok)
	}
	if _, ok := catalog.OrnamentReward(8, 1); ok {
		t.Fatal("unmapped ornament received a fallback")
	}
}

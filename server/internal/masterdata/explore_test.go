package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestExploreMappingsLoadFromMasterData(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadExploreCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if catalog.UnlockQuestIds[1] == 0 || catalog.LowerDifficulty[11] != 1 {
		t.Fatalf("explore unlock mappings = quests %#v, difficulty %#v", catalog.UnlockQuestIds, catalog.LowerDifficulty)
	}
}

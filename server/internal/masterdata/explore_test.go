package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestExploreDifficultyMappingsLoadFromMasterData(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadExploreCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if catalog.LowerDifficulty[11] != 1 {
		t.Fatalf("explore difficulty mappings = %#v", catalog.LowerDifficulty)
	}
	if catalog.FirstExploreId != 1 {
		t.Fatalf("first explore ID = %d, want 1", catalog.FirstExploreId)
	}
}

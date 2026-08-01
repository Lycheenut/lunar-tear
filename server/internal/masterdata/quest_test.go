package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestLoadQuestCatalogResolvesEventUnlockQuests(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	parts, err := LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := LoadConditionResolver()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.EventChapterById) == 0 || len(catalog.EventUnlockQuestIdsByType) == 0 {
		t.Fatal("event chapters or normalized unlock quests were not loaded")
	}
}

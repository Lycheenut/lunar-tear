package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
)

func TestLoadGachaCatalogBuildsScopedEventBoxes(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	parts, err := LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog(parts)
	if err != nil {
		t.Fatal(err)
	}
	var eventCount int
	for _, entry := range entries {
		if entry.GachaLabelType != model.GachaLabelEvent {
			continue
		}
		eventCount++
		if len(entry.BoxItems) == 0 {
			t.Fatalf("event gacha %d has no scoped box items", entry.GachaId)
		}
	}
	if eventCount == 0 {
		t.Fatal("catalog did not contain an event gacha")
	}
}

package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
)

func TestLoadGachaCatalogDoesNotSynthesizeEventBoxInventory(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.GachaLabelType == model.GachaLabelEvent {
			t.Fatalf("event gacha %d was exposed without authoritative inventory", entry.GachaId)
		}
	}
	if len(entries) == 0 {
		t.Fatal("non-event gachas were removed with event gachas")
	}
}

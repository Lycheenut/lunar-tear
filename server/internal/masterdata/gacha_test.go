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

func TestLoadGachaCatalogIncludesGuaranteedFourStarWeaponGacha(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.GachaId != model.GachaIdGuaranteedFourStarWeapon {
			continue
		}
		if entry.BannerAssetName != guaranteedFourStarWeaponGachaAssetName ||
			entry.RequiredConsumableItemId != model.ConsumableIdGuaranteedFourStarWeaponTicket {
			t.Fatalf("unexpected guaranteed weapon Gacha: %+v", entry)
		}
		if len(entry.PricePhases) != 1 {
			t.Fatalf("price phase count = %d, want 1", len(entry.PricePhases))
		}
		phase := entry.PricePhases[0]
		if phase.PhaseId != 600031 ||
			phase.PriceType != model.PriceTypeConsumableItem ||
			phase.PriceId != model.ConsumableIdGuaranteedFourStarWeaponTicket ||
			phase.Price != 1 || phase.DrawCount != 1 ||
			phase.FixedRarityMin != model.RaritySSRare || phase.FixedCount != 1 {
			t.Fatalf("unexpected guaranteed weapon price phase: %+v", phase)
		}
		return
	}
	t.Fatalf("Gacha %d was not added to the catalog", model.GachaIdGuaranteedFourStarWeapon)
}

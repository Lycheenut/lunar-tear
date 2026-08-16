package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestLoadGachaCatalogDoesNotSynthesizeEventBoxInventory(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog()
	if err != nil {
		t.Fatal(err)
	}
	hasMamaBanner := false
	for _, entry := range entries {
		if entry.GachaLabelType == model.GachaLabelEvent {
			t.Fatalf("event gacha %d was exposed without authoritative inventory", entry.GachaId)
		}
		hasMamaBanner = hasMamaBanner || entry.IsMamaBanner
	}
	if len(entries) == 0 {
		t.Fatal("non-event gachas were removed with event gachas")
	}
	if !hasMamaBanner {
		t.Fatal("m_mom_banner entries were not marked as Mama banners")
	}
}

func TestLoadGachaCatalogIncludesGuaranteedTicketGachas(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	entries, _, err := LoadGachaCatalog()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		gachaId       int32
		ticketId      int32
		assetName     string
		minimumRarity model.RarityType
	}{
		{model.GachaIdGuaranteedThreeStarOrHigher, model.ConsumableIdGuaranteedThreeStarOrHigherTicket, guaranteedThreeStarOrHigherGachaAssetName, model.RaritySRare},
		{model.GachaIdGuaranteedFourStar, model.ConsumableIdGuaranteedFourStarTicket, guaranteedFourStarGachaAssetName, model.RaritySSRare},
	}
	byId := make(map[int32]store.GachaCatalogEntry, len(entries))
	maxMasterSortOrder := int32(0)
	var catalogStartDatetime int64
	var catalogEndDatetime int64
	for _, entry := range entries {
		byId[entry.GachaId] = entry
		if model.IsGuaranteedTicketGacha(entry.GachaId) {
			continue
		}
		if entry.SortOrder > maxMasterSortOrder {
			maxMasterSortOrder = entry.SortOrder
		}
		if entry.StartDatetime > 0 && (catalogStartDatetime == 0 || entry.StartDatetime < catalogStartDatetime) {
			catalogStartDatetime = entry.StartDatetime
		}
		if entry.EndDatetime > catalogEndDatetime {
			catalogEndDatetime = entry.EndDatetime
		}
	}
	for _, tt := range tests {
		entry, ok := byId[tt.gachaId]
		if !ok {
			t.Fatalf("Gacha %d was not added to the catalog", tt.gachaId)
		}
		if entry.BannerAssetName != tt.assetName || entry.RequiredConsumableItemId != tt.ticketId {
			t.Fatalf("unexpected guaranteed Gacha %d: %+v", tt.gachaId, entry)
		}
		if entry.IsMamaBanner {
			t.Fatalf("ticket-only Gacha %d was marked as a Mama banner", tt.gachaId)
		}
		if entry.SortOrder <= maxMasterSortOrder {
			t.Fatalf("ticket-only Gacha %d sort order %d is not after master-data Gachas ending at %d", tt.gachaId, entry.SortOrder, maxMasterSortOrder)
		}
		if entry.StartDatetime != catalogStartDatetime || entry.EndDatetime != catalogEndDatetime {
			t.Fatalf("ticket-only Gacha %d availability = %d..%d, want catalog availability %d..%d", tt.gachaId, entry.StartDatetime, entry.EndDatetime, catalogStartDatetime, catalogEndDatetime)
		}
		if len(entry.PricePhases) != 1 {
			t.Fatalf("Gacha %d price phase count = %d, want 1", tt.gachaId, len(entry.PricePhases))
		}
		phase := entry.PricePhases[0]
		if phase.PhaseId != tt.gachaId*model.PhaseIdMultiplier+1 ||
			phase.PriceType != model.PriceTypeConsumableItem ||
			phase.PriceId != tt.ticketId ||
			phase.Price != 1 || phase.RegularPrice != 0 || phase.DrawCount != 1 ||
			phase.FixedRarityMin != tt.minimumRarity || phase.FixedCount != 1 {
			t.Fatalf("unexpected guaranteed Gacha %d price phase: %+v", tt.gachaId, phase)
		}
	}
	if byId[model.GachaIdGuaranteedFourStar].SortOrder <= byId[model.GachaIdGuaranteedThreeStarOrHigher].SortOrder {
		t.Fatal("four-star guaranteed Gacha was not ordered after the three-star guaranteed Gacha")
	}
}

package masterdata

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"lunar-tear/server/internal/assettext"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

func TestEventTicketTiersMatchInstalledAssets(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if err := memorydb.Init(path); err != nil {
		t.Fatal(err)
	}
	texts, err := assettext.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	items, err := utils.ReadTable[EntityMConsumableItem]("m_consumable_item")
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[int32]EntityMConsumableItem)
	for _, item := range items {
		byID[item.ConsumableItemId] = item
	}
	for baseID, tickets := range eventGachaTickets {
		entries := eventGachaTierEntries(store.GachaCatalogEntry{GachaId: baseID, RelatedEventQuestChapterId: 99, StartDatetime: 10, EndDatetime: 20})
		for i, entry := range entries {
			item := byID[tickets[i]]
			name := texts["ja"][fmt.Sprintf("consumable_item.name.%03d%03d", item.AssetCategoryId, item.AssetVariationId)]
			title := texts["ja"]["gacha.title."+entry.BannerAssetName]
			tier := []string{"・銅", "・銀", "・金"}[i]
			if !strings.Contains(name, tier) || !strings.Contains(title, tier) {
				t.Fatalf("event %d tier %s ticket %d: %q / %q", baseID, tier, tickets[i], name, title)
			}
			if entry.EventGachaBaseId != baseID || entry.GroupId != entry.GachaId || entry.RequiredConsumableItemId != tickets[i] || entry.RelatedEventQuestChapterId != 99 || entry.StartDatetime != 10 || entry.EndDatetime != 20 {
				t.Fatalf("incorrect tier metadata: %+v", entry)
			}
			for _, phase := range entry.PricePhases {
				if phase.PriceId != tickets[i] || phase.Price != phase.DrawCount || phase.PriceType != model.PriceTypeConsumableItem {
					t.Fatalf("incorrect tier price: %+v", phase)
				}
			}
		}
	}
}

func TestEventCatalogIncludesAllTiersWithoutInventingSingleTicketTiers(t *testing.T) {
	chapters := map[int32]EntityMEventQuestChapter{329001: {EventQuestChapterId: 300}, 332001: {EventQuestChapterId: 332}, 311001: {EventQuestChapterId: 311}}
	links := map[int32]EntityMEventQuestLink{329001: {PossessionType: 6, PossessionId: 6055}, 332001: {PossessionType: 6, PossessionId: 6065}, 311001: {PossessionType: 6, PossessionId: 6011}}
	entries := buildEventGachaEntries(chapters, links)
	want := []int32{311001, 311011, 329001, 329011, 329021, 332001}
	if len(entries) != len(want) {
		t.Fatalf("entries = %+v", entries)
	}
	for i, id := range want {
		if entries[i].GachaId != id {
			t.Fatalf("entry %d = %d, want %d", i, entries[i].GachaId, id)
		}
	}
}

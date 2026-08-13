package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
)

func TestEnrichDupExchangeAddsAwakeningStone(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	pool, err := LoadGachaPool()
	if err != nil {
		t.Fatal(err)
	}
	dup, err := LoadDupExchange()
	if err != nil {
		t.Fatal(err)
	}
	configuredFourStar := append([]model.DupExchangeEntry(nil), dup[31000]...)
	if len(configuredFourStar) != 2 {
		t.Fatalf("configured four-star fixture = %+v, want two exchange entries", configuredFourStar)
	}
	partialThreeStars := map[int32]struct {
		original []model.DupExchangeEntry
		stoneId  int32
	}{
		21000: {original: append([]model.DupExchangeEntry(nil), dup[21000]...), stoneId: 313016},
		22002: {original: append([]model.DupExchangeEntry(nil), dup[22002]...), stoneId: 313023},
	}
	for costumeId, partial := range partialThreeStars {
		if len(partial.original) != 1 {
			t.Fatalf("partial three-star fixture %d = %+v, want only the limit-break material", costumeId, partial.original)
		}
	}
	if _, exists := dup[22000]; exists {
		t.Fatal("test fixture no longer exercises the duplicate-exchange fallback")
	}
	if _, err := EnrichDupExchange(dup, pool); err != nil {
		t.Fatal(err)
	}

	entries := dup[22000]
	if len(entries) != 2 {
		t.Fatalf("fallback entries = %+v, want limit-break material and awakening stone", entries)
	}
	if entries[0].PossessionType != int32(model.PossessionTypeMaterial) || entries[0].Count != 10 {
		t.Fatalf("limit-break fallback = %+v", entries[0])
	}
	if entries[1] != (model.DupExchangeEntry{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 313021, Count: 1}) {
		t.Fatalf("awakening-stone fallback = %+v", entries[1])
	}
	for costumeId, partial := range partialThreeStars {
		partialEntries := dup[costumeId]
		if len(partialEntries) != 2 || partialEntries[0] != partial.original[0] {
			t.Fatalf("partial three-star exchange %d changed from %+v to %+v", costumeId, partial.original, partialEntries)
		}
		if partialEntries[1] != (model.DupExchangeEntry{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: partial.stoneId, Count: 1}) {
			t.Fatalf("partial three-star awakening-stone fallback %d = %+v", costumeId, partialEntries[1])
		}
	}
	if got := dup[31000]; len(got) != len(configuredFourStar) || got[0] != configuredFourStar[0] || got[1] != configuredFourStar[1] {
		t.Fatalf("configured four-star exchange changed from %+v to %+v", configuredFourStar, got)
	}
	if added, err := EnrichDupExchange(dup, pool); err != nil {
		t.Fatal(err)
	} else if added != 0 {
		t.Fatalf("second enrichment changed %d costumes", added)
	}
}

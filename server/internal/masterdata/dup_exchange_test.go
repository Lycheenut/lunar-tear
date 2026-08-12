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
	if got := dup[31000]; len(got) != len(configuredFourStar) || got[0] != configuredFourStar[0] || got[1] != configuredFourStar[1] {
		t.Fatalf("configured four-star exchange changed from %+v to %+v", configuredFourStar, got)
	}
}

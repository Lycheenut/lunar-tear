package masterdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestPartsMainStatusPoolsCoverAllPartsAndRarities(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for id, part := range catalog.PartsById {
		// Check the resulting stat kinds against each actual item's slot, rather
		// than deriving expected IDs from the lottery-group encoding.
		want := map[[2]int32]bool{
			{2, 1}: true, {2, 2}: true, // Attack, flat and percentage.
			{7, 1}: true, {7, 2}: true, // Defense, flat and percentage.
			{6, 1}: true, {6, 2}: true, // HP, flat and percentage.
		}
		if part.PartsGroupId >= 1 && part.PartsGroupId <= 54 {
			switch part.PartsGroupId % 3 {
			case 2:
				want[[2]int32{4, 1}] = true // Critical rate.
				want[[2]int32{3, 1}] = true // Critical damage.
			case 0:
				want[[2]int32{1, 1}] = true // Agility.
			}
		}
		pool := catalog.MainStatusPool[part.PartsStatusMainLotteryGroupId]
		if len(pool) != len(want) {
			t.Fatalf("parts %d has %d main statuses, want %d", id, len(pool), len(want))
		}
		for _, stat := range pool {
			definition, ok := catalog.PartsStatusMainById[stat]
			if !ok {
				t.Fatalf("parts %d references unknown main status %d", id, stat)
			}
			kind := [2]int32{definition.StatusKindType, definition.StatusCalculationType}
			if !want[kind] {
				t.Fatalf("parts %d has ineligible or duplicate main status %+v", id, definition)
			}
			delete(want, kind)
			if (stat-1)%4+1 != part.RarityType/10 {
				t.Fatalf("parts %d rarity=%d has wrong-tier main status %d", id, part.RarityType, stat)
			}
		}
	}
}

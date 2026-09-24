package masterdata

import (
	"math"
	"testing"

	"lunar-tear/server/internal/model"
)

func TestPartsSubStatusTable(t *testing.T) {
	defs, pools := buildPartsStatusSub()
	if len(defs) != 36 || len(pools) != 6 {
		t.Fatalf("definitions=%d pools=%d, want 36 and 6", len(defs), len(pools))
	}
	for id, def := range defs {
		for _, r := range []model.PartsSubStatusRange{def.Initial, def.Growth} {
			if r.Min <= 0 || r.Max < r.Min || r.Step <= 0 || (r.Max-r.Min)%r.Step != 0 {
				t.Fatalf("sub-status %d has invalid range %+v", id, r)
			}
			for i := 0; i < 100; i++ {
				value := r.Roll()
				if value < r.Min || value > r.Max || (value-r.Min)%r.Step != 0 {
					t.Fatalf("sub-status %d rolled %d outside %+v", id, value, r)
				}
			}
		}
		if int64(def.Initial.Max)+5*int64(def.Growth.Max) > math.MaxInt32 {
			t.Fatalf("sub-status %d overflows after five enhancements", id)
		}
	}

	wantEffects := map[[2]int32]bool{
		{2, 1}: true, {7, 1}: true, {2, 2}: true, {7, 2}: true, {6, 2}: true,
		{6, 1}: true, {4, 1}: true, {3, 1}: true, {1, 1}: true,
	}
	seenIDs := make(map[int32]bool)
	for poolID, tier := range map[int32]int32{1: 1, 2: 2, 3: 3, 4: 4, 11: 1, 12: 2} {
		pool := pools[poolID]
		if len(pool) != len(wantEffects) {
			t.Fatalf("pool %d has %d effects, want %d", poolID, len(pool), len(wantEffects))
		}
		seenEffects := make(map[[2]int32]bool)
		for _, id := range pool {
			def, ok := defs[id]
			if !ok || id < 1 || id > 36 || (id-1)%4+1 != tier {
				t.Fatalf("pool %d has invalid sub-status ID %d for tier %d", poolID, id, tier)
			}
			effect := [2]int32{def.StatusKindType, def.StatusCalculationType}
			if !wantEffects[effect] || seenEffects[effect] {
				t.Fatalf("pool %d has unknown or duplicate effect %v", poolID, effect)
			}
			seenEffects[effect] = true
			seenIDs[id] = true
		}
	}
	if len(seenIDs) != len(defs) {
		t.Fatal("some sub-status definitions are not reachable from any pool")
	}
}

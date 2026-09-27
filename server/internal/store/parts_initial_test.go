package store

import (
	"maps"
	"testing"

	"lunar-tear/server/internal/model"
)

func TestPartsRepairRequiresFourUniqueSubStatuses(t *testing.T) {
	for _, test := range []struct {
		pool       []int32
		wantRepair bool
	}{{[]int32{1, 2, 3, 4}, true}, {[]int32{1, 1, 2, 3}, false}, {[]int32{99}, false}} {
		g := &PossessionGranter{
			PartsById:          map[int32]PartsRef{36: {PartsStatusSubLotteryGroupId: 4}},
			PartsSubStatusPool: map[int32][]int32{4: test.pool},
			PartsSubStatusDefs: map[int32]model.PartsStatusSubDef{},
		}
		for id := int32(1); id <= 4; id++ {
			g.PartsSubStatusDefs[id] = model.PartsStatusSubDef{Initial: model.PartsSubStatusRange{Min: 10, Max: 10, Step: 1}}
		}
		user := SeedUserState(1, "repair", 1, model.ClientPlatform{})
		user.Parts["empty"] = PartsState{UserPartsUuid: "empty", PartsId: 36, PartsStatusMainId: 28, Level: 1}
		before := maps.Clone(user.Parts)
		repaired := g.RepairPartsWithoutSubStatuses(user, 1000)
		if test.wantRepair {
			if repaired != 1 || len(user.PartsStatusSubs) != 4 {
				t.Fatalf("repair=%d sub-count=%d, want one complete repair with four sub-stats", repaired, len(user.PartsStatusSubs))
			}
		} else if repaired != 0 || len(user.PartsStatusSubs) != 0 {
			t.Fatalf("incomplete pool %v partially repaired data: repaired=%d subs=%v", test.pool, repaired, user.PartsStatusSubs)
		}
		if !maps.Equal(before, user.Parts) {
			t.Fatal("repair changed the part")
		}
	}
}

func TestPartsInitializationAlwaysHasSubStatus(t *testing.T) {
	// The reported Sincerity instances had partsId=36, main status=28 and
	// no sub-status rows. The client treats an empty sub-status list as a
	// preview: no detail dialog, hidden/stale rank, and rank 1 in enhancement.
	g := &PossessionGranter{
		PartsById: map[int32]PartsRef{36: {
			PartsGroupId: 2, RarityType: 40, PartsInitialLotteryId: 1,
			PartsStatusMainLotteryGroupId: 42, PartsStatusSubLotteryGroupId: 4,
		}},
		PartsMainStatusPool: map[int32][]int32{42: {28}},
		PartsSubStatusPool:  map[int32][]int32{4: {4}},
		PartsSubStatusDefs: map[int32]model.PartsStatusSubDef{
			4: {StatusKindType: 2, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 30, Max: 30, Step: 1}},
		},
		PartsSellPriceL1ByRarity: map[int32]int32{40: 100},
		GoldConsumableItemId:     99,
	}
	for _, source := range []string{"grant", "drop", "auto-sale", "only-rank-1-selected"} {
		t.Run(source, func(t *testing.T) {
			user := SeedUserState(1, "parts", 1, model.ClientPlatform{})
			if source == "grant" {
				g.GrantParts(user, 36, 1000)
			} else {
				_, _, sold := g.GrantOrSellPartsDrop(user, 36, map[int32]bool{40: true}, map[int32]bool{1: source == "only-rank-1-selected", 2: source == "auto-sale"}, 1000, 1000)
				if sold != (source == "auto-sale") {
					t.Fatalf("auto-sale used the empty template's rank: sold=%v", sold)
				}
			}
			if source == "auto-sale" {
				if len(user.Parts)+len(user.PartsStatusSubs) != 0 || user.ConsumableItems[99] != 100 {
					t.Fatal("auto-sale left equipment or failed to award gold")
				}
				return
			}
			if len(user.Parts) != 1 || len(user.PartsStatusSubs) != 1 {
				t.Fatalf("parts=%d sub-statuses=%d, want 1 each", len(user.Parts), len(user.PartsStatusSubs))
			}
			for uuid, part := range user.Parts {
				sub := user.PartsStatusSubs[PartsStatusSubKey{UserPartsUuid: uuid, StatusIndex: 1}]
				if part.PartsStatusMainId != 28 || sub.Level != 1 || sub.StatusChangeValue != 30 {
					t.Fatalf("invalid initialized part=%+v sub=%+v", part, sub)
				}
			}
		})
	}
}

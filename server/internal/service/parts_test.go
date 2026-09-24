package service

import (
	"fmt"
	"maps"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
)

func partsSubStatusTestCatalog() *masterdata.PartsCatalog {
	defs := map[int32]model.PartsStatusSubDef{
		4:  {StatusKindType: 2, StatusCalculationType: 1},
		8:  {StatusKindType: 7, StatusCalculationType: 1},
		12: {StatusKindType: 2, StatusCalculationType: 2},
		16: {StatusKindType: 7, StatusCalculationType: 2},
		20: {StatusKindType: 6, StatusCalculationType: 2},
		24: {StatusKindType: 6, StatusCalculationType: 1},
		28: {StatusKindType: 4, StatusCalculationType: 1},
		32: {StatusKindType: 3, StatusCalculationType: 1},
		36: {StatusKindType: 1, StatusCalculationType: 1},
	}
	// Distinct fixed rolls distinguish initial values from growth added to existing totals.
	for id, def := range defs {
		def.Initial = model.PartsSubStatusRange{Min: 13, Max: 13, Step: 1}
		def.Growth = model.PartsSubStatusRange{Min: 77, Max: 77, Step: 1}
		defs[id] = def
	}
	return &masterdata.PartsCatalog{
		PartsStatusSubById: defs,
		SubStatusPool:      map[int32][]int32{4: {4, 8, 12, 16, 20, 24, 28, 32, 36}},
	}
}

func TestGrantPartsSubStatuses(t *testing.T) {
	catalog := partsSubStatusTestCatalog()
	partDef := masterdata.EntityMParts{RarityType: model.RaritySSRare, PartsStatusSubLotteryGroupId: 4}
	const uuid = "target"
	const nowMillis = int64(1000)

	for _, tc := range []struct {
		name         string
		level        int32
		slots        []int32
		wantAdded    int
		wantEnhanced int
	}{
		{name: "level 3 unlocks first", level: 3, wantAdded: 1},
		{name: "level 3 fills missing slots", level: 3, slots: []int32{1}, wantAdded: 1},
		{name: "level 3 unlocks fourth", level: 3, slots: []int32{1, 2, 3}, wantAdded: 1},
		{name: "level 3 boosts full rank", level: 3, slots: []int32{1, 2, 3, 4}, wantEnhanced: 1},
		{name: "fills lowest missing slot", level: 3, slots: []int32{2, 3}, wantAdded: 1},
		{name: "level 6 unlocks second", level: 6, slots: []int32{1}, wantAdded: 1},
		{name: "level 6 unlocks third", level: 6, slots: []int32{1, 2}, wantAdded: 1},
		{name: "level 6 unlocks fourth", level: 6, slots: []int32{1, 2, 3}, wantAdded: 1},
		{name: "level 6 boosts full rank", level: 6, slots: []int32{1, 2, 3, 4}, wantEnhanced: 1},
		{name: "level 9 unlocks third", level: 9, slots: []int32{1, 2}, wantAdded: 1},
		{name: "level 9 unlocks fourth", level: 9, slots: []int32{1, 2, 3}, wantAdded: 1},
		{name: "level 9 excess count", level: 9, slots: []int32{1, 2, 3, 4}, wantEnhanced: 1},
		{name: "level 12 unlocks fourth", level: 12, slots: []int32{1, 2, 3}, wantAdded: 1},
		{name: "level 12 equal count", level: 12, slots: []int32{1, 2, 3, 4}, wantEnhanced: 1},
		{name: "ordinary level", level: 4, slots: []int32{1, 2, 3}},
		{name: "max level grows", level: 15, slots: []int32{1, 2, 3, 4}, wantEnhanced: 1},
		{name: "max level can still unlock", level: 15, slots: []int32{1, 2, 3}, wantAdded: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := &store.UserState{PartsStatusSubs: make(map[store.PartsStatusSubKey]store.PartsStatusSubState)}
			for _, slot := range tc.slots {
				lotteryId := slot * 4
				def := catalog.PartsStatusSubById[lotteryId]
				user.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: uuid, StatusIndex: slot}] = store.PartsStatusSubState{
					UserPartsUuid:           uuid,
					StatusIndex:             slot,
					PartsStatusSubLotteryId: lotteryId,
					Level:                   1,
					StatusKindType:          def.StatusKindType,
					StatusCalculationType:   def.StatusCalculationType,
					StatusChangeValue:       900,
					LatestVersion:           1,
				}
			}
			otherKey := store.PartsStatusSubKey{UserPartsUuid: "other", StatusIndex: 1}
			user.PartsStatusSubs[otherKey] = store.PartsStatusSubState{
				UserPartsUuid: "other", StatusIndex: 1, PartsStatusSubLotteryId: 4,
				Level: 1, StatusChangeValue: 62, LatestVersion: 1,
			}
			before := maps.Clone(user.PartsStatusSubs)

			grantPartsSubStatuses(catalog, user, uuid, store.PartsState{Level: tc.level}, partDef, nowMillis)

			if got, want := len(user.PartsStatusSubs), len(before)+tc.wantAdded; got != want {
				t.Fatalf("sub-status count = %d, want %d", got, want)
			}
			if user.PartsStatusSubs[otherKey] != before[otherKey] {
				t.Fatal("another part's sub-status was changed")
			}
			added, enhanced := 0, 0
			seen := make(map[int32]bool)
			for key, sub := range user.PartsStatusSubs {
				if key.UserPartsUuid != uuid {
					continue
				}
				if seen[sub.PartsStatusSubLotteryId] {
					t.Fatalf("duplicate sub-status lottery %d", sub.PartsStatusSubLotteryId)
				}
				seen[sub.PartsStatusSubLotteryId] = true
				if old, exists := before[key]; exists {
					if sub == old {
						continue
					}
					enhanced++
					if sub.StatusChangeValue != old.StatusChangeValue+77 {
						t.Errorf("sub-status value = %d, want %d + growth 77", sub.StatusChangeValue, old.StatusChangeValue)
					}
					old.Level = sub.Level
					old.StatusChangeValue = sub.StatusChangeValue
					old.LatestVersion = sub.LatestVersion
					if sub != old {
						t.Errorf("enhancement changed sub-status identity: %+v", sub)
					}
				} else {
					added++
					if sub.StatusChangeValue != 13 {
						t.Errorf("unlocked value = %d, want initial 13", sub.StatusChangeValue)
					}
					for slot := int32(1); slot < sub.StatusIndex; slot++ {
						if _, exists := before[store.PartsStatusSubKey{UserPartsUuid: uuid, StatusIndex: slot}]; !exists {
							t.Errorf("skipped empty slot %d", slot)
						}
					}
				}
				if sub.Level != tc.level || sub.LatestVersion != nowMillis {
					t.Errorf("updated sub-status = %+v, want level %d, version %d", sub, tc.level, nowMillis)
				}
			}
			if added != tc.wantAdded || enhanced != tc.wantEnhanced {
				t.Errorf("added=%d enhanced=%d, want added=%d enhanced=%d", added, enhanced, tc.wantAdded, tc.wantEnhanced)
			}
		})
	}
}

func TestPartsSubStatusGrowthFromEveryInitialRank(t *testing.T) {
	for rank := int32(1); rank <= 5; rank++ {
		t.Run(fmt.Sprintf("rank_%d", rank), func(t *testing.T) {
			catalog := partsSubStatusTestCatalog()
			partDef := masterdata.EntityMParts{PartsId: 1, PartsGroupId: 1, RarityType: 40, PartsInitialLotteryId: rank, PartsStatusSubLotteryGroupId: 4}
			catalog.PartsById = map[int32]masterdata.EntityMParts{1: partDef}
			granter := questflow.BuildGranter(&masterdata.QuestCatalog{PartsCatalog: catalog}, nil)
			user := store.SeedUserState(1, "parts", 1, model.ClientPlatform{})
			granter.GrantParts(user, 1, 1000)
			if len(user.PartsStatusSubs) != int(rank-1) {
				t.Fatalf("initial count = %d", len(user.PartsStatusSubs))
			}
			for uuid, part := range user.Parts {
				for level := int32(2); level <= model.PartsMaxLevel; level++ {
					part.Level = level
					grantPartsSubStatuses(catalog, user, uuid, part, partDef, int64(level))
				}
			}
			var total int32
			seen := map[int32]bool{}
			for _, sub := range user.PartsStatusSubs {
				if seen[sub.PartsStatusSubLotteryId] {
					t.Fatal("duplicate effect")
				}
				seen[sub.PartsStatusSubLotteryId] = true
				total += sub.StatusChangeValue
			}
			if len(user.PartsStatusSubs) != 4 || total != 4*13+rank*77 {
				t.Fatalf("final count=%d total=%d, want 4 initial rolls plus %d growth rolls", len(user.PartsStatusSubs), total, rank)
			}
		})
	}
}

func TestPartsSubStatusUsesFixedMilestonesAndSlotCap(t *testing.T) {
	catalog := partsSubStatusTestCatalog()
	user := store.SeedUserState(1, "parts", 1, model.ClientPlatform{})
	def := masterdata.EntityMParts{PartsStatusSubLotteryGroupId: 4}
	for _, tc := range []struct {
		level int32
		count int
	}{{0, 0}, {1, 0}, {2, 0}, {3, 1}, {4, 1}, {6, 2}, {7, 2}, {9, 3}, {12, 4}, {14, 4}, {15, 4}, {16, 4}, {18, 4}} {
		before := maps.Clone(user.PartsStatusSubs)
		grantPartsSubStatuses(catalog, user, "target", store.PartsState{Level: tc.level}, def, int64(tc.level))
		if len(user.PartsStatusSubs) != tc.count {
			t.Fatalf("level %d: count=%d", tc.level, len(user.PartsStatusSubs))
		}
		changed := !maps.Equal(before, user.PartsStatusSubs)
		if want := tc.level == 3 || tc.level == 6 || tc.level == 9 || tc.level == 12 || tc.level == 15; changed != want {
			t.Fatalf("level %d: changed=%v", tc.level, changed)
		}
	}
}

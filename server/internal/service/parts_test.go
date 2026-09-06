package service

import (
	"maps"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestGrantPartsSubStatuses(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	catalog, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
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
		{name: "level 3 equal count", level: 3, slots: []int32{1}, wantEnhanced: 1},
		{name: "level 3 excess count", level: 3, slots: []int32{1, 2, 3}, wantEnhanced: 1},
		{name: "count does not depend on slot index", level: 3, slots: []int32{2, 3}, wantEnhanced: 1},
		{name: "level 6 unlocks second", level: 6, slots: []int32{1}, wantAdded: 1},
		{name: "level 6 equal count", level: 6, slots: []int32{1, 2}, wantEnhanced: 1},
		{name: "level 6 excess count", level: 6, slots: []int32{1, 2, 3}, wantEnhanced: 1},
		{name: "level 9 unlocks third", level: 9, slots: []int32{1, 2}, wantAdded: 1},
		{name: "level 9 equal count", level: 9, slots: []int32{1, 2, 3}, wantEnhanced: 1},
		{name: "level 9 excess count", level: 9, slots: []int32{1, 2, 3, 4}, wantEnhanced: 1},
		{name: "level 12 unlocks fourth", level: 12, slots: []int32{1, 2, 3}, wantAdded: 1},
		{name: "level 12 equal count", level: 12, slots: []int32{1, 2, 3, 4}, wantEnhanced: 1},
		{name: "ordinary level", level: 4, slots: []int32{1, 2, 3}},
		{name: "max level", level: 15, slots: []int32{1, 2, 3, 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := &store.UserState{PartsStatusSubs: make(map[store.PartsStatusSubKey]store.PartsStatusSubState)}
			for _, slot := range tc.slots {
				lotteryId := slot * 4
				def := catalog.PartsStatusMainById[lotteryId]
				f, ok := catalog.FuncResolver.Resolve(def.StatusNumericalFunctionId)
				if !ok {
					t.Fatalf("missing numerical function for lottery %d", lotteryId)
				}
				user.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: uuid, StatusIndex: slot}] = store.PartsStatusSubState{
					UserPartsUuid:           uuid,
					StatusIndex:             slot,
					PartsStatusSubLotteryId: lotteryId,
					Level:                   1,
					StatusKindType:          def.StatusKindType,
					StatusCalculationType:   def.StatusCalculationType,
					StatusChangeValue:       f.Evaluate(1),
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
					if sub.StatusChangeValue <= old.StatusChangeValue {
						t.Errorf("sub-status value = %d, want greater than %d", sub.StatusChangeValue, old.StatusChangeValue)
					}
					old.Level = sub.Level
					old.StatusChangeValue = sub.StatusChangeValue
					old.LatestVersion = sub.LatestVersion
					if sub != old {
						t.Errorf("enhancement changed sub-status identity: %+v", sub)
					}
				} else {
					added++
				}
				def := catalog.PartsStatusMainById[sub.PartsStatusSubLotteryId]
				f, ok := catalog.FuncResolver.Resolve(def.StatusNumericalFunctionId)
				if !ok {
					t.Fatalf("missing numerical function for lottery %d", sub.PartsStatusSubLotteryId)
				}
				if sub.Level != tc.level || sub.StatusChangeValue != f.Evaluate(tc.level) || sub.LatestVersion != nowMillis {
					t.Errorf("updated sub-status = %+v, want level %d, value %d, version %d", sub, tc.level, f.Evaluate(tc.level), nowMillis)
				}
			}
			if added != tc.wantAdded || enhanced != tc.wantEnhanced {
				t.Errorf("added=%d enhanced=%d, want added=%d enhanced=%d", added, enhanced, tc.wantAdded, tc.wantEnhanced)
			}
		})
	}
}

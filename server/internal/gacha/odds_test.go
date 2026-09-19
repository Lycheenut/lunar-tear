package gacha

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestPremiumOddsNormalTenthAndPickup(t *testing.T) {
	source, entries, config := testPremiumSource()
	config.Banners[100] = BannerConfig{BannerAssetName: "limited_100", StartDatetime: 1, EndDatetime: 2, PickupWeaponIds: []int32{1}}
	catalog, err := BuildPremiumCatalog(config, source, entries, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	catalog.Banners[100].Groups[0].NonPickup = append(catalog.Banners[100].Groups[0].NonPickup, PoolItem{WeaponId: 12, CostumeId: 112, RarityType: 40})
	phase := store.GachaPricePhaseEntry{DrawCount: 10, FixedCount: 1, FixedRarityMin: model.RaritySRare}
	odds, err := PremiumOdds(catalog.Banners[100], entries[0], phase)
	if err != nil {
		t.Fatal(err)
	}
	if len(odds) != 2 || odds[0].First != 1 || odds[0].Last != 9 || odds[1].First != 10 || odds[1].Last != 10 {
		t.Fatalf("slots: %+v", odds)
	}
	want := [][]float64{{.02, .03, .05, .10, .80}, {.02, .03, .05, .90, 0}}
	for col, slot := range odds {
		var total float64
		for i, group := range slot.Groups {
			assertOddsEqual(t, group.Rate, want[col][i])
			var groupTotal float64
			for _, item := range group.Items {
				groupTotal += item.Rate
			}
			assertOddsEqual(t, groupTotal, group.Rate)
			total += group.Rate
		}
		assertOddsEqual(t, total, 1)
		assertOddsEqual(t, slot.Groups[0].Items[0].Rate, .01)
		assertOddsEqual(t, slot.Groups[0].Items[1].Rate, .005)
		assertOddsEqual(t, slot.Groups[0].Items[2].Rate, .005)
		if !slot.Groups[0].Items[0].Pickup {
			t.Fatal("pickup marker lost")
		}
	}
}

func TestPremiumOddsMatchDrawsIncludingGuarantees(t *testing.T) {
	source, entries, config := testPremiumSource()
	catalog, err := BuildPremiumCatalog(config, source, entries, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	banner := catalog.Banners[100]
	cases := []struct {
		name                                  string
		id, mode, step, draws, minimum, fixed int32
	}{
		{"normal", 100, model.GachaModeBasic, 0, 10, 30, 1},
		{"two guaranteed four-stars", 100, model.GachaModeStepup, 5, 10, 40, 2},
		{"three-star ticket", model.GachaIdGuaranteedThreeStarOrHigher, model.GachaModeBasic, 0, 1, 30, 1},
		{"four-star ticket", model.GachaIdGuaranteedFourStar, model.GachaModeBasic, 0, 1, 40, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := store.GachaCatalogEntry{GachaId: tc.id, GachaModeType: tc.mode}
			phase := store.GachaPricePhaseEntry{DrawCount: tc.draws, StepNumber: tc.step, FixedRarityMin: tc.minimum, FixedCount: tc.fixed}
			odds, err := PremiumOdds(banner, entry, phase)
			if err != nil {
				t.Fatal(err)
			}
			rng := rand.New(rand.NewSource(614))
			const executions = 50000
			counts := make([]map[int32]int, tc.draws)
			for i := range counts {
				counts[i] = make(map[int32]int)
			}
			for range executions {
				items, err := drawPremiumWithOptions(banner, int(tc.draws), tc.minimum, int(tc.fixed), premiumRateMultiplier(entry, phase), tc.id == model.GachaIdGuaranteedThreeStarOrHigher, rng.Intn)
				if err != nil {
					t.Fatal(err)
				}
				for slot, item := range items {
					counts[slot][item.PossessionId]++
				}
			}
			for _, slot := range odds {
				var total float64
				for _, group := range slot.Groups {
					for _, item := range group.Items {
						total += item.Rate
						id := item.DrawnItem().PossessionId
						for i := slot.First - 1; i < slot.Last; i++ {
							got := float64(counts[i][id]) / executions
							if math.Abs(got-item.Rate) > .009 {
								t.Fatalf("slot %d, item %d: observed %.5f, displayed %.5f", i+1, id, got, item.Rate)
							}
							if item.Rate == 0 && counts[i][id] != 0 {
								t.Fatal("zero-rate item drawn")
							}
						}
					}
				}
				assertOddsEqual(t, total, 1)
			}
		})
	}
}

func TestBoxOddsLiveCountsExhaustionAndMonthlyReset(t *testing.T) {
	now := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC).UnixMilli()
	entry := store.GachaCatalogEntry{GachaLabelType: model.GachaLabelChapter, BoxItems: []store.GachaBoxItemEntry{
		{PossessionId: 1, CounterId: 11, Count: 5, MaxCount: 3},
		{PossessionId: 1, CounterId: 12, Count: 10, MaxCount: 1},
		{PossessionId: 2, Count: 2, Weight: 1},
		{PossessionId: 3, Count: 3, Weight: 3},
	}}
	counts := map[int32]int32{model.ChapterGachaMonthCounterId: gametime.BusinessMonthKey(now), 11: 1}
	state := store.GachaBannerState{BoxDrewCounts: counts}
	items := BoxOdds(entry, state, now)
	assertOddsEqual(t, items[0].Rate, .8*2/3)
	assertOddsEqual(t, items[1].Rate, .8/3)
	assertOddsEqual(t, items[2].Rate, .05)
	assertOddsEqual(t, items[3].Rate, .15)
	if items[0].Remaining != 2 || items[0].Count != 5 || !items[2].Unlimited {
		t.Fatalf("quantities: %+v", items)
	}
	counts[11], counts[12] = 3, 1
	items = BoxOdds(entry, state, now)
	assertOddsEqual(t, items[0].Rate, 0)
	assertOddsEqual(t, items[2].Rate, .25)
	assertOddsEqual(t, items[3].Rate, .75)
	items = BoxOdds(entry, state, gametime.StartOfNextBusinessMonthAtMillis(now))
	assertOddsEqual(t, items[0].Rate, .6)
	if items[0].Remaining != 3 || !reflect.DeepEqual(counts, map[int32]int32{model.ChapterGachaMonthCounterId: 202609, 11: 3, 12: 1}) {
		t.Fatal("view mutated counters or failed to reset")
	}
	entry.GachaLabelType = model.GachaLabelEvent
	counts[11], counts[12] = 2, 0
	items = BoxOdds(entry, state, now)
	assertOddsEqual(t, items[0].Rate, .5)
	assertOddsEqual(t, items[1].Rate, .5)
	if items[2].Unlimited || items[2].Rate != 0 {
		t.Fatal("event rewards became unlimited")
	}
	counts[11], counts[12] = 3, 1
	for _, item := range BoxOdds(entry, state, now) {
		assertOddsEqual(t, item.Rate, 0)
	}
}

func assertOddsEqual(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("rate = %.12f, want %.12f", got, want)
	}
}

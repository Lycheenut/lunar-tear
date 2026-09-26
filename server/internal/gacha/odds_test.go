package gacha

import (
	"fmt"
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
	entry := store.GachaCatalogEntry{GachaId: 1, GachaLabelType: model.GachaLabelChapter}
	box := BoxConfig{
		GroupWeights: BoxGroupWeights{Limited: 6000, Unlimited: 4000},
		LimitedRewards: []BoxRewardConfig{
			{PossessionId: 1, Count: 5, MaxCount: 3},
			{PossessionId: 1, Count: 10, MaxCount: 1},
		},
		UnlimitedRewards: []BoxRewardConfig{
			{PossessionId: 2, Count: 2, Weight: 1},
			{PossessionId: 3, Count: 3, Weight: 3},
		},
	}
	config := DefaultConfig()
	config.ChapterBanners[1] = box
	h := &GachaHandler{Premium: &PremiumCatalog{Config: config}}
	counts := map[int32]int32{model.ChapterGachaMonthCounterId: gametime.BusinessMonthKey(now), 1: 1}
	state := store.GachaBannerState{BoxDrewCounts: counts}
	getItems := func(at int64) []BoxItemOdds {
		t.Helper()
		odds, err := h.BoxOdds(entry, state, at)
		if err != nil {
			t.Fatal(err)
		}
		return odds.Items
	}
	items := getItems(now)
	assertOddsEqual(t, items[0].Rate, .6*2/3)
	assertOddsEqual(t, items[1].Rate, .6/3)
	assertOddsEqual(t, items[2].Rate, .10)
	assertOddsEqual(t, items[3].Rate, .30)
	if items[0].Remaining != 2 || items[0].Count != 5 || !items[2].Unlimited {
		t.Fatalf("quantities: %+v", items)
	}
	counts[1], counts[2] = 3, 1
	items = getItems(now)
	assertOddsEqual(t, items[0].Rate, 0)
	assertOddsEqual(t, items[2].Rate, .25)
	assertOddsEqual(t, items[3].Rate, .75)
	items = getItems(gametime.StartOfNextBusinessMonthAtMillis(now))
	assertOddsEqual(t, items[0].Rate, .45)
	if items[0].Remaining != 3 || !reflect.DeepEqual(counts, map[int32]int32{model.ChapterGachaMonthCounterId: 202609, 1: 3, 2: 1}) {
		t.Fatal("view mutated counters or failed to reset")
	}
	entry.GachaLabelType = model.GachaLabelEvent
	box.GroupWeights = BoxGroupWeights{Limited: GroupWeightTotal}
	config.EventBanners[1] = EventBoxConfig{Boxes: []BoxConfig{box}}
	counts[1], counts[2] = 2, 0
	items = getItems(now)
	assertOddsEqual(t, items[0].Rate, .5)
	assertOddsEqual(t, items[1].Rate, .5)
	if !items[2].Unlimited || items[2].Rate != 0 {
		t.Fatal("disabled unlimited group has nonzero probability")
	}
	counts[1], counts[2] = 3, 1
	for _, item := range getItems(now) {
		assertOddsEqual(t, item.Rate, 0)
	}
}

func TestBoxOddsMatchConfiguredDraws(t *testing.T) {
	const month = 202609
	now := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC).UnixMilli()
	for _, weights := range []BoxGroupWeights{{Limited: 3000, Unlimited: 7000}, {Limited: 10000}, {Unlimited: 10000}} {
		t.Run(fmt.Sprintf("%d-%d", weights.Limited, weights.Unlimited), func(t *testing.T) {
			box := BoxConfig{
				GroupWeights: weights,
				LimitedRewards: []BoxRewardConfig{
					{PossessionId: 1, Count: 1, MaxCount: 4},
					{PossessionId: 2, Count: 1, MaxCount: 1},
				},
				UnlimitedRewards: []BoxRewardConfig{
					{PossessionId: 3, Count: 1, Weight: 1},
					{PossessionId: 4, Count: 1, Weight: 3},
				},
			}
			h := &GachaHandler{Premium: &PremiumCatalog{Config: &Config{ChapterBanners: map[int32]BoxConfig{1: box}}}}
			state := store.GachaBannerState{BoxDrewCounts: map[int32]int32{model.ChapterGachaMonthCounterId: month, 1: 1}}
			odds, err := h.BoxOdds(store.GachaCatalogEntry{GachaId: 1, GachaLabelType: model.GachaLabelChapter}, state, now)
			if err != nil {
				t.Fatal(err)
			}
			rng := rand.New(rand.NewSource(123))
			observed := make(map[int32]int)
			const trials = 30000
			items := boxItems(box)
			for range trials {
				counts := map[int32]int32{model.ChapterGachaMonthCounterId: month, 1: 1}
				draw, err := drawChapterWithIntn(items, weights, counts, 1, month, rng.Intn)
				if err != nil {
					t.Fatal(err)
				}
				observed[draw[0].CounterId]++
			}
			var total float64
			for _, item := range odds.Items {
				actual := float64(observed[item.CounterId]) / trials
				if math.Abs(actual-item.Rate) > .012 {
					t.Fatalf("item %d: displayed %f, observed %f", item.CounterId, item.Rate, actual)
				}
				total += item.Rate
			}
			assertOddsEqual(t, total, 1)
		})
	}
}

func TestBoxOddsSelectsCurrentEventBox(t *testing.T) {
	first := BoxConfig{GroupWeights: BoxGroupWeights{Limited: GroupWeightTotal}, LimitedRewards: []BoxRewardConfig{{PossessionId: 1, Count: 1, MaxCount: 5}}}
	second := BoxConfig{
		GroupWeights:     BoxGroupWeights{Limited: 2500, Unlimited: 7500},
		LimitedRewards:   []BoxRewardConfig{{PossessionId: 2, Count: 2, MaxCount: 3}},
		UnlimitedRewards: []BoxRewardConfig{{PossessionId: 3, Count: 3, Weight: 1}},
	}
	h := &GachaHandler{Premium: &PremiumCatalog{Config: &Config{EventBanners: map[int32]EventBoxConfig{1: {Boxes: []BoxConfig{first, second}}}}}}
	entry := store.GachaCatalogEntry{GachaId: 1, GachaLabelType: model.GachaLabelEvent, BoxItems: boxItems(first)}
	for _, number := range []int32{2, 99} {
		state := store.GachaBannerState{BoxNumber: number, BoxDrewCounts: map[int32]int32{1: 1}}
		odds, err := h.BoxOdds(entry, state, 0)
		if err != nil {
			t.Fatal(err)
		}
		if odds.BoxNumber != 2 || len(odds.Items) != 2 || odds.Items[0].PossessionId != 2 || odds.Items[0].Remaining != 2 || !odds.Items[1].Unlimited {
			t.Fatalf("current box = %+v", odds)
		}
		assertOddsEqual(t, odds.Items[0].Rate, .25)
		assertOddsEqual(t, odds.Items[1].Rate, .75)
		if state.BoxNumber != number || state.BoxDrewCounts[1] != 1 || entry.BoxItems[0].PossessionId != 1 {
			t.Fatal("odds changed the player or shared catalog")
		}
	}
	delete(h.Premium.Config.EventBanners, 1)
	if _, err := h.BoxOdds(entry, store.GachaBannerState{}, 0); err == nil {
		t.Fatal("missing box configuration was accepted")
	}
}

func assertOddsEqual(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("rate = %.12f, want %.12f", got, want)
	}
}

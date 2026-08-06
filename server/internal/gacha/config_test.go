package gacha

import (
	"encoding/json"
	"math/rand"
	"strings"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestBuildPremiumCatalogUsesDefaultStandardAndPerGroupPickup(t *testing.T) {
	source, entries, config := testPremiumSource()
	config.LimitedSets["limited_a"] = LimitedSetConfig{DisplayName: "Limited A"}
	config.Weapons[1] = WeaponConfig{Availability: AvailabilityLimited, LimitedSet: "limited_a"}
	config.Banners[100] = BannerConfig{LimitedSets: []string{"limited_a"}, PickupWeaponIds: []int32{1}}

	catalog, err := BuildPremiumCatalog(config, source, entries, BuildOptions{
		RequireComplete:       true,
		CurrentMasterDataHash: "sha256:test",
	})
	if err != nil {
		t.Fatal(err)
	}
	banner := catalog.Banners[100]
	group := findPremiumGroup(t, banner, GroupCharacterWeapon4)
	if len(group.Pickup) != 1 || group.Pickup[0].WeaponId != 1 {
		t.Fatalf("unexpected pickup group: %+v", group.Pickup)
	}
	if len(group.NonPickup) != 1 || group.NonPickup[0].WeaponId != 2 {
		t.Fatalf("unexpected non-pickup group: %+v", group.NonPickup)
	}
	if _, exists := banner.ItemsByWeaponId[11]; exists {
		t.Fatal("event weapon entered the banner pool")
	}
}

func TestConfiguredPromotionsFollowPickupOrderAndPairedWeapon(t *testing.T) {
	source, entries, config := testPremiumSource()
	config.Banners[100] = BannerConfig{PickupWeaponIds: []int32{3, 1}}
	catalog, err := BuildPremiumCatalog(config, source, entries, BuildOptions{
		RequireComplete:       true,
		CurrentMasterDataHash: "sha256:test",
	})
	if err != nil {
		t.Fatal(err)
	}
	ApplyConfiguredPromotions(entries, catalog)
	items := entries[0].PromotionItems
	if len(items) != 2 {
		t.Fatalf("promotion item count = %d, want 2", len(items))
	}
	if items[0].PossessionType != int32(model.PossessionTypeWeapon) || items[0].PossessionId != 3 {
		t.Fatalf("first promotion item = %+v, want weapon 3", items[0])
	}
	if items[1].PossessionType != int32(model.PossessionTypeCostume) || items[1].PossessionId != 101 || items[1].BonusPossessionId != 1 {
		t.Fatalf("second promotion item = %+v, want costume 101 with weapon 1", items[1])
	}
}

func TestBuildPremiumCatalogRejectsAllPickupGroup(t *testing.T) {
	source, entries, config := testPremiumSource()
	config.Banners[100] = BannerConfig{PickupWeaponIds: []int32{1, 2}}
	_, err := BuildPremiumCatalog(config, source, entries, BuildOptions{
		RequireComplete:       true,
		CurrentMasterDataHash: "sha256:test",
	})
	if err == nil || !strings.Contains(err.Error(), "no non-pickup") {
		t.Fatalf("all-pickup group error = %v", err)
	}
}

func TestBuildPremiumCatalogDefaultsMissingWeaponToStandard(t *testing.T) {
	source, entries, config := testPremiumSource()
	delete(config.Weapons, 3)
	catalog, err := BuildPremiumCatalog(config, source, entries, BuildOptions{
		RequireComplete:       true,
		CurrentMasterDataHash: "sha256:test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := catalog.Banners[100].ItemsByWeaponId[3]; !ok {
		t.Fatal("weapon without an override did not enter the standard pool")
	}
}

func TestEncodeConfigOmitsExplicitStandardWeapons(t *testing.T) {
	config := DefaultConfig()
	config.Weapons[1] = WeaponConfig{Availability: AvailabilityStandard}
	config.Weapons[2] = WeaponConfig{Availability: AvailabilityEvent}
	if _, _, err := EncodeConfig(config); err != nil {
		t.Fatal(err)
	}
	if _, ok := config.Weapons[1]; ok {
		t.Fatal("explicit standard weapon was not normalized to the default")
	}
	if config.Weapons[2].Availability != AvailabilityEvent {
		t.Fatal("event override was removed during normalization")
	}
}

func TestConfigWithoutAutomaticEventWeaponsRemovesOverridesAndPickups(t *testing.T) {
	source, _, config := testPremiumSource()
	config.Weapons[12] = WeaponConfig{Availability: AvailabilityEvent}
	config.Banners[100] = BannerConfig{
		LimitedSets:     []string{"limited_a"},
		PickupWeaponIds: []int32{1, 11, 12},
	}

	filtered := ConfigWithoutAutomaticEventWeapons(config, source)
	if _, exists := filtered.Weapons[11]; exists {
		t.Fatal("automatically excluded weapon override was preserved")
	}
	if _, exists := filtered.Weapons[12]; !exists {
		t.Fatal("unknown weapon was silently removed instead of being left for validation")
	}
	if got := filtered.Banners[100].PickupWeaponIds; len(got) != 2 || got[0] != 1 || got[1] != 12 {
		t.Fatalf("filtered pickup IDs = %v, want [1 12]", got)
	}
	if _, exists := config.Weapons[11]; !exists {
		t.Fatal("source config was mutated")
	}
	if got := config.Banners[100].PickupWeaponIds; len(got) != 3 {
		t.Fatalf("source pickup IDs were mutated: %v", got)
	}
	raw, _, err := EncodeConfig(filtered)
	if err != nil {
		t.Fatal(err)
	}
	var exported Config
	if err := json.Unmarshal(raw, &exported); err != nil {
		t.Fatal(err)
	}
	if _, exists := exported.Weapons[11]; exists {
		t.Fatal("automatically excluded weapon was written to exported JSON")
	}
	if got := exported.Banners[100].PickupWeaponIds; len(got) != 2 || got[0] != 1 || got[1] != 12 {
		t.Fatalf("exported pickup IDs = %v, want [1 12]", got)
	}
}

func TestBuildPremiumCatalogRejectsMissingTenthDrawTransferTarget(t *testing.T) {
	source, entries, config := testPremiumSource()
	config.GroupWeights = GroupWeights{WeaponOnly: RarityWeights{TwoStar: 10000}}
	config.Weapons[7] = WeaponConfig{Availability: AvailabilityEvent}
	config.Weapons[8] = WeaponConfig{Availability: AvailabilityEvent}
	_, err := BuildPremiumCatalog(config, source, entries, BuildOptions{
		RequireComplete:       true,
		CurrentMasterDataHash: "sha256:test",
	})
	if err == nil || !strings.Contains(err.Error(), "tenth-draw") {
		t.Fatalf("missing tenth-draw transfer target error = %v", err)
	}
}

func TestBuildPremiumCatalogRejectsMissingFixedRarityGroup(t *testing.T) {
	source, entries, config := testPremiumSource()
	config.GroupWeights = GroupWeights{
		CharacterWeapon: RarityWeights{ThreeStar: 500},
		WeaponOnly:      RarityWeights{TwoStar: 8500, ThreeStar: 1000},
	}
	entries[0].PricePhases = []store.GachaPricePhaseEntry{{PhaseId: 7, FixedRarityMin: model.RaritySSRare, FixedCount: 1}}
	_, err := BuildPremiumCatalog(config, source, entries, BuildOptions{
		RequireComplete:       true,
		CurrentMasterDataHash: "sha256:test",
	})
	if err == nil || !strings.Contains(err.Error(), "fixed rarity") {
		t.Fatalf("missing fixed-rarity group error = %v", err)
	}
}

func TestConfiguredDrawSplitsPickupAndNonPickupEvenly(t *testing.T) {
	banner := &PremiumBannerPool{Groups: []PremiumGroup{{
		Id:        GroupWeaponOnly4,
		GrantType: GrantWeaponOnly,
		Star:      4,
		Rarity:    model.RaritySSRare,
		Weight:    1,
		Pickup:    []PoolItem{{WeaponId: 1, RarityType: model.RaritySSRare}, {WeaponId: 2, RarityType: model.RaritySSRare}},
		NonPickup: []PoolItem{{WeaponId: 3, RarityType: model.RaritySSRare}, {WeaponId: 4, RarityType: model.RaritySSRare}},
	}}}
	pickup, err := drawPremiumWithIntn(banner, 1, 0, 0, 1, sequenceIntn(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if pickup[0].PossessionId != 2 {
		t.Fatalf("pickup draw = %d, want 2", pickup[0].PossessionId)
	}
	nonPickup, err := drawPremiumWithIntn(banner, 1, 0, 0, 1, sequenceIntn(0, 1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if nonPickup[0].PossessionId != 3 {
		t.Fatalf("non-pickup draw = %d, want 3", nonPickup[0].PossessionId)
	}
}

func TestConfiguredTenPullLastSlotTransfersTwoStarWeight(t *testing.T) {
	banner := &PremiumBannerPool{Groups: []PremiumGroup{
		{Id: GroupWeaponOnly2, GrantType: GrantWeaponOnly, Star: 2, Rarity: model.RarityRare, Weight: 8000, NonPickup: []PoolItem{{WeaponId: 2, RarityType: model.RarityRare}}},
		{Id: GroupWeaponOnly3, GrantType: GrantWeaponOnly, Star: 3, Rarity: model.RaritySRare, Weight: 2000, NonPickup: []PoolItem{{WeaponId: 3, RarityType: model.RaritySRare}}},
	}}
	items, err := drawPremiumWithIntn(banner, 10, 0, 0, 1, sequenceIntn(make([]int, 30)...))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 9; i++ {
		if items[i].RarityType != model.RarityRare {
			t.Fatalf("slot %d rarity = %d, want 2-star", i+1, items[i].RarityType)
		}
	}
	if items[9].RarityType != model.RaritySRare {
		t.Fatalf("slot 10 rarity = %d, want transferred 3-star", items[9].RarityType)
	}
}

func TestTenthDrawWeightsPreserveFourStarAndTransferByGrantType(t *testing.T) {
	groups := []PremiumGroup{
		{GrantType: GrantCharacterWeapon, Star: 4},
		{GrantType: GrantCharacterWeapon, Star: 3},
		{GrantType: GrantCharacterWeapon, Star: 2},
		{GrantType: GrantWeaponOnly, Star: 4},
		{GrantType: GrantWeaponOnly, Star: 3},
		{GrantType: GrantWeaponOnly, Star: 2},
	}
	got := transferTwoStarWeightsToThreeStar(groups, []int{200, 500, 100, 300, 1000, 7900})
	want := []int{200, 600, 0, 300, 8900, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tenth-draw weight %d = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestConfiguredTenthDrawDistribution(t *testing.T) {
	banner := &PremiumBannerPool{Groups: []PremiumGroup{
		{Id: GroupWeaponOnly4, GrantType: GrantWeaponOnly, Star: 4, Rarity: model.RaritySSRare, Weight: 500, NonPickup: []PoolItem{{WeaponId: 4, RarityType: model.RaritySSRare}}},
		{Id: GroupWeaponOnly3, GrantType: GrantWeaponOnly, Star: 3, Rarity: model.RaritySRare, Weight: 1500, NonPickup: []PoolItem{{WeaponId: 3, RarityType: model.RaritySRare}}},
		{Id: GroupWeaponOnly2, GrantType: GrantWeaponOnly, Star: 2, Rarity: model.RarityRare, Weight: 8000, NonPickup: []PoolItem{{WeaponId: 2, RarityType: model.RarityRare}}},
	}}
	random := rand.New(rand.NewSource(2))
	counts := make(map[int32]int)
	const draws = 20000
	for range draws {
		items, err := drawPremiumWithIntn(banner, 10, 0, 0, 1, random.Intn)
		if err != nil {
			t.Fatal(err)
		}
		counts[items[9].PossessionId]++
	}
	assertRateNear(t, counts[4], draws, 0.05, 0.01)
	assertRateNear(t, counts[3], draws, 0.95, 0.01)
	if counts[2] != 0 {
		t.Fatalf("tenth draw produced %d 2-star weapons", counts[2])
	}
}

func TestConfiguredDrawDistribution(t *testing.T) {
	banner := &PremiumBannerPool{Groups: []PremiumGroup{
		{Id: GroupWeaponOnly4, GrantType: GrantWeaponOnly, Star: 4, Rarity: model.RaritySSRare, Weight: 2000, Pickup: []PoolItem{{WeaponId: 1, RarityType: model.RaritySSRare}}, NonPickup: []PoolItem{{WeaponId: 2, RarityType: model.RaritySSRare}}},
		{Id: GroupWeaponOnly2, GrantType: GrantWeaponOnly, Star: 2, Rarity: model.RarityRare, Weight: 8000, NonPickup: []PoolItem{{WeaponId: 3, RarityType: model.RarityRare}}},
	}}
	random := rand.New(rand.NewSource(1))
	counts := make(map[int32]int)
	const draws = 50000
	for range draws {
		items, err := drawPremiumWithIntn(banner, 1, 0, 0, 1, random.Intn)
		if err != nil {
			t.Fatal(err)
		}
		counts[items[0].PossessionId]++
	}
	assertRateNear(t, counts[1], draws, 0.10, 0.01)
	assertRateNear(t, counts[2], draws, 0.10, 0.01)
	assertRateNear(t, counts[3], draws, 0.80, 0.01)
}

func testPremiumSource() (*masterdata.GachaCatalog, []store.GachaCatalogEntry, *Config) {
	source := &masterdata.GachaCatalog{
		ConfigurableWeaponById: make(map[int32]masterdata.GachaPoolItem),
		EligibleWeaponById:     make(map[int32]masterdata.GachaPoolItem),
		CostumeByWeaponId:      make(map[int32]masterdata.GachaPoolItem),
	}
	add := func(id int32, rarity model.RarityType, costumeId, characterId int32) {
		weapon := masterdata.GachaPoolItem{PossessionType: int32(model.PossessionTypeWeapon), PossessionId: id, RarityType: rarity}
		source.ConfigurableWeaponById[id] = weapon
		source.EligibleWeaponById[id] = weapon
		if costumeId != 0 {
			source.CostumeByWeaponId[id] = masterdata.GachaPoolItem{PossessionType: int32(model.PossessionTypeCostume), PossessionId: costumeId, RarityType: rarity, CharacterId: characterId}
		}
	}
	add(1, model.RaritySSRare, 101, 1001)
	add(2, model.RaritySSRare, 102, 1002)
	add(3, model.RaritySSRare, 0, 0)
	add(4, model.RaritySSRare, 0, 0)
	add(5, model.RaritySRare, 105, 1005)
	add(6, model.RaritySRare, 106, 1006)
	add(7, model.RaritySRare, 0, 0)
	add(8, model.RaritySRare, 0, 0)
	add(9, model.RarityRare, 0, 0)
	add(10, model.RarityRare, 0, 0)
	source.ConfigurableWeaponById[11] = masterdata.GachaPoolItem{PossessionType: int32(model.PossessionTypeWeapon), PossessionId: 11, RarityType: model.RaritySSRare}

	config := DefaultConfig()
	config.SourceMasterDataHash = "sha256:test"
	for id := int32(1); id <= 10; id++ {
		config.Weapons[id] = WeaponConfig{Availability: AvailabilityStandard}
	}
	config.Weapons[11] = WeaponConfig{Availability: AvailabilityEvent}
	entries := []store.GachaCatalogEntry{{GachaId: 100, GachaLabelType: model.GachaLabelPremium}}
	return source, entries, config
}

func findPremiumGroup(t *testing.T, banner *PremiumBannerPool, id GroupId) PremiumGroup {
	t.Helper()
	for _, group := range banner.Groups {
		if group.Id == id {
			return group
		}
	}
	t.Fatalf("group %s not found", id)
	return PremiumGroup{}
}

func sequenceIntn(values ...int) func(int) int {
	index := 0
	return func(limit int) int {
		value := 0
		if index < len(values) {
			value = values[index]
		}
		index++
		if limit <= 0 {
			return 0
		}
		return value % limit
	}
}

func assertRateNear(t *testing.T, count, total int, want, tolerance float64) {
	t.Helper()
	got := float64(count) / float64(total)
	if got < want-tolerance || got > want+tolerance {
		t.Fatalf("rate = %.4f, want %.4f ± %.4f", got, want, tolerance)
	}
}

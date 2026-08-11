package gacha

import (
	"math"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestHandleDrawRejectsMissingPhaseAndInsufficientPrice(t *testing.T) {
	h := &GachaHandler{}
	entry := store.GachaCatalogEntry{GachaId: 1, PricePhases: []store.GachaPricePhaseEntry{{PhaseId: 10, PriceType: model.PriceTypeGem, Price: 100, DrawCount: 1}}}
	user := &store.UserState{}
	user.EnsureMaps()
	if _, err := h.HandleDraw(user, entry, 999, 1); err == nil {
		t.Fatal("missing phase was accepted")
	}
	if _, err := h.HandleDraw(user, entry, 10, 1); err == nil {
		t.Fatal("insufficient gems were accepted")
	}
	if user.Gacha.BannerStates[1].DrawCount != 0 {
		t.Fatal("failed draw changed banner state")
	}
}

func TestHandleDrawRejectsOversizedRequestWithoutMutation(t *testing.T) {
	h := &GachaHandler{Pool: &masterdata.GachaCatalog{}, Granter: &store.PossessionGranter{}}
	entry := eventBoxEntry()
	entry.PricePhases[0].DrawCount = 10
	user := &store.UserState{}
	user.EnsureMaps()
	if _, err := h.HandleDraw(user, entry, entry.PricePhases[0].PhaseId, math.MaxInt32); err == nil {
		t.Fatal("oversized draw request was accepted")
	}
	if user.Materials[99] != 0 || len(user.Gacha.BannerStates) != 0 {
		t.Fatal("rejected draw mutated user state")
	}
}

func TestHandleDrawEnforcesStepUpOrderAndAdvancesFromFirstStep(t *testing.T) {
	h := &GachaHandler{Pool: &masterdata.GachaCatalog{}, Granter: &store.PossessionGranter{}}
	entry := eventBoxEntry()
	entry.GachaModeType = model.GachaModeStepup
	entry.MaxStepNumber = 2
	entry.PricePhases = []store.GachaPricePhaseEntry{
		{PhaseId: 11, DrawCount: 1, LimitExecCount: 1, StepNumber: 1},
		{PhaseId: 12, DrawCount: 1, LimitExecCount: 1, StepNumber: 2},
	}
	entry.BoxItems[0].MaxCount = 4
	user := &store.UserState{}
	user.EnsureMaps()

	if _, err := h.HandleDraw(user, entry, 12, 1); err == nil {
		t.Fatal("second step was accepted before first step")
	}
	if _, err := h.HandleDraw(user, entry, 11, 1); err != nil {
		t.Fatalf("first step failed: %v", err)
	}
	if got := user.Gacha.BannerStates[entry.GachaId].StepNumber; got != 2 {
		t.Fatalf("first step advanced to %d, want 2", got)
	}
	if _, err := h.HandleDraw(user, entry, 11, 1); err == nil {
		t.Fatal("first step was accepted twice")
	}
	if _, err := h.HandleDraw(user, entry, 12, 1); err != nil {
		t.Fatalf("second step failed: %v", err)
	}
	state := user.Gacha.BannerStates[entry.GachaId]
	if state.StepNumber != 1 || state.LoopCount != 1 {
		t.Fatalf("step-up did not loop correctly: %+v", state)
	}
}

func TestEventGachaDrawsOnlyItsConfiguredBoxItems(t *testing.T) {
	h := &GachaHandler{
		Pool:    &masterdata.GachaCatalog{Materials: []masterdata.GachaPoolItem{{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 10}}},
		Granter: &store.PossessionGranter{},
	}
	entry := eventBoxEntry()
	user := &store.UserState{}
	user.EnsureMaps()
	result, err := h.HandleDraw(user, entry, entry.PricePhases[0].PhaseId, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].PossessionId != 99 || user.Materials[99] != 1 {
		t.Fatalf("event box used the wrong pool: items=%v materials=%v", result.Items, user.Materials)
	}
	if user.Materials[10] != 0 {
		t.Fatal("global material pool leaked into event box")
	}
	if _, err := h.HandleDraw(user, entry, entry.PricePhases[0].PhaseId, 1); err == nil {
		t.Fatal("exhausted event box charged another draw")
	}
}

func TestHandleResetBoxInitializesPersistentBannerIdentity(t *testing.T) {
	h := &GachaHandler{}
	entry := eventBoxEntry()
	user := &store.UserState{}
	user.EnsureMaps()
	if err := h.HandleResetBox(user, entry); err != nil {
		t.Fatal(err)
	}
	state := user.Gacha.BannerStates[entry.GachaId]
	if state.GachaId != entry.GachaId || state.BoxNumber != 2 {
		t.Fatalf("unexpected reset state: %+v", state)
	}
}

func TestDupExchangesForGradeUsesTierCount(t *testing.T) {
	exchanges := []model.DupExchangeEntry{
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 501, Count: 10},
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 502, Count: 1},
	}
	tests := []struct {
		grade int32
		count int32
	}{
		{grade: 1, count: 20},
		{grade: 2, count: 16},
		{grade: 3, count: 14},
		{grade: 4, count: 12},
		{grade: 5, count: 10},
	}

	for _, tt := range tests {
		got := dupExchangesForGrade(exchanges, tt.grade)
		if got[0].Count != tt.count {
			t.Errorf("grade %d exchange count = %d, want %d", tt.grade, got[0].Count, tt.count)
		}
		if got[1].Count != 1 {
			t.Errorf("grade %d fixed bonus count = %d, want 1", tt.grade, got[1].Count)
		}
	}
	if exchanges[0].Count != 10 {
		t.Fatalf("source exchange count changed to %d", exchanges[0].Count)
	}
}

func TestDupGradeDistributionUsesConfiguredPercentages(t *testing.T) {
	counts := [5]int{}
	for roll := 0; roll < 100; roll++ {
		grade := dupGradeForRoll(roll)
		if grade < 1 || grade > 5 {
			t.Fatalf("roll %d produced invalid grade %d", roll, grade)
		}
		counts[grade-1]++
	}
	want := [5]int{3, 8, 14, 30, 45}
	if counts != want {
		t.Fatalf("grade counts = %v, want %v", counts, want)
	}
}

func TestDrawPremiumAppliesGuaranteePerExecution(t *testing.T) {
	previousRates := premiumRates
	premiumRates = []RateTier{{Weight: 10000, PossessionType: int32(model.PossessionTypeWeapon), RarityType: model.RarityRare}}
	defer func() { premiumRates = previousRates }()

	banner := &masterdata.BannerPool{
		CostumesByRarity: map[int32][]masterdata.GachaPoolItem{},
		WeaponsByRarity: map[int32][]masterdata.GachaPoolItem{
			model.RarityRare:   {{PossessionType: int32(model.PossessionTypeWeapon), PossessionId: 1, RarityType: model.RarityRare}},
			model.RaritySSRare: {{PossessionType: int32(model.PossessionTypeWeapon), PossessionId: 2, RarityType: model.RaritySSRare}},
		},
	}
	h := &GachaHandler{Pool: &masterdata.GachaCatalog{BannerPools: map[int32]*masterdata.BannerPool{1: banner}}}
	entry := store.GachaCatalogEntry{GachaId: 1}
	phase := store.GachaPricePhaseEntry{DrawCount: 10, FixedRarityMin: model.RaritySSRare, FixedCount: 1}

	items := h.drawPremium(entry, phase, 2)
	if len(items) != 20 {
		t.Fatalf("draw count = %d, want 20", len(items))
	}
	for _, index := range []int{9, 19} {
		if items[index].RarityType < model.RaritySSRare {
			t.Fatalf("execution ending at index %d was not guaranteed: %+v", index, items[index])
		}
	}
}

func eventBoxEntry() store.GachaCatalogEntry {
	return store.GachaCatalogEntry{
		GachaId:        1,
		GachaLabelType: model.GachaLabelEvent,
		GachaModeType:  model.GachaModeBox,
		PricePhases:    []store.GachaPricePhaseEntry{{PhaseId: 10, DrawCount: 1}},
		BoxItems:       []store.GachaBoxItemEntry{{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 99, RarityType: int32(model.RarityNormal), Count: 1, MaxCount: 1}},
	}
}

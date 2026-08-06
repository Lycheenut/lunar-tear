package gacha

import (
	"math"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestHandleDrawRejectsMissingPhaseAndInsufficientPrice(t *testing.T) {
	banner := &PremiumBannerPool{
		GachaId: 1,
		Groups: []PremiumGroup{{
			Id:        GroupWeaponOnly4,
			GrantType: GrantWeaponOnly,
			Rarity:    model.RaritySSRare,
			Weight:    1,
			NonPickup: []PoolItem{{WeaponId: 1, RarityType: model.RaritySSRare}},
		}},
	}
	h := &GachaHandler{Premium: &PremiumCatalog{Banners: map[int32]*PremiumBannerPool{1: banner}}}
	entry := store.GachaCatalogEntry{GachaId: 1, GachaLabelType: model.GachaLabelPremium, PricePhases: []store.GachaPricePhaseEntry{{PhaseId: 10, PriceType: model.PriceTypeGem, Price: 100, DrawCount: 1}}}
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
	if user.Gem != (store.UserGemState{}) {
		t.Fatal("failed draw changed gem balance")
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

func eventBoxEntry() store.GachaCatalogEntry {
	return store.GachaCatalogEntry{
		GachaId:        1,
		GachaLabelType: model.GachaLabelEvent,
		GachaModeType:  model.GachaModeBox,
		PricePhases:    []store.GachaPricePhaseEntry{{PhaseId: 10, DrawCount: 1}},
		BoxItems:       []store.GachaBoxItemEntry{{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 99, RarityType: int32(model.RarityNormal), Count: 1, MaxCount: 1}},
	}
}

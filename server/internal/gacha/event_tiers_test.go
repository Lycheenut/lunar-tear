package gacha

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestEventTicketPoolsReadDrawAndResetIndependently(t *testing.T) {
	config := DefaultConfig()
	var entries []store.GachaCatalogEntry
	for i, id := range []int32{329001, 329011, 329021} {
		box := BoxConfig{GroupWeights: BoxGroupWeights{Limited: GroupWeightTotal}, LimitedRewards: []BoxRewardConfig{{PossessionType: 5, PossessionId: int32(100 + i), Count: 2, MaxCount: 1, Jackpot: true}}}
		config.EventBanners[id] = EventBoxConfig{Boxes: []BoxConfig{box, box}}
		entries = append(entries, store.GachaCatalogEntry{GachaId: id, EventGachaBaseId: 329001, GachaLabelType: model.GachaLabelEvent, GachaModeType: model.GachaModeBox, PricePhases: []store.GachaPricePhaseEntry{{PhaseId: id*10 + 1, PriceType: model.PriceTypeConsumableItem, PriceId: int32(6055 + i), Price: 1, DrawCount: 1}}})
	}
	config.EventSchedules = map[int32]EventSchedule{329001: {StartDatetime: 10, EndDatetime: 20}, 329021: {StartDatetime: 12, EndDatetime: 18}}
	raw, hash, err := EncodeConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "gacha.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, loadedHash, exists, err := ReadConfig(path)
	if err != nil || !exists || hash != loadedHash || !reflect.DeepEqual(config, loaded) {
		t.Fatalf("tier config round trip: %v", err)
	}
	if err := validateBoxConfigs(loaded, entries, nil); err != nil {
		t.Fatal(err)
	}
	ApplyConfiguredBoxes(entries, loaded)
	ApplyEventSchedules(entries, loaded)
	if entries[1].StartDatetime != 10 || entries[1].EndDatetime != 20 || entries[2].StartDatetime != 12 || entries[2].EndDatetime != 18 {
		t.Fatalf("tier schedules: %+v", entries)
	}
	h := &GachaHandler{Pool: &masterdata.GachaCatalog{}, Premium: &PremiumCatalog{Config: loaded}, Granter: &store.PossessionGranter{}}
	user := &store.UserState{}
	user.EnsureMaps()
	user.ConsumableItems[6055] = 3
	// Owning bronze must not pay for a silver draw or mutate its stock.
	if _, err := h.HandleDraw(user, entries[1], entries[1].PricePhases[0].PhaseId, 1); err == nil {
		t.Fatal("silver draw accepted bronze ticket")
	}
	if len(user.Gacha.BannerStates) != 0 || user.ConsumableItems[6055] != 3 {
		t.Fatal("failed draw changed state")
	}
	for i, entry := range entries {
		ticket := int32(6055 + i)
		user.ConsumableItems[ticket] = 3
		if err := h.HandleResetBox(user, entry); err == nil {
			t.Fatal("reset before jackpot accepted")
		}
		result, err := h.HandleDraw(user, entry, entry.PricePhases[0].PhaseId, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Items) != 1 || result.Items[0].PossessionId != int32(100+i) || !result.Items[0].IsTarget || user.ConsumableItems[ticket] != 2 {
			t.Fatalf("wrong tier draw: %+v", result)
		}
		if err := h.HandleResetBox(user, entry); err != nil {
			t.Fatal(err)
		}
		state := user.Gacha.BannerStates[entry.GachaId]
		if state.BoxNumber != 2 || len(state.BoxDrewCounts) != 0 {
			t.Fatalf("reset: %+v", state)
		}
		odds, err := h.BoxOdds(entry, state, 0)
		if err != nil || odds.BoxNumber != 2 || len(odds.Items) != 1 || odds.Items[0].Remaining != 1 || odds.Items[0].Rate != 1 {
			t.Fatalf("current tier odds: %+v, %v", odds, err)
		}
		for j := i + 1; j < len(entries); j++ {
			if _, ok := user.Gacha.BannerStates[entries[j].GachaId]; ok {
				t.Fatal("draw/reset changed another tier")
			}
		}
	}
}

func TestLegacyEventConfigKeepsOnlyConfiguredTierActive(t *testing.T) {
	config := DefaultConfig()
	config.EventBanners[329001] = EventBoxConfig{Boxes: []BoxConfig{{GroupWeights: BoxGroupWeights{Limited: GroupWeightTotal}, LimitedRewards: []BoxRewardConfig{{PossessionType: 5, PossessionId: 100, Count: 1, MaxCount: 1, Jackpot: true}}}}}
	entries := []store.GachaCatalogEntry{{GachaId: 329001, GachaLabelType: model.GachaLabelEvent}, {GachaId: 329011, GachaLabelType: model.GachaLabelEvent}, {GachaId: 329021, GachaLabelType: model.GachaLabelEvent}}
	ApplyConfiguredBoxes(entries, config)
	if entries[0].BoxCount != 1 || entries[1].BoxCount != 0 || entries[2].BoxCount != 0 {
		t.Fatalf("legacy tier visibility: %+v", entries)
	}
	config.EventBanners[999999] = config.EventBanners[329001]
	if err := validateBoxConfigs(config, entries, nil); err == nil {
		t.Fatal("unknown tier accepted")
	}
}

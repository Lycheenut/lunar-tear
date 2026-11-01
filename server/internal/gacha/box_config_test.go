package gacha

import (
	"testing"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestValidateEventBoxesRequiresJackpot(t *testing.T) {
	config := DefaultConfig()
	config.EventBanners[300001] = EventBoxConfig{Boxes: []BoxConfig{{
		GroupWeights:   BoxGroupWeights{Limited: GroupWeightTotal},
		LimitedRewards: []BoxRewardConfig{{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 100, Count: 1, MaxCount: 1}},
	}}}
	entries := []store.GachaCatalogEntry{{GachaId: 300001, GachaLabelType: model.GachaLabelEvent}}
	if err := validateBoxConfigs(config, entries, nil); err == nil {
		t.Fatal("Event box without a jackpot was accepted")
	}
	box := config.EventBanners[300001]
	box.Boxes[0].LimitedRewards[0].Jackpot = true
	config.EventBanners[300001] = box
	if err := validateBoxConfigs(config, entries, nil); err != nil {
		t.Fatalf("valid Event box was rejected: %v", err)
	}
}

func TestApplyConfiguredBoxesDisablesUnconfiguredChapter(t *testing.T) {
	entries := []store.GachaCatalogEntry{{
		GachaId:        200001,
		GachaLabelType: model.GachaLabelChapter,
		BoxCount:       1,
		BoxItems:       []store.GachaBoxItemEntry{{PossessionId: 100}},
		PromotionItems: []store.GachaPromotionItem{{PossessionId: 100}},
	}}
	ApplyConfiguredBoxes(entries, DefaultConfig())
	if entries[0].BoxCount != 0 || len(entries[0].BoxItems) != 0 || len(entries[0].PromotionItems) != 0 {
		t.Fatalf("unconfigured Chapter remains active: %+v", entries[0])
	}
}

func TestApplyConfiguredBoxesUsesFeaturedRewards(t *testing.T) {
	config := DefaultConfig()
	config.ChapterBanners[200001] = BoxConfig{
		GroupWeights: BoxGroupWeights{Limited: 8000, Unlimited: 2000},
		LimitedRewards: []BoxRewardConfig{{
			PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 100, Count: 2, MaxCount: 3, Weight: 7, Featured: true,
		}},
		UnlimitedRewards: []BoxRewardConfig{{
			PossessionType: int32(model.PossessionTypeConsumableItem), PossessionId: 1, Count: 5, Weight: 2,
		}},
	}
	entries := []store.GachaCatalogEntry{{GachaId: 200001, GachaLabelType: model.GachaLabelChapter}}
	ApplyConfiguredBoxes(entries, config)
	if len(entries[0].BoxItems) != 2 || entries[0].BoxItems[0].Weight != 0 || entries[0].BoxItems[1].Weight != 2 || entries[0].BoxItems[1].MaxCount != 0 {
		t.Fatalf("configured box items = %+v", entries[0].BoxItems)
	}
	if len(entries[0].PromotionItems) != 1 || entries[0].PromotionItems[0].PossessionId != 100 {
		t.Fatalf("configured featured rewards = %+v", entries[0].PromotionItems)
	}
}

func TestValidateBoxRewardsOnlyRequiresUnlimitedWeights(t *testing.T) {
	config := DefaultConfig()
	config.ChapterBanners[200001] = BoxConfig{
		GroupWeights: BoxGroupWeights{Limited: 8000, Unlimited: 2000},
		LimitedRewards: []BoxRewardConfig{{
			PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 100, Count: 1, MaxCount: 2,
		}},
		UnlimitedRewards: []BoxRewardConfig{{
			PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 200, Count: 1, Weight: 1,
		}},
	}
	entries := []store.GachaCatalogEntry{{GachaId: 200001, GachaLabelType: model.GachaLabelChapter}}
	if err := validateBoxConfigs(config, entries, nil); err != nil {
		t.Fatalf("limited reward without weight was rejected: %v", err)
	}

	box := config.ChapterBanners[200001]
	box.UnlimitedRewards[0].Weight = 0
	config.ChapterBanners[200001] = box
	if err := validateBoxConfigs(config, entries, nil); err == nil {
		t.Fatal("unlimited reward without weight was accepted")
	}
}

func TestAvailableConfiguredBoxDrawCountIncludesWeightlessLimitedStock(t *testing.T) {
	items := []store.GachaBoxItemEntry{{MaxCount: 2, CounterId: 1}}
	if got := availableConfiguredBoxDrawCount(items, BoxGroupWeights{Limited: GroupWeightTotal}, nil); got != 2 {
		t.Fatalf("available draw count = %d, want 2", got)
	}
}

func TestEventPromotionsDistinguishFeaturedAndJackpotRewards(t *testing.T) {
	items := []store.GachaBoxItemEntry{
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 100, Count: 1, MaxCount: 1, IsFeatured: true},
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 200, Count: 1, MaxCount: 1, IsJackpot: true},
	}
	promotions := boxPromotions(items, true)
	if len(promotions) != 2 || promotions[0].IsTarget || !promotions[1].IsTarget {
		t.Fatalf("Event featured/jackpot promotions = %+v", promotions)
	}
}

func TestEventBoxValidatesAndGrantsDebris(t *testing.T) {
	const thoughtID = 79100832
	config := DefaultConfig()
	config.EventBanners[334001] = EventBoxConfig{Boxes: []BoxConfig{{
		GroupWeights: BoxGroupWeights{Limited: GroupWeightTotal},
		LimitedRewards: []BoxRewardConfig{{PossessionType: int32(model.PossessionTypeThought), PossessionId: thoughtID,
			Count: 1, MaxCount: 1, Featured: true, Jackpot: true}},
	}}}
	entries := []store.GachaCatalogEntry{{GachaId: 334001, GachaLabelType: model.GachaLabelEvent,
		PricePhases: []store.GachaPricePhaseEntry{{PhaseId: 1, PriceType: model.PriceTypeConsumableItem, PriceId: 6066, Price: 1, DrawCount: 1}}}}
	valid := map[[2]int32]bool{{int32(model.PossessionTypeThought), thoughtID}: true}
	if err := validateBoxConfigs(config, entries, valid); err != nil {
		t.Fatalf("valid Debris rejected: %v", err)
	}
	if err := validateBoxConfigs(config, entries, map[[2]int32]bool{}); err == nil {
		t.Fatal("unknown Debris accepted")
	}
	ApplyConfiguredBoxes(entries, config)
	h := &GachaHandler{Premium: &PremiumCatalog{Config: config}, Granter: &store.PossessionGranter{}}
	user := &store.UserState{}
	user.EnsureMaps()
	user.ConsumableItems[6066] = 1
	result, err := h.HandleDraw(user, entries[0], 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].PossessionId != thoughtID || !result.Items[0].IsTarget || len(user.Thoughts) != 1 || user.ConsumableItems[6066] != 0 {
		t.Fatalf("Debris was not granted: %+v / %+v", result, user.Thoughts)
	}
	for _, thought := range user.Thoughts {
		if thought.ThoughtId != thoughtID {
			t.Fatalf("wrong Debris: %+v", thought)
		}
	}
	if err := h.HandleResetBox(user, entries[0]); err != nil {
		t.Fatalf("completed Debris box did not reset: %v", err)
	}
}

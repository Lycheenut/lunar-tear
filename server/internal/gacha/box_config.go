package gacha

import (
	"fmt"
	"math"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type BoxGroupWeights struct {
	Limited   int `json:"limited"`
	Unlimited int `json:"unlimited"`
}

type BoxRewardConfig struct {
	PossessionType int32 `json:"possessionType"`
	PossessionId   int32 `json:"possessionId,omitempty"`
	RarityType     int32 `json:"rarityType,omitempty"`
	Count          int32 `json:"count"`
	MaxCount       int32 `json:"maxCount,omitempty"`
	Weight         int32 `json:"weight,omitempty"`
	Featured       bool  `json:"featured,omitempty"`
	Jackpot        bool  `json:"jackpot,omitempty"`
}

type BoxConfig struct {
	GroupWeights     BoxGroupWeights   `json:"groupWeights"`
	LimitedRewards   []BoxRewardConfig `json:"limitedRewards"`
	UnlimitedRewards []BoxRewardConfig `json:"unlimitedRewards"`
}

type EventBoxConfig struct {
	Boxes []BoxConfig `json:"boxes"`
}

func validateBoxConfigs(config *Config, entries []store.GachaCatalogEntry, validRewards map[[2]int32]bool) error {
	labels := make(map[int32]int32, len(entries))
	for _, entry := range entries {
		labels[entry.GachaId] = entry.GachaLabelType
	}
	for gachaId, box := range config.ChapterBanners {
		if labels[gachaId] != model.GachaLabelChapter {
			return fmt.Errorf("configured Chapter Gacha %d is not a chapter banner", gachaId)
		}
		if err := validateBoxConfig(gachaId, 1, box, false, validRewards); err != nil {
			return err
		}
	}
	for gachaId, event := range config.EventBanners {
		if labels[gachaId] != model.GachaLabelEvent {
			return fmt.Errorf("configured Event Gacha %d is not an event banner", gachaId)
		}
		if len(event.Boxes) == 0 {
			return fmt.Errorf("Event Gacha %d must contain at least one box", gachaId)
		}
		for i, box := range event.Boxes {
			if err := validateBoxConfig(gachaId, i+1, box, true, validRewards); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBoxConfig(gachaId int32, boxNumber int, box BoxConfig, event bool, validRewards map[[2]int32]bool) error {
	if box.GroupWeights.Limited < 0 || box.GroupWeights.Unlimited < 0 ||
		box.GroupWeights.Limited+box.GroupWeights.Unlimited != GroupWeightTotal {
		return fmt.Errorf("Gacha %d box %d limited and unlimited group weights must be non-negative and total %d", gachaId, boxNumber, GroupWeightTotal)
	}
	if box.GroupWeights.Limited > 0 && len(box.LimitedRewards) == 0 {
		return fmt.Errorf("Gacha %d box %d has positive limited weight but no limited rewards", gachaId, boxNumber)
	}
	if box.GroupWeights.Unlimited > 0 && len(box.UnlimitedRewards) == 0 {
		return fmt.Errorf("Gacha %d box %d has positive unlimited weight but no unlimited rewards", gachaId, boxNumber)
	}
	if err := validateRewardGroup(gachaId, boxNumber, "limited", box.LimitedRewards, true, event, validRewards); err != nil {
		return err
	}
	if err := validateRewardGroup(gachaId, boxNumber, "unlimited", box.UnlimitedRewards, false, event, validRewards); err != nil {
		return err
	}
	if event {
		if box.GroupWeights.Limited <= 0 {
			return fmt.Errorf("Event Gacha %d box %d limited reward group must have positive probability", gachaId, boxNumber)
		}
		jackpots := 0
		for _, reward := range box.LimitedRewards {
			if reward.Jackpot {
				jackpots++
			}
		}
		if jackpots == 0 {
			return fmt.Errorf("Event Gacha %d box %d must have at least one jackpot reward", gachaId, boxNumber)
		}
	}
	return nil
}

func validateRewardGroup(gachaId int32, boxNumber int, group string, rewards []BoxRewardConfig, limited, event bool, validRewards map[[2]int32]bool) error {
	var totalWeight int64
	for i, reward := range rewards {
		if !supportedBoxPossessionType(reward.PossessionType) || (reward.PossessionType != int32(model.PossessionTypeFreeGem) && reward.PossessionId <= 0) {
			return fmt.Errorf("Gacha %d box %d %s reward %d has an invalid possession", gachaId, boxNumber, group, i+1)
		}
		if validRewards != nil && !validRewards[[2]int32{reward.PossessionType, reward.PossessionId}] {
			return fmt.Errorf("Gacha %d box %d %s reward %d does not exist in master data", gachaId, boxNumber, group, i+1)
		}
		if reward.Count <= 0 {
			return fmt.Errorf("Gacha %d box %d %s reward %d count must be positive", gachaId, boxNumber, group, i+1)
		}
		if limited && reward.MaxCount <= 0 {
			return fmt.Errorf("Gacha %d box %d limited reward %d stock must be positive", gachaId, boxNumber, i+1)
		}
		if !limited && reward.MaxCount != 0 {
			return fmt.Errorf("Gacha %d box %d unlimited reward %d must not specify stock", gachaId, boxNumber, i+1)
		}
		if !limited {
			if reward.Weight <= 0 {
				return fmt.Errorf("Gacha %d box %d %s reward %d weight must be positive", gachaId, boxNumber, group, i+1)
			}
			totalWeight += int64(reward.Weight)
			if totalWeight > math.MaxInt {
				return fmt.Errorf("Gacha %d box %d %s reward weights are too large", gachaId, boxNumber, group)
			}
		}
		if reward.Jackpot && (!event || !limited) {
			return fmt.Errorf("Gacha %d box %d reward %d can only be a jackpot in an Event Gacha limited group", gachaId, boxNumber, i+1)
		}
	}
	return nil
}

func supportedBoxPossessionType(possessionType int32) bool {
	switch model.PossessionType(possessionType) {
	case model.PossessionTypeMaterial, model.PossessionTypeWeapon, model.PossessionTypeCompanion,
		model.PossessionTypeConsumableItem, model.PossessionTypeFreeGem:
		return true
	default:
		return false
	}
}

func boxItems(box BoxConfig) []store.GachaBoxItemEntry {
	items := make([]store.GachaBoxItemEntry, 0, len(box.LimitedRewards)+len(box.UnlimitedRewards))
	appendRewards := func(rewards []BoxRewardConfig, limited bool) {
		for _, reward := range rewards {
			maxCount := int32(0)
			weight := int32(0)
			if limited {
				maxCount = reward.MaxCount
			} else {
				weight = reward.Weight
			}
			items = append(items, store.GachaBoxItemEntry{
				PossessionType: reward.PossessionType,
				PossessionId:   reward.PossessionId,
				RarityType:     reward.RarityType,
				Count:          reward.Count,
				MaxCount:       maxCount,
				CounterId:      int32(len(items) + 1),
				Weight:         weight,
				IsFeatured:     reward.Featured,
				IsJackpot:      reward.Jackpot,
			})
		}
	}
	appendRewards(box.LimitedRewards, true)
	appendRewards(box.UnlimitedRewards, false)
	return items
}

func boxPromotions(items []store.GachaBoxItemEntry, event bool) []store.GachaPromotionItem {
	var promotions []store.GachaPromotionItem
	for _, item := range items {
		if !item.IsFeatured && !item.IsJackpot {
			continue
		}
		promotions = append(promotions, store.GachaPromotionItem{
			PossessionType:   item.PossessionType,
			PossessionId:     item.PossessionId,
			Count:            item.Count,
			MaxDrawableCount: item.MaxCount,
			CounterId:        item.CounterId,
			IsTarget:         !event || item.IsJackpot,
		})
	}
	return promotions
}

// ApplyConfiguredBoxes installs the first box for catalog visibility. Event
// boxes are replaced with the user's current box by GachaHandler.EntryForState.
func ApplyConfiguredBoxes(entries []store.GachaCatalogEntry, config *Config) {
	if config == nil {
		return
	}
	for i := range entries {
		var box BoxConfig
		var configured bool
		switch entries[i].GachaLabelType {
		case model.GachaLabelChapter:
			box, configured = config.ChapterBanners[entries[i].GachaId]
			if configured {
				entries[i].BoxCount = 1
			} else {
				entries[i].BoxCount = 0
				entries[i].BoxItems = nil
				entries[i].PromotionItems = nil
			}
		case model.GachaLabelEvent:
			event, ok := config.EventBanners[entries[i].GachaId]
			if ok && len(event.Boxes) > 0 {
				box, configured = event.Boxes[0], true
				entries[i].BoxCount = int32(len(event.Boxes))
			} else {
				entries[i].BoxCount = 0
				entries[i].BoxItems = nil
				entries[i].PromotionItems = nil
			}
		}
		if configured {
			entries[i].BoxItems = boxItems(box)
			entries[i].PromotionItems = boxPromotions(entries[i].BoxItems, entries[i].GachaLabelType == model.GachaLabelEvent)
		}
	}
}

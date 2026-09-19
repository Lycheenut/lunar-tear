package gacha

import (
	"fmt"
	"slices"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type PremiumItemOdds struct {
	PoolItem
	Pickup bool
	Rate   float64
}

type PremiumGroupOdds struct {
	GrantType GrantType
	Star      int32
	Rate      float64
	Items     []PremiumItemOdds
}

// PremiumSlotOdds combines consecutive positions with identical distributions.
type PremiumSlotOdds struct {
	First, Last int
	Groups      []PremiumGroupOdds
}

// PremiumOdds uses the same slot weights, rarity filter and pickup split as DrawPremium.
// Rates are fractions of one, before display rounding.
func PremiumOdds(bp *PremiumBannerPool, entry store.GachaCatalogEntry, phase store.GachaPricePhaseEntry) ([]PremiumSlotOdds, error) {
	if bp == nil || phase.DrawCount <= 0 {
		return nil, fmt.Errorf("premium Gacha pool or price phase is unavailable")
	}
	weights := adjustedGroupWeights(bp.Groups, premiumRateMultiplier(entry, phase))
	var result []PremiumSlotOdds
	var previousWeights []int
	var previousMinimum int32
	for slot := 0; slot < int(phase.DrawCount); slot++ {
		slotWeights, minimum := premiumSlotWeights(bp.Groups, weights, int(phase.DrawCount), slot,
			phase.FixedRarityMin, int(phase.FixedCount), entry.GachaId == model.GachaIdGuaranteedThreeStarOrHigher)
		if len(result) > 0 && minimum == previousMinimum && slices.Equal(slotWeights, previousWeights) {
			result[len(result)-1].Last = slot + 1
			continue
		}
		total := 0
		for i, group := range bp.Groups {
			if group.Rarity >= minimum && group.ItemCount() > 0 && slotWeights[i] > 0 {
				total += slotWeights[i]
			}
		}
		if total <= 0 {
			return nil, fmt.Errorf("premium Gacha has no available rewards for slot %d", slot+1)
		}
		odds := PremiumSlotOdds{First: slot + 1, Last: slot + 1}
		for i, group := range bp.Groups {
			g := PremiumGroupOdds{GrantType: group.GrantType, Star: group.Star}
			if group.Rarity >= minimum && group.ItemCount() > 0 && slotWeights[i] > 0 {
				g.Rate = float64(slotWeights[i]) / float64(total)
			}
			ordinaryShare := 1.0
			if len(group.Pickup) > 0 {
				ordinaryShare = 0.5
				for _, item := range group.Pickup {
					g.Items = append(g.Items, PremiumItemOdds{item, true, g.Rate * 0.5 / float64(len(group.Pickup))})
				}
			}
			for _, item := range group.NonPickup {
				g.Items = append(g.Items, PremiumItemOdds{item, false, g.Rate * ordinaryShare / float64(len(group.NonPickup))})
			}
			odds.Groups = append(odds.Groups, g)
		}
		result = append(result, odds)
		previousWeights, previousMinimum = slotWeights, minimum
	}
	return result, nil
}

type BoxItemOdds struct {
	store.GachaBoxItemEntry
	Remaining int32
	Unlimited bool
	Rate      float64
}

type BoxOdds struct {
	BoxNumber int32
	Items     []BoxItemOdds
}

// BoxOdds uses the same current box and group weights as draws, without changing
// player state. Chapter counters are treated as reset at the monthly boundary.
func (h *GachaHandler) BoxOdds(entry store.GachaCatalogEntry, state store.GachaBannerState, nowMillis int64) (BoxOdds, error) {
	box, boxCount, configured := h.configuredBox(entry, &state)
	if !configured {
		return BoxOdds{}, fmt.Errorf("box Gacha %d is not configured", entry.GachaId)
	}
	counts := state.BoxDrewCounts
	chapter := entry.GachaLabelType == model.GachaLabelChapter
	if chapter && counts[model.ChapterGachaMonthCounterId] != gametime.BusinessMonthKey(nowMillis) {
		counts = nil
	}
	var limited, unlimited int64
	rewards := boxItems(box)
	items := make([]BoxItemOdds, 0, len(rewards))
	for i, item := range rewards {
		odds := BoxItemOdds{GachaBoxItemEntry: item, Unlimited: item.MaxCount <= 0}
		odds.Count = positiveCount(item.Count)
		if odds.Unlimited {
			if box.GroupWeights.Unlimited > 0 {
				unlimited += int64(max(item.Weight, 0))
			}
		} else {
			odds.Remaining = max(item.MaxCount-counts[chapterCounterId(item, i)], 0)
			if box.GroupWeights.Limited > 0 {
				limited += int64(odds.Remaining)
			}
		}
		items = append(items, odds)
	}
	limitedShare := 1.0
	if unlimited > 0 {
		limitedShare = 0
		if limited > 0 {
			limitedShare = float64(box.GroupWeights.Limited) / float64(box.GroupWeights.Limited+box.GroupWeights.Unlimited)
		}
	}
	for i := range items {
		item := &items[i]
		if item.Unlimited && unlimited > 0 {
			item.Rate = (1 - limitedShare) * float64(max(item.Weight, 0)) / float64(unlimited)
		} else if !item.Unlimited && limited > 0 {
			item.Rate = limitedShare * float64(item.Remaining) / float64(limited)
		}
	}
	return BoxOdds{BoxNumber: currentBoxNumber(&state, boxCount), Items: items}, nil
}

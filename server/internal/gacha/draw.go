package gacha

import (
	"fmt"
	"math/rand"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type DrawnItem struct {
	PossessionType int32
	PossessionId   int32
	RarityType     model.RarityType
	CharacterId    int32
	Count          int32
	CounterId      int32
}

func DrawPremium(bp *PremiumBannerPool, count int, fixedRarityMin int32, fixedCount int, rateMultiplier float64) ([]DrawnItem, error) {
	return drawPremiumWithIntn(bp, count, fixedRarityMin, fixedCount, rateMultiplier, rand.Intn)
}

func DrawPremiumTenthSlot(bp *PremiumBannerPool, rateMultiplier float64) ([]DrawnItem, error) {
	return drawPremiumTenthSlotWithIntn(bp, rateMultiplier, rand.Intn)
}

func drawPremiumTenthSlotWithIntn(bp *PremiumBannerPool, rateMultiplier float64, intn func(int) int) ([]DrawnItem, error) {
	return drawPremiumWithOptions(bp, 1, 0, 0, rateMultiplier, true, intn)
}

func drawPremiumWithIntn(bp *PremiumBannerPool, count int, fixedRarityMin int32, fixedCount int, rateMultiplier float64, intn func(int) int) ([]DrawnItem, error) {
	return drawPremiumWithOptions(bp, count, fixedRarityMin, fixedCount, rateMultiplier, false, intn)
}

func drawPremiumWithOptions(bp *PremiumBannerPool, count int, fixedRarityMin int32, fixedCount int, rateMultiplier float64, forceTenthSlot bool, intn func(int) int) ([]DrawnItem, error) {
	if bp == nil {
		return nil, fmt.Errorf("premium Gacha pool is not configured")
	}
	result := make([]DrawnItem, 0, count)
	weights := adjustedGroupWeights(bp.Groups, rateMultiplier)

	for i := range count {
		isGuaranteeSlot := fixedCount > 0 && i >= count-fixedCount
		slotWeights := weights
		if forceTenthSlot || (i+1)%int(model.PremiumMultiPullCount) == 0 {
			slotWeights = transferTwoStarWeightsToThreeStar(bp.Groups, weights)
		}
		minimumRarity := int32(0)
		if isGuaranteeSlot && fixedRarityMin > minimumRarity {
			minimumRarity = fixedRarityMin
		}
		item, err := rollConfiguredItem(bp.Groups, slotWeights, minimumRarity, intn)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func transferTwoStarWeightsToThreeStar(groups []PremiumGroup, weights []int) []int {
	transferred := append([]int(nil), weights...)
	threeStarByGrantType := make(map[GrantType]int)
	for i, group := range groups {
		if group.Star == 3 {
			threeStarByGrantType[group.GrantType] = i
		}
	}
	for i, group := range groups {
		if group.Star != 2 || i >= len(transferred) {
			continue
		}
		weight := transferred[i]
		transferred[i] = 0
		if target, ok := threeStarByGrantType[group.GrantType]; ok && target < len(transferred) {
			transferred[target] += weight
		}
	}
	return transferred
}

func DrawBox(items []BoxItem, count int) []DrawnItem {
	var available []int
	for i, item := range items {
		remaining := item.MaxCount - item.DrewCount
		for range remaining {
			available = append(available, i)
		}
	}

	result := make([]DrawnItem, 0, count)
	for i := 0; i < count && len(available) > 0; i++ {
		pick := rand.Intn(len(available))
		idx := available[pick]
		item := items[idx]
		result = append(result, DrawnItem{
			PossessionType: item.PossessionType,
			PossessionId:   item.PossessionId,
			RarityType:     item.RarityType,
			Count:          positiveCount(item.Count),
			CounterId:      item.CounterId,
		})
		items[idx].DrewCount++
		available = append(available[:pick], available[pick+1:]...)
	}
	return result
}

func DrawReward(materials []masterdata.GachaPoolItem, count int) []DrawnItem {
	if len(materials) == 0 {
		return nil
	}
	result := make([]DrawnItem, 0, count)
	for range count {
		m := materials[rand.Intn(len(materials))]
		result = append(result, DrawnItem{
			PossessionType: m.PossessionType,
			PossessionId:   m.PossessionId,
			RarityType:     m.RarityType,
			Count:          1,
		})
	}
	return result
}

type BoxItem struct {
	PossessionType int32
	PossessionId   int32
	RarityType     model.RarityType
	Count          int32
	MaxCount       int32
	DrewCount      int32
	IsTarget       bool
	CounterId      int32
}

const (
	chapterProbabilityTotal     = 100
	chapterUnlimitedProbability = 20
	chapterLimitedProbability   = chapterProbabilityTotal - chapterUnlimitedProbability
)

func drawChapterWithIntn(items []store.GachaBoxItemEntry, drewCounts map[int32]int32, count int, monthKey int32, intn func(int) int) ([]DrawnItem, error) {
	if drewCounts[model.ChapterGachaMonthCounterId] != monthKey {
		clear(drewCounts)
		drewCounts[model.ChapterGachaMonthCounterId] = monthKey
	}

	result := make([]DrawnItem, 0, count)
	for range count {
		limitedWeight := 0
		unlimitedWeight := 0
		for i, item := range items {
			counterId := chapterCounterId(item, i)
			if item.Weight > 0 && (item.MaxCount <= 0 || drewCounts[counterId] < item.MaxCount) {
				if item.MaxCount > 0 {
					limitedWeight += int(item.Weight)
				} else {
					unlimitedWeight += int(item.Weight)
				}
			}
		}
		if limitedWeight <= 0 && unlimitedWeight <= 0 {
			return nil, fmt.Errorf("chapter Gacha has no available rewards")
		}

		drawLimited := limitedWeight > 0
		if limitedWeight > 0 && unlimitedWeight > 0 {
			drawLimited = intn(chapterProbabilityTotal) < chapterLimitedProbability
		} else if limitedWeight <= 0 {
			drawLimited = false
		}
		totalWeight := unlimitedWeight
		if drawLimited {
			totalWeight = limitedWeight
		}
		roll := intn(totalWeight)
		selected := -1
		for i, item := range items {
			counterId := chapterCounterId(item, i)
			if item.Weight <= 0 || (item.MaxCount > 0) != drawLimited || (item.MaxCount > 0 && drewCounts[counterId] >= item.MaxCount) {
				continue
			}
			if roll >= int(item.Weight) {
				roll -= int(item.Weight)
				continue
			}
			selected = i
			break
		}
		if selected < 0 {
			return nil, fmt.Errorf("chapter Gacha reward selection failed")
		}

		item := items[selected]
		counterId := chapterCounterId(item, selected)
		result = append(result, DrawnItem{
			PossessionType: item.PossessionType,
			PossessionId:   item.PossessionId,
			RarityType:     model.RarityType(item.RarityType),
			Count:          positiveCount(item.Count),
			CounterId:      counterId,
		})
		if item.MaxCount > 0 {
			drewCounts[counterId]++
		}
	}
	return result, nil
}

func chapterCounterId(item store.GachaBoxItemEntry, index int) int32 {
	if item.CounterId > 0 {
		return item.CounterId
	}
	if item.PossessionId > 0 {
		return item.PossessionId
	}
	return int32(index + 1)
}

func positiveCount(count int32) int32 {
	if count > 0 {
		return count
	}
	return 1
}

func adjustedGroupWeights(groups []PremiumGroup, multiplier float64) []int {
	adjusted := make([]int, len(groups))
	for i, group := range groups {
		adjusted[i] = group.Weight
	}
	if multiplier == 1.0 || multiplier == 0 {
		return adjusted
	}

	var fourStarExtra int
	var nonFourStar int
	for i, group := range groups {
		if group.Rarity >= model.RaritySSRare {
			extra := int(float64(adjusted[i]) * (multiplier - 1.0))
			adjusted[i] += extra
			fourStarExtra += extra
		} else {
			nonFourStar += adjusted[i]
		}
	}
	if nonFourStar > 0 && fourStarExtra > 0 {
		for i, group := range groups {
			if group.Rarity < model.RaritySSRare && adjusted[i] > 0 {
				reduction := fourStarExtra * adjusted[i] / nonFourStar
				adjusted[i] -= reduction
				if adjusted[i] < 1 {
					adjusted[i] = 1
				}
			}
		}
	}
	return adjusted
}

func rollConfiguredItem(groups []PremiumGroup, weights []int, minimumRarity model.RarityType, intn func(int) int) (DrawnItem, error) {
	totalWeight := 0
	for i, group := range groups {
		if group.Rarity >= minimumRarity && group.ItemCount() > 0 && weights[i] > 0 {
			totalWeight += weights[i]
		}
	}
	if totalWeight <= 0 {
		return DrawnItem{}, fmt.Errorf("configured Gacha has no group available for minimum rarity %d", minimumRarity)
	}

	roll := intn(totalWeight)
	for i, group := range groups {
		if group.Rarity < minimumRarity || group.ItemCount() == 0 || weights[i] <= 0 {
			continue
		}
		if roll >= weights[i] {
			roll -= weights[i]
			continue
		}
		items := group.NonPickup
		if len(group.Pickup) > 0 && intn(2) == 0 {
			items = group.Pickup
		}
		if len(items) == 0 {
			return DrawnItem{}, fmt.Errorf("configured Gacha group %s selected an empty item subset", group.Id)
		}
		return items[intn(len(items))].DrawnItem(), nil
	}
	return DrawnItem{}, fmt.Errorf("configured Gacha group selection failed")
}

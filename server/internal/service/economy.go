package service

import (
	"math"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

const (
	// The normal Glorious Success chance is 2%; campaign rates are normalized to permil.
	standardGreatSuccessRatePermil = int32(20)
	// A Glorious Success doubles the effective enhancement material experience.
	greatSuccessExpMultiplier = int64(2)
)

func finalizeEnhancementExp(baseExp int64, ratePermil int32, roll int) (int32, bool, error) {
	isGreatSuccess := baseExp > 0 && roll < int(ratePermil)
	if isGreatSuccess {
		baseExp *= greatSuccessExpMultiplier
	}
	if baseExp < 0 || baseExp > math.MaxInt32 {
		return 0, false, status.Error(codes.InvalidArgument, "enhancement experience is too large")
	}
	return int32(baseExp), isGreatSuccess, nil
}

type enhancementSelection struct {
	count      int32
	expPerUnit int64
}

type enhancementMaterialSelection struct {
	materialId int32
	enhancementSelection
}

func sortEnhancementMaterialSelections(selections []enhancementMaterialSelection) {
	sort.Slice(selections, func(i, j int) bool {
		if selections[i].expPerUnit != selections[j].expPerUnit {
			return selections[i].expPerUnit > selections[j].expPerUnit
		}
		return selections[i].materialId < selections[j].materialId
	})
}

func consumeEnhancementMaterials(selections []enhancementMaterialSelection, requiredExp, multiplier int64) (int64, int32, map[int32]int32) {
	sortEnhancementMaterialSelections(selections)
	consumptionSelections := make([]enhancementSelection, len(selections))
	for i := range selections {
		consumptionSelections[i] = selections[i].enhancementSelection
	}
	consumed, effectiveExp, consumedCount := consumeEnhancementSelections(consumptionSelections, requiredExp, multiplier)
	surplus := make(map[int32]int32)
	for i, selection := range selections {
		if count := selection.count - consumed[i]; count > 0 {
			surplus[selection.materialId] = count
		}
		selection.count = consumed[i]
		selections[i] = selection
	}
	return effectiveExp, consumedCount, surplus
}

func consumeEnhancementSelections(selections []enhancementSelection, requiredExp, multiplier int64) ([]int32, int64, int32) {
	consumed := make([]int32, len(selections))
	var effectiveExp int64
	var consumedCount int32
	for i, selection := range selections {
		if effectiveExp >= requiredExp {
			break
		}
		effectiveExpPerUnit := selection.expPerUnit * multiplier
		remainingExp := requiredExp - effectiveExp
		count := int64(selection.count)
		if needed := (remainingExp + effectiveExpPerUnit - 1) / effectiveExpPerUnit; count > needed {
			count = needed
		}
		consumed[i] = int32(count)
		consumedCount += int32(count)
		effectiveExp += count * effectiveExpPerUnit
	}
	return consumed, effectiveExp, consumedCount
}

func enhancementExpCap(thresholds []int32, maxLevel int32) (int32, bool) {
	if len(thresholds) == 0 {
		return 0, false
	}
	capIndex := len(thresholds) - 1
	if maxLevel > 0 && int(maxLevel) < len(thresholds) {
		capIndex = int(maxLevel)
	}
	return thresholds[capIndex], true
}

func enhancementCampaignFilter(catalog *campaign.Catalog, user *store.UserState, nowMillis int64) campaign.Filter {
	return catalog.FilterForUser(userCampaignStatusContext(user, nowMillis))
}

func materialCost(materialId, count int32) store.PossessionCost {
	return store.PossessionCost{
		PossessionType: model.PossessionTypeMaterial,
		PossessionId:   materialId,
		Count:          count,
	}
}

func consumableCost(consumableItemId, count int32) store.PossessionCost {
	return store.PossessionCost{
		PossessionType: model.PossessionTypeConsumableItem,
		PossessionId:   consumableItemId,
		Count:          count,
	}
}

func deductUpgradeCosts(user *store.UserState, operation string, costs []store.PossessionCost) error {
	if len(costs) == 0 {
		return nil
	}
	if err := store.DeductPossessions(user, costs); err != nil {
		return status.Errorf(codes.FailedPrecondition, "%s: %v", operation, err)
	}
	return nil
}

func selectedMaterialCosts(materials map[int32]int32, options []masterdata.MaterialOption) ([]store.PossessionCost, int32, error) {
	if len(materials) == 0 || len(options) == 0 {
		return nil, 0, status.Error(codes.InvalidArgument, "required materials are missing")
	}
	requiredByMaterial := make(map[int32]int64, len(options))
	commonUnits := int64(1)
	for _, option := range options {
		if option.Count <= 0 {
			return nil, 0, status.Error(codes.FailedPrecondition, "material requirement is invalid")
		}
		required := int64(option.Count)
		requiredByMaterial[option.MaterialId] = required
		commonUnits = commonUnits / greatestCommonDivisor(commonUnits, required) * required
	}

	costs := make([]store.PossessionCost, 0, len(materials))
	var consumedUnits int64
	for materialId, count := range materials {
		requiredPerStep, allowed := requiredByMaterial[materialId]
		if count <= 0 || !allowed {
			return nil, 0, status.Errorf(codes.InvalidArgument, "invalid material selection %d", materialId)
		}
		consumedUnits += int64(count) * (commonUnits / requiredPerStep)
		costs = append(costs, materialCost(materialId, count))
	}
	if consumedUnits%commonUnits != 0 {
		return nil, 0, status.Error(codes.InvalidArgument, "material count does not satisfy a complete upgrade step")
	}
	steps := consumedUnits / commonUnits
	if steps > int64(^uint32(0)>>1) {
		return nil, 0, status.Error(codes.InvalidArgument, "material count is too large")
	}
	return costs, int32(steps), nil
}

func greatestCommonDivisor(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

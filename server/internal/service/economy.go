package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

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

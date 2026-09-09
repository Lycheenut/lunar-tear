package store

import (
	"github.com/google/uuid"

	"lunar-tear/server/internal/model"
)

type WeaponEnhancedRef struct {
	WeaponId        int32
	Level           int32
	Exp             int32
	LimitBreakCount int32
	SkillLevels     map[int32]int32 // slot -> level
	AbilityLevels   map[int32]int32 // slot -> level
}

type PartsEnhancedRef struct {
	PartsId                int32
	PartsStatusMainId      int32
	Level                  int32
	IsRandomSubStatusCount bool
	SubStatusCount         int32
	SubStatuses            []PartsStatusSubState
}

func (g *PossessionGranter) validEnhancedParts(enhanced PartsEnhancedRef) bool {
	part, ok := g.PartsById[enhanced.PartsId]
	if !ok || enhanced.Level < 1 || enhanced.PartsStatusMainId <= 0 || enhanced.SubStatusCount < 0 || enhanced.SubStatusCount > 4 {
		return false
	}
	usedSlots, usedIDs := map[int32]bool{}, map[int32]bool{}
	for _, sub := range enhanced.SubStatuses {
		if sub.StatusIndex < 1 || sub.StatusIndex > 4 || sub.Level < 1 || usedSlots[sub.StatusIndex] || usedIDs[sub.PartsStatusSubLotteryId] {
			return false
		}
		usedSlots[sub.StatusIndex], usedIDs[sub.PartsStatusSubLotteryId] = true, true
	}
	count := enhanced.SubStatusCount
	if enhanced.IsRandomSubStatusCount {
		count = 4
	}
	for slot := range usedSlots {
		count = max(count, slot)
	}
	for _, id := range g.PartsSubStatusPool[part.PartsStatusSubLotteryGroupId] {
		if _, ok := g.PartsSubStatusDefs[id]; !ok {
			return false
		}
		usedIDs[id] = true
	}
	return len(usedIDs) >= int(count)
}

func (g *PossessionGranter) grantEnhancedParts(user *UserState, enhanced PartsEnhancedRef, nowMillis int64) {
	part := g.PartsById[enhanced.PartsId]
	count := enhanced.SubStatusCount
	if enhanced.IsRandomSubStatusCount {
		// Use the ordinary rank lottery for the count, while retaining the template's item and main status.
		_, rolled, _ := g.rollPartsVariant(enhanced.PartsId)
		count = max(0, rolled.PartsInitialLotteryId-1)
	}
	key := uuid.New().String()
	user.Parts[key] = PartsState{
		UserPartsUuid: key, PartsId: enhanced.PartsId, Level: enhanced.Level,
		PartsStatusMainId: enhanced.PartsStatusMainId, AcquisitionDatetime: nowMillis,
	}
	if _, exists := user.PartsGroupNotes[part.PartsGroupId]; !exists {
		user.PartsGroupNotes[part.PartsGroupId] = PartsGroupNoteState{
			PartsGroupId: part.PartsGroupId, FirstAcquisitionDatetime: nowMillis, LatestVersion: nowMillis,
		}
	}
	for _, sub := range enhanced.SubStatuses {
		sub.UserPartsUuid, sub.LatestVersion = key, nowMillis
		user.PartsStatusSubs[PartsStatusSubKey{UserPartsUuid: key, StatusIndex: sub.StatusIndex}] = sub
		count = max(count, sub.StatusIndex)
	}
	pool := g.PartsSubStatusPool[part.PartsStatusSubLotteryGroupId]
	for slot := int32(1); slot <= count; slot++ {
		subKey := PartsStatusSubKey{UserPartsUuid: key, StatusIndex: slot}
		if _, exists := user.PartsStatusSubs[subKey]; exists {
			continue
		}
		id, ok := PickUniquePartsSubStatus(pool, user, key)
		if !ok {
			break
		}
		def := g.PartsSubStatusDefs[id]
		value := def.StatusChangeInitialValue
		if def.StatusFunc != nil {
			value = def.StatusFunc(enhanced.Level)
		}
		user.PartsStatusSubs[subKey] = PartsStatusSubState{
			UserPartsUuid: key, StatusIndex: slot, PartsStatusSubLotteryId: id, Level: enhanced.Level,
			StatusKindType: def.StatusKindType, StatusCalculationType: def.StatusCalculationType,
			StatusChangeValue: value, LatestVersion: nowMillis,
		}
	}
}

func (g *PossessionGranter) GrantOrSellEnhancedPartsDrop(user *UserState, enhancedID int32, raritySet, rankSet map[int32]bool, nowMillis int64) (int32, bool) {
	enhanced, ok := g.PartsEnhancedById[enhancedID]
	if !ok || !g.validEnhancedParts(enhanced) {
		return enhancedID, false
	}
	part := g.PartsById[enhanced.PartsId]
	price := g.PartsSellPriceByRarity[part.RarityType]
	if price != nil && raritySet[part.RarityType] && rankSet[part.PartsInitialLotteryId] {
		user.ConsumableItems[g.GoldConsumableItemId] += price(enhanced.Level)
		AddMissionCount(user, int32(model.MissionClearConditionTypePossessionAddByCount), 1, enhancedID, int32(model.PossessionTypePartsEnhanced))
		return enhanced.PartsId, true
	}
	g.GrantFull(user, model.PossessionTypePartsEnhanced, enhancedID, 1, nowMillis)
	return enhanced.PartsId, false
}

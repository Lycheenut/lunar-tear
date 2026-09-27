package store

import "lunar-tear/server/internal/model"

// RepairPartsWithoutSubStatuses is the offline compatibility repair used by
// cmd/repair-parts. Fill all four sub-status slots; keep the item and main status
// and never reroll existing stats. An incomplete roll leaves the part untouched.
func (g *PossessionGranter) RepairPartsWithoutSubStatuses(user *UserState, nowMillis int64) int {
	hasSubs := make(map[string]bool)
	for key := range user.PartsStatusSubs {
		hasSubs[key.UserPartsUuid] = true
	}
	repaired := 0
	for uuid, part := range user.Parts {
		ref, ok := g.PartsById[part.PartsId]
		if !ok || part.PartsStatusMainId <= 0 || hasSubs[uuid] {
			continue
		}
		pending := &UserState{PartsStatusSubs: make(map[PartsStatusSubKey]PartsStatusSubState)}
		for slot := int32(1); slot <= model.PartsMaxSubStatusCount; slot++ {
			if !g.grantInitialPartsSubStatus(pending, uuid, ref, slot, nowMillis) {
				break
			}
		}
		if len(pending.PartsStatusSubs) != int(model.PartsMaxSubStatusCount) {
			continue
		}
		for key, sub := range pending.PartsStatusSubs {
			user.PartsStatusSubs[key] = sub
		}
		repaired++
	}
	return repaired
}

func (g *PossessionGranter) grantInitialPartsSubStatus(user *UserState, uuid string, ref PartsRef, slot int32, nowMillis int64) bool {
	id, ok := PickUniquePartsSubStatus(g.PartsSubStatusPool[ref.PartsStatusSubLotteryGroupId], user, uuid)
	if !ok {
		return false
	}
	def, ok := g.PartsSubStatusDefs[id]
	if !ok {
		return false
	}
	user.PartsStatusSubs[PartsStatusSubKey{UserPartsUuid: uuid, StatusIndex: slot}] = PartsStatusSubState{
		UserPartsUuid: uuid, StatusIndex: slot, PartsStatusSubLotteryId: id, Level: 1,
		StatusKindType: def.StatusKindType, StatusCalculationType: def.StatusCalculationType,
		StatusChangeValue: def.Initial.Roll(), LatestVersion: nowMillis,
	}
	return true
}

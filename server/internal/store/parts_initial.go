package store

// RepairPartsWithoutSubStatuses is the offline compatibility repair used by
// cmd/repair-parts. Add one sub-status so the unmodified client's thumbnails
// show details; keep the item and main status and never reroll existing stats.
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
		if g.grantInitialPartsSubStatus(user, uuid, ref, 1, nowMillis) {
			repaired++
		}
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

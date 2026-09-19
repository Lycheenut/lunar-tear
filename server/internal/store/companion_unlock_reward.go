package store

// GrantCompanionUnlockReward fills the non-main-quest companion collection.
// The audited roster is CompanionId 31-53; 49-51 cannot be enhanced normally.
// Existing companions are preserved rather than converted into duplicate rewards.
func GrantCompanionUnlockReward(user *UserState, granter *PossessionGranter, nowMillis int64) {
	user.EnsureMaps()
	owned := make(map[int32]string, len(user.Companions))
	for key, companion := range user.Companions {
		owned[companion.CompanionId] = key
	}
	for id := int32(31); id <= 53; id++ {
		level := int32(1)
		if id >= 49 && id <= 51 {
			level = 50
		}
		if key, exists := owned[id]; exists {
			companion := user.Companions[key]
			if companion.Level < level {
				companion.Level = level
				companion.LatestVersion = max(companion.LatestVersion+1, nowMillis)
				user.Companions[key] = companion
			}
			continue
		}
		granter.grantCompanion(user, id, level, nowMillis)
	}
}

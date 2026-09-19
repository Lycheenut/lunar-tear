package store

import "lunar-tear/server/internal/model"

// The client enables login bonuses when CalculatorOutgame.IsEndMomMenuTutorial
// sees MenuSecond at or beyond MomMenuEditDeck. See docs/LOGIN_BONUS_UNLOCK_REWARD.md.
func IsLoginBonusUnlocked(user *UserState) bool {
	return user.Tutorials[int32(model.TutorialTypeMenuSecond)].ProgressPhase >= int32(model.TutorialPhaseMomMenuEditDeck)
}

// GrantLoginBonusUnlockReward grants the bundle unconditionally. Tutorial callers
// must check the locked-to-unlocked transition; the repair command runs only once.
func GrantLoginBonusUnlockReward(user *UserState, granter *PossessionGranter, nowMillis int64) {
	user.EnsureMaps()
	// The costume and its weapon are separate rewards; GrantFull does not pair them.
	granter.GrantFull(user, model.PossessionTypeCostume, 24008, 1, nowMillis)
	granter.GrantFull(user, model.PossessionTypeMaterial, 311211, 40, nowMillis)
	granter.GrantFull(user, model.PossessionTypeMaterial, 313197, 5, nowMillis)
	granter.GrantFull(user, model.PossessionTypeWeapon, 240271, 1, nowMillis)
	granter.GrantFull(user, model.PossessionTypeMaterial, 312011, 4, nowMillis)
	// Owning companions does not unlock their deck slots; the client checks
	// Companion tutorial progress independently. See docs/COMPANION_UNLOCK_REWARD.md.
	GrantCompanionUnlockReward(user, granter, nowMillis)
}

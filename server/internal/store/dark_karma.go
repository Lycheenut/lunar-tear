package store

import "lunar-tear/server/internal/model"

const darkCostumeKarmaGrant int32 = 30

// ReconcileDarkKarma rebuilds the balance from distinct owned costumes. Older
// saves may record an unlocked slot in the costume count or in the effect rows.
func (g *PossessionGranter) ReconcileDarkKarma(user *UserState) {
	remaining := make(map[int32]int32)
	for uuid, costume := range user.Costumes {
		costs := g.DarkKarmaSlotCosts[costume.CostumeId]
		if len(costs) == 0 {
			continue
		}
		balance := darkCostumeKarmaGrant
		for slot, cost := range costs {
			_, hasEffect := user.CostumeLotteryEffects[CostumeLotteryEffectKey{UserCostumeUuid: uuid, SlotNumber: slot}]
			if hasEffect || slot <= costume.CostumeLotteryEffectUnlockedSlotCount {
				balance -= cost
			}
		}
		if previous, exists := remaining[costume.CostumeId]; !exists || balance < previous {
			remaining[costume.CostumeId] = max(0, balance)
		}
	}
	var total int32
	for _, balance := range remaining {
		total += balance
	}
	if total == 0 {
		delete(user.Materials, model.MaterialIdDarkKarmaStone)
	} else {
		user.Materials[model.MaterialIdDarkKarmaStone] = total
	}
}

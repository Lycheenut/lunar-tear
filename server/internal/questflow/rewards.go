package questflow

import (
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/gameutil"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questdrop"
	"lunar-tear/server/internal/store"
)

func (h *QuestHandler) isQuestCleared(user *store.UserState, questId int32) bool {
	quest, ok := user.Quests[questId]
	if !ok {
		return false
	}
	return quest.QuestStateType == model.UserQuestStateTypeCleared
}

func appendMissionRewards(dst []RewardGrant, src []masterdata.EntityMQuestMissionReward) []RewardGrant {
	for _, r := range src {
		dst = append(dst, RewardGrant{
			PossessionType: model.PossessionType(r.PossessionType),
			PossessionId:   r.PossessionId,
			Count:          r.Count,
		})
	}
	return dst
}

func (h *QuestHandler) firstClearRewardGroupId(user *store.UserState, questDef masterdata.EntityMQuest) int32 {
	rewardGroupId := questDef.QuestFirstClearRewardGroupId
	for _, switchRow := range h.FirstClearRewardSwitchesByQuestId[questDef.QuestId] {
		if h.isQuestCleared(user, switchRow.SwitchConditionClearQuestId) {
			rewardGroupId = switchRow.QuestFirstClearRewardGroupId
			break
		}
	}
	return rewardGroupId
}

func (h *QuestHandler) questMissionPowerBonusApplies(user *store.UserState, questId int32, questDef masterdata.EntityMQuest) bool {
	if !questDef.IsBigWinTarget || h.Config == nil || h.Config.QuestMissionBigWinBonusPower <= 0 {
		return false
	}

	deckType := model.DeckTypeQuest
	if questDef.QuestDeckRestrictionGroupId != 0 {
		deckType = model.DeckTypeRestrictedQuest
	}
	deckNumber := user.Quests[questId].UserDeckNumber
	deck, ok := user.Decks[store.DeckKey{DeckType: deckType, UserDeckNumber: deckNumber}]
	if !ok {
		return false
	}

	requiredPower := int64(questDef.RecommendedDeckPower) + int64(h.Config.QuestMissionBigWinBonusPower)
	return int64(deck.Power) >= requiredPower
}

func (h *QuestHandler) evaluateFinishOutcome(user *store.UserState, questId int32, target campaign.QuestTarget, nowMillis int64) FinishOutcome {
	outcome := FinishOutcome{}
	questState, ok := user.Quests[questId]
	if !ok {
		log.Printf("[evaluateFinishOutcome] quest %d has no user state", questId)
		return outcome
	}
	questDef, ok := h.QuestById[questId]
	if !ok {
		log.Printf("[evaluateFinishOutcome] unknown quest %d", questId)
		return outcome
	}

	isReplay := model.IsReplayQuestFlowType(user.MainQuest.CurrentQuestFlowType)

	if !questState.IsRewardGranted && !isReplay {
		rewardGroupId := h.firstClearRewardGroupId(user, questDef)
		for _, reward := range h.FirstClearRewardsByGroupId[rewardGroupId] {
			outcome.FirstClearRewards = append(outcome.FirstClearRewards, RewardGrant{
				PossessionType: model.PossessionType(reward.PossessionType),
				PossessionId:   reward.PossessionId,
				Count:          reward.Count,
			})
		}
	}

	if isReplay && questDef.QuestReplayFlowRewardGroupId > 0 {
		_, alreadyReceived := user.QuestReplayFlowRewards[questDef.QuestReplayFlowRewardGroupId]
		if !alreadyReceived {
			outcome.ReplayRewardGroupId = questDef.QuestReplayFlowRewardGroupId
			for _, reward := range h.ReplayFlowRewardsByGroupId[questDef.QuestReplayFlowRewardGroupId] {
				outcome.ReplayFlowFirstClearRewards = append(outcome.ReplayFlowFirstClearRewards, RewardGrant{
					PossessionType: model.PossessionType(reward.PossessionType),
					PossessionId:   reward.PossessionId,
					Count:          reward.Count,
				})
			}
		}
	}

	// Mission rewards / BigWin are first-clear concepts. Reference
	// IUserQuestMissionTable has no rows for replay-variant ids (30000+):
	// the popup is empty on replay in the original game.
	if !isReplay {
		powerBonusApplies := h.questMissionPowerBonusApplies(user, questId, questDef)
		regularMissionCount := 0
		clearedOrSatisfied := 0
		for _, questMissionId := range h.MissionIdsByQuestId[questId] {
			missionDef, ok := h.MissionById[questMissionId]
			if !ok || model.QuestMissionConditionType(missionDef.QuestMissionConditionType) == model.QuestMissionConditionTypeComplete {
				continue
			}
			regularMissionCount++

			key := store.QuestMissionKey{QuestId: questId, QuestMissionId: questMissionId}
			mission := user.QuestMissions[key]
			if mission.IsClear {
				clearedOrSatisfied++
			} else if powerBonusApplies || h.questMissionSatisfied(user, questId, missionDef) {
				clearedOrSatisfied++
				outcome.ClearedQuestMissionIds = append(outcome.ClearedQuestMissionIds, questMissionId)
				outcome.MissionClearRewards = appendMissionRewards(
					outcome.MissionClearRewards,
					h.MissionRewardsByMissionId[missionDef.QuestMissionRewardId],
				)
			}
		}

		allRegularWillClear := regularMissionCount > 0 && clearedOrSatisfied == regularMissionCount
		if allRegularWillClear {
			for _, questMissionId := range h.MissionIdsByQuestId[questId] {
				missionDef, ok := h.MissionById[questMissionId]
				if !ok || model.QuestMissionConditionType(missionDef.QuestMissionConditionType) != model.QuestMissionConditionTypeComplete {
					continue
				}
				key := store.QuestMissionKey{QuestId: questId, QuestMissionId: questMissionId}
				if !user.QuestMissions[key].IsClear {
					outcome.ClearedQuestMissionIds = append(outcome.ClearedQuestMissionIds, questMissionId)
					outcome.MissionClearCompleteRewards = appendMissionRewards(
						outcome.MissionClearCompleteRewards,
						h.MissionRewardsByMissionId[missionDef.QuestMissionRewardId],
					)
					outcome.BigWinClearedQuestMissionIds = append(outcome.BigWinClearedQuestMissionIds, questMissionId)
				}
			}
			outcome.IsBigWin = len(outcome.BigWinClearedQuestMissionIds) > 0
		}
	}

	outcome.DropRewards = h.computeDropRewards(user, questDef, target, nowMillis)
	return outcome
}

var autoSaleRarityTiers = map[int32]bool{10: true, 20: true, 30: true, 40: true, 50: true}

// Rarity tiers (10..50) and ranks (1..5) are disjoint, so the delimited values
// are classified by range — independent of the client's map key or delimiter.
func parseAutoSaleRules(settings map[int32]store.AutoSaleSettingState) (raritySet, rankSet map[int32]bool) {
	raritySet = map[int32]bool{}
	rankSet = map[int32]bool{}
	for _, s := range settings {
		for _, n := range extractInts(s.PossessionAutoSaleItemValue) {
			switch {
			case autoSaleRarityTiers[n]:
				raritySet[n] = true
			case n >= 1 && n <= 5:
				rankSet[n] = true
			}
		}
	}
	return raritySet, rankSet
}

func extractInts(s string) []int32 {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
	out := make([]int32, 0, len(fields))
	for _, f := range fields {
		if v, err := strconv.Atoi(f); err == nil {
			out = append(out, int32(v))
		}
	}
	return out
}

func (h *QuestHandler) grantDropRewards(user *store.UserState, drops []RewardGrant, raritySet, rankSet map[int32]bool, nowMillis int64) {
	for i := range drops {
		d := drops[i]
		if d.PossessionType == model.PossessionTypeParts || d.PossessionType == model.PossessionTypePartsEnhanced {
			chosenId, sold := h.Granter.GrantOrSellPartsDrop(user, d.PossessionId, raritySet, rankSet, nowMillis)
			if sold {
				// Sold parts have no inventory row, so the popup needs the rolled
				// variant id; kept parts read theirs from the parts table diff.
				drops[i].PossessionId = chosenId
				drops[i].IsAutoSale = true
			}
			continue
		}
		h.applyRewardPossession(user, d.PossessionType, d.PossessionId, d.Count, nowMillis)
	}
}

func battleDropSeed(userId int64, questId int32, runSeed int64) int64 {
	value := uint64(userId) ^ uint64(runSeed) ^ uint64(uint32(questId))*0x9e3779b97f4a7c15
	value ^= value >> 30
	value *= 0xbf58476d1ce4e5b9
	value ^= value >> 27
	value *= 0x94d049bb133111eb
	value ^= value >> 31
	return int64(value)
}

func (h *QuestHandler) battleDropPlan(user *store.UserState, questId int32, runSeed int64) []masterdata.BattleDropInfo {
	quest, ok := h.QuestById[questId]
	if !ok {
		return nil
	}
	candidates := h.BattleDropsByQuestId[questId]
	if len(candidates) == 0 {
		return nil
	}
	random := rand.New(rand.NewSource(battleDropSeed(user.UserId, questId, runSeed)))
	if pool, configured := h.DropRewardsByQuestID[questId]; configured {
		return h.weightedBattleDropPlan(candidates, pool, random)
	}

	// Unconfigured quests intentionally retain the original master-data row
	// lottery. Duplicate pickup rows remain duplicate tickets, including in the
	// second draw within the revealed rarity.
	pool := h.PickupRewardIdsByGroupId[quest.QuestPickupRewardGroupId]
	if len(pool) == 0 {
		return nil
	}
	plan := make([]masterdata.BattleDropInfo, 0, len(candidates))
	for _, candidate := range candidates {
		probeRewardID := pool[random.Intn(len(pool))]
		effectID := h.BattleDropEffectIdByRewardId[probeRewardID]
		subset := h.PickupRewardIdsByGroupAndEffectId[quest.QuestPickupRewardGroupId][effectID]
		if len(subset) == 0 {
			continue
		}
		candidate.BattleDropEffectId = effectID
		candidate.BattleDropRewardId = subset[random.Intn(len(subset))]
		plan = append(plan, candidate)
	}
	return plan
}

func (h *QuestHandler) weightedBattleDropPlan(candidates []masterdata.BattleDropInfo, pool []questdrop.Reward, random *rand.Rand) []masterdata.BattleDropInfo {
	if len(pool) == 0 {
		return nil
	}
	byEffectID := make(map[int32][]questdrop.Reward)
	effectWeights := make(map[int32]int64)
	var effectIDs []int32
	for _, reward := range pool {
		effectID := h.BattleDropEffectIdByRewardId[reward.BattleDropRewardID]
		if _, exists := byEffectID[effectID]; !exists {
			effectIDs = append(effectIDs, effectID)
		}
		byEffectID[effectID] = append(byEffectID[effectID], reward)
		effectWeights[effectID] += int64(reward.Weight)
	}
	sort.Slice(effectIDs, func(i, j int) bool { return effectIDs[i] < effectIDs[j] })

	plan := make([]masterdata.BattleDropInfo, 0, len(candidates))
	for _, candidate := range candidates {
		// The battle reveal is selected from each rarity's total configured
		// weight. The concrete reward then uses its own weight within that rarity.
		effectID := weightedEffectID(random, effectIDs, effectWeights)
		rewardID := weightedRewardID(random, byEffectID[effectID])
		if effectID == 0 || rewardID == 0 {
			continue
		}
		candidate.BattleDropEffectId = effectID
		candidate.BattleDropRewardId = rewardID
		plan = append(plan, candidate)
	}
	return plan
}

func weightedEffectID(random *rand.Rand, effectIDs []int32, weights map[int32]int64) int32 {
	var total int64
	for _, effectID := range effectIDs {
		total += weights[effectID]
	}
	if total <= 0 {
		return 0
	}
	draw := random.Int63n(total)
	for _, effectID := range effectIDs {
		draw -= weights[effectID]
		if draw < 0 {
			return effectID
		}
	}
	return 0
}

func weightedRewardID(random *rand.Rand, rewards []questdrop.Reward) int32 {
	var total int64
	for _, reward := range rewards {
		total += int64(reward.Weight)
	}
	if total <= 0 {
		return 0
	}
	draw := random.Int63n(total)
	for _, reward := range rewards {
		draw -= int64(reward.Weight)
		if draw < 0 {
			return reward.BattleDropRewardID
		}
	}
	return 0
}

func (h *QuestHandler) computeDropRewards(user *store.UserState, questDef masterdata.EntityMQuest, target campaign.QuestTarget, nowMillis int64) []RewardGrant {
	runSeed := user.Quests[questDef.QuestId].LatestStartDatetime
	return h.computeDropRewardsForRun(user, questDef, target, nowMillis, runSeed)
}

func (h *QuestHandler) computeDropRewardsForRun(
	user *store.UserState,
	questDef masterdata.EntityMQuest,
	target campaign.QuestTarget,
	nowMillis, runSeed int64,
) []RewardGrant {
	var drops []RewardGrant
	var dropRate campaign.DropRateMul
	var dropCount campaign.DropCountMul
	if h.Campaigns != nil {
		dropRate = h.Campaigns.QuestDropRate(target, h.campaignFilter(user, nowMillis))
		dropCount = h.Campaigns.QuestDropCount(target, h.campaignFilter(user, nowMillis))
	}
	for _, planned := range h.battleDropPlan(user, questDef.QuestId, runSeed) {
		if bdr, ok := h.BattleDropRewardById[planned.BattleDropRewardId]; ok {
			itemDropRate := dropRate
			itemDropCount := dropCount
			if h.ImportantItemEffects != nil {
				ratePermil, countPermil := h.ImportantItemEffects.QuestBonuses(
					user.ImportantItems, target, model.PossessionType(bdr.PossessionType), bdr.PossessionId, nowMillis)
				itemDropRate = itemDropRate.WithBonusPermil(ratePermil)
				itemDropCount = itemDropCount.WithBonusPermil(countPermil)
			}
			drops = append(drops, RewardGrant{
				PossessionType: model.PossessionType(bdr.PossessionType),
				PossessionId:   bdr.PossessionId,
				Count:          itemDropCount.Apply(itemDropRate.Apply(bdr.Count)),
				RewardEffectId: planned.BattleDropEffectId,
			})
		}
	}
	drops = append(drops, h.questBonusDropRewards(user, questDef, nowMillis)...)
	return h.appendBonusDrops(user, drops, target, nowMillis)
}

func (h *QuestHandler) applyExpRewards(user *store.UserState, questId int32, nowMillis int64) {
	questDef, ok := h.QuestById[questId]
	if !ok {
		return
	}

	oldLevel := user.Status.Level
	user.Status.Exp += questDef.UserExp
	user.Status.Level, user.Status.Exp = gameutil.LevelAndCap(user.Status.Exp, h.UserExpThresholds)
	log.Printf("[applyExpRewards] questId=%d user: +%d exp -> total=%d level=%d", questId, questDef.UserExp, user.Status.Exp, user.Status.Level)

	if user.Status.Level > oldLevel {
		if maxStamina, ok := h.MaxStaminaByLevel[user.Status.Level]; ok {
			maxStaminaMillis := maxStamina * 1000
			store.RecoverStamina(user, maxStaminaMillis, maxStaminaMillis, nowMillis)
		}
	}

	if h.RentalQuestIds[questId] {
		log.Printf("[applyExpRewards] questId=%d skipping character/costume exp (rental deck)", questId)
		return
	}

	if questDef.CharacterExp == 0 && questDef.CostumeExp == 0 {
		return
	}

	deckCostumeUuids, deckCharacterIds := h.resolveDeckUnits(user, questId)
	if deckCostumeUuids == nil {
		log.Printf("[applyExpRewards] questId=%d skipping character/costume exp (deck not resolved)", questId)
		return
	}
	characterBonusByCostume, costumeBonusByCostume := h.questBonusExpPermilByCostume(user, questDef, nowMillis)
	characterBonusByCharacter := make(map[int32]int32)
	for costumeUuid, bonusPermil := range characterBonusByCostume {
		costume, ok := user.Costumes[costumeUuid]
		if !ok {
			continue
		}
		characterId := h.CostumeById[costume.CostumeId].CharacterId
		characterBonusByCharacter[characterId] += bonusPermil
	}

	if questDef.CharacterExp != 0 {
		for id := range deckCharacterIds {
			row := user.Characters[id]
			gainedExp := int32(int64(questDef.CharacterExp) * int64(1000+characterBonusByCharacter[id]) / 1000)
			row.Exp += gainedExp
			row.Level, row.Exp = gameutil.LevelAndCap(row.Exp, h.CharacterExpThresholds)
			user.Characters[id] = row
			log.Printf("[applyExpRewards] questId=%d character=%d: +%d exp -> total=%d level=%d", questId, id, gainedExp, row.Exp, row.Level)
		}
	}

	if questDef.CostumeExp != 0 {
		for key := range deckCostumeUuids {
			row := user.Costumes[key]
			cm, ok := h.CostumeById[row.CostumeId]
			if !ok {
				continue
			}
			var maxLevel int32
			if maxLevelFunc, hasMax := h.CostumeMaxLevelByRarity[cm.RarityType]; hasMax {
				maxLevel = maxLevelFunc.Evaluate(row.LimitBreakCount) +
					h.CharacterRebirth.CostumeLevelLimitUp(cm.CharacterId, user.CharacterRebirths[cm.CharacterId].RebirthCount)
				if row.Level >= maxLevel {
					log.Printf("[applyExpRewards] questId=%d costume=%d (key=%s): at max level %d, skipping", questId, row.CostumeId, key, row.Level)
					continue
				}
			}
			gainedExp := int32(int64(questDef.CostumeExp) * int64(1000+costumeBonusByCostume[key]) / 1000)
			row.Exp += gainedExp
			if thresholds, ok := h.CostumeExpByRarity[cm.RarityType]; ok {
				row.Level, row.Exp = gameutil.ApplyExpWithMaxLevel(row.Exp, thresholds, maxLevel)
			}
			user.Costumes[key] = row
			log.Printf("[applyExpRewards] questId=%d costume=%d (key=%s): +%d exp -> total=%d level=%d", questId, row.CostumeId, key, gainedExp, row.Exp, row.Level)
		}
	}
}

func (h *QuestHandler) resolveDeckUnits(user *store.UserState, questId int32) (costumeUuids map[string]bool, characterIds map[int32]bool) {
	costumeUuids = make(map[string]bool)
	characterIds = make(map[int32]bool)
	for _, unit := range h.questBonusDeckUnits(user, questId) {
		if unit.costumeUuid == "" {
			continue
		}
		costumeUuids[unit.costumeUuid] = true
		if unit.characterId != 0 {
			characterIds[unit.characterId] = true
		}
	}

	if len(costumeUuids) == 0 {
		return nil, nil
	}
	return costumeUuids, characterIds
}

func (h *QuestHandler) applyExpAndGoldRewards(user *store.UserState, questId int32, target campaign.QuestTarget, nowMillis int64) {
	questDef, ok := h.QuestById[questId]
	if !ok {
		return
	}

	h.applyExpRewards(user, questId, nowMillis)

	if questDef.Gold != 0 {
		gold := h.goldWithCampaign(user, questDef.Gold, target, nowMillis)
		user.ConsumableItems[h.Config.ConsumableItemIdForGold] += gold
		log.Printf("[applyQuestRewards] questId=%d gold: +%d -> total=%d", questId, gold, user.ConsumableItems[h.Config.ConsumableItemIdForGold])
	}
}

func (h *QuestHandler) applyFirstClearItemRewards(user *store.UserState, questId int32, nowMillis int64) {
	questDef, ok := h.QuestById[questId]
	if !ok {
		return
	}
	rewardGroupId := h.firstClearRewardGroupId(user, questDef)
	for _, reward := range h.FirstClearRewardsByGroupId[rewardGroupId] {
		h.applyRewardPossession(user, model.PossessionType(reward.PossessionType), reward.PossessionId, reward.Count, nowMillis)
	}
}

func (h *QuestHandler) applyQuestRewards(user *store.UserState, questId int32, nowMillis int64) {
	h.applyExpAndGoldRewards(user, questId, h.targetForMain(questId), nowMillis)
	h.applyFirstClearItemRewards(user, questId, nowMillis)
}

func (h *QuestHandler) applyRewardPossession(user *store.UserState, possType model.PossessionType, possId, count int32, nowMillis int64) {
	h.Granter.GrantFull(user, possType, possId, count, nowMillis)
}

func (h *QuestHandler) grantWeaponStoryUnlock(user *store.UserState, weaponId, storyIndex int32, nowMillis int64) bool {
	return store.GrantWeaponStoryUnlock(user, weaponId, storyIndex, nowMillis)
}

var tutorialCompanionChoices = map[int32]int32{
	1: 2,  // bear + fire (Cat=1, Attr=2)
	2: 1,  // bear + wind (Cat=1, Attr=6)
	3: 7,  // doll + fire (Cat=3, Attr=2)
	4: 10, // doll + wind (Cat=3, Attr=6)
}

func (h *QuestHandler) ApplyTutorialReward(user *store.UserState, tutorialType model.TutorialType, choiceId int32, nowMillis int64) []RewardGrant {
	switch tutorialType {
	case model.TutorialTypeCompanion:
		return h.applyCompanionTutorialReward(user, choiceId, nowMillis)
	default:
		return nil
	}
}

func (h *QuestHandler) applyCompanionTutorialReward(user *store.UserState, choiceId int32, nowMillis int64) []RewardGrant {
	companionId, ok := tutorialCompanionChoices[choiceId]
	if !ok {
		log.Printf("[QuestHandler] unknown companion tutorial choiceId=%d", choiceId)
		return nil
	}
	h.Granter.GrantCompanion(user, companionId, nowMillis)
	return []RewardGrant{{
		PossessionType: model.PossessionTypeCompanion,
		PossessionId:   companionId,
		Count:          1,
	}}
}

func (h *QuestHandler) BattleDropRewards(user *store.UserState, questId int32) []masterdata.BattleDropInfo {
	return h.battleDropPlan(user, questId, user.Quests[questId].LatestStartDatetime)
}

func (h *QuestHandler) grantWeaponStoryUnlocksForQuestScene(user *store.UserState, questId int32, resultType model.QuestResultType, nowMillis int64) []int32 {
	var changedIds []int32
	if resultType == model.QuestResultTypeHalfResult {
		questDef, ok := h.QuestById[questId]
		if !ok {
			return nil
		}
		rewardGroupId := h.firstClearRewardGroupId(user, questDef)
		for _, reward := range h.FirstClearRewardsByGroupId[rewardGroupId] {
			if model.PossessionType(reward.PossessionType) != model.PossessionTypeWeapon {
				continue
			}
			weaponId := reward.PossessionId
			weapon, ok := h.WeaponById[weaponId]
			if !ok || weapon.WeaponStoryReleaseConditionGroupId == 0 {
				continue
			}
			groupId := weapon.WeaponStoryReleaseConditionGroupId
			for _, cond := range h.ReleaseConditionsByGroupId[groupId] {
				if model.WeaponStoryReleaseConditionType(cond.WeaponStoryReleaseConditionType) == model.WeaponStoryReleaseConditionTypeAcquisition && cond.ConditionValue == 0 {
					if h.grantWeaponStoryUnlock(user, weaponId, cond.StoryIndex, nowMillis) {
						changedIds = append(changedIds, weaponId)
					}
				}
			}
		}
		return changedIds
	}
	if resultType == model.QuestResultTypeFullResult {
		for groupId, conditions := range h.ReleaseConditionsByGroupId {
			for _, cond := range conditions {
				if model.WeaponStoryReleaseConditionType(cond.WeaponStoryReleaseConditionType) == model.WeaponStoryReleaseConditionTypeQuestClear && cond.ConditionValue == questId {
					for _, weaponId := range h.WeaponIdsByReleaseConditionGroupId[groupId] {
						if h.grantWeaponStoryUnlock(user, weaponId, cond.StoryIndex, nowMillis) {
							changedIds = append(changedIds, weaponId)
						}
					}
					break
				}
			}
		}
	}
	return changedIds
}

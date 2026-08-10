package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"

	"github.com/google/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/gameutil"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

type CostumeServiceServer struct {
	pb.UnimplementedCostumeServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewCostumeServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *CostumeServiceServer {
	return &CostumeServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *CostumeServiceServer) RegisterLevelBonusConfirmed(ctx context.Context, req *pb.RegisterLevelBonusConfirmedRequest) (*pb.RegisterLevelBonusConfirmedResponse, error) {
	if req.Level < 0 {
		return nil, status.Error(codes.InvalidArgument, "level must not be negative")
	}
	catalog := s.holder.Get().Costume
	if _, ok := catalog.Costumes[req.CostumeId]; !ok {
		return nil, status.Error(codes.NotFound, "costume not found")
	}
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		var ownedLevel int32
		for _, costume := range user.Costumes {
			if costume.CostumeId == req.CostumeId && costume.Level > ownedLevel {
				ownedLevel = costume.Level
			}
		}
		if ownedLevel == 0 || req.Level > ownedLevel {
			validationErr = status.Error(codes.FailedPrecondition, "costume level has not been reached")
			return
		}
		lastReleased := int32(0)
		for _, level := range catalog.LevelBonusLevelsByCostume[req.CostumeId] {
			if level <= ownedLevel {
				lastReleased = level
			}
		}
		registerCostumeLevelBonusStatus(user, req.CostumeId, req.Level, lastReleased, nowMillis)
	})
	if err != nil {
		return nil, fmt.Errorf("register costume level bonus: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}
	return &pb.RegisterLevelBonusConfirmedResponse{}, nil
}

func registerCostumeLevelBonusStatus(user *store.UserState, costumeId, confirmedLevel, lastReleasedLevel int32, nowMillis int64) {
	rec := user.CostumeLevelBonusReleaseStatuses[costumeId]
	rec.CostumeId = costumeId
	if lastReleasedLevel > rec.LastReleasedBonusLevel {
		rec.LastReleasedBonusLevel = lastReleasedLevel
	}
	if confirmedLevel > rec.ConfirmedBonusLevel {
		rec.ConfirmedBonusLevel = confirmedLevel
	}
	rec.LatestVersion = nowMillis
	user.CostumeLevelBonusReleaseStatuses[costumeId] = rec
}

func recomputeCostumeLotteryEffectResults(user *store.UserState, catalog *masterdata.CostumeCatalog, userCostumeUuid string, nowMillis int64) {
	for key := range user.CostumeLotteryEffectAbilities {
		if key.UserCostumeUuid == userCostumeUuid {
			delete(user.CostumeLotteryEffectAbilities, key)
		}
	}
	for key := range user.CostumeLotteryEffectStatusUps {
		if key.UserCostumeUuid == userCostumeUuid {
			delete(user.CostumeLotteryEffectStatusUps, key)
		}
	}

	costume, ok := user.Costumes[userCostumeUuid]
	if !ok {
		return
	}
	for key, effectState := range user.CostumeLotteryEffects {
		if key.UserCostumeUuid != userCostumeUuid || effectState.OddsNumber == 0 {
			continue
		}
		effect, ok := catalog.LotteryEffects[[2]int32{costume.CostumeId, key.SlotNumber}]
		if !ok {
			continue
		}
		odds, ok := catalog.LotteryEffectOddsByNumber[[2]int32{effect.CostumeLotteryEffectOddsGroupId, effectState.OddsNumber}]
		if !ok {
			continue
		}
		switch model.CostumeLotteryEffectType(odds.CostumeLotteryEffectType) {
		case model.CostumeLotteryEffectTypeAbility:
			ability, ok := catalog.LotteryEffectTargetAbilities[odds.CostumeLotteryEffectTargetId]
			if !ok {
				continue
			}
			user.CostumeLotteryEffectAbilities[key] = store.CostumeLotteryEffectAbilityState{
				UserCostumeUuid: userCostumeUuid,
				SlotNumber:      key.SlotNumber,
				AbilityId:       ability.AbilityId,
				AbilityLevel:    ability.AbilityLevel,
				LatestVersion:   nowMillis,
			}
		case model.CostumeLotteryEffectTypeStatusUp:
			for _, statusRow := range catalog.LotteryEffectTargetStatusUps[odds.CostumeLotteryEffectTargetId] {
				statusKey := store.CostumeLotteryEffectStatusKey{
					UserCostumeUuid:       userCostumeUuid,
					StatusCalculationType: model.StatusCalculationType(statusRow.StatusCalculationType),
				}
				statusState := user.CostumeLotteryEffectStatusUps[statusKey]
				statusState.UserCostumeUuid = userCostumeUuid
				statusState.StatusCalculationType = statusKey.StatusCalculationType
				switch model.StatusKindType(statusRow.StatusKindType) {
				case model.StatusKindTypeHp:
					statusState.Hp += statusRow.EffectValue
				case model.StatusKindTypeAttack:
					statusState.Attack += statusRow.EffectValue
				case model.StatusKindTypeVitality:
					statusState.Vitality += statusRow.EffectValue
				case model.StatusKindTypeAgility:
					statusState.Agility += statusRow.EffectValue
				case model.StatusKindTypeCriticalRatio:
					statusState.CriticalRatio += statusRow.EffectValue
				case model.StatusKindTypeCriticalAttack:
					statusState.CriticalAttack += statusRow.EffectValue
				}
				statusState.LatestVersion = nowMillis
				user.CostumeLotteryEffectStatusUps[statusKey] = statusState
			}
		}
	}
}

func (s *CostumeServiceServer) Enhance(ctx context.Context, req *pb.EnhanceRequest) (*pb.EnhanceResponse, error) {
	log.Printf("[CostumeService] Enhance: uuid=%s materials=%v", req.UserCostumeUuid, req.Materials)

	cat := s.holder.Get()
	catalog := cat.Costume
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	var validationErr error
	var isGreatSuccess bool
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		costume, ok := user.Costumes[req.UserCostumeUuid]
		if !ok {
			log.Printf("[CostumeService] Enhance: costume uuid=%s not found", req.UserCostumeUuid)
			return
		}

		cm, ok := catalog.Costumes[costume.CostumeId]
		if !ok {
			log.Printf("[CostumeService] Enhance: costume master id=%d not found", costume.CostumeId)
			return
		}

		rateBonus := cat.Campaign.CostumeRateBonus(campaign.CostumeTarget{
			CostumeId:          costume.CostumeId,
			CharacterId:        cm.CharacterId,
			SkillfulWeaponType: cm.SkillfulWeaponType,
		}, enhancementCampaignFilter(cat.Campaign, user, nowMillis))

		totalExp := int64(0)
		totalMaterialCount := int32(0)
		costs := make([]store.PossessionCost, 0, len(req.Materials)+1)
		for materialId, count := range req.Materials {
			if count <= 0 {
				validationErr = status.Errorf(codes.InvalidArgument, "invalid material count for %d", materialId)
				return
			}
			mat, ok := catalog.Materials[materialId]
			if !ok {
				validationErr = status.Errorf(codes.InvalidArgument, "invalid costume enhancement material %d", materialId)
				return
			}
			if count > math.MaxInt32-totalMaterialCount {
				validationErr = status.Error(codes.InvalidArgument, "material count is too large")
				return
			}
			costs = append(costs, materialCost(materialId, count))
			totalMaterialCount += count

			expPerUnit := int64(mat.EffectValue)
			if mat.WeaponType != 0 && mat.WeaponType == cm.SkillfulWeaponType {
				expPerUnit = expPerUnit * int64(config.MaterialSameWeaponExpCoefficientPermil) / 1000
			}
			totalExp += expPerUnit * int64(count)
		}

		greatSuccessRate := rateBonus.Apply(standardGreatSuccessRatePermil)
		finalExp, greatSuccess, outcomeErr := finalizeEnhancementExp(totalExp, greatSuccessRate, rand.Intn(1000))
		if outcomeErr != nil {
			validationErr = outcomeErr
			return
		}
		isGreatSuccess = greatSuccess

		if costFunc, ok := catalog.EnhanceCostByRarity[cm.RarityType]; ok && totalMaterialCount > 0 {
			goldCost := costFunc.Evaluate(totalMaterialCount)
			costs = append(costs, consumableCost(config.ConsumableItemIdForGold, goldCost))
			log.Printf("[CostumeService] Enhance: gold cost=%d (materials=%d)", goldCost, totalMaterialCount)
		}
		if err := deductUpgradeCosts(user, "costume enhancement cost", costs); err != nil {
			validationErr = err
			return
		}

		costume.Exp += finalExp

		if thresholds, ok := catalog.ExpByRarity[cm.RarityType]; ok {
			var maxLevel int32
			if maxLevelFunc, hasMax := catalog.MaxLevelByRarity[cm.RarityType]; hasMax {
				maxLevel = maxLevelFunc.Evaluate(costume.LimitBreakCount) +
					cat.CharacterRebirth.CostumeLevelLimitUp(cm.CharacterId, user.CharacterRebirths[cm.CharacterId].RebirthCount)
			}
			costume.Level, costume.Exp = gameutil.ApplyExpWithMaxLevel(costume.Exp, thresholds, maxLevel)
		}

		costume.LatestVersion = nowMillis
		user.Costumes[req.UserCostumeUuid] = costume
		log.Printf("[CostumeService] Enhance: costumeId=%d +%d exp greatSuccess=%v -> total=%d level=%d", costume.CostumeId, finalExp, isGreatSuccess, costume.Exp, costume.Level)
	})
	if err != nil {
		return nil, fmt.Errorf("costume enhance: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.EnhanceResponse{
		IsGreatSuccess:         isGreatSuccess,
		SurplusEnhanceMaterial: map[int32]int32{},
	}, nil
}

func (s *CostumeServiceServer) Awaken(ctx context.Context, req *pb.AwakenRequest) (*pb.AwakenResponse, error) {
	log.Printf("[CostumeService] Awaken: uuid=%s materials=%v", req.UserCostumeUuid, req.Materials)

	cat := s.holder.Get()
	catalog := cat.Costume
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		costume, ok := user.Costumes[req.UserCostumeUuid]
		if !ok {
			log.Printf("[CostumeService] Awaken: costume uuid=%s not found", req.UserCostumeUuid)
			return
		}

		awakenRow, ok := catalog.AwakenByCostumeId[costume.CostumeId]
		if !ok {
			log.Printf("[CostumeService] Awaken: no awaken data for costumeId=%d", costume.CostumeId)
			return
		}

		nextStep := costume.AwakenCount + 1
		if nextStep > config.CostumeAwakenAvailableCount {
			validationErr = status.Error(codes.FailedPrecondition, "costume is already fully awakened")
			return
		}

		costs, awakenSteps, costErr := selectedMaterialCosts(req.Materials, catalog.AwakenMaterialOptions(awakenRow.CostumeAwakenStepMaterialGroupId, nextStep))
		if costErr != nil {
			validationErr = costErr
			return
		}
		if awakenSteps != 1 {
			validationErr = status.Error(codes.InvalidArgument, "materials must cover exactly one awakening step")
			return
		}
		gold, ok := catalog.AwakenGold(awakenRow.CostumeAwakenPriceGroupId, nextStep)
		if !ok {
			validationErr = status.Error(codes.FailedPrecondition, "costume awakening price is unavailable")
			return
		}
		costs = append(costs, consumableCost(config.ConsumableItemIdForGold, gold))
		log.Printf("[CostumeService] Awaken: gold cost=%d", gold)
		if err := deductUpgradeCosts(user, "costume awakening cost", costs); err != nil {
			validationErr = err
			return
		}

		costume.AwakenCount = nextStep
		costume.LatestVersion = nowMillis
		user.Costumes[req.UserCostumeUuid] = costume
		log.Printf("[CostumeService] Awaken: costumeId=%d awakenCount=%d", costume.CostumeId, nextStep)

		effectSteps, ok := catalog.AwakenEffectsByGroupAndStep[awakenRow.CostumeAwakenEffectGroupId]
		if !ok {
			return
		}
		effect, ok := effectSteps[nextStep]
		if !ok {
			return
		}

		switch model.CostumeAwakenEffectType(effect.CostumeAwakenEffectType) {
		case model.CostumeAwakenEffectTypeStatusUp:
			applyCostumeAwakenStatusUp(catalog, user, req.UserCostumeUuid, effect.CostumeAwakenEffectId, nowMillis)
		case model.CostumeAwakenEffectTypeAbility:
			log.Printf("[CostumeService] Awaken: ability effect id=%d (client-resolved)", effect.CostumeAwakenEffectId)
		case model.CostumeAwakenEffectTypeItemAcquire:
			applyCostumeAwakenItemAcquire(catalog, user, effect.CostumeAwakenEffectId, nowMillis)
		default:
			log.Printf("[CostumeService] Awaken: unknown effect type=%d", effect.CostumeAwakenEffectType)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("costume awaken: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.AwakenResponse{}, nil
}

func applyCostumeAwakenStatusUp(catalog *masterdata.CostumeCatalog, user *store.UserState, costumeUuid string, statusUpGroupId int32, nowMillis int64) {
	rows, ok := catalog.AwakenStatusUpByGroup[statusUpGroupId]
	if !ok {
		log.Printf("[CostumeService] Awaken: status up group %d not found", statusUpGroupId)
		return
	}

	for _, row := range rows {
		calcType := model.StatusCalculationType(row.StatusCalculationType)
		key := store.CostumeAwakenStatusKey{
			UserCostumeUuid:       costumeUuid,
			StatusCalculationType: calcType,
		}
		state := user.CostumeAwakenStatusUps[key]
		state.UserCostumeUuid = costumeUuid
		state.StatusCalculationType = calcType

		switch model.StatusKindType(row.StatusKindType) {
		case model.StatusKindTypeHp:
			state.Hp += row.EffectValue
		case model.StatusKindTypeAttack:
			state.Attack += row.EffectValue
		case model.StatusKindTypeVitality:
			state.Vitality += row.EffectValue
		case model.StatusKindTypeAgility:
			state.Agility += row.EffectValue
		case model.StatusKindTypeCriticalRatio:
			state.CriticalRatio += row.EffectValue
		case model.StatusKindTypeCriticalAttack:
			state.CriticalAttack += row.EffectValue
		}

		state.LatestVersion = nowMillis
		user.CostumeAwakenStatusUps[key] = state
	}
}

func applyCostumeAwakenItemAcquire(catalog *masterdata.CostumeCatalog, user *store.UserState, itemAcquireId int32, nowMillis int64) {
	acq, ok := catalog.AwakenItemAcquireById[itemAcquireId]
	if !ok {
		log.Printf("[CostumeService] Awaken: item acquire id=%d not found", itemAcquireId)
		return
	}

	for _, t := range user.Thoughts {
		if t.ThoughtId == acq.PossessionId {
			return
		}
	}
	key := uuid.New().String()
	user.Thoughts[key] = store.ThoughtState{
		UserThoughtUuid:     key,
		ThoughtId:           acq.PossessionId,
		AcquisitionDatetime: nowMillis,
		LatestVersion:       nowMillis,
	}
	log.Printf("[CostumeService] Awaken: granted thought id=%d", acq.PossessionId)
}

func (s *CostumeServiceServer) EnhanceActiveSkill(ctx context.Context, req *pb.EnhanceActiveSkillRequest) (*pb.EnhanceActiveSkillResponse, error) {
	log.Printf("[CostumeService] EnhanceActiveSkill: uuid=%s addLevel=%d", req.UserCostumeUuid, req.AddLevelCount)

	cat := s.holder.Get()
	catalog := cat.Costume
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	if req.AddLevelCount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "add level count must be positive")
	}
	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		costume, ok := user.Costumes[req.UserCostumeUuid]
		if !ok {
			log.Printf("[CostumeService] EnhanceActiveSkill: costume uuid=%s not found", req.UserCostumeUuid)
			return
		}

		cm, ok := catalog.Costumes[costume.CostumeId]
		if !ok {
			log.Printf("[CostumeService] EnhanceActiveSkill: costume master id=%d not found", costume.CostumeId)
			return
		}

		groupRows := catalog.ActiveSkillGroupsByGroupId[cm.CostumeActiveSkillGroupId]
		enhanceMatId := int32(-1)
		for _, g := range groupRows {
			if g.CostumeLimitBreakCountLowerLimit <= costume.LimitBreakCount {
				enhanceMatId = g.CostumeActiveSkillEnhancementMaterialId
				break
			}
		}
		if enhanceMatId < 0 {
			log.Printf("[CostumeService] EnhanceActiveSkill: no skill group for costumeId=%d groupId=%d lb=%d",
				costume.CostumeId, cm.CostumeActiveSkillGroupId, costume.LimitBreakCount)
			return
		}

		skill := user.CostumeActiveSkills[req.UserCostumeUuid]
		currentLevel := skill.Level

		maxLevelFunc, ok := catalog.ActiveSkillMaxLevelByRarity[cm.RarityType]
		if !ok {
			log.Printf("[CostumeService] EnhanceActiveSkill: no max level func for rarity=%d", cm.RarityType)
			return
		}
		maxLevel := maxLevelFunc.Evaluate(1)

		addCount := req.AddLevelCount
		if addCount > maxLevel-currentLevel {
			addCount = maxLevel - currentLevel
		}
		if addCount <= 0 {
			log.Printf("[CostumeService] EnhanceActiveSkill: already at max level %d", currentLevel)
			return
		}

		costs := make([]store.PossessionCost, 0)
		for lvl := currentLevel; lvl < currentLevel+addCount; lvl++ {
			key := [2]int32{enhanceMatId, lvl}
			mats := catalog.ActiveSkillEnhanceMats[key]
			for _, mat := range mats {
				costs = append(costs, materialCost(mat.MaterialId, mat.Count))
			}

			if costFunc, ok := catalog.ActiveSkillCostByRarity[cm.RarityType]; ok {
				goldCost := costFunc.Evaluate(lvl + 1)
				costs = append(costs, consumableCost(config.ConsumableItemIdForGold, goldCost))
			}
		}
		if err := deductUpgradeCosts(user, "costume active skill enhancement cost", costs); err != nil {
			validationErr = err
			return
		}

		skill.UserCostumeUuid = req.UserCostumeUuid
		skill.Level = currentLevel + addCount
		skill.LatestVersion = nowMillis
		user.CostumeActiveSkills[req.UserCostumeUuid] = skill
		log.Printf("[CostumeService] EnhanceActiveSkill: costumeId=%d level %d -> %d", costume.CostumeId, currentLevel, skill.Level)
	})
	if err != nil {
		return nil, fmt.Errorf("costume enhance active skill: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.EnhanceActiveSkillResponse{}, nil
}

func (s *CostumeServiceServer) LimitBreak(ctx context.Context, req *pb.LimitBreakRequest) (*pb.LimitBreakResponse, error) {
	log.Printf("[CostumeService] LimitBreak: uuid=%s materials=%v", req.UserCostumeUuid, req.Materials)

	cat := s.holder.Get()
	catalog := cat.Costume
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		costume, ok := user.Costumes[req.UserCostumeUuid]
		if !ok {
			log.Printf("[CostumeService] LimitBreak: costume uuid=%s not found", req.UserCostumeUuid)
			return
		}

		if costume.LimitBreakCount >= config.CostumeLimitBreakAvailableCount {
			log.Printf("[CostumeService] LimitBreak: already at max limit break %d", costume.LimitBreakCount)
			return
		}

		cm, ok := catalog.Costumes[costume.CostumeId]
		if !ok {
			log.Printf("[CostumeService] LimitBreak: costume master id=%d not found", costume.CostumeId)
			return
		}

		costs, limitBreakSteps, costErr := selectedMaterialCosts(req.Materials, catalog.LimitBreakMaterialsByCostume[costume.CostumeId])
		if costErr != nil {
			validationErr = costErr
			return
		}
		if limitBreakSteps != 1 {
			validationErr = status.Error(codes.InvalidArgument, "materials must cover exactly one limit break")
			return
		}

		if costFunc, ok := catalog.LimitBreakCostByRarity[cm.RarityType]; ok {
			goldCost := costFunc.Evaluate(limitBreakSteps)
			costs = append(costs, consumableCost(config.ConsumableItemIdForGold, goldCost))
			log.Printf("[CostumeService] LimitBreak: gold cost=%d", goldCost)
		}
		if err := deductUpgradeCosts(user, "costume limit break cost", costs); err != nil {
			validationErr = err
			return
		}

		costume.LimitBreakCount++
		costume.LatestVersion = nowMillis
		user.Costumes[req.UserCostumeUuid] = costume
		log.Printf("[CostumeService] LimitBreak: costumeId=%d limitBreak -> %d", costume.CostumeId, costume.LimitBreakCount)
	})
	if err != nil {
		return nil, fmt.Errorf("costume limit break: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.LimitBreakResponse{}, nil
}

func (s *CostumeServiceServer) UnlockLotteryEffectSlot(ctx context.Context, req *pb.UnlockLotteryEffectSlotRequest) (*pb.UnlockLotteryEffectSlotResponse, error) {
	log.Printf("[CostumeService] UnlockLotteryEffectSlot: uuid=%s slot=%d", req.UserCostumeUuid, req.SlotNumber)

	cat := s.holder.Get()
	catalog := cat.Costume
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		costume, ok := user.Costumes[req.UserCostumeUuid]
		if !ok {
			log.Printf("[CostumeService] UnlockLotteryEffectSlot: costume uuid=%s not found", req.UserCostumeUuid)
			return
		}

		effectRow, ok := catalog.LotteryEffects[[2]int32{costume.CostumeId, req.SlotNumber}]
		if !ok {
			log.Printf("[CostumeService] UnlockLotteryEffectSlot: no lottery effect for costumeId=%d slot=%d", costume.CostumeId, req.SlotNumber)
			return
		}
		key := store.CostumeLotteryEffectKey{
			UserCostumeUuid: req.UserCostumeUuid,
			SlotNumber:      req.SlotNumber,
		}
		if _, unlocked := user.CostumeLotteryEffects[key]; unlocked {
			validationErr = status.Error(codes.FailedPrecondition, "costume lottery effect slot is already unlocked")
			return
		}
		if req.SlotNumber != costume.CostumeLotteryEffectUnlockedSlotCount+1 {
			validationErr = status.Error(codes.FailedPrecondition, "costume lottery effect slots must be unlocked in order")
			return
		}

		costs := []store.PossessionCost{consumableCost(config.ConsumableItemIdForGold, config.CostumeLotteryEffectUnlockSlotConsumeGold)}
		mats := catalog.LotteryEffectMats[effectRow.CostumeLotteryEffectUnlockMaterialGroupId]
		for _, mat := range mats {
			costs = append(costs, materialCost(mat.MaterialId, mat.Count))
		}
		if err := deductUpgradeCosts(user, "costume lottery slot unlock cost", costs); err != nil {
			validationErr = err
			return
		}

		user.CostumeLotteryEffects[key] = store.CostumeLotteryEffectState{
			UserCostumeUuid: req.UserCostumeUuid,
			SlotNumber:      req.SlotNumber,
			OddsNumber:      0,
			LatestVersion:   nowMillis,
		}
		recomputeCostumeLotteryEffectResults(user, catalog, req.UserCostumeUuid, nowMillis)

		costume.CostumeLotteryEffectUnlockedSlotCount++
		costume.LatestVersion = nowMillis
		user.Costumes[req.UserCostumeUuid] = costume
		log.Printf("[CostumeService] UnlockLotteryEffectSlot: costumeId=%d slot=%d unlocked slotCount=%d", costume.CostumeId, req.SlotNumber, costume.CostumeLotteryEffectUnlockedSlotCount)
	})
	if err != nil {
		return nil, fmt.Errorf("costume unlock lottery effect slot: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.UnlockLotteryEffectSlotResponse{}, nil
}

func (s *CostumeServiceServer) DrawLotteryEffect(ctx context.Context, req *pb.DrawLotteryEffectRequest) (*pb.DrawLotteryEffectResponse, error) {
	log.Printf("[CostumeService] DrawLotteryEffect: uuid=%s slot=%d", req.UserCostumeUuid, req.SlotNumber)

	cat := s.holder.Get()
	catalog := cat.Costume
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		costume, ok := user.Costumes[req.UserCostumeUuid]
		if !ok {
			log.Printf("[CostumeService] DrawLotteryEffect: costume uuid=%s not found", req.UserCostumeUuid)
			return
		}

		effectRow, ok := catalog.LotteryEffects[[2]int32{costume.CostumeId, req.SlotNumber}]
		if !ok {
			log.Printf("[CostumeService] DrawLotteryEffect: no lottery effect for costumeId=%d slot=%d", costume.CostumeId, req.SlotNumber)
			return
		}
		key := store.CostumeLotteryEffectKey{
			UserCostumeUuid: req.UserCostumeUuid,
			SlotNumber:      req.SlotNumber,
		}
		if _, unlocked := user.CostumeLotteryEffects[key]; !unlocked {
			validationErr = status.Error(codes.FailedPrecondition, "costume lottery effect slot is not unlocked")
			return
		}

		oddsPool := catalog.LotteryEffectOdds[effectRow.CostumeLotteryEffectOddsGroupId]
		if len(oddsPool) == 0 {
			log.Printf("[CostumeService] DrawLotteryEffect: empty odds pool for groupId=%d", effectRow.CostumeLotteryEffectOddsGroupId)
			return
		}

		costs := []store.PossessionCost{consumableCost(config.ConsumableItemIdForGold, config.CostumeLotteryEffectDrawSlotConsumeGold)}
		mats := catalog.LotteryEffectMats[effectRow.CostumeLotteryEffectDrawMaterialGroupId]
		for _, mat := range mats {
			costs = append(costs, materialCost(mat.MaterialId, mat.Count))
		}
		if err := deductUpgradeCosts(user, "costume lottery draw cost", costs); err != nil {
			validationErr = err
			return
		}

		totalWeight := int32(0)
		for _, row := range oddsPool {
			totalWeight += row.Weight
		}
		roll := rand.Int31n(totalWeight)
		var picked masterdata.EntityMCostumeLotteryEffectOddsGroup
		for _, row := range oddsPool {
			roll -= row.Weight
			if roll < 0 {
				picked = row
				break
			}
		}

		existing := user.CostumeLotteryEffects[key]
		if existing.OddsNumber == 0 {
			existing.UserCostumeUuid = req.UserCostumeUuid
			existing.SlotNumber = req.SlotNumber
			existing.OddsNumber = picked.OddsNumber
			existing.LatestVersion = nowMillis
			user.CostumeLotteryEffects[key] = existing
			recomputeCostumeLotteryEffectResults(user, catalog, req.UserCostumeUuid, nowMillis)
		} else {
			user.CostumeLotteryEffectPending[req.UserCostumeUuid] = store.CostumeLotteryEffectPendingState{
				UserCostumeUuid: req.UserCostumeUuid,
				SlotNumber:      req.SlotNumber,
				OddsNumber:      picked.OddsNumber,
				LatestVersion:   nowMillis,
			}
		}

		log.Printf("[CostumeService] DrawLotteryEffect: costumeId=%d slot=%d drew oddsNumber=%d type=%d targetId=%d firstDraw=%v",
			costume.CostumeId, req.SlotNumber, picked.OddsNumber, picked.CostumeLotteryEffectType, picked.CostumeLotteryEffectTargetId, existing.OddsNumber == 0)
	})
	if err != nil {
		return nil, fmt.Errorf("costume draw lottery effect: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.DrawLotteryEffectResponse{}, nil
}

func (s *CostumeServiceServer) ConfirmLotteryEffect(ctx context.Context, req *pb.ConfirmLotteryEffectRequest) (*pb.ConfirmLotteryEffectResponse, error) {
	log.Printf("[CostumeService] ConfirmLotteryEffect: uuid=%s accept=%v", req.UserCostumeUuid, req.IsAccept)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	catalog := s.holder.Get().Costume

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		pending, ok := user.CostumeLotteryEffectPending[req.UserCostumeUuid]
		if !ok {
			log.Printf("[CostumeService] ConfirmLotteryEffect: no pending for uuid=%s", req.UserCostumeUuid)
			return
		}

		if req.IsAccept {
			key := store.CostumeLotteryEffectKey{
				UserCostumeUuid: pending.UserCostumeUuid,
				SlotNumber:      pending.SlotNumber,
			}
			effect := user.CostumeLotteryEffects[key]
			effect.UserCostumeUuid = pending.UserCostumeUuid
			effect.SlotNumber = pending.SlotNumber
			effect.OddsNumber = pending.OddsNumber
			effect.LatestVersion = nowMillis
			user.CostumeLotteryEffects[key] = effect
			recomputeCostumeLotteryEffectResults(user, catalog, req.UserCostumeUuid, nowMillis)
			log.Printf("[CostumeService] ConfirmLotteryEffect: accepted oddsNumber=%d for slot=%d", pending.OddsNumber, pending.SlotNumber)
		} else {
			log.Printf("[CostumeService] ConfirmLotteryEffect: rejected oddsNumber=%d for slot=%d", pending.OddsNumber, pending.SlotNumber)
		}

		delete(user.CostumeLotteryEffectPending, req.UserCostumeUuid)
	})
	if err != nil {
		return nil, fmt.Errorf("costume confirm lottery effect: %w", err)
	}

	return &pb.ConfirmLotteryEffectResponse{}, nil
}

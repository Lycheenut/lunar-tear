package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

type PartsServiceServer struct {
	pb.UnimplementedPartsServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewPartsServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *PartsServiceServer {
	return &PartsServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *PartsServiceServer) Protect(ctx context.Context, req *pb.PartsProtectRequest) (*pb.PartsProtectResponse, error) {
	log.Printf("[PartsService] Protect: uuids=%v", req.UserPartsUuid)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, uuid := range req.UserPartsUuid {
			part, ok := user.Parts[uuid]
			if !ok {
				log.Printf("[PartsService] Protect: part uuid=%s not found", uuid)
				continue
			}
			part.IsProtected = true
			part.LatestVersion = nowMillis
			user.Parts[uuid] = part
		}
	})

	return &pb.PartsProtectResponse{}, nil
}

func (s *PartsServiceServer) Unprotect(ctx context.Context, req *pb.PartsUnprotectRequest) (*pb.PartsUnprotectResponse, error) {
	log.Printf("[PartsService] Unprotect: uuids=%v", req.UserPartsUuid)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, uuid := range req.UserPartsUuid {
			part, ok := user.Parts[uuid]
			if !ok {
				log.Printf("[PartsService] Unprotect: part uuid=%s not found", uuid)
				continue
			}
			part.IsProtected = false
			part.LatestVersion = nowMillis
			user.Parts[uuid] = part
		}
	})

	return &pb.PartsUnprotectResponse{}, nil
}

func (s *PartsServiceServer) Sell(ctx context.Context, req *pb.PartsSellRequest) (*pb.PartsSellResponse, error) {
	log.Printf("[PartsService] Sell: %d part(s)", len(req.UserPartsUuid))

	cat := s.holder.Get()
	catalog := cat.Parts
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		totalGold := int32(0)
		for _, uuid := range req.UserPartsUuid {
			part, ok := user.Parts[uuid]
			if !ok {
				log.Printf("[PartsService] Sell: uuid=%s not found, skipping", uuid)
				continue
			}
			if part.IsProtected {
				log.Printf("[PartsService] Sell: uuid=%s is protected, skipping", uuid)
				continue
			}

			partDef, ok := catalog.PartsById[part.PartsId]
			if !ok {
				log.Printf("[PartsService] Sell: partsId=%d not in catalog, skipping", part.PartsId)
				continue
			}

			sellFunc, ok := catalog.SellPriceByRarity[partDef.RarityType]
			if !ok {
				log.Printf("[PartsService] Sell: no sell price func for rarity=%d, skipping", partDef.RarityType)
				continue
			}

			gold := sellFunc.Evaluate(part.Level)
			totalGold += gold
			delete(user.Parts, uuid)
			for k := range user.PartsStatusSubs {
				if k.UserPartsUuid == uuid {
					delete(user.PartsStatusSubs, k)
				}
			}
			log.Printf("[PartsService] Sell: uuid=%s partsId=%d level=%d -> %d gold", uuid, part.PartsId, part.Level, gold)
		}

		if totalGold > 0 {
			user.ConsumableItems[config.ConsumableItemIdForGold] += totalGold
			log.Printf("[PartsService] Sell: total gold +%d", totalGold)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("parts sell: %w", err)
	}

	return &pb.PartsSellResponse{}, nil
}

func (s *PartsServiceServer) Enhance(ctx context.Context, req *pb.PartsEnhanceRequest) (*pb.PartsEnhanceResponse, error) {
	log.Printf("[PartsService] Enhance: uuid=%s", req.UserPartsUuid)

	cat := s.holder.Get()
	catalog := cat.Parts
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	isSuccess := false

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		part, ok := user.Parts[req.UserPartsUuid]
		if !ok {
			log.Printf("[PartsService] Enhance: part uuid=%s not found", req.UserPartsUuid)
			return
		}

		if part.Level >= model.PartsMaxLevel {
			log.Printf("[PartsService] Enhance: part uuid=%s already at max level %d", req.UserPartsUuid, part.Level)
			return
		}

		partDef, ok := catalog.PartsById[part.PartsId]
		if !ok {
			log.Printf("[PartsService] Enhance: part master id=%d not found", part.PartsId)
			return
		}

		rarity, ok := catalog.RarityByRarityType[partDef.RarityType]
		if !ok {
			log.Printf("[PartsService] Enhance: rarity type=%d not found", partDef.RarityType)
			return
		}

		goldCost := int32(0)
		if prices, ok := catalog.PriceByGroupAndLevel[rarity.PartsLevelUpPriceGroupId]; ok {
			goldCost = prices[part.Level]
		}

		currentGold := user.ConsumableItems[config.ConsumableItemIdForGold]
		if currentGold < goldCost {
			log.Printf("[PartsService] Enhance: insufficient gold have=%d need=%d", currentGold, goldCost)
			return
		}

		user.ConsumableItems[config.ConsumableItemIdForGold] -= goldCost

		successRate := int32(1000)
		if rates, ok := catalog.RateByGroupAndLevel[rarity.PartsLevelUpRateGroupId]; ok {
			if r, ok := rates[part.Level]; ok {
				successRate = r
			}
		}
		baseRate := successRate
		successRate = cat.Campaign.PartsRateBonus(campaign.PartsTarget{
			PartsId:      part.PartsId,
			PartsGroupId: partDef.PartsGroupId,
			Rarity:       model.RarityType(partDef.RarityType),
		}, campaign.Filter{NowMillis: nowMillis, UserStatus: campaign.TargetUserStatusAll}).Apply(baseRate)

		if rand.Intn(1000) < int(successRate) {
			part.Level++
			isSuccess = true
			log.Printf("[PartsService] Enhance: SUCCESS partsId=%d level %d -> %d (rate=%d‰ base=%d‰, cost=%d gold)",
				part.PartsId, part.Level-1, part.Level, successRate, baseRate, goldCost)

			grantPartsSubStatuses(catalog, user, req.UserPartsUuid, part, partDef, nowMillis)
		} else {
			log.Printf("[PartsService] Enhance: FAIL partsId=%d stays level %d (rate=%d‰ base=%d‰, cost=%d gold)",
				part.PartsId, part.Level, successRate, baseRate, goldCost)
		}

		part.LatestVersion = nowMillis
		user.Parts[req.UserPartsUuid] = part
	})
	if err != nil {
		return nil, fmt.Errorf("parts enhance: %w", err)
	}

	return &pb.PartsEnhanceResponse{
		IsSuccess: isSuccess,
	}, nil
}

func grantPartsSubStatuses(catalog *masterdata.PartsCatalog, user *store.UserState, uuid string, part store.PartsState, partDef masterdata.EntityMParts, nowMillis int64) {
	if !model.IsPartsSubStatusEnhancementLevel(part.Level) {
		return
	}
	existingKeys := make([]store.PartsStatusSubKey, 0, model.PartsMaxSubStatusCount)
	var emptySlot int32
	for slot := int32(1); slot <= model.PartsMaxSubStatusCount; slot++ {
		key := store.PartsStatusSubKey{UserPartsUuid: uuid, StatusIndex: slot}
		if _, exists := user.PartsStatusSubs[key]; exists {
			existingKeys = append(existingKeys, key)
		} else if emptySlot == 0 {
			emptySlot = slot
		}
	}
	// Every milestone first fills one missing slot, regardless of initial rank.
	if emptySlot != 0 {
		pool := catalog.SubStatusPool[partDef.PartsStatusSubLotteryGroupId]
		pick, picked := store.PickUniquePartsSubStatus(pool, user, uuid)
		if !picked {
			return
		}
		def := catalog.PartsStatusSubById[pick]
		key := store.PartsStatusSubKey{UserPartsUuid: uuid, StatusIndex: emptySlot}
		user.PartsStatusSubs[key] = store.PartsStatusSubState{
			UserPartsUuid: uuid, StatusIndex: emptySlot, PartsStatusSubLotteryId: pick,
			Level: part.Level, StatusKindType: def.StatusKindType, StatusCalculationType: def.StatusCalculationType,
			StatusChangeValue: def.Initial.Roll(), LatestVersion: nowMillis,
		}
		return
	}
	key := existingKeys[rand.Intn(len(existingKeys))]
	sub := user.PartsStatusSubs[key]
	def, ok := catalog.PartsStatusSubById[sub.PartsStatusSubLotteryId]
	if !ok {
		return
	}
	sub.Level = part.Level
	sub.StatusChangeValue += def.Growth.Roll()
	sub.LatestVersion = nowMillis
	user.PartsStatusSubs[key] = sub
}

func (s *PartsServiceServer) ReplacePreset(ctx context.Context, req *pb.PartsReplacePresetRequest) (*pb.PartsReplacePresetResponse, error) {
	log.Printf("[PartsService] ReplacePreset: preset=%d uuids=[%s, %s, %s]",
		req.UserPartsPresetNumber, req.UserPartsUuid01, req.UserPartsUuid02, req.UserPartsUuid03)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		preset := user.PartsPresets[req.UserPartsPresetNumber]
		preset.UserPartsPresetNumber = req.UserPartsPresetNumber
		preset.UserPartsUuid01 = req.UserPartsUuid01
		preset.UserPartsUuid02 = req.UserPartsUuid02
		preset.UserPartsUuid03 = req.UserPartsUuid03
		preset.LatestVersion = nowMillis
		user.PartsPresets[req.UserPartsPresetNumber] = preset
	})
	if err != nil {
		return nil, fmt.Errorf("parts replace preset: %w", err)
	}

	return &pb.PartsReplacePresetResponse{}, nil
}

func (s *PartsServiceServer) UpdatePresetName(ctx context.Context, req *pb.PartsUpdatePresetNameRequest) (*pb.PartsUpdatePresetNameResponse, error) {
	log.Printf("[PartsService] UpdatePresetName: preset=%d name=%q", req.UserPartsPresetNumber, req.Name)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		preset := user.PartsPresets[req.UserPartsPresetNumber]
		preset.UserPartsPresetNumber = req.UserPartsPresetNumber
		preset.Name = req.Name
		preset.LatestVersion = nowMillis
		user.PartsPresets[req.UserPartsPresetNumber] = preset
	})
	if err != nil {
		return nil, fmt.Errorf("parts update preset name: %w", err)
	}

	return &pb.PartsUpdatePresetNameResponse{}, nil
}

func (s *PartsServiceServer) UpdatePresetTagNumber(ctx context.Context, req *pb.PartsUpdatePresetTagNumberRequest) (*pb.PartsUpdatePresetTagNumberResponse, error) {
	log.Printf("[PartsService] UpdatePresetTagNumber: preset=%d tag=%d", req.UserPartsPresetNumber, req.UserPartsPresetTagNumber)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		preset := user.PartsPresets[req.UserPartsPresetNumber]
		preset.UserPartsPresetNumber = req.UserPartsPresetNumber
		preset.UserPartsPresetTagNumber = req.UserPartsPresetTagNumber
		preset.LatestVersion = nowMillis
		user.PartsPresets[req.UserPartsPresetNumber] = preset
	})
	if err != nil {
		return nil, fmt.Errorf("parts update preset tag number: %w", err)
	}

	return &pb.PartsUpdatePresetTagNumberResponse{}, nil
}

func (s *PartsServiceServer) UpdatePresetTagName(ctx context.Context, req *pb.PartsUpdatePresetTagNameRequest) (*pb.PartsUpdatePresetTagNameResponse, error) {
	log.Printf("[PartsService] UpdatePresetTagName: tag=%d name=%q", req.UserPartsPresetTagNumber, req.Name)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		tag := user.PartsPresetTags[req.UserPartsPresetTagNumber]
		tag.UserPartsPresetTagNumber = req.UserPartsPresetTagNumber
		tag.Name = req.Name
		tag.LatestVersion = nowMillis
		user.PartsPresetTags[req.UserPartsPresetTagNumber] = tag
	})
	if err != nil {
		return nil, fmt.Errorf("parts update preset tag name: %w", err)
	}

	return &pb.PartsUpdatePresetTagNameResponse{}, nil
}

func (s *PartsServiceServer) CopyPreset(ctx context.Context, req *pb.PartsCopyPresetRequest) (*pb.PartsCopyPresetResponse, error) {
	log.Printf("[PartsService] CopyPreset: from=%d to=%d", req.FromUserPartsPresetNumber, req.ToUserPartsPresetNumber)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		from, ok := user.PartsPresets[req.FromUserPartsPresetNumber]
		if !ok {
			log.Printf("[PartsService] CopyPreset: source preset=%d not found, skipping", req.FromUserPartsPresetNumber)
			return
		}
		to := from
		to.UserPartsPresetNumber = req.ToUserPartsPresetNumber
		to.LatestVersion = nowMillis
		user.PartsPresets[req.ToUserPartsPresetNumber] = to
	})
	if err != nil {
		return nil, fmt.Errorf("parts copy preset: %w", err)
	}

	return &pb.PartsCopyPresetResponse{}, nil
}

func (s *PartsServiceServer) RemovePreset(ctx context.Context, req *pb.PartsRemovePresetRequest) (*pb.PartsRemovePresetResponse, error) {
	log.Printf("[PartsService] RemovePreset: preset=%d", req.UserPartsPresetNumber)

	userId := CurrentUserId(ctx, s.users, s.sessions)

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		delete(user.PartsPresets, req.UserPartsPresetNumber)
	})
	if err != nil {
		return nil, fmt.Errorf("parts remove preset: %w", err)
	}

	return &pb.PartsRemovePresetResponse{}, nil
}

package service

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

const companionMaxLevel = int32(50)

type CompanionServiceServer struct {
	pb.UnimplementedCompanionServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewCompanionServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *CompanionServiceServer {
	return &CompanionServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *CompanionServiceServer) Enhance(ctx context.Context, req *pb.CompanionEnhanceRequest) (*pb.CompanionEnhanceResponse, error) {
	log.Printf("[CompanionService] Enhance: uuid=%s addLevel=%d", req.UserCompanionUuid, req.AddLevelCount)

	cat := s.holder.Get()
	catalog := cat.Companion
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	if req.AddLevelCount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "add level count must be positive")
	}
	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		companion, ok := user.Companions[req.UserCompanionUuid]
		if !ok {
			log.Printf("[CompanionService] Enhance: companion uuid=%s not found", req.UserCompanionUuid)
			return
		}

		compDef, ok := catalog.CompanionById[companion.CompanionId]
		if !ok {
			log.Printf("[CompanionService] Enhance: companion master id=%d not found", companion.CompanionId)
			return
		}

		addLevelCount := req.AddLevelCount
		if addLevelCount > companionMaxLevel-companion.Level {
			addLevelCount = companionMaxLevel - companion.Level
		}
		targetLevel := companion.Level + addLevelCount

		costs := make([]store.PossessionCost, 0)
		for lvl := companion.Level; lvl < targetLevel; lvl++ {
			if costFunc, ok := catalog.GoldCostByCategory[compDef.CompanionCategoryType]; ok {
				goldCost := costFunc.Evaluate(lvl)
				costs = append(costs, consumableCost(config.ConsumableItemIdForGold, goldCost))
			}

			matKey := masterdata.CompanionLevelKey{CategoryType: compDef.CompanionCategoryType, Level: lvl}
			if mat, ok := catalog.MaterialsByKey[matKey]; ok {
				costs = append(costs, materialCost(mat.MaterialId, mat.Count))
			}
		}
		if err := deductUpgradeCosts(user, "companion enhancement cost", costs); err != nil {
			validationErr = err
			return
		}

		companion.Level = targetLevel
		companion.LatestVersion = nowMillis
		user.Companions[req.UserCompanionUuid] = companion
		log.Printf("[CompanionService] Enhance: companionId=%d level -> %d", companion.CompanionId, targetLevel)
	})
	if err != nil {
		return nil, fmt.Errorf("companion enhance: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.CompanionEnhanceResponse{}, nil
}

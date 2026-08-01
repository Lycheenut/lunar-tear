package service

import (
	"context"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

type CharacterServiceServer struct {
	pb.UnimplementedCharacterServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewCharacterServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *CharacterServiceServer {
	return &CharacterServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *CharacterServiceServer) Rebirth(ctx context.Context, req *pb.RebirthRequest) (*pb.RebirthResponse, error) {
	log.Printf("[CharacterService] Rebirth: characterId=%d rebirthCount=%d", req.CharacterId, req.RebirthCount)
	if req.RebirthCount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "rebirth count must be positive")
	}

	cat := s.holder.Get()
	catalog := cat.CharacterRebirth
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	stepGroupId, ok := catalog.StepGroupByCharacterId[req.CharacterId]
	if !ok {
		return nil, status.Error(codes.NotFound, "character rebirth configuration not found")
	}

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if _, owned := user.Characters[req.CharacterId]; !owned {
			validationErr = status.Error(codes.FailedPrecondition, "character is not owned")
			return
		}
		current := user.CharacterRebirths[req.CharacterId]
		currentCount := current.RebirthCount
		if currentCount >= config.CharacterRebirthAvailableCount || req.RebirthCount > config.CharacterRebirthAvailableCount-currentCount {
			validationErr = status.Error(codes.FailedPrecondition, "character rebirth limit exceeded")
			return
		}
		targetCount := currentCount + req.RebirthCount

		costs := make([]store.PossessionCost, 0)
		for count := currentCount; count < targetCount; count++ {
			step, ok := catalog.StepByGroupAndCount[masterdata.StepKey{GroupId: stepGroupId, BeforeRebirthCount: count}]
			if !ok {
				validationErr = status.Error(codes.FailedPrecondition, "character rebirth step is unavailable")
				return
			}

			costs = append(costs, consumableCost(config.ConsumableItemIdForGold, config.CharacterRebirthConsumeGold))

			materials := catalog.MaterialsByGroupId[step.CharacterRebirthMaterialGroupId]
			for _, mat := range materials {
				costs = append(costs, materialCost(mat.MaterialId, mat.Count))
			}
		}
		if err := deductUpgradeCosts(user, "character rebirth cost", costs); err != nil {
			validationErr = err
			return
		}

		log.Printf("[CharacterService] Rebirth: characterId=%d count %d -> %d", req.CharacterId, currentCount, targetCount)
		user.CharacterRebirths[req.CharacterId] = store.CharacterRebirthState{
			CharacterId:   req.CharacterId,
			RebirthCount:  targetCount,
			LatestVersion: nowMillis,
		}
	})
	if err != nil {
		log.Printf("[CharacterService] Rebirth error: %v", err)
		return nil, err
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.RebirthResponse{}, nil
}

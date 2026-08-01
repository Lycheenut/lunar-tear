package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

// The released master data does not contain Explore's reward pool. Keep the
// existing compatibility rewards until a reliable source is available.
const (
	exploreStaminaRecovery  = 1000
	exploreRewardMaterialId = 100001
	exploreRewardBaseCount  = 1
)

type ExploreServiceServer struct {
	pb.UnimplementedExploreServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewExploreServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *ExploreServiceServer {
	return &ExploreServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *ExploreServiceServer) StartExplore(ctx context.Context, req *pb.StartExploreRequest) (*pb.StartExploreResponse, error) {
	log.Printf("[ExploreService] StartExplore: exploreId=%d useConsumableItemId=%d", req.ExploreId, req.UseConsumableItemId)

	cats := s.holder.Get()
	catalog := cats.Explore
	if _, ok := catalog.Explores[req.ExploreId]; !ok {
		return nil, fmt.Errorf("explore id=%d not found", req.ExploreId)
	}

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		explore := catalog.Explores[req.ExploreId]
		if user.Explore.PlayingExploreId != 0 {
			validationErr = status.Error(codes.FailedPrecondition, "an explore is already in progress")
			return
		}
		if nowMillis < explore.StartDatetime || !exploreUnlocked(user, catalog, explore) {
			validationErr = status.Error(codes.FailedPrecondition, "explore is not unlocked")
			return
		}
		useTicket := req.UseConsumableItemId != 0
		if useTicket {
			if req.UseConsumableItemId != cats.GameConfig.ConsumableItemIdForExploreTicket || explore.ConsumeItemCount <= 0 {
				validationErr = status.Error(codes.InvalidArgument, "invalid explore ticket")
				return
			}
			if user.ConsumableItems[req.UseConsumableItemId] < explore.ConsumeItemCount {
				validationErr = status.Error(codes.FailedPrecondition, "not enough explore tickets")
				return
			}
			user.ConsumableItems[req.UseConsumableItemId] -= explore.ConsumeItemCount
		} else if user.Explore.LatestPlayDatetime != 0 {
			interval := time.Duration(cats.GameConfig.ExplorePlayIntervalMinute) * time.Minute
			if nowMillis-user.Explore.LatestPlayDatetime < interval.Milliseconds() {
				validationErr = status.Error(codes.FailedPrecondition, "explore cooldown has not elapsed")
				return
			}
		}

		user.Explore = store.ExploreState{
			PlayingExploreId:   req.ExploreId,
			IsUseExploreTicket: useTicket,
			LatestPlayDatetime: nowMillis,
			LatestVersion:      nowMillis,
		}
	})
	if err != nil {
		return nil, fmt.Errorf("start explore: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.StartExploreResponse{}, nil
}

func (s *ExploreServiceServer) FinishExplore(ctx context.Context, req *pb.FinishExploreRequest) (*pb.FinishExploreResponse, error) {
	log.Printf("[ExploreService] FinishExplore: exploreId=%d score=%d", req.ExploreId, req.Score)

	catalog := s.holder.Get().Explore
	explore, ok := catalog.Explores[req.ExploreId]
	if !ok {
		return nil, fmt.Errorf("explore id=%d not found", req.ExploreId)
	}

	assetGradeIconId := catalog.GradeForScore(req.ExploreId, req.Score)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	if req.Score < 0 {
		return nil, status.Error(codes.InvalidArgument, "score must not be negative")
	}
	rewardCount := int32(exploreRewardBaseCount) * explore.RewardLotteryCount

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if user.Explore.PlayingExploreId != req.ExploreId {
			validationErr = status.Error(codes.FailedPrecondition, "explore is not in progress")
			return
		}
		existing, exists := user.ExploreScores[req.ExploreId]
		if !exists || req.Score > existing.MaxScore {
			user.ExploreScores[req.ExploreId] = store.ExploreScoreState{
				ExploreId:              req.ExploreId,
				MaxScore:               req.Score,
				MaxScoreUpdateDatetime: nowMillis,
				LatestVersion:          nowMillis,
			}
		}

		user.Explore = store.ExploreState{
			PlayingExploreId:   0,
			IsUseExploreTicket: false,
			LatestPlayDatetime: user.Explore.LatestPlayDatetime,
			LatestVersion:      nowMillis,
		}

		user.Status.StaminaMilliValue += exploreStaminaRecovery
		user.Status.StaminaUpdateDatetime = nowMillis
		user.Status.LatestVersion = nowMillis
		log.Printf("[ExploreService] FinishExplore: stamina +%d -> %d", exploreStaminaRecovery, user.Status.StaminaMilliValue)

		user.Materials[exploreRewardMaterialId] += rewardCount
	})
	if err != nil {
		return nil, fmt.Errorf("finish explore: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	rewards := []*pb.ExploreReward{
		{
			PossessionType: int32(model.PossessionTypeMaterial),
			PossessionId:   exploreRewardMaterialId,
			Count:          rewardCount,
		},
	}

	return &pb.FinishExploreResponse{
		AcquireStaminaCount: exploreStaminaRecovery,
		ExploreReward:       rewards,
		AssetGradeIconId:    assetGradeIconId,
	}, nil
}

func (s *ExploreServiceServer) RetireExplore(ctx context.Context, req *pb.RetireExploreRequest) (*pb.RetireExploreResponse, error) {
	log.Printf("[ExploreService] RetireExplore: exploreId=%d", req.ExploreId)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if user.Explore.PlayingExploreId != req.ExploreId {
			validationErr = status.Error(codes.FailedPrecondition, "explore is not in progress")
			return
		}
		user.Explore = store.ExploreState{
			PlayingExploreId:   0,
			IsUseExploreTicket: false,
			LatestPlayDatetime: user.Explore.LatestPlayDatetime,
			LatestVersion:      nowMillis,
		}
	})
	if err != nil {
		return nil, fmt.Errorf("retire explore: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.RetireExploreResponse{}, nil
}

func exploreUnlocked(user *store.UserState, catalog *masterdata.ExploreCatalog, explore masterdata.EntityMExplore) bool {
	if explore.ExploreUnlockConditionId == 0 {
		return true
	}
	condition, ok := catalog.UnlockConditions[explore.ExploreUnlockConditionId]
	if !ok {
		return false
	}
	switch condition.ExploreUnlockConditionType {
	case 1:
		questId := catalog.UnlockQuestIds[condition.ExploreUnlockConditionId]
		quest, exists := user.Quests[questId]
		return questId != 0 && exists && quest.QuestStateType == model.UserQuestStateTypeCleared
	case 2:
		lowerExploreId := catalog.LowerDifficulty[explore.ExploreId]
		score, exists := user.ExploreScores[lowerExploreId]
		return lowerExploreId != 0 && exists && score.MaxScore >= condition.ConditionValue
	default:
		return false
	}
}

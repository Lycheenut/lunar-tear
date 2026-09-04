package service

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

type LabyrinthServiceServer struct {
	pb.UnimplementedLabyrinthServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func labyrinthStageCleared(user *store.UserState, questIds []int32) bool {
	if len(questIds) == 0 {
		return false
	}
	for _, questId := range questIds {
		if user.Quests[questId].QuestStateType != model.UserQuestStateTypeCleared {
			return false
		}
	}
	return true
}

func NewLabyrinthServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *LabyrinthServiceServer {
	if holder == nil {
		panic("runtime holder is required")
	}
	return &LabyrinthServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *LabyrinthServiceServer) ReceiveStageAccumulationReward(ctx context.Context, req *pb.ReceiveStageAccumulationRewardRequest) (*pb.ReceiveStageAccumulationRewardResponse, error) {
	log.Printf("[LabyrinthService] ReceiveStageAccumulationReward: chapter=%d stage=%d questMissionClearCount=%d",
		req.EventQuestChapterId, req.StageOrder, req.QuestMissionClearCount)

	cat := s.holder.Get()
	laby := cat.Labyrinth
	granter := cat.QuestHandler.Granter
	if !laby.HasStage(req.EventQuestChapterId, req.StageOrder) {
		return nil, status.Error(codes.InvalidArgument, "labyrinth stage not found")
	}
	questIds, ok := laby.StageQuestIds(cat.Quest, req.EventQuestChapterId, req.StageOrder)
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "labyrinth stage quests are unavailable")
	}

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	key := store.LabyrinthStageKey{
		EventQuestChapterId: req.EventQuestChapterId,
		StageOrder:          req.StageOrder,
	}

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		rec := user.LabyrinthStages[key]
		old := rec.AccumulationRewardReceivedQuestMissionCount
		actualClearCount := cat.QuestHandler.ClearedQuestMissionCount(user, questIds)

		items, highest := laby.CollectAccumulationRewards(req.EventQuestChapterId, req.StageOrder, old, actualClearCount)
		if highest <= old {
			log.Printf("[LabyrinthService] ReceiveStageAccumulationReward: nothing to grant for chapter=%d stage=%d (claimed=%d, target=%d)",
				req.EventQuestChapterId, req.StageOrder, old, actualClearCount)
			return
		}

		for _, it := range items {
			granter.GrantFull(user, model.PossessionType(it.PossessionType), it.PossessionId, it.Count, nowMillis)
		}

		rec.EventQuestChapterId = req.EventQuestChapterId
		rec.StageOrder = req.StageOrder
		rec.AccumulationRewardReceivedQuestMissionCount = highest
		rec.LatestVersion = nowMillis
		user.LabyrinthStages[key] = rec

		log.Printf("[LabyrinthService] ReceiveStageAccumulationReward: chapter=%d stage=%d granted %d item(s), claimed %d -> %d",
			req.EventQuestChapterId, req.StageOrder, len(items), old, highest)
	})
	if err != nil {
		return nil, fmt.Errorf("receive labyrinth accumulation reward: %w", err)
	}

	return &pb.ReceiveStageAccumulationRewardResponse{}, nil
}

func (s *LabyrinthServiceServer) ReceiveStageClearReward(ctx context.Context, req *pb.ReceiveStageClearRewardRequest) (*pb.ReceiveStageClearRewardResponse, error) {
	log.Printf("[LabyrinthService] ReceiveStageClearReward: chapter=%d stage=%d",
		req.EventQuestChapterId, req.StageOrder)

	cat := s.holder.Get()
	laby := cat.Labyrinth
	granter := cat.QuestHandler.Granter
	if !laby.HasStage(req.EventQuestChapterId, req.StageOrder) {
		return nil, status.Error(codes.InvalidArgument, "labyrinth stage not found")
	}
	questIds, ok := laby.StageQuestIds(cat.Quest, req.EventQuestChapterId, req.StageOrder)
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "labyrinth stage quests are unavailable")
	}
	items := laby.StageClearReward(req.EventQuestChapterId, req.StageOrder)
	if len(items) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "labyrinth stage reward is unavailable")
	}

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	key := store.LabyrinthStageKey{
		EventQuestChapterId: req.EventQuestChapterId,
		StageOrder:          req.StageOrder,
	}

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		rec := user.LabyrinthStages[key]
		if rec.IsReceivedStageClearReward {
			log.Printf("[LabyrinthService] ReceiveStageClearReward: already claimed chapter=%d stage=%d",
				req.EventQuestChapterId, req.StageOrder)
			return
		}
		if !labyrinthStageCleared(user, questIds) {
			validationErr = status.Error(codes.FailedPrecondition, "labyrinth stage is not cleared")
			return
		}
		for _, it := range items {
			granter.GrantFull(user, model.PossessionType(it.PossessionType), it.PossessionId, it.Count, nowMillis)
		}

		rec.EventQuestChapterId = req.EventQuestChapterId
		rec.StageOrder = req.StageOrder
		rec.IsReceivedStageClearReward = true
		rec.LatestVersion = nowMillis
		user.LabyrinthStages[key] = rec

		log.Printf("[LabyrinthService] ReceiveStageClearReward: chapter=%d stage=%d granted %d item(s)",
			req.EventQuestChapterId, req.StageOrder, len(items))
	})
	if err != nil {
		return nil, fmt.Errorf("receive labyrinth stage clear reward: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.ReceiveStageClearRewardResponse{}, nil
}

func (s *LabyrinthServiceServer) UpdateSeasonData(ctx context.Context, req *pb.UpdateSeasonDataRequest) (*pb.UpdateSeasonDataResponse, error) {
	cat := s.holder.Get()
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var seasonResult []*pb.LabyrinthSeasonResult
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if result := updateLabyrinthSeasonData(cat, user, req.EventQuestChapterId, nowMillis); result != nil {
			seasonResult = append(seasonResult, result)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("update labyrinth season data: %w", err)
	}

	log.Printf("[LabyrinthService] UpdateSeasonData: chapter=%d -> %d result(s)",
		req.EventQuestChapterId, len(seasonResult))
	return &pb.UpdateSeasonDataResponse{SeasonResult: seasonResult}, nil
}

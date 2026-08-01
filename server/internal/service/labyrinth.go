package service

import (
	"context"
	"fmt"
	"log"

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

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	key := store.LabyrinthStageKey{
		EventQuestChapterId: req.EventQuestChapterId,
		StageOrder:          req.StageOrder,
	}

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		rec := user.LabyrinthStages[key]
		old := rec.AccumulationRewardReceivedQuestMissionCount

		items, highest := laby.CollectAccumulationRewards(req.EventQuestChapterId, req.StageOrder, old, req.QuestMissionClearCount)
		if highest <= old {
			log.Printf("[LabyrinthService] ReceiveStageAccumulationReward: nothing to grant for chapter=%d stage=%d (claimed=%d, target=%d)",
				req.EventQuestChapterId, req.StageOrder, old, req.QuestMissionClearCount)
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

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	key := store.LabyrinthStageKey{
		EventQuestChapterId: req.EventQuestChapterId,
		StageOrder:          req.StageOrder,
	}

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		rec := user.LabyrinthStages[key]
		if rec.IsReceivedStageClearReward {
			log.Printf("[LabyrinthService] ReceiveStageClearReward: already claimed chapter=%d stage=%d",
				req.EventQuestChapterId, req.StageOrder)
			return
		}
		items := laby.StageClearReward(req.EventQuestChapterId, req.StageOrder)
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

	return &pb.ReceiveStageClearRewardResponse{}, nil
}

func (s *LabyrinthServiceServer) UpdateSeasonData(ctx context.Context, req *pb.UpdateSeasonDataRequest) (*pb.UpdateSeasonDataResponse, error) {
	laby := s.holder.Get().Labyrinth

	var seasonResult []*pb.LabyrinthSeasonResult
	for _, m := range laby.SeasonMilestones(req.EventQuestChapterId) {
		rewards := make([]*pb.LabyrinthReward, 0, len(m.Rewards))
		for _, it := range m.Rewards {
			rewards = append(rewards, &pb.LabyrinthReward{
				PossessionType: it.PossessionType,
				PossessionId:   it.PossessionId,
				Count:          it.Count,
			})
		}
		seasonResult = append(seasonResult, &pb.LabyrinthSeasonResult{
			EventQuestChapterId: req.EventQuestChapterId,
			HeadQuestId:         m.HeadQuestId,
			SeasonReward:        rewards,
			HeadStageOrder:      m.HeadStageOrder,
		})
	}

	log.Printf("[LabyrinthService] UpdateSeasonData: chapter=%d -> %d milestone(s)",
		req.EventQuestChapterId, len(seasonResult))
	return &pb.UpdateSeasonDataResponse{SeasonResult: seasonResult}, nil
}

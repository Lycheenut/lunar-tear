package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func (s *QuestServiceServer) ReceiveTowerAccumulationReward(ctx context.Context, req *pb.ReceiveTowerAccumulationRewardRequest) (*pb.ReceiveTowerAccumulationRewardResponse, error) {
	log.Printf("[QuestService] ReceiveTowerAccumulationReward: eventQuestChapterId=%d targetMissionClearCount=%d",
		req.EventQuestChapterId, req.TargetMissionClearCount)

	cat := s.holder.Get()
	tower := cat.Tower
	granter := cat.QuestHandler.Granter

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		rec := user.TowerAccumulationRewards[req.EventQuestChapterId]
		old := rec.LatestRewardReceiveQuestMissionClearCount

		items, highest := tower.CollectRewards(req.EventQuestChapterId, old, req.TargetMissionClearCount)
		if highest <= old {
			log.Printf("[QuestService] ReceiveTowerAccumulationReward: nothing to grant for chapter=%d (claimed=%d, target=%d)",
				req.EventQuestChapterId, old, req.TargetMissionClearCount)
			return
		}

		for _, it := range items {
			granter.GrantFull(user, model.PossessionType(it.PossessionType), it.PossessionId, it.Count, nowMillis)
		}

		rec.EventQuestChapterId = req.EventQuestChapterId
		rec.LatestRewardReceiveQuestMissionClearCount = highest
		rec.LatestVersion = nowMillis
		user.TowerAccumulationRewards[req.EventQuestChapterId] = rec

		log.Printf("[QuestService] ReceiveTowerAccumulationReward: chapter=%d granted %d item(s), claimed %d -> %d",
			req.EventQuestChapterId, len(items), old, highest)
	})
	if err != nil {
		return nil, fmt.Errorf("receive tower accumulation reward: %w", err)
	}

	return &pb.ReceiveTowerAccumulationRewardResponse{}, nil
}

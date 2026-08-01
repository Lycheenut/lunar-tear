package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type QuestServiceServer struct {
	pb.UnimplementedQuestServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewQuestServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *QuestServiceServer {
	if holder == nil {
		panic("runtime holder is required")
	}
	return &QuestServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *QuestServiceServer) UpdateMainFlowSceneProgress(ctx context.Context, req *pb.UpdateMainFlowSceneProgressRequest) (*pb.UpdateMainFlowSceneProgressResponse, error) {
	log.Printf("[QuestService] UpdateMainFlowSceneProgress: questSceneId=%d", req.QuestSceneId)

	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		engine.HandleMainFlowSceneProgress(user, req.QuestSceneId, gametime.NowMillis())
	})

	return &pb.UpdateMainFlowSceneProgressResponse{}, nil
}

func (s *QuestServiceServer) UpdateReplayFlowSceneProgress(ctx context.Context, req *pb.UpdateReplayFlowSceneProgressRequest) (*pb.UpdateReplayFlowSceneProgressResponse, error) {
	log.Printf("[QuestService] UpdateReplayFlowSceneProgress: questSceneId=%d", req.QuestSceneId)

	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		engine.HandleReplayFlowSceneProgress(user, req.QuestSceneId, gametime.NowMillis())
	})

	return &pb.UpdateReplayFlowSceneProgressResponse{}, nil
}

func (s *QuestServiceServer) UpdateMainQuestSceneProgress(ctx context.Context, req *pb.UpdateMainQuestSceneProgressRequest) (*pb.UpdateMainQuestSceneProgressResponse, error) {
	log.Printf("[QuestService] UpdateMainQuestSceneProgress: questSceneId=%d", req.QuestSceneId)

	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		engine.HandleMainQuestSceneProgress(user, req.QuestSceneId)
	})

	return &pb.UpdateMainQuestSceneProgressResponse{}, nil
}

func (s *QuestServiceServer) StartMainQuest(ctx context.Context, req *pb.StartMainQuestRequest) (*pb.StartMainQuestResponse, error) {
	log.Printf("[QuestService] StartMainQuest: questId=%d isMainFlow=%v isReplayFlow=%v isBattleOnly=%v maxAutoOrbitCount=%d",
		req.QuestId, req.IsMainFlow, req.IsReplayFlow, req.IsBattleOnly, req.MaxAutoOrbitCount)

	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	s.users.UpdateUser(userId, func(user *store.UserState) {
		if req.IsReplayFlow {
			engine.HandleQuestStartReplay(user, req.QuestId, req.IsBattleOnly, req.UserDeckNumber, nowMillis)
		} else {
			engine.HandleQuestStart(user, req.QuestId, req.IsBattleOnly, req.IsMainFlow, req.UserDeckNumber, nowMillis)
		}
		startAutoOrbit(user, model.QuestTypeMain, 0, req.QuestId, req.MaxAutoOrbitCount, nowMillis)
	})

	drops := engine.BattleDropRewards(req.QuestId)
	pbDrops := make([]*pb.BattleDropReward, len(drops))
	for i, d := range drops {
		pbDrops[i] = &pb.BattleDropReward{
			QuestSceneId:         d.QuestSceneId,
			BattleDropCategoryId: d.BattleDropCategoryId,
			BattleDropEffectId:   1,
		}
	}

	return &pb.StartMainQuestResponse{
		BattleDropReward: pbDrops,
	}, nil
}

func emptyAutoOrbitReward() *pb.QuestAutoOrbitResult {
	return &pb.QuestAutoOrbitResult{
		DropReward:               []*pb.QuestReward{},
		UserStatusCampaignReward: []*pb.QuestReward{},
	}
}

func autoOrbitDropsToProto(drops []store.AutoOrbitDropEntry) []*pb.QuestReward {
	out := make([]*pb.QuestReward, len(drops))
	for i, d := range drops {
		out[i] = &pb.QuestReward{
			PossessionType: d.PossessionType,
			PossessionId:   d.PossessionId,
			Count:          d.Count,
			IsAutoSale:     d.IsAutoSale,
		}
	}
	return out
}

func toProtoRewards(grants []questflow.RewardGrant) []*pb.QuestReward {
	if len(grants) == 0 {
		return []*pb.QuestReward{}
	}
	out := make([]*pb.QuestReward, len(grants))
	for i, g := range grants {
		out[i] = &pb.QuestReward{
			PossessionType: int32(g.PossessionType),
			PossessionId:   g.PossessionId,
			Count:          g.Count,
			IsAutoSale:     g.IsAutoSale,
		}
	}
	return out
}

func (s *QuestServiceServer) FinishMainQuest(ctx context.Context, req *pb.FinishMainQuestRequest) (*pb.FinishMainQuestResponse, error) {
	log.Printf("[QuestService] FinishMainQuest: questId=%d isMainFlow=%v isRetired=%v isAnnihilated=%v isAutoOrbit=%v storySkipType=%d",
		req.QuestId, req.IsMainFlow, req.IsRetired, req.IsAnnihilated, req.IsAutoOrbit, req.StorySkipType)

	nowMillis := gametime.NowMillis()
	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var outcome questflow.FinishOutcome
	var endedDrops []store.AutoOrbitDropEntry
	var loopEnded bool
	s.users.UpdateUser(userId, func(user *store.UserState) {
		outcome = engine.HandleQuestFinish(user, req.QuestId, req.IsRetired, req.IsAnnihilated, nowMillis)
		endedDrops, loopEnded = finishAutoOrbit(user, req.IsAutoOrbit, req.IsRetired, req.IsAnnihilated, model.QuestTypeMain, 0, req.QuestId, nowMillis, outcome.DropRewards)
	})

	autoOrbitReward := emptyAutoOrbitReward()
	if loopEnded {
		autoOrbitReward.DropReward = autoOrbitDropsToProto(endedDrops)
	}

	return &pb.FinishMainQuestResponse{
		DropReward:                      toProtoRewards(outcome.DropRewards),
		FirstClearReward:                toProtoRewards(outcome.FirstClearRewards),
		MissionClearReward:              toProtoRewards(outcome.MissionClearRewards),
		MissionClearCompleteReward:      toProtoRewards(outcome.MissionClearCompleteRewards),
		AutoOrbitResult:                 []*pb.QuestReward{},
		IsBigWin:                        outcome.IsBigWin,
		BigWinClearedQuestMissionIdList: outcome.BigWinClearedQuestMissionIds,
		ReplayFlowFirstClearReward:      toProtoRewards(outcome.ReplayFlowFirstClearRewards),
		UserStatusCampaignReward:        []*pb.QuestReward{},
		AutoOrbitReward:                 autoOrbitReward,
	}, nil
}

func (s *QuestServiceServer) RestartMainQuest(ctx context.Context, req *pb.RestartMainQuestRequest) (*pb.RestartMainQuestResponse, error) {
	log.Printf("[QuestService] RestartMainQuest: questId=%d isMainFlow=%v", req.QuestId, req.IsMainFlow)

	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var deckNumber int32
	s.users.UpdateUser(userId, func(user *store.UserState) {
		engine.HandleQuestRestart(user, req.QuestId, gametime.NowMillis())
		deckNumber = user.Quests[req.QuestId].UserDeckNumber
	})

	drops := engine.BattleDropRewards(req.QuestId)
	pbDrops := make([]*pb.BattleDropReward, len(drops))
	for i, d := range drops {
		pbDrops[i] = &pb.BattleDropReward{
			QuestSceneId:         d.QuestSceneId,
			BattleDropCategoryId: d.BattleDropCategoryId,
			BattleDropEffectId:   1,
		}
	}

	return &pb.RestartMainQuestResponse{
		BattleDropReward: pbDrops,
		DeckNumber:       deckNumber,
	}, nil
}

func (s *QuestServiceServer) FinishAutoOrbit(ctx context.Context, req *emptypb.Empty) (*pb.FinishAutoOrbitResponse, error) {
	log.Printf("[QuestService] FinishAutoOrbit")
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var drops []store.AutoOrbitDropEntry
	s.users.UpdateUser(userId, func(user *store.UserState) {
		drops = consumeAutoOrbitRewards(user)
	})
	pbDrops := make([]*pb.QuestReward, len(drops))
	for i, d := range drops {
		pbDrops[i] = &pb.QuestReward{
			PossessionType: d.PossessionType,
			PossessionId:   d.PossessionId,
			Count:          d.Count,
		}
	}
	return &pb.FinishAutoOrbitResponse{
		AutoOrbitResult: []*pb.QuestReward{},
		AutoOrbitReward: &pb.QuestAutoOrbitResult{
			DropReward:               pbDrops,
			UserStatusCampaignReward: []*pb.QuestReward{},
		},
	}, nil
}

func (s *QuestServiceServer) SkipQuest(ctx context.Context, req *pb.SkipQuestRequest) (*pb.SkipQuestResponse, error) {
	log.Printf("[QuestService] SkipQuest: questId=%d skipCount=%d useEffectItems=%d", req.QuestId, req.SkipCount, len(req.UseEffectItem))

	nowMillis := gametime.NowMillis()
	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var outcome questflow.FinishOutcome
	s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, item := range req.UseEffectItem {
			log.Printf("[QuestService] SkipQuest UseEffectItem: consumableItemId=%d count=%d", item.ConsumableItemId, item.Count)
			user.ConsumableItems[item.ConsumableItemId] -= item.Count
			if user.ConsumableItems[item.ConsumableItemId] < 0 {
				user.ConsumableItems[item.ConsumableItemId] = 0
			}
		}
		outcome = engine.HandleQuestSkip(user, req.QuestId, req.SkipCount, nowMillis)
	})

	return &pb.SkipQuestResponse{
		DropReward:               toProtoRewards(outcome.DropRewards),
		UserStatusCampaignReward: []*pb.QuestReward{},
	}, nil
}

func (s *QuestServiceServer) SetRoute(ctx context.Context, req *pb.SetRouteRequest) (*pb.SetRouteResponse, error) {
	log.Printf("[QuestService] SetRoute: mainQuestRouteId=%d", req.MainQuestRouteId)

	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		user.MainQuest.CurrentMainQuestRouteId = req.MainQuestRouteId
		if seasonId, ok := engine.SeasonIdByRouteId[req.MainQuestRouteId]; ok {
			user.MainQuest.MainQuestSeasonId = seasonId
		}
		now := gametime.NowMillis()
		user.PortalCageStatus.IsCurrentProgress = false
		user.PortalCageStatus.LatestVersion = now
		if user.SideStoryActiveProgress.CurrentSideStoryQuestId != 0 {
			user.SideStoryActiveProgress = store.SideStoryActiveProgress{
				LatestVersion: now,
			}
		}
	})

	return &pb.SetRouteResponse{}, nil
}

func (s *QuestServiceServer) SetQuestSceneChoice(ctx context.Context, req *pb.SetQuestSceneChoiceRequest) (*pb.SetQuestSceneChoiceResponse, error) {
	log.Printf("[QuestService] SetQuestSceneChoice: questSceneId=%d choiceNumber=%d",
		req.QuestSceneId, req.ChoiceNumber)
	return &pb.SetQuestSceneChoiceResponse{}, nil
}

func (s *QuestServiceServer) ResetLimitContentQuestProgress(ctx context.Context, req *pb.ResetLimitContentQuestProgressRequest) (*pb.ResetLimitContentQuestProgressResponse, error) {
	log.Printf("[QuestService] ResetLimitContentQuestProgress: eventQuestChapterId=%d questId=%d",
		req.EventQuestChapterId, req.QuestId)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if _, exists := user.SideStoryQuests[req.QuestId]; exists {
			user.SideStoryQuests[req.QuestId] = store.SideStoryQuestProgress{
				HeadSideStoryQuestSceneId: 0,
				SideStoryQuestStateType:   model.SideStoryQuestStateUnknown,
				LatestVersion:             nowMillis,
			}
		}

		delete(user.QuestLimitContentStatus, req.QuestId)
		for id, restricted := range user.DeckLimitContentRestricted {
			if restricted.EventQuestChapterId == req.EventQuestChapterId {
				delete(user.DeckLimitContentRestricted, id)
			}
		}

		if user.SideStoryActiveProgress.CurrentSideStoryQuestId == req.QuestId {
			user.SideStoryActiveProgress = store.SideStoryActiveProgress{
				LatestVersion: nowMillis,
			}
		}
	})
	if err != nil {
		return nil, fmt.Errorf("reset limit content quest progress: %w", err)
	}

	return &pb.ResetLimitContentQuestProgressResponse{}, nil
}

func (s *QuestServiceServer) SetAutoSaleSetting(ctx context.Context, req *pb.SetAutoSaleSettingRequest) (*pb.SetAutoSaleSettingResponse, error) {
	log.Printf("[QuestService] SetAutoSaleSetting: items=%d", len(req.AutoSaleSettingItem))

	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		user.AutoSaleSettings = make(map[int32]store.AutoSaleSettingState, len(req.AutoSaleSettingItem))
		for itemType, itemValue := range req.AutoSaleSettingItem {
			user.AutoSaleSettings[itemType] = store.AutoSaleSettingState{
				PossessionAutoSaleItemType:  itemType,
				PossessionAutoSaleItemValue: itemValue,
			}
		}
	})

	return &pb.SetAutoSaleSettingResponse{}, nil
}

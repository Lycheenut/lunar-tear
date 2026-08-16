package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	var validationErr error
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		validationErr = engine.HandleMainFlowSceneProgress(user, req.QuestSceneId, gametime.NowMillis())
	})
	if updateErr != nil {
		return nil, fmt.Errorf("update main flow scene: %w", updateErr)
	}
	if validationErr != nil {
		return nil, status.Error(codes.InvalidArgument, validationErr.Error())
	}

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
	var validationErr error
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		validationErr = engine.HandleMainQuestSceneProgress(user, req.QuestSceneId)
	})
	if updateErr != nil {
		return nil, fmt.Errorf("update main quest scene: %w", updateErr)
	}
	if validationErr != nil {
		return nil, status.Error(codes.InvalidArgument, validationErr.Error())
	}

	return &pb.UpdateMainQuestSceneProgressResponse{}, nil
}

func (s *QuestServiceServer) StartMainQuest(ctx context.Context, req *pb.StartMainQuestRequest) (*pb.StartMainQuestResponse, error) {
	log.Printf("[QuestService] StartMainQuest: questId=%d isMainFlow=%v isReplayFlow=%v isBattleOnly=%v maxAutoOrbitCount=%d",
		req.QuestId, req.IsMainFlow, req.IsReplayFlow, req.IsBattleOnly, req.MaxAutoOrbitCount)

	cageUpdate, err := parseCageMeasurableValues(req.CageMeasurableValues)
	if err != nil {
		return nil, err
	}
	catalogs := s.holder.Get()
	engine := catalogs.QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var validationErr error
	var drops []masterdata.BattleDropInfo
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		var beforeCageUpdate store.UserState
		if cageUpdate.present {
			beforeCageUpdate = store.CloneUserState(*user)
			cageUpdate.apply(catalogs, user, nowMillis)
		}
		if req.IsReplayFlow {
			validationErr = engine.HandleQuestStartReplay(user, req.QuestId, req.IsBattleOnly, req.UserDeckNumber, nowMillis)
		} else {
			validationErr = engine.HandleQuestStart(user, req.QuestId, req.IsBattleOnly, req.IsMainFlow, req.UserDeckNumber, nowMillis)
		}
		if validationErr != nil {
			if cageUpdate.present {
				*user = beforeCageUpdate
			}
			return
		}
		startAutoOrbit(user, model.QuestTypeMain, 0, req.QuestId, req.MaxAutoOrbitCount, nowMillis)
		drops = engine.BattleDropRewards(user, req.QuestId)
	})
	if updateErr != nil {
		return nil, fmt.Errorf("start main quest: %w", updateErr)
	}
	if validationErr != nil {
		return nil, status.Error(codes.FailedPrecondition, validationErr.Error())
	}

	return &pb.StartMainQuestResponse{
		BattleDropReward: toProtoBattleDrops(drops),
	}, nil
}

func toProtoBattleDrops(drops []masterdata.BattleDropInfo) []*pb.BattleDropReward {
	if len(drops) == 0 {
		return []*pb.BattleDropReward{}
	}
	out := make([]*pb.BattleDropReward, len(drops))
	for i, drop := range drops {
		out[i] = &pb.BattleDropReward{
			QuestSceneId:         drop.QuestSceneId,
			BattleDropCategoryId: drop.BattleDropCategoryId,
			BattleDropEffectId:   drop.BattleDropEffectId,
		}
	}
	return out
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
			RewardEffectId: g.RewardEffectId,
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
	var validationErr error
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		if err := engine.ValidateQuestContinuation(user, req.QuestId); err != nil {
			validationErr = err
			return
		}
		outcome = engine.HandleQuestFinish(user, req.QuestId, req.IsRetired, req.IsAnnihilated, nowMillis)
		endedDrops, loopEnded = finishAutoOrbit(user, req.IsAutoOrbit, req.IsRetired, req.IsAnnihilated, model.QuestTypeMain, 0, req.QuestId, nowMillis, outcome.DropRewards)
	})
	if updateErr != nil {
		return nil, fmt.Errorf("finish main quest: %w", updateErr)
	}
	if validationErr != nil {
		return nil, status.Error(codes.FailedPrecondition, validationErr.Error())
	}

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
	nowMillis := gametime.NowMillis()
	var deckNumber int32
	var battleBinary []byte
	var validationErr error
	var drops []masterdata.BattleDropInfo
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		if err := engine.HandleQuestRestart(user, req.QuestId, nowMillis); err != nil {
			validationErr = err
			return
		}
		deckNumber = user.Quests[req.QuestId].UserDeckNumber
		battleBinary = battleCheckpoint(user)
		drops = engine.BattleDropRewards(user, req.QuestId)
	})
	if updateErr != nil {
		return nil, fmt.Errorf("restart main quest: %w", updateErr)
	}
	if validationErr != nil {
		return nil, status.Error(codes.FailedPrecondition, validationErr.Error())
	}

	return &pb.RestartMainQuestResponse{
		BattleDropReward: toProtoBattleDrops(drops),
		BattleBinary:     battleBinary,
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
	var validationErr error
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		candidate := store.CloneUserState(*user)
		if err := applyQuestUseEffectItems(s.holder.Get(), &candidate, req.UseEffectItem, nowMillis); err != nil {
			validationErr = err
			return
		}
		outcome, validationErr = engine.HandleQuestSkip(&candidate, req.QuestId, req.SkipCount, nowMillis)
		if validationErr == nil {
			*user = candidate
		}
	})
	if updateErr != nil {
		return nil, fmt.Errorf("skip quest: %w", updateErr)
	}
	if validationErr != nil {
		return nil, status.Error(codes.FailedPrecondition, validationErr.Error())
	}

	return &pb.SkipQuestResponse{
		DropReward:               toProtoRewards(outcome.DropRewards),
		UserStatusCampaignReward: []*pb.QuestReward{},
	}, nil
}

func applyQuestUseEffectItems(cat *runtime.Catalogs, user *store.UserState, items []*pb.UseEffectItem, nowMillis int64) error {
	maxStaminaMillis := cat.Quest.MaxStaminaByLevel[user.Status.Level] * 1000
	for _, item := range items {
		if item.Count <= 0 {
			return fmt.Errorf("effect item count must be positive")
		}
		if _, ok := cat.ConsumableItem.All[item.ConsumableItemId]; !ok {
			return fmt.Errorf("unknown effect item %d", item.ConsumableItemId)
		}
		effects := cat.ConsumableItem.Effects[item.ConsumableItemId]
		hasStaminaRecovery := false
		for _, effect := range effects {
			if effect.EffectTargetType == model.EffectTargetStaminaRecovery {
				hasStaminaRecovery = true
				millis := store.ResolveStaminaEffectMillis(effect.EffectValueType, effect.EffectValue, maxStaminaMillis)
				if millis <= 0 || int64(millis)*int64(item.Count) > int64(^uint32(0)>>1) {
					return fmt.Errorf("invalid stamina recovery effect item %d", item.ConsumableItemId)
				}
			}
		}
		if !hasStaminaRecovery {
			return fmt.Errorf("consumable item %d is not a stamina recovery item", item.ConsumableItemId)
		}
		if user.ConsumableItems[item.ConsumableItemId] < item.Count {
			return fmt.Errorf("insufficient effect item %d", item.ConsumableItemId)
		}
		user.ConsumableItems[item.ConsumableItemId] -= item.Count
		for _, effect := range effects {
			if effect.EffectTargetType == model.EffectTargetStaminaRecovery {
				millis := store.ResolveStaminaEffectMillis(effect.EffectValueType, effect.EffectValue, maxStaminaMillis)
				store.RecoverStamina(user, millis*item.Count, maxStaminaMillis, nowMillis)
			}
		}
	}
	return nil
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
	key := store.QuestSceneChoiceKey{QuestSceneId: req.QuestSceneId, QuestFlowType: req.QuestFlowType}
	_, ok := s.holder.Get().Quest.SceneChoiceByKey[masterdata.QuestSceneChoiceKey{QuestSceneId: req.QuestSceneId, QuestFlowType: req.QuestFlowType, ChoiceNumber: req.ChoiceNumber}]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid quest scene choice")
	}
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		state := store.QuestSceneChoiceState{QuestSceneId: req.QuestSceneId, QuestFlowType: req.QuestFlowType, ChoiceNumber: req.ChoiceNumber, ChoiceDatetime: nowMillis, LatestVersion: nowMillis}
		user.QuestSceneChoices[key] = state
		user.QuestSceneChoiceHistory[store.QuestSceneChoiceHistoryKey{QuestSceneId: req.QuestSceneId, QuestFlowType: req.QuestFlowType, ChoiceNumber: req.ChoiceNumber}] = state
	})
	if err != nil {
		return nil, fmt.Errorf("set quest scene choice: %w", err)
	}
	return &pb.SetQuestSceneChoiceResponse{}, nil
}

func (s *QuestServiceServer) SkipQuestBulk(ctx context.Context, req *pb.SkipQuestBulkRequest) (*pb.SkipQuestBulkResponse, error) {
	log.Printf("[QuestService] SkipQuestBulk: quests=%d", len(req.SkipQuestInfo))
	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var outcome questflow.FinishOutcome
	var validationErr error
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		candidate := store.CloneUserState(*user)
		if err := applyQuestUseEffectItems(s.holder.Get(), &candidate, req.UseEffectItem, nowMillis); err != nil {
			validationErr = err
			return
		}
		questIds, counts := make([]int32, len(req.SkipQuestInfo)), make([]int32, len(req.SkipQuestInfo))
		for i, info := range req.SkipQuestInfo {
			questIds[i], counts[i] = info.QuestId, info.SkipCount
		}
		outcome, validationErr = engine.HandleQuestSkipBulk(&candidate, questIds, counts, nowMillis)
		if validationErr == nil {
			*user = candidate
		}
	})
	if updateErr != nil {
		return nil, fmt.Errorf("bulk skip quests: %w", updateErr)
	}
	if validationErr != nil {
		return nil, status.Error(codes.FailedPrecondition, validationErr.Error())
	}
	return &pb.SkipQuestBulkResponse{DropReward: toProtoRewards(outcome.DropRewards), UserStatusCampaignReward: []*pb.QuestReward{}}, nil
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

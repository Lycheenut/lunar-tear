package service

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func (s *QuestServiceServer) StartEventQuest(ctx context.Context, req *pb.StartEventQuestRequest) (*pb.StartEventQuestResponse, error) {
	log.Printf("[QuestService] StartEventQuest: chapterId=%d questId=%d isBattleOnly=%v maxAutoOrbitCount=%d",
		req.EventQuestChapterId, req.QuestId, req.IsBattleOnly, req.MaxAutoOrbitCount)

	cat := s.holder.Get()
	engine := cat.QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var validationErr error
	var drops []masterdata.BattleDropInfo
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if err := validateLimitContentDeck(user, cat.LimitContent, req.EventQuestChapterId, req.UserDeckNumber, nowMillis); err != nil {
			validationErr = err
			return
		}
		if err := engine.HandleEventQuestStart(user, req.EventQuestChapterId, req.QuestId, req.IsBattleOnly, req.UserDeckNumber, nowMillis); err != nil {
			validationErr = status.Error(codes.FailedPrecondition, err.Error())
			return
		}
		startAutoOrbit(user, model.QuestTypeEvent, req.EventQuestChapterId, req.QuestId, req.MaxAutoOrbitCount, nowMillis)
		drops = engine.BattleDropRewards(user, req.QuestId)
	})
	if err != nil {
		return nil, fmt.Errorf("start event quest: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.StartEventQuestResponse{
		BattleDropReward: toProtoBattleDrops(drops),
	}, nil
}

func (s *QuestServiceServer) FinishEventQuest(ctx context.Context, req *pb.FinishEventQuestRequest) (*pb.FinishEventQuestResponse, error) {
	log.Printf("[QuestService] FinishEventQuest: chapterId=%d questId=%d isRetired=%v isAnnihilated=%v isAutoOrbit=%v",
		req.EventQuestChapterId, req.QuestId, req.IsRetired, req.IsAnnihilated, req.IsAutoOrbit)

	nowMillis := gametime.NowMillis()
	cat := s.holder.Get()
	engine := cat.QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var outcome questflow.FinishOutcome
	var endedDrops []store.AutoOrbitDropEntry
	var loopEnded bool
	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if err := engine.ValidateEventQuestContinuation(user, req.EventQuestChapterId, req.QuestId, nowMillis); err != nil {
			validationErr = status.Error(codes.FailedPrecondition, err.Error())
			return
		}
		deckNumber := user.Quests[req.QuestId].UserDeckNumber
		if !req.IsRetired && !req.IsAnnihilated {
			if err := validateLimitContentDeck(user, cat.LimitContent, req.EventQuestChapterId, deckNumber, nowMillis); err != nil {
				validationErr = err
				return
			}
		}
		outcome = engine.HandleEventQuestFinish(user, req.EventQuestChapterId, req.QuestId, req.IsRetired, req.IsAnnihilated, nowMillis)
		if !req.IsRetired && !req.IsAnnihilated {
			if err := recordLimitContentDeck(user, cat.LimitContent, req.EventQuestChapterId, req.QuestId, deckNumber, nowMillis); err != nil {
				validationErr = err
				return
			}
		}
		endedDrops, loopEnded = finishAutoOrbit(user, req.IsAutoOrbit, req.IsRetired, req.IsAnnihilated, model.QuestTypeEvent, req.EventQuestChapterId, req.QuestId, nowMillis, outcome.DropRewards)
	})
	if err != nil {
		return nil, fmt.Errorf("finish event quest: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	autoOrbitReward := emptyAutoOrbitReward()
	if loopEnded {
		autoOrbitReward.DropReward = autoOrbitDropsToProto(endedDrops)
	}

	return &pb.FinishEventQuestResponse{
		DropReward:                      toProtoRewards(outcome.DropRewards),
		FirstClearReward:                toProtoRewards(outcome.FirstClearRewards),
		MissionClearReward:              toProtoRewards(outcome.MissionClearRewards),
		MissionClearCompleteReward:      toProtoRewards(outcome.MissionClearCompleteRewards),
		AutoOrbitResult:                 []*pb.QuestReward{},
		IsBigWin:                        outcome.IsBigWin,
		BigWinClearedQuestMissionIdList: outcome.BigWinClearedQuestMissionIds,
		UserStatusCampaignReward:        []*pb.QuestReward{},
		AutoOrbitReward:                 autoOrbitReward,
	}, nil
}

type limitContentDeckTarget struct {
	possessionType int32
	uuid           string
}

func limitContentDeckTargets(user *store.UserState, deckNumber int32) ([]limitContentDeckTarget, error) {
	deck, ok := user.Decks[store.DeckKey{DeckType: model.DeckTypeRestrictedLimitContentQuest, UserDeckNumber: deckNumber}]
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "limit-content deck does not exist")
	}
	deckCharacterUuids := []string{deck.UserDeckCharacterUuid01, deck.UserDeckCharacterUuid02, deck.UserDeckCharacterUuid03}
	var targets []limitContentDeckTarget
	seen := make(map[string]bool)
	add := func(possessionType int32, targetUuid string) {
		key := fmt.Sprintf("%d:%s", possessionType, targetUuid)
		if targetUuid != "" && !seen[key] {
			seen[key] = true
			targets = append(targets, limitContentDeckTarget{possessionType, targetUuid})
		}
	}
	for _, deckCharacterUuid := range deckCharacterUuids {
		if deckCharacterUuid == "" {
			continue
		}
		character, ok := user.DeckCharacters[deckCharacterUuid]
		if !ok {
			return nil, status.Error(codes.FailedPrecondition, "limit-content deck contains an unknown character")
		}
		if character.UserCostumeUuid == "" {
			return nil, status.Error(codes.FailedPrecondition, "limit-content deck character has no costume")
		}
		if _, ok := user.Costumes[character.UserCostumeUuid]; !ok {
			return nil, status.Error(codes.FailedPrecondition, "limit-content deck contains an unknown costume")
		}
		if character.MainUserWeaponUuid == "" {
			return nil, status.Error(codes.FailedPrecondition, "limit-content deck character has no main weapon")
		}
		if _, ok := user.Weapons[character.MainUserWeaponUuid]; !ok {
			return nil, status.Error(codes.FailedPrecondition, "limit-content deck contains an unknown main weapon")
		}
		add(int32(model.PossessionTypeCostume), character.UserCostumeUuid)
		add(int32(model.PossessionTypeWeapon), character.MainUserWeaponUuid)
		for _, weaponUuid := range user.DeckSubWeapons[deckCharacterUuid] {
			if _, ok := user.Weapons[weaponUuid]; !ok {
				return nil, status.Error(codes.FailedPrecondition, "limit-content deck contains an unknown sub weapon")
			}
			add(int32(model.PossessionTypeWeapon), weaponUuid)
		}
	}
	if len(targets) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "limit-content deck is empty")
	}
	return targets, nil
}

func restrictionPossessionType(restrictionType int32) int32 {
	if restrictionType == 1 {
		return int32(model.PossessionTypeCostume)
	}
	if restrictionType == 2 {
		return int32(model.PossessionTypeWeapon)
	}
	return 0
}

func validateLimitContentDeck(user *store.UserState, catalog *masterdata.LimitContentCatalog, chapterId, deckNumber int32, nowMillis int64) error {
	if catalog == nil {
		return nil
	}
	restrictedTypes := catalog.ActiveRestrictionTypes(chapterId, nowMillis)
	if len(restrictedTypes) == 0 {
		return nil
	}
	targets, err := limitContentDeckTargets(user, deckNumber)
	if err != nil {
		return err
	}
	for _, restrictionType := range restrictedTypes {
		possessionType := restrictionPossessionType(restrictionType)
		for _, target := range targets {
			if target.possessionType != possessionType {
				continue
			}
			for _, used := range user.DeckLimitContentRestricted {
				if used.EventQuestChapterId == chapterId && used.PossessionType == possessionType && used.TargetUuid == target.uuid {
					return status.Error(codes.FailedPrecondition, "deck contains content already used in this limit quest")
				}
			}
		}
	}
	return nil
}

func recordLimitContentDeck(user *store.UserState, catalog *masterdata.LimitContentCatalog, chapterId, questId, deckNumber int32, nowMillis int64) error {
	if catalog == nil {
		return nil
	}
	restrictionTypes := catalog.ActiveRestrictionTypes(chapterId, nowMillis)
	if len(restrictionTypes) == 0 {
		return nil
	}
	targets, err := limitContentDeckTargets(user, deckNumber)
	if err != nil {
		return err
	}
	for _, restrictionType := range restrictionTypes {
		possessionType := restrictionPossessionType(restrictionType)
		for _, target := range targets {
			if target.possessionType != possessionType {
				continue
			}
			alreadyRecorded := false
			for _, used := range user.DeckLimitContentRestricted {
				if used.EventQuestChapterId == chapterId && used.PossessionType == possessionType && used.TargetUuid == target.uuid {
					alreadyRecorded = true
					break
				}
			}
			if alreadyRecorded {
				continue
			}
			id := uuid.NewString()
			user.DeckLimitContentRestricted[id] = store.DeckLimitContentRestrictedState{DeckRestrictedUuid: id, EventQuestChapterId: chapterId, QuestId: questId, PossessionType: possessionType, TargetUuid: target.uuid, LatestVersion: nowMillis}
		}
	}
	return nil
}

func (s *QuestServiceServer) RestartEventQuest(ctx context.Context, req *pb.RestartEventQuestRequest) (*pb.RestartEventQuestResponse, error) {
	log.Printf("[QuestService] RestartEventQuest: chapterId=%d questId=%d", req.EventQuestChapterId, req.QuestId)

	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var validationErr error
	var battleBinary []byte
	var deckNumber int32
	var drops []masterdata.BattleDropInfo
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		if err := engine.ValidateEventQuestContinuation(user, req.EventQuestChapterId, req.QuestId, nowMillis); err != nil {
			validationErr = err
			return
		}
		engine.HandleEventQuestRestart(user, req.EventQuestChapterId, req.QuestId, nowMillis)
		battleBinary = battleCheckpoint(user)
		deckNumber = user.Quests[req.QuestId].UserDeckNumber
		drops = engine.BattleDropRewards(user, req.QuestId)
	})
	if updateErr != nil {
		return nil, fmt.Errorf("restart event quest: %w", updateErr)
	}
	if validationErr != nil {
		return nil, status.Error(codes.FailedPrecondition, validationErr.Error())
	}

	return &pb.RestartEventQuestResponse{
		BattleDropReward: toProtoBattleDrops(drops),
		BattleBinary:     battleBinary,
		DeckNumber:       deckNumber,
	}, nil
}

func (s *QuestServiceServer) UpdateEventQuestSceneProgress(ctx context.Context, req *pb.UpdateEventQuestSceneProgressRequest) (*pb.UpdateEventQuestSceneProgressResponse, error) {
	log.Printf("[QuestService] UpdateEventQuestSceneProgress: questSceneId=%d", req.QuestSceneId)

	engine := s.holder.Get().QuestHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		engine.HandleEventQuestSceneProgress(user, req.QuestSceneId, gametime.NowMillis())
	})

	return &pb.UpdateEventQuestSceneProgressResponse{}, nil
}

const defaultGuerrillaFreeOpenMinutes = int32(60)

func (s *QuestServiceServer) StartGuerrillaFreeOpen(ctx context.Context, req *emptypb.Empty) (*pb.StartGuerrillaFreeOpenResponse, error) {
	log.Printf("[QuestService] StartGuerrillaFreeOpen")

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	s.users.UpdateUser(userId, func(user *store.UserState) {
		user.GuerrillaFreeOpen.StartDatetime = nowMillis
		user.GuerrillaFreeOpen.OpenMinutes = defaultGuerrillaFreeOpenMinutes
		user.GuerrillaFreeOpen.DailyOpenedCount++
		user.GuerrillaFreeOpen.LatestVersion = nowMillis
	})

	return &pb.StartGuerrillaFreeOpenResponse{}, nil
}

func (s *QuestServiceServer) ReceiveDailyQuestGroupCompleteReward(ctx context.Context, _ *emptypb.Empty) (*pb.ReceiveDailyQuestGroupCompleteRewardResponse, error) {
	log.Printf("[QuestService] ReceiveDailyQuestGroupCompleteReward")
	cat := s.holder.Get()
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	today := gametime.StartOfBusinessDayMillis()
	var validationErr error
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		var active *masterdata.EventQuestDailyGroup
		for i := range cat.Quest.EventDailyGroups {
			group := &cat.Quest.EventDailyGroups[i]
			if nowMillis >= group.Definition.StartDatetime && (group.Definition.EndDatetime == 0 || nowMillis < group.Definition.EndDatetime) {
				active = group
				break
			}
		}
		if active == nil {
			validationErr = status.Error(codes.FailedPrecondition, "no active daily quest group")
			return
		}
		if received := user.EventQuestDailyRewards[active.Definition.EventQuestDailyGroupId]; received.RewardReceiveDatetime >= today {
			validationErr = status.Error(codes.AlreadyExists, "daily quest group reward already received")
			return
		}
		for _, chapterId := range active.ChapterIds {
			for _, questId := range cat.Quest.EventQuestIdsByChapterId[chapterId] {
				quest := user.Quests[questId]
				if quest.QuestStateType != model.UserQuestStateTypeCleared || quest.LastClearDatetime < today {
					validationErr = status.Error(codes.FailedPrecondition, "daily quest group is incomplete")
					return
				}
			}
		}
		for _, reward := range active.Rewards {
			cat.QuestHandler.Granter.GrantFull(user, model.PossessionType(reward.PossessionType), reward.PossessionId, reward.Count, nowMillis)
		}
		id := active.Definition.EventQuestDailyGroupId
		user.EventQuestDailyRewards[id] = store.EventQuestDailyRewardState{EventQuestDailyGroupId: id, RewardReceiveDatetime: nowMillis, LatestVersion: nowMillis}
	})
	if updateErr != nil {
		return nil, fmt.Errorf("receive daily quest group reward: %w", updateErr)
	}
	if validationErr != nil {
		return nil, validationErr
	}
	return &pb.ReceiveDailyQuestGroupCompleteRewardResponse{}, nil
}

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
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type GimmickServiceServer struct {
	pb.UnimplementedGimmickServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewGimmickServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *GimmickServiceServer {
	return &GimmickServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *GimmickServiceServer) UpdateSequence(ctx context.Context, req *pb.UpdateSequenceRequest) (*pb.UpdateSequenceResponse, error) {
	log.Printf("[GimmickService] UpdateSequence: scheduleId=%d sequenceId=%d",
		req.GimmickSequenceScheduleId, req.GimmickSequenceId)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if !s.holder.Get().Gimmick.SequenceAvailable(user, req.GimmickSequenceScheduleId, req.GimmickSequenceId, gametime.NowMillis()) {
			validationErr = status.Error(codes.FailedPrecondition, "gimmick sequence is not available")
			return
		}
		key := store.GimmickSequenceKey{
			GimmickSequenceScheduleId: req.GimmickSequenceScheduleId,
			GimmickSequenceId:         req.GimmickSequenceId,
		}
		sequence := user.Gimmick.Sequences[key]
		sequence.Key = key
		user.Gimmick.Sequences[key] = sequence
	})
	if err != nil {
		return nil, fmt.Errorf("update gimmick sequence: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}
	return &pb.UpdateSequenceResponse{}, nil
}

func (s *GimmickServiceServer) UpdateGimmickProgress(ctx context.Context, req *pb.UpdateGimmickProgressRequest) (*pb.UpdateGimmickProgressResponse, error) {
	log.Printf("[GimmickService] UpdateGimmickProgress: scheduleId=%d sequenceId=%d gimmickId=%d ornamentIndex=%d progressValueBit=%d flowType=%d",
		req.GimmickSequenceScheduleId, req.GimmickSequenceId, req.GimmickId, req.GimmickOrnamentIndex, req.ProgressValueBit, req.FlowType)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	cat := s.holder.Get()

	var ornamentRewards []*pb.GimmickReward
	var sequenceCleared bool
	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		nowMillis := gametime.NowMillis()
		if !cat.Gimmick.GimmickAvailable(user, req.GimmickSequenceScheduleId, req.GimmickSequenceId, req.GimmickId, nowMillis) {
			validationErr = status.Error(codes.FailedPrecondition, "gimmick is not available")
			return
		}
		progressKey := store.GimmickKey{
			GimmickSequenceScheduleId: req.GimmickSequenceScheduleId,
			GimmickSequenceId:         req.GimmickSequenceId,
			GimmickId:                 req.GimmickId,
		}
		progress := user.Gimmick.Progress[progressKey]
		progress.Key = progressKey
		progress.StartDatetime = nowMillis

		ornamentKey := store.GimmickOrnamentKey{
			GimmickSequenceScheduleId: req.GimmickSequenceScheduleId,
			GimmickSequenceId:         req.GimmickSequenceId,
			GimmickId:                 req.GimmickId,
			GimmickOrnamentIndex:      req.GimmickOrnamentIndex,
		}
		_, ornamentAlreadyProcessed := user.Gimmick.OrnamentProgress[ornamentKey]
		ornament := user.Gimmick.OrnamentProgress[ornamentKey]
		ornament.Key = ornamentKey
		ornament.ProgressValueBit = req.ProgressValueBit
		ornament.BaseDatetime = nowMillis
		ornament.LatestVersion = nowMillis
		persistOrnamentProgress := true

		// Per-type branches:
		//   * Report (type 9, "Hidden Stories")            — mark gimmick + sequence
		//     cleared, grant SequenceRewards (ImportantItem type 3, library reads it).
		//   * MapOnlyCageTreasureHunt (type 7, "Hidden Black Birds") — same as Report
		//     but the per-tap reward also comes back from m_cage_ornament_reward via
		//     GimmickOrnamentViewId.
		//   * CageMemory (type 10, "Lost Archives")        — resolve an ImportantItem
		//     (type 4) from the gimmick's monitor texture and grant it. IsGimmickCleared
		//     stays false (matches original userdata; only ornament progress flips).
		//   * CageTreasureHunt                          — use the ornament reward map.
		//   * CageIntervalDropItem*                    — record interval progress only;
		//     the available interval table has no possession mapping.
		switch cat.Gimmick.GimmickType(req.GimmickId) {
		case model.GimmickTypeReport:
			progress.IsGimmickCleared = true
			sequenceCleared = markSequenceClearedOnce(user, cat, req.GimmickSequenceScheduleId, req.GimmickSequenceId, nowMillis)

		case model.GimmickTypeCageTreasureHunt, model.GimmickTypeMapOnlyCageTreasureHunt:
			if ornamentAlreadyProcessed {
				persistOrnamentProgress = false
				break
			}
			r, ok := cat.Gimmick.OrnamentReward(req.GimmickId, req.GimmickOrnamentIndex)
			if !ok {
				persistOrnamentProgress = false
				log.Printf("[GimmickService] UpdateGimmickProgress: hidden-bird %d ornament %d has no reward mapping, skipping",
					req.GimmickId, req.GimmickOrnamentIndex)
				break
			}
			cat.QuestHandler.Granter.GrantFull(user, model.PossessionType(r.PossessionType), r.PossessionId, r.Count, nowMillis)
			ornamentRewards = append(ornamentRewards, &pb.GimmickReward{
				PossessionType: r.PossessionType,
				PossessionId:   r.PossessionId,
				Count:          r.Count,
			})
			progress.IsGimmickCleared = true
			sequenceCleared = markSequenceClearedOnce(user, cat, req.GimmickSequenceScheduleId, req.GimmickSequenceId, nowMillis)

		case model.GimmickTypeCageMemory:
			itemId, ok := cat.Gimmick.CageMemoryImportantItem(req.GimmickId)
			if !ok {
				log.Printf("[GimmickService] UpdateGimmickProgress: cage memory %d has no important-item mapping, skipping grant",
					req.GimmickId)
				break
			}
			if _, owned := user.ImportantItems[itemId]; owned {
				break
			}
			cat.QuestHandler.Granter.GrantFull(user, model.PossessionTypeImportantItem, itemId, 1, nowMillis)
			ornamentRewards = append(ornamentRewards, &pb.GimmickReward{
				PossessionType: int32(model.PossessionTypeImportantItem),
				PossessionId:   itemId,
				Count:          1,
			})

		case model.GimmickTypeCageIntervalDropItem,
			model.GimmickTypeMapOnlyCageIntervalDrop:
			// The interval table controls timing only and has no possession mapping.
			// Persist progress but do not fabricate a material reward.
		}
		if persistOrnamentProgress {
			user.Gimmick.OrnamentProgress[ornamentKey] = ornament
		}
		progress.LatestVersion = nowMillis
		user.Gimmick.Progress[progressKey] = progress
	})
	if err != nil {
		return nil, fmt.Errorf("update gimmick progress: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	var clearReward []*pb.GimmickReward
	if sequenceCleared {
		for _, r := range cat.Gimmick.SequenceRewards(req.GimmickSequenceId) {
			clearReward = append(clearReward, &pb.GimmickReward{
				PossessionType: r.PossessionType,
				PossessionId:   r.PossessionId,
				Count:          r.Count,
			})
		}
	}
	return &pb.UpdateGimmickProgressResponse{
		GimmickOrnamentReward:      ornamentRewards,
		IsSequenceCleared:          sequenceCleared,
		GimmickSequenceClearReward: clearReward,
	}, nil
}

func markSequenceClearedOnce(user *store.UserState, cat *runtime.Catalogs, scheduleId, sequenceId int32, nowMillis int64) bool {
	seqKey := store.GimmickSequenceKey{
		GimmickSequenceScheduleId: scheduleId,
		GimmickSequenceId:         sequenceId,
	}
	sequence := user.Gimmick.Sequences[seqKey]
	sequence.Key = seqKey
	defer func() { user.Gimmick.Sequences[seqKey] = sequence }()

	if sequence.IsGimmickSequenceCleared {
		return false
	}
	sequence.IsGimmickSequenceCleared = true
	sequence.ClearDatetime = nowMillis
	for _, r := range cat.Gimmick.SequenceRewards(sequenceId) {
		cat.QuestHandler.Granter.GrantFull(user, model.PossessionType(r.PossessionType), r.PossessionId, r.Count, nowMillis)
	}
	return true
}

func (s *GimmickServiceServer) InitSequenceSchedule(ctx context.Context, _ *emptypb.Empty) (*pb.InitSequenceScheduleResponse, error) {
	log.Printf("[GimmickService] InitSequenceSchedule")
	userId := CurrentUserId(ctx, s.users, s.sessions)
	now := gametime.NowMillis()
	s.users.UpdateUser(userId, func(user *store.UserState) {
		eligible := s.holder.Get().Gimmick.ActiveScheduleKeys(*user, now)
		eligibleSet := make(map[store.GimmickSequenceKey]struct{}, len(eligible))
		for _, key := range eligible {
			eligibleSet[key] = struct{}{}
		}
		pruned := 0
		for key, entry := range user.Gimmick.Sequences {
			if _, ok := eligibleSet[key]; ok {
				continue
			}
			if entry.IsGimmickSequenceCleared {
				continue
			}
			delete(user.Gimmick.Sequences, key)
			pruned++
		}

		added := 0
		for _, key := range eligible {
			if len(user.Gimmick.Sequences) >= masterdata.MaxUserGimmickRows {
				break
			}
			if _, exists := user.Gimmick.Sequences[key]; !exists {
				user.Gimmick.Sequences[key] = store.GimmickSequenceState{Key: key}
				added++
			}
		}
		if pruned > 0 || added > 0 {
			log.Printf("[GimmickService] InitSequenceSchedule: pruned %d stale, added %d sequences (total %d, eligible %d, cap %d)",
				pruned, added, len(user.Gimmick.Sequences), len(eligible), masterdata.MaxUserGimmickRows)
		}
	})
	return &pb.InitSequenceScheduleResponse{}, nil
}

func (s *GimmickServiceServer) Unlock(ctx context.Context, req *pb.UnlockRequest) (*pb.UnlockResponse, error) {
	log.Printf("[GimmickService] Unlock: gimmickKeys=%d", len(req.GimmickKey))
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		nowMillis := gametime.NowMillis()
		for _, item := range req.GimmickKey {
			if !s.holder.Get().Gimmick.GimmickAvailable(user, item.GimmickSequenceScheduleId, item.GimmickSequenceId, item.GimmickId, nowMillis) {
				validationErr = status.Error(codes.FailedPrecondition, "gimmick is not available")
				return
			}
		}
		for _, item := range req.GimmickKey {
			key := store.GimmickKey{
				GimmickSequenceScheduleId: item.GimmickSequenceScheduleId,
				GimmickSequenceId:         item.GimmickSequenceId,
				GimmickId:                 item.GimmickId,
			}
			unlock := user.Gimmick.Unlocks[key]
			unlock.Key = key
			unlock.IsUnlocked = true
			user.Gimmick.Unlocks[key] = unlock
		}
	})
	if err != nil {
		return nil, fmt.Errorf("unlock gimmick: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}
	return &pb.UnlockResponse{}, nil
}

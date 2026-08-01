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

type MissionServiceServer struct {
	pb.UnimplementedMissionServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewMissionServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *MissionServiceServer {
	return &MissionServiceServer{users: users, sessions: sessions, holder: holder}
}

func missionIsActive(c *masterdata.MissionCatalog, mission masterdata.EntityMMission, nowMillis int64) bool {
	if mission.MissionTermId == 0 {
		return true
	}
	term, ok := c.TermById[mission.MissionTermId]
	return ok && nowMillis >= term.StartDatetime && (term.EndDatetime == 0 || nowMillis < term.EndDatetime)
}

func missionUnlocked(c *masterdata.MissionCatalog, user *store.UserState, mission masterdata.EntityMMission) bool {
	if mission.MissionUnlockConditionId == 0 {
		return true
	}
	condition, ok := c.UnlockById[mission.MissionUnlockConditionId]
	if !ok {
		return false
	}
	switch condition.MissionUnlockConditionType {
	case 0, 1, 6:
		return true
	case 2:
		return user.Quests[condition.ConditionValue].QuestStateType == model.UserQuestStateTypeCleared
	case 5:
		return user.Status.Level >= condition.ConditionValue
	default:
		return false
	}
}

func syncMissionProgress(c *masterdata.MissionCatalog, user *store.UserState, req *pb.UpdateMissionProgressRequest, nowMillis int64) error {
	measuredByType := make(map[int32]int32)
	if req.CageMeasurableValues != nil {
		measuredByType[37] = req.CageMeasurableValues.RunningDistanceMeters
		measuredByType[38] = req.CageMeasurableValues.MamaTappedCount
	}
	if req.PictureBookMeasurableValues != nil {
		measuredByType[39] = req.PictureBookMeasurableValues.DefeatWizardCount
		if rhythm := req.PictureBookMeasurableValues.RhythmInteractionMeasurableValues; rhythm != nil {
			if rhythm.LiveTypeId <= 0 {
				return status.Error(codes.InvalidArgument, "rhythm live type must be positive")
			}
			measuredByType[36] = rhythm.TapCount
		}
	}
	for missionType, measured := range measuredByType {
		if measured < 0 {
			return status.Errorf(codes.InvalidArgument, "mission metric type %d must not be negative", missionType)
		}
	}

	for missionType, measured := range measuredByType {
		for _, id := range c.MeasurableMissionIdsByType[missionType] {
			mission := c.MissionById[id]
			if !missionIsActive(c, mission, nowMillis) || !missionUnlocked(c, user, mission) {
				continue
			}
			state := user.Missions[id]
			if state.MissionId == 0 {
				state = store.UserMissionState{MissionId: id, StartDatetime: nowMillis, MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress), LatestVersion: nowMillis}
			}
			if state.MissionProgressStatusType >= int32(model.MissionProgressStatusTypeClear) {
				user.Missions[id] = state
				continue
			}
			if measured > state.ProgressValue {
				state.ProgressValue = measured
				state.LatestVersion = nowMillis
			}
			if state.ProgressValue >= mission.ClearConditionValue {
				state.MissionProgressStatusType = int32(model.MissionProgressStatusTypeClear)
				state.ClearDatetime = nowMillis
				state.LatestVersion = nowMillis
			}
			user.Missions[id] = state
		}
	}
	return nil
}

func (s *MissionServiceServer) UpdateMissionProgress(ctx context.Context, req *pb.UpdateMissionProgressRequest) (*pb.UpdateMissionProgressResponse, error) {
	log.Printf("[MissionService] UpdateMissionProgress: cage=%v pictureBook=%v", req.CageMeasurableValues, req.PictureBookMeasurableValues)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		validationErr = syncMissionProgress(s.holder.Get().Mission, user, req, gametime.NowMillis())
	})
	if err != nil {
		return nil, fmt.Errorf("update mission progress: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}
	return &pb.UpdateMissionProgressResponse{}, nil
}

func grantMissionPossession(cat *runtime.Catalogs, user *store.UserState, possessionType, possessionId, count int32, nowMillis int64) {
	if model.PossessionType(possessionType) == model.PossessionTypeMissionPassPoint {
		state := user.MissionPassPoints[possessionId]
		state.MissionPassId = possessionId
		state.Point += count
		state.LatestVersion = nowMillis
		user.MissionPassPoints[possessionId] = state
		return
	}
	cat.QuestHandler.Granter.GrantFull(user, model.PossessionType(possessionType), possessionId, count, nowMillis)
}

func missionExpired(c *masterdata.MissionCatalog, mission masterdata.EntityMMission, nowMillis int64) bool {
	term, ok := c.TermById[mission.MissionTermId]
	if !ok || term.EndDatetime == 0 {
		return false
	}
	return nowMillis >= term.EndDatetime+int64(mission.MinExpirationDays)*int64(24*time.Hour/time.Millisecond)
}

func missionPassActive(pass masterdata.EntityMMissionPass, nowMillis int64) bool {
	return nowMillis >= pass.StartDatetime && (pass.EndDatetime == 0 || nowMillis < pass.EndDatetime)
}

func missionPassEnded(pass masterdata.EntityMMissionPass, nowMillis int64) bool {
	return pass.EndDatetime > 0 && nowMillis >= pass.EndDatetime
}

func (s *MissionServiceServer) ReceiveMissionRewardsById(ctx context.Context, req *pb.ReceiveMissionRewardsByIdRequest) (*pb.ReceiveMissionRewardsResponse, error) {
	cat := s.holder.Get()
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var received, expired []*pb.MissionReward
	var validationErr error
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		seen := make(map[int32]bool)
		for _, id := range req.MissionId {
			if seen[id] {
				validationErr = status.Errorf(codes.InvalidArgument, "duplicate mission %d", id)
				return
			}
			seen[id] = true
			_, ok := cat.Mission.MissionById[id]
			if !ok {
				validationErr = status.Errorf(codes.InvalidArgument, "unknown mission %d", id)
				return
			}
			state := user.Missions[id]
			if state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
				validationErr = status.Errorf(codes.FailedPrecondition, "mission %d is not claimable", id)
				return
			}
		}
		if validationErr != nil {
			return
		}
		for _, id := range req.MissionId {
			mission := cat.Mission.MissionById[id]
			state := user.Missions[id]
			isExpired := missionExpired(cat.Mission, mission, nowMillis)
			for _, reward := range cat.Mission.RewardsById[mission.MissionRewardId] {
				row := &pb.MissionReward{PossessionType: reward.PossessionType, PossessionId: reward.PossessionId, Count: reward.Count}
				if isExpired {
					expired = append(expired, row)
				} else {
					grantMissionPossession(cat, user, reward.PossessionType, reward.PossessionId, reward.Count, nowMillis)
					received = append(received, row)
				}
			}
			state.MissionProgressStatusType = int32(model.MissionProgressStatusTypeRewardReceived)
			state.LatestVersion = nowMillis
			user.Missions[id] = state
		}
	})
	if updateErr != nil {
		return nil, fmt.Errorf("receive mission rewards: %w", updateErr)
	}
	if validationErr != nil {
		return nil, validationErr
	}
	return &pb.ReceiveMissionRewardsResponse{ReceivedPossession: received, ExpiredPossession: expired, OverflowPossession: []*pb.MissionReward{}}, nil
}

func claimMissionPassRewards(cat *runtime.Catalogs, user *store.UserState, passId int32, nowMillis int64, allowEnded bool) ([]*pb.MissionPassReward, error) {
	pass, ok := cat.Mission.PassById[passId]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "unknown mission pass")
	}
	if !allowEnded && !missionPassActive(pass.Definition, nowMillis) {
		return nil, status.Error(codes.FailedPrecondition, "mission pass is inactive")
	}
	point := user.MissionPassPoints[passId].Point
	necessary := make(map[int32]int32)
	for _, level := range pass.Levels {
		necessary[level.Level] = level.NecessaryPoint
	}
	claimedKeys := make(map[store.MissionPassRewardKey]bool)
	var received []*pb.MissionPassReward
	for _, reward := range pass.Rewards {
		key := store.MissionPassRewardKey{MissionPassId: passId, Level: reward.Level, IsPremium: reward.IsPremium}
		if _, claimed := user.MissionPassRewards[key]; claimed || point < necessary[reward.Level] {
			continue
		}
		if reward.IsPremium && user.PremiumItems[pass.Definition.PremiumItemId] == 0 {
			continue
		}
		grantMissionPossession(cat, user, reward.PossessionType, reward.PossessionId, reward.Count, nowMillis)
		received = append(received, &pb.MissionPassReward{PossessionType: reward.PossessionType, PossessionId: reward.PossessionId, Count: reward.Count})
		claimedKeys[key] = true
	}
	for key := range claimedKeys {
		user.MissionPassRewards[key] = store.MissionPassRewardState{MissionPassId: key.MissionPassId, Level: key.Level, IsPremium: key.IsPremium, RewardReceiveDatetime: nowMillis, LatestVersion: nowMillis}
	}
	return received, nil
}

func (s *MissionServiceServer) ReceiveMissionPassRewards(ctx context.Context, req *pb.ReceiveMissionPassRewardsRequest) (*pb.ReceiveMissionPassRewardsResponse, error) {
	cat := s.holder.Get()
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var received []*pb.MissionPassReward
	var claimErr error
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		received, claimErr = claimMissionPassRewards(cat, user, req.MissionPassId, nowMillis, false)
	})
	if updateErr != nil {
		return nil, fmt.Errorf("receive mission pass rewards: %w", updateErr)
	}
	if claimErr != nil {
		return nil, claimErr
	}
	return &pb.ReceiveMissionPassRewardsResponse{ReceivedPossession: received, OverflowPossession: []*pb.MissionPassReward{}}, nil
}

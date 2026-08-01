package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

type CageOrnamentServiceServer struct {
	pb.UnimplementedCageOrnamentServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewCageOrnamentServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *CageOrnamentServiceServer {
	return &CageOrnamentServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *CageOrnamentServiceServer) ReceiveReward(ctx context.Context, req *pb.ReceiveRewardRequest) (*pb.ReceiveRewardResponse, error) {
	log.Printf("[CageOrnamentService] ReceiveReward: cageOrnamentId=%d", req.CageOrnamentId)

	cat := s.holder.Get()
	reward, ok := cat.CageOrnament.LookupReward(req.CageOrnamentId)
	if !ok {
		log.Printf("[CageOrnamentService] ReceiveReward: no reward mapping for cageOrnamentId=%d, returning empty",
			req.CageOrnamentId)
		return &pb.ReceiveRewardResponse{}, nil
	}

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	granted := false
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		granted = claimCageOrnamentReward(user, req.CageOrnamentId, reward, cat.QuestHandler.Granter, nowMillis)
	})
	if err != nil {
		return nil, fmt.Errorf("receive cage ornament reward: %w", err)
	}

	if !granted {
		return &pb.ReceiveRewardResponse{}, nil
	}

	return &pb.ReceiveRewardResponse{
		CageOrnamentReward: []*pb.CageOrnamentReward{
			{
				PossessionType: reward.PossessionType,
				PossessionId:   reward.PossessionId,
				Count:          reward.Count,
			},
		},
	}, nil
}

func claimCageOrnamentReward(user *store.UserState, cageOrnamentId int32, reward masterdata.CageOrnamentReward, granter *store.PossessionGranter, nowMillis int64) bool {
	if _, exists := user.CageOrnamentRewards[cageOrnamentId]; exists {
		return false
	}
	user.CageOrnamentRewards[cageOrnamentId] = store.CageOrnamentRewardState{CageOrnamentId: cageOrnamentId, AcquisitionDatetime: nowMillis, LatestVersion: nowMillis}
	granter.GrantFull(user, model.PossessionType(reward.PossessionType), reward.PossessionId, reward.Count, nowMillis)
	return true
}

func (s *CageOrnamentServiceServer) RecordAccess(ctx context.Context, req *pb.RecordAccessRequest) (*pb.RecordAccessResponse, error) {
	log.Printf("[CageOrnamentService] RecordAccess: cageOrnamentId=%d", req.CageOrnamentId)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		access := user.CageOrnamentAccesses[req.CageOrnamentId]
		access.CageOrnamentId = req.CageOrnamentId
		if access.FirstAccessDatetime == 0 {
			access.FirstAccessDatetime = nowMillis
		}
		access.LatestAccessDatetime = nowMillis
		access.LatestVersion = nowMillis
		user.CageOrnamentAccesses[req.CageOrnamentId] = access
	})
	if err != nil {
		return nil, fmt.Errorf("record cage ornament access: %w", err)
	}
	return &pb.RecordAccessResponse{}, nil
}

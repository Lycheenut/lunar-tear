package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"

	"google.golang.org/protobuf/types/known/emptypb"
)

// PvpServiceServer keeps the local Arena implementation intentionally small,
// but records the result semantics needed by every PVP mission condition.
type PvpServiceServer struct {
	pb.UnimplementedPvpServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
}

func NewPvpServiceServer(users store.UserRepository, sessions store.SessionRepository) *PvpServiceServer {
	return &PvpServiceServer{users: users, sessions: sessions}
}

func (s *PvpServiceServer) GetTopData(ctx context.Context, _ *emptypb.Empty) (*pb.GetTopDataResponse, error) {
	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("load PVP user: %w", err)
	}
	return &pb.GetTopDataResponse{Rank: user.Profile.CurrentPvpRank}, nil
}

func (s *PvpServiceServer) GetMatchingList(context.Context, *emptypb.Empty) (*pb.GetMatchingListResponse, error) {
	return &pb.GetMatchingListResponse{Matching: []*pb.MatchingOpponent{}}, nil
}

func (s *PvpServiceServer) UpdateMatchingList(context.Context, *emptypb.Empty) (*pb.UpdateMatchingListResponse, error) {
	return &pb.UpdateMatchingListResponse{Matching: []*pb.MatchingOpponent{}}, nil
}

func (s *PvpServiceServer) StartBattle(context.Context, *pb.StartBattleRequest) (*pb.StartBattleResponse, error) {
	return &pb.StartBattleResponse{OpponentDeckCharacter: []*pb.PvpDeckCharacter{}}, nil
}

func (s *PvpServiceServer) FinishBattle(ctx context.Context, req *pb.FinishBattleRequest) (*pb.FinishBattleResponse, error) {
	log.Printf("[PvpService] FinishBattle: opponentPlayerId=%d victory=%v", req.OpponentPlayerId, req.IsVictory)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var beforeRank, afterRank int32
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		beforeRank = user.Profile.CurrentPvpRank
		if user.Profile.CurrentPvpRank == 0 {
			// A single-player local server has one ranked participant.
			user.Profile.CurrentPvpRank = 1
		}
		afterRank = user.Profile.CurrentPvpRank

		recordPvpMissionEvents(user, req.IsVictory)
	})
	if err != nil {
		return nil, fmt.Errorf("finish PVP battle: %w", err)
	}
	return &pb.FinishBattleResponse{BeforeRank: beforeRank, AfterRank: afterRank}, nil
}

func recordPvpMissionEvents(user *store.UserState, victory bool) {
	store.AddMissionCount(user, int32(model.MissionClearConditionTypePvpFinishByCount), 1, 0, 0)
	if victory {
		store.AddMissionCount(user, int32(model.MissionClearConditionTypePvpFinishByWinCount), 1, 0, 0)
		store.AddMissionCount(user, int32(model.MissionClearConditionTypePvpFinishByWinStreakCount), 1, 0, 0)
		store.AddMissionCount(user, int32(model.MissionClearConditionTypePvpFinishByWinStreakCountFromUnlock), 1, 0, 0)
		return
	}
	store.ResetMissionValue(user, int32(model.MissionClearConditionTypePvpFinishByWinStreakCount), 0, 0)
	store.ResetMissionValue(user, int32(model.MissionClearConditionTypePvpFinishByWinStreakCountFromUnlock), 0, 0)
}

func (s *PvpServiceServer) GetRanking(context.Context, *pb.GetRankingRequest) (*pb.GetRankingResponse, error) {
	return &pb.GetRankingResponse{RankingUser: []*pb.RankingUser{}}, nil
}

func (s *PvpServiceServer) GetSeasonResult(context.Context, *emptypb.Empty) (*pb.GetSeasonResultResponse, error) {
	return &pb.GetSeasonResultResponse{}, nil
}

func (s *PvpServiceServer) GetAttackLogList(context.Context, *emptypb.Empty) (*pb.GetAttackLogListResponse, error) {
	return &pb.GetAttackLogListResponse{AttackLog: []*pb.BattleLog{}}, nil
}

func (s *PvpServiceServer) GetDefenseLogList(context.Context, *emptypb.Empty) (*pb.GetDefenseLogListResponse, error) {
	return &pb.GetDefenseLogListResponse{DefenseLog: []*pb.BattleLog{}}, nil
}

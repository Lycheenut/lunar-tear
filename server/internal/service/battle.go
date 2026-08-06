package service

import (
	"bytes"
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/store"
)

type BattleServiceServer struct {
	pb.UnimplementedBattleServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
}

func NewBattleServiceServer(users store.UserRepository, sessions store.SessionRepository) *BattleServiceServer {
	return &BattleServiceServer{users: users, sessions: sessions}
}

func (s *BattleServiceServer) StartWave(ctx context.Context, req *pb.StartWaveRequest) (*pb.StartWaveResponse, error) {
	log.Printf("[BattleService] StartWave: userParty=%d npcParty=%d", len(req.UserPartyInitialInfoList), len(req.NpcPartyInitialInfoList))
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		user.Battle = store.BattleState{
			IsActive:           true,
			StartCount:         user.Battle.StartCount + 1,
			FinishCount:        user.Battle.FinishCount,
			LastStartedAt:      gametime.NowMillis(),
			LastFinishedAt:     user.Battle.LastFinishedAt,
			LastUserPartyCount: int32(len(req.UserPartyInitialInfoList)),
			LastNpcPartyCount:  int32(len(req.NpcPartyInitialInfoList)),
			MissionDetail:      store.BattleMissionDetailState{},
		}
	})
	return &pb.StartWaveResponse{}, nil
}

func (s *BattleServiceServer) FinishWave(ctx context.Context, req *pb.FinishWaveRequest) (*pb.FinishWaveResponse, error) {
	log.Printf("[BattleService] FinishWave: battleBinary=%d userParty=%d npcParty=%d elapsedFrames=%d",
		len(req.BattleBinary), len(req.UserPartyResultInfoList), len(req.NpcPartyResultInfoList), req.ElapsedFrameCount)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		user.Battle.IsActive = false
		user.Battle.FinishCount++
		user.Battle.LastFinishedAt = gametime.NowMillis()
		user.Battle.LastUserPartyCount = int32(len(req.UserPartyResultInfoList))
		user.Battle.LastNpcPartyCount = int32(len(req.NpcPartyResultInfoList))
		user.Battle.LastBattleBinarySize = int32(len(req.BattleBinary))
		user.BattleBinary = bytes.Clone(req.BattleBinary)
		user.Battle.LastElapsedFrameCount = req.ElapsedFrameCount
		user.Battle.MissionDetail = store.BattleMissionDetailState{}
		if detail := req.BattleDetail; detail != nil {
			missionDetail := store.BattleMissionDetailState{
				IsValid:                true,
				CharacterDeathCount:    detail.CharacterDeathCount,
				MaxDamage:              int64(detail.MaxDamage),
				CostumeSkillUseCount:   detail.PlayerCostumeActiveSkillUsedCount,
				WeaponSkillUseCount:    detail.PlayerWeaponActiveSkillUsedCount,
				CompanionSkillUseCount: detail.PlayerCompanionSkillUsedCount,
				CriticalCount:          detail.CriticalCount,
				ComboCount:             detail.ComboCount,
				ComboMaxDamage:         int64(detail.ComboMaxDamage),
				TotalRecoverPoint:      detail.TotalRecoverPoint,
			}
			for _, info := range detail.CostumeBattleInfo {
				index := int(info.DeckCharacterNumber) - 1
				if index < 0 || index >= len(missionDetail.CostumeResults) {
					continue
				}
				if info.MaxHp > 0 && missionDetail.CostumeResults[index].MaxHp == 0 {
					missionDetail.CostumeResultCount++
				}
				missionDetail.CostumeResults[index] = store.CostumeBattleResultState{
					IsAlive: info.IsAlive, MaxHp: info.MaxHp, RemainingHp: info.RemainingHp,
				}
			}
			user.Battle.MissionDetail = missionDetail
		}
	})
	return &pb.FinishWaveResponse{}, nil
}

func battleCheckpoint(user *store.UserState) []byte {
	return bytes.Clone(user.BattleBinary)
}

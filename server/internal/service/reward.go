package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type RewardServiceServer struct {
	pb.UnimplementedRewardServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewRewardServiceServer(
	users store.UserRepository,
	sessions store.SessionRepository,
	holder *runtime.Holder,
) *RewardServiceServer {
	return &RewardServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *RewardServiceServer) ReceiveBigHuntReward(ctx context.Context, _ *emptypb.Empty) (*pb.ReceiveBigHuntRewardResponse, error) {
	log.Printf("[RewardService] ReceiveBigHuntReward")

	cat := s.holder.Get()
	bhCatalog := cat.BigHunt
	granter := cat.QuestHandler.Granter
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	weeklyVersion := gametime.BusinessWeeklyVersion(nowMillis)
	today := gametime.StartOfBusinessDayMillis()

	var weeklyScoreResults []*pb.WeeklyScoreResult
	var weeklyRewards []*pb.BigHuntReward
	isReceived := false

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		for bossQuestId, bossQuest := range bhCatalog.BossQuestById {
			st := user.BigHuntStatuses[bossQuestId]
			if st.LastDailyRewardReceivedDayVersion >= today {
				continue
			}
			rewardGroupId := bhCatalog.ResolveActiveScoreRewardGroupId(bossQuest.BigHuntScoreRewardGroupScheduleId, nowMillis)
			if rewardGroupId == 0 {
				continue
			}
			maxScore := user.BigHuntScheduleMaxScores[store.BigHuntScheduleScoreKey{
				BigHuntScheduleId: bhCatalog.ActiveScheduleId,
				BigHuntBossId:     bossQuest.BigHuntBossId,
			}].MaxScore
			if maxScore <= 0 {
				continue
			}
			items := bhCatalog.CollectNewRewards(rewardGroupId, 0, maxScore)
			for _, item := range items {
				count := bigHuntDailyRewardCount(cat, user, item, nowMillis)
				granter.GrantFull(user, model.PossessionType(item.PossessionType), item.PossessionId, count, nowMillis)
			}
			if len(items) > 0 {
				log.Printf("[RewardService] ReceiveBigHuntReward: bossQuestId=%d granted %d daily rewards (maxScore=%d, group=%d)",
					bossQuestId, len(items), maxScore, rewardGroupId)
			}
			st.LastDailyRewardReceivedDayVersion = today
			st.LatestVersion = nowMillis
			user.BigHuntStatuses[bossQuestId] = st
		}

		ws := user.BigHuntWeeklyStatuses[weeklyVersion]
		isReceived = ws.IsReceivedWeeklyReward

		for _, boss := range bhCatalog.BossByBossId {
			key := store.BigHuntWeeklyScoreKey{
				BigHuntWeeklyVersion: weeklyVersion,
				AttributeType:        boss.AttributeType,
			}
			wms := user.BigHuntWeeklyMaxScores[key]
			gradeIcon := bhCatalog.ResolveGradeIconId(boss.BigHuntBossId, wms.MaxScore)
			weeklyScoreResults = append(weeklyScoreResults, &pb.WeeklyScoreResult{
				AttributeType:           boss.AttributeType,
				BeforeMaxScore:          wms.MaxScore,
				CurrentMaxScore:         wms.MaxScore,
				BeforeAssetGradeIconId:  gradeIcon,
				CurrentAssetGradeIconId: gradeIcon,
				AfterMaxScore:           wms.MaxScore,
				AfterAssetGradeIconId:   gradeIcon,
			})
		}

		if !isReceived {
			for _, boss := range bhCatalog.BossByBossId {
				rewardGroupId := bhCatalog.ResolveActiveWeeklyRewardGroupIdByAttr(boss.AttributeType, nowMillis)
				if rewardGroupId == 0 {
					continue
				}

				weekKey := store.BigHuntWeeklyScoreKey{
					BigHuntWeeklyVersion: weeklyVersion,
					AttributeType:        boss.AttributeType,
				}
				maxScore := user.BigHuntWeeklyMaxScores[weekKey].MaxScore

				items := bhCatalog.CollectNewRewards(rewardGroupId, 0, maxScore)
				for _, item := range items {
					granter.GrantFull(user, model.PossessionType(item.PossessionType), item.PossessionId, item.Count, nowMillis)
					weeklyRewards = append(weeklyRewards, &pb.BigHuntReward{
						PossessionType: item.PossessionType,
						PossessionId:   item.PossessionId,
						Count:          item.Count,
					})
				}
			}

			ws.IsReceivedWeeklyReward = true
			ws.LatestVersion = nowMillis
			user.BigHuntWeeklyStatuses[weeklyVersion] = ws
			isReceived = true
		}
	})
	if err != nil {
		return nil, fmt.Errorf("receive big hunt reward: %w", err)
	}

	if weeklyRewards == nil {
		weeklyRewards = []*pb.BigHuntReward{}
	}
	if weeklyScoreResults == nil {
		weeklyScoreResults = []*pb.WeeklyScoreResult{}
	}

	return &pb.ReceiveBigHuntRewardResponse{
		WeeklyScoreResult:           weeklyScoreResults,
		WeeklyScoreReward:           weeklyRewards,
		IsReceivedWeeklyScoreReward: isReceived,
		LastWeekWeeklyScoreReward:   []*pb.BigHuntReward{},
	}, nil
}

func bigHuntDailyRewardCount(cat *runtime.Catalogs, user *store.UserState, item masterdata.RewardItem, nowMillis int64) int32 {
	if cat.ImportantItems == nil {
		return item.Count
	}
	_, bonusPermil := cat.ImportantItems.QuestBonuses(
		user.ImportantItems,
		campaign.QuestTarget{QuestType: campaign.QuestTypeBigHunt},
		model.PossessionType(item.PossessionType),
		item.PossessionId,
		nowMillis,
	)
	modifier := campaign.DropCountMul{}
	return modifier.WithBonusPermil(bonusPermil).Apply(item.Count)
}

func (s *RewardServiceServer) ReceivePvpReward(ctx context.Context, _ *emptypb.Empty) (*pb.ReceivePvpRewardResponse, error) {
	log.Printf("[RewardService] ReceivePvpReward (stub)")
	return &pb.ReceivePvpRewardResponse{
		DiffUserData: map[string]*pb.DiffData{},
	}, nil
}

func (s *RewardServiceServer) ReceiveLabyrinthSeasonReward(ctx context.Context, _ *emptypb.Empty) (*pb.ReceiveLabyrinthSeasonRewardResponse, error) {
	log.Printf("[RewardService] ReceiveLabyrinthSeasonReward")
	cat := s.holder.Get()
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var results []*pb.LabyrinthSeasonResult
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, chapter := range cat.Labyrinth.ChaptersByOrder {
			chapterId := chapter.EventQuestChapterId
			if result := receivePendingLabyrinthSeasonReward(cat, user, chapterId, nowMillis); result != nil {
				results = append(results, result)
			}
		}
	})
	if err != nil {
		return nil, fmt.Errorf("receive labyrinth season reward: %w", err)
	}
	return &pb.ReceiveLabyrinthSeasonRewardResponse{
		SeasonResult: results,
		DiffUserData: map[string]*pb.DiffData{},
	}, nil
}

func (s *RewardServiceServer) ReceiveMissionPassRemainingReward(ctx context.Context, _ *emptypb.Empty) (*pb.ReceiveMissionPassRemainingRewardResponse, error) {
	log.Printf("[RewardService] ReceiveMissionPassRemainingReward")
	cat := s.holder.Get()
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var receivedPassId int32
	var claimErr error
	_, updateErr := s.users.UpdateUser(userId, func(user *store.UserState) {
		receivedPassId, claimErr = claimMissionPassRemainingReward(cat, user, nowMillis)
	})
	if updateErr != nil {
		return nil, fmt.Errorf("receive remaining mission pass reward: %w", updateErr)
	}
	if claimErr != nil {
		return nil, claimErr
	}
	return &pb.ReceiveMissionPassRemainingRewardResponse{RewardReceivedMissionPassId: receivedPassId}, nil
}

func claimMissionPassRemainingReward(cat *runtime.Catalogs, user *store.UserState, nowMillis int64) (int32, error) {
	var receivedPassId int32
	var latestEnd int64
	for passId, pass := range cat.Mission.PassById {
		if !missionPassEnded(pass.Definition, nowMillis) || user.MissionPassRemaining[passId].RewardReceived {
			continue
		}
		if receivedPassId == 0 || pass.Definition.EndDatetime > latestEnd || (pass.Definition.EndDatetime == latestEnd && passId < receivedPassId) {
			latestEnd = pass.Definition.EndDatetime
			receivedPassId = passId
		}
	}
	if receivedPassId == 0 {
		return 0, nil
	}
	if _, err := claimMissionPassRewards(cat, user, receivedPassId, nowMillis, true); err != nil {
		return 0, err
	}
	user.MissionPassRemaining[receivedPassId] = store.MissionPassRemainingState{MissionPassId: receivedPassId, RewardReceived: true, RewardReceiveDatetime: nowMillis, LatestVersion: nowMillis}
	return receivedPassId, nil
}

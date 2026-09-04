package service

import (
	"time"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

const labyrinthSeasonRewardClaimPeriod = int64(30 * 24 * time.Hour / time.Millisecond)

type labyrinthQuestProgress struct {
	order      int
	stageOrder int32
}

func updateLabyrinthSeasonData(cat *runtime.Catalogs, user *store.UserState, chapterId int32, nowMillis int64) *pb.LabyrinthSeasonResult {
	currentSeason, ok := cat.Labyrinth.LatestStartedSeason(chapterId, nowMillis)
	if !ok {
		return nil
	}

	state := user.LabyrinthSeasons[chapterId]
	pendingSeasonNumber := state.LastJoinSeasonNumber
	if pendingSeasonNumber == 0 {
		// The projection exposes the latest season for users without a stored row.
		// Use the same season here so their first completed season can be settled.
		pendingSeasonNumber = currentSeason.SeasonNumber
	}
	result := settleLabyrinthSeasonReward(cat, user, chapterId, pendingSeasonNumber, nowMillis)

	state = user.LabyrinthSeasons[chapterId]
	if state.LastJoinSeasonNumber != currentSeason.SeasonNumber {
		state.EventQuestChapterId = chapterId
		state.LastJoinSeasonNumber = currentSeason.SeasonNumber
		state.LatestVersion = nowMillis
		user.LabyrinthSeasons[chapterId] = state
	}
	return result
}

func receivePendingLabyrinthSeasonReward(cat *runtime.Catalogs, user *store.UserState, chapterId int32, nowMillis int64) *pb.LabyrinthSeasonResult {
	seasonNumber := user.LabyrinthSeasons[chapterId].LastJoinSeasonNumber
	if seasonNumber == 0 {
		season, ok := cat.Labyrinth.LatestStartedSeason(chapterId, nowMillis)
		if !ok {
			return nil
		}
		seasonNumber = season.SeasonNumber
	}
	return settleLabyrinthSeasonReward(cat, user, chapterId, seasonNumber, nowMillis)
}

func settleLabyrinthSeasonReward(cat *runtime.Catalogs, user *store.UserState, chapterId, seasonNumber int32, nowMillis int64) *pb.LabyrinthSeasonResult {
	state := user.LabyrinthSeasons[chapterId]
	if seasonNumber == 0 || seasonNumber <= state.LastSeasonRewardReceivedSeasonNumber {
		return nil
	}
	season, ok := cat.Labyrinth.Season(chapterId, seasonNumber)
	if !ok || nowMillis < season.EndDatetime || nowMillis > season.EndDatetime+labyrinthSeasonRewardClaimPeriod {
		return nil
	}

	state.EventQuestChapterId = chapterId
	if state.LastJoinSeasonNumber == 0 {
		state.LastJoinSeasonNumber = seasonNumber
	}
	state.LastSeasonRewardReceivedSeasonNumber = seasonNumber
	state.LatestVersion = nowMillis
	user.LabyrinthSeasons[chapterId] = state

	earned, headStageOrder, ok := earnedLabyrinthSeasonMilestone(cat, user, chapterId, season)
	if !ok || len(earned.Rewards) == 0 {
		return nil
	}

	result := &pb.LabyrinthSeasonResult{
		EventQuestChapterId: chapterId,
		HeadQuestId:         earned.HeadQuestId,
		HeadStageOrder:      headStageOrder,
	}
	for _, reward := range earned.Rewards {
		cat.QuestHandler.Granter.GrantFull(user, model.PossessionType(reward.PossessionType), reward.PossessionId, reward.Count, nowMillis)
		result.SeasonReward = append(result.SeasonReward, &pb.LabyrinthReward{
			PossessionType: reward.PossessionType,
			PossessionId:   reward.PossessionId,
			Count:          reward.Count,
		})
	}
	return result
}

func earnedLabyrinthSeasonMilestone(cat *runtime.Catalogs, user *store.UserState, chapterId int32, season masterdata.EntityMEventQuestLabyrinthSeason) (masterdata.LabyrinthSeasonMilestone, int32, bool) {
	milestones := cat.Labyrinth.SeasonMilestonesFor(season)
	progress := labyrinthQuestProgressById(cat, chapterId)
	bestOrder := -1
	var earned masterdata.LabyrinthSeasonMilestone
	var headStageOrder int32
	found := false
	for index, milestone := range milestones {
		quest, exists := user.Quests[milestone.HeadQuestId]
		if !exists || quest.QuestStateType != model.UserQuestStateTypeCleared {
			continue
		}
		order := index
		stageOrder := milestone.HeadStageOrder
		if value, exists := progress[milestone.HeadQuestId]; exists {
			order = value.order
			stageOrder = value.stageOrder
		}
		if !found || order > bestOrder {
			bestOrder = order
			earned = milestone
			headStageOrder = stageOrder
			found = true
		}
	}
	return earned, headStageOrder, found
}

func labyrinthQuestProgressById(cat *runtime.Catalogs, chapterId int32) map[int32]labyrinthQuestProgress {
	progress := make(map[int32]labyrinthQuestProgress)
	if cat.Quest == nil {
		return progress
	}
	for _, chapter := range cat.Labyrinth.ChaptersByOrder {
		if chapter.EventQuestChapterId != chapterId {
			continue
		}
		order := 0
		for _, stageOrder := range chapter.StageOrders {
			questIds, ok := cat.Labyrinth.StageQuestIds(cat.Quest, chapterId, stageOrder)
			if !ok {
				continue
			}
			for _, questId := range questIds {
				progress[questId] = labyrinthQuestProgress{order: order, stageOrder: stageOrder}
				order++
			}
		}
		break
	}
	return progress
}

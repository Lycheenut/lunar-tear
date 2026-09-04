package service

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

func TestLabyrinthStageClearUsesStoredQuestState(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Quests[100] = store.UserQuestState{QuestStateType: model.UserQuestStateTypeCleared}
	if labyrinthStageCleared(user, []int32{100, 200}) {
		t.Fatal("partially cleared stage was accepted")
	}
	user.Quests[200] = store.UserQuestState{QuestStateType: model.UserQuestStateTypeCleared}
	if !labyrinthStageCleared(user, []int32{100, 200}) {
		t.Fatal("fully cleared stage was rejected")
	}
}

func TestUpdateLabyrinthSeasonDataGrantsOneReachedRewardOnce(t *testing.T) {
	cat := labyrinthSeasonTestCatalog()
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.LabyrinthSeasons[10] = store.LabyrinthSeasonState{
		EventQuestChapterId:  10,
		LastJoinSeasonNumber: 1,
	}
	user.Quests[101] = store.UserQuestState{QuestId: 101, QuestStateType: model.UserQuestStateTypeCleared}
	user.Quests[102] = store.UserQuestState{QuestId: 102, QuestStateType: model.UserQuestStateTypeCleared}

	result := updateLabyrinthSeasonData(cat, user, 10, 100)
	if result == nil || result.HeadQuestId != 102 || result.HeadStageOrder != 2 {
		t.Fatalf("season result = %+v, want the single highest reached quest", result)
	}
	if len(result.SeasonReward) != 1 || user.Materials[500] != 5 {
		t.Fatalf("reward = %+v, material balance = %d", result.SeasonReward, user.Materials[500])
	}
	state := user.LabyrinthSeasons[10]
	if state.LastJoinSeasonNumber != 2 || state.LastSeasonRewardReceivedSeasonNumber != 1 {
		t.Fatalf("season state = %+v", state)
	}

	if duplicate := updateLabyrinthSeasonData(cat, user, 10, 101); duplicate != nil {
		t.Fatalf("duplicate season result = %+v", duplicate)
	}
	if user.Materials[500] != 5 {
		t.Fatalf("duplicate entrance changed material balance to %d", user.Materials[500])
	}
}

func TestReceiveLabyrinthSeasonRewardUsesJoinedSeason(t *testing.T) {
	cat := labyrinthSeasonTestCatalog()
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.LabyrinthSeasons[10] = store.LabyrinthSeasonState{
		EventQuestChapterId:  10,
		LastJoinSeasonNumber: 1,
	}
	user.Quests[101] = store.UserQuestState{QuestId: 101, QuestStateType: model.UserQuestStateTypeCleared}

	result := receivePendingLabyrinthSeasonReward(cat, user, 10, 201)
	if result == nil || result.HeadQuestId != 101 || user.Materials[500] != 5 {
		t.Fatalf("joined-season result = %+v, material balance = %d", result, user.Materials[500])
	}
	if user.Materials[600] != 0 {
		t.Fatal("reward from a season the player did not join was granted")
	}
}

func TestUpdateLabyrinthSeasonDataSkipsExpiredReward(t *testing.T) {
	cat := labyrinthSeasonTestCatalog()
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.LabyrinthSeasons[10] = store.LabyrinthSeasonState{
		EventQuestChapterId:  10,
		LastJoinSeasonNumber: 1,
	}
	user.Quests[101] = store.UserQuestState{QuestId: 101, QuestStateType: model.UserQuestStateTypeCleared}
	nowMillis := int64(100) + labyrinthSeasonRewardClaimPeriod + 1

	if result := updateLabyrinthSeasonData(cat, user, 10, nowMillis); result != nil {
		t.Fatalf("expired season result = %+v", result)
	}
	if user.Materials[500] != 0 || user.LabyrinthSeasons[10].LastJoinSeasonNumber != 2 {
		t.Fatalf("expired reward changed balance or failed to join current season: balance=%d state=%+v", user.Materials[500], user.LabyrinthSeasons[10])
	}
}

func labyrinthSeasonTestCatalog() *runtime.Catalogs {
	return &runtime.Catalogs{
		Labyrinth: &masterdata.LabyrinthCatalog{
			ChaptersByOrder: []masterdata.LabyrinthChapter{{EventQuestChapterId: 10}},
			SeasonsByChapter: map[int32]map[int32]masterdata.EntityMEventQuestLabyrinthSeason{
				10: {
					1: {EventQuestChapterId: 10, SeasonNumber: 1, StartDatetime: 0, EndDatetime: 100, SeasonRewardGroupId: 11},
					2: {EventQuestChapterId: 10, SeasonNumber: 2, StartDatetime: 100, EndDatetime: labyrinthSeasonRewardClaimPeriod, SeasonRewardGroupId: 12},
				},
			},
			SeasonMilestonesByRewardGroup: map[int32][]masterdata.LabyrinthSeasonMilestone{
				11: {
					{HeadQuestId: 101, HeadStageOrder: 1, Rewards: []masterdata.RewardItem{{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 500, Count: 5}}},
					{HeadQuestId: 102, HeadStageOrder: 2, Rewards: []masterdata.RewardItem{{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 500, Count: 5}}},
				},
				12: {{HeadQuestId: 102, HeadStageOrder: 2, Rewards: []masterdata.RewardItem{{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 600, Count: 7}}}},
			},
		},
		Quest: &masterdata.QuestCatalog{EventQuestIdsByChapterId: map[int32][]int32{10: {101, 102}}},
		QuestHandler: &questflow.QuestHandler{
			Granter: &store.PossessionGranter{},
		},
	}
}

package missionprogress

import (
	"fmt"
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

func TestEveryClearConditionEnumAcceptsProgress(t *testing.T) {
	for raw := int32(1); raw <= 74; raw++ {
		if raw == 4 || raw == 24 || raw == 34 {
			if model.MissionClearConditionType(raw).IsKnown() {
				t.Fatalf("reserved condition ID %d is marked known", raw)
			}
			continue
		}
		if !model.MissionClearConditionType(raw).IsKnown() {
			t.Fatalf("condition ID %d is not marked known", raw)
		}

		mission := masterdata.EntityMMission{MissionId: raw, MissionClearConditionType: raw, ClearConditionValue: 1}
		catalogs := testCatalog(mission)
		user := &store.UserState{}
		user.EnsureMaps()
		before := store.CloneUserState(*user)
		Apply(catalogs, &before, user, []store.MissionEvent{{ConditionType: raw, Count: 1}}, 100)
		if state := user.Missions[raw]; state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
			t.Fatalf("condition ID %d did not clear from its matching event: %+v", raw, state)
		}
	}

	if model.MissionClearConditionTypeUnknown.IsKnown() || model.MissionClearConditionType(75).IsKnown() {
		t.Fatal("unknown clear condition was marked known")
	}
}

func TestEveryUnlockConditionEnum(t *testing.T) {
	resolver := loadConditionResolver(t)
	tests := []struct {
		name  string
		type_ model.MissionUnlockConditionType
		setup func(*runtime.Catalogs, *store.UserState)
	}{
		{name: "grant", type_: model.MissionUnlockConditionTypeGrant},
		{name: "quest clear", type_: model.MissionUnlockConditionTypeQuestClear, setup: func(_ *runtime.Catalogs, user *store.UserState) {
			user.Quests[200] = store.UserQuestState{QuestId: 200, QuestStateType: model.UserQuestStateTypeCleared}
		}},
		{name: "mission clear", type_: model.MissionUnlockConditionTypeMissionClearById, setup: func(_ *runtime.Catalogs, user *store.UserState) {
			user.Missions[200] = clearedMission(200)
		}},
		{name: "all daily", type_: model.MissionUnlockConditionTypeMissionClearForAllDaily, setup: func(catalogs *runtime.Catalogs, user *store.UserState) {
			catalogs.Mission.MissionById[200] = masterdata.EntityMMission{MissionId: 200, MissionGroupId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeTitleTransitionByCount), ClearConditionValue: 1}
			user.Missions[200] = clearedMission(200)
		}},
		{name: "webview page", type_: model.MissionUnlockConditionTypeWebviewPanelMissionClearByPageNumber, setup: func(catalogs *runtime.Catalogs, user *store.UserState) {
			catalogs.Mission.WebviewPageNumberByPageId[300] = 200
			user.WebviewPanelMissions[300] = store.WebviewPanelMissionState{WebviewPanelMissionPageId: 300, RewardReceiveDatetime: 1}
		}},
		{name: "evaluate", type_: model.MissionUnlockConditionTypeEvaluate, setup: func(catalogs *runtime.Catalogs, _ *store.UserState) {
			catalogs.ConditionResolver = resolver
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mission := masterdata.EntityMMission{MissionId: 100, MissionUnlockConditionId: 1}
			catalogs := testCatalog(mission)
			conditionValue := int32(200)
			if tt.type_ == model.MissionUnlockConditionTypeEvaluate {
				conditionValue = 0
			}
			catalogs.Mission.UnlockById[1] = masterdata.EntityMMissionUnlockCondition{MissionUnlockConditionId: 1, MissionUnlockConditionType: int32(tt.type_), ConditionValue: conditionValue}
			user := &store.UserState{}
			user.EnsureMaps()
			if tt.setup != nil {
				tt.setup(catalogs, user)
			}
			if !tt.type_.IsKnown() || !unlocked(catalogs, user, mission, 100) {
				t.Fatalf("unlock condition %d was not satisfied", tt.type_)
			}
		})
	}

	catalogs := testCatalog()
	mission := masterdata.EntityMMission{MissionId: 100, MissionUnlockConditionId: 1}
	catalogs.Mission.UnlockById[1] = masterdata.EntityMMissionUnlockCondition{MissionUnlockConditionId: 1}
	user := &store.UserState{}
	user.EnsureMaps()
	if unlocked(catalogs, user, mission, 100) || model.MissionUnlockConditionTypeUnknown.IsKnown() {
		t.Fatal("UNKNOWN unlock condition was accepted")
	}
}

func TestDailyQuestProgressDoesNotCarryAcrossBusinessDays(t *testing.T) {
	mission := masterdata.EntityMMission{MissionId: 1, MissionGroupId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount), ClearConditionValue: 1}
	catalogs := testCatalog(mission)
	user := &store.UserState{}
	user.EnsureMaps()
	user.Missions[1] = store.UserMissionState{MissionId: 1, StartDatetime: 1, ProgressValue: 1, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
	user.Quests[10] = store.UserQuestState{QuestId: 10, QuestStateType: model.UserQuestStateTypeCleared, ClearCount: 3, DailyClearCount: 1, LastClearDatetime: 1}

	const nowMillis = int64(10*24*60*60*1000 + 12345)
	Sync(catalogs, user, nowMillis)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("old quest clears leaked into the new day: %+v", state)
	}
	wantStart := gametime.StartOfBusinessDayAtMillis(nowMillis)
	if state := user.Missions[1]; state.StartDatetime != wantStart {
		t.Fatalf("daily mission start time = %d, want business-day boundary %d", state.StartDatetime, wantStart)
	}

	quest := user.Quests[10]
	quest.DailyClearCount = 1
	quest.LastClearDatetime = nowMillis
	user.Quests[10] = quest
	Sync(catalogs, user, nowMillis)
	if state := user.Missions[1]; state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("today's quest clear did not satisfy daily mission: %+v", state)
	}
}

func TestAllDailyIgnoresMissionPassDaily(t *testing.T) {
	aggregate := masterdata.EntityMMission{MissionId: 1, MissionGroupId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeMissionClearForAllDaily), ClearConditionValue: 1}
	daily := masterdata.EntityMMission{MissionId: 2, MissionGroupId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeTitleTransitionByCount), ClearConditionValue: 1}
	passDaily := masterdata.EntityMMission{MissionId: 3, MissionGroupId: 9, MissionClearConditionType: int32(model.MissionClearConditionTypeTitleTransitionByCount), ClearConditionValue: 1}
	catalogs := testCatalog(aggregate, daily, passDaily)
	user := &store.UserState{}
	user.EnsureMaps()
	user.Missions[2] = clearedMission(2)
	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("mission-pass daily task blocked standard daily aggregate: %+v", state)
	}
}

func TestExplorationMissionWaitsForExplorationUnlock(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 65, MissionGroupId: 1,
		MissionClearConditionType: int32(model.MissionClearConditionTypeMissionClearForAllDailyBySubCategoryId),
		ClearConditionValue:       1,
		RelatedMainFunctionType:   mainFunctionTypeExploration,
	}
	daily := masterdata.EntityMMission{
		MissionId: 64, MissionGroupId: 1,
		MissionClearConditionType: int32(model.MissionClearConditionTypeShopBuyByCount),
		ClearConditionValue:       1,
	}
	catalogs := testCatalog(mission, daily)
	dailyGroup := catalogs.Mission.GroupById[1]
	dailyGroup.MissionSubCategoryId = 1
	catalogs.Mission.GroupById[1] = dailyGroup
	catalogs.Explore = explorationUnlockCatalog()
	user := &store.UserState{}
	user.EnsureMaps()
	user.Missions[64] = clearedMission(64)

	Sync(catalogs, user, 100)
	if _, exists := user.Missions[65]; exists {
		t.Fatal("exploration-related mission was created before Exploration unlocked")
	}

	user.Quests[31] = store.UserQuestState{QuestId: 31, QuestStateType: model.UserQuestStateTypeCleared}
	Sync(catalogs, user, 200)
	if state := user.Missions[65]; state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("exploration-related mission did not unlock normally: %+v", state)
	}
}

func TestSyncRemovesStaleMissionForLockedExploration(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 65, MissionGroupId: 1,
		MissionClearConditionType: int32(model.MissionClearConditionTypeMissionClearForAllDailyBySubCategoryId),
		ClearConditionValue:       1,
		RelatedMainFunctionType:   mainFunctionTypeExploration,
	}
	catalogs := testCatalog(mission)
	catalogs.Explore = explorationUnlockCatalog()
	user := &store.UserState{}
	user.EnsureMaps()
	user.Missions[65] = clearedMission(65)

	Sync(catalogs, user, 100)
	if _, exists := user.Missions[65]; exists {
		t.Fatal("stale completed mission survived while Exploration was locked")
	}
}

func TestCompletedDailyAggregateDoesNotChangeDuringSync(t *testing.T) {
	aggregate := masterdata.EntityMMission{
		MissionId: 1, MissionGroupId: 1,
		MissionClearConditionType: int32(model.MissionClearConditionTypeMissionClearForAllDailyBySubCategoryId),
		ClearConditionValue:       1,
	}
	catalogs := testCatalog(aggregate)
	user := &store.UserState{}
	user.EnsureMaps()
	user.Missions[1] = store.UserMissionState{
		MissionId:                 1,
		StartDatetime:             1,
		ProgressValue:             1,
		MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear),
		ClearDatetime:             50,
		LatestVersion:             50,
	}

	want := user.Missions[1]
	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state != want {
		t.Fatalf("completed daily aggregate changed during sync: got %+v, want %+v", state, want)
	}
}

func TestFromUnlockLoginDoesNotInheritUnlockingLogin(t *testing.T) {
	mission := masterdata.EntityMMission{MissionId: 1, MissionUnlockConditionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeUserLoginByCountFromUnlock), ClearConditionValue: 1}
	catalogs := testCatalog(mission)
	catalogs.Mission.UnlockById[1] = masterdata.EntityMMissionUnlockCondition{MissionUnlockConditionId: 1, MissionUnlockConditionType: int32(model.MissionUnlockConditionTypeQuestClear), ConditionValue: 10}
	before := &store.UserState{}
	before.EnsureMaps()
	// A stale/pre-created row must not make a locked FROM_UNLOCK mission inherit
	// the event that unlocks it.
	before.Missions[1] = store.UserMissionState{MissionId: 1, MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress)}
	after := store.CloneUserState(*before)
	after.Quests[10] = store.UserQuestState{QuestId: 10, QuestStateType: model.UserQuestStateTypeCleared}
	after.Login.TotalLoginCount = 1
	Apply(catalogs, before, &after, nil, 100)
	if state := after.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("unlocking login was inherited: %+v", state)
	}

	nextBefore := store.CloneUserState(after)
	after.Login.TotalLoginCount = 2
	Apply(catalogs, &nextBefore, &after, nil, 200)
	if state := after.Missions[1]; state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("post-unlock login did not clear mission: %+v", state)
	}
}

func TestPvpRankKeepsBestLowerRank(t *testing.T) {
	mission := masterdata.EntityMMission{MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypePvpRank), ClearConditionValue: 1000}
	catalogs := testCatalog(mission)
	user := &store.UserState{}
	user.EnsureMaps()
	user.Profile.CurrentPvpRank = 1500
	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 1500 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("initial PVP rank was not recorded: %+v", state)
	}
	user.Profile.CurrentPvpRank = 900
	Sync(catalogs, user, 200)
	if state := user.Missions[1]; state.ProgressValue != 900 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("improved lower PVP rank did not clear mission: %+v", state)
	}
}

func TestNewWeaponSkillsDoNotCountAsSkillEnhancement(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeWeaponEnhanceSkillByCount), ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	before := &store.UserState{}
	before.EnsureMaps()
	after := store.CloneUserState(*before)
	after.Weapons["new"] = store.WeaponState{UserWeaponUuid: "new", WeaponId: 100}
	after.WeaponSkills["new"] = []store.WeaponSkillState{{UserWeaponUuid: "new", SlotNumber: 1, Level: 1}}

	Apply(catalogs, before, &after, nil, 100)
	if state := after.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("new weapon skills advanced enhancement mission: %+v", state)
	}

	nextBefore := store.CloneUserState(after)
	skills := append([]store.WeaponSkillState(nil), after.WeaponSkills["new"]...)
	skills[0].Level++
	after.WeaponSkills["new"] = skills
	Apply(catalogs, &nextBefore, &after, nil, 200)
	if state := after.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("real weapon skill enhancement was not counted: %+v", state)
	}
}

func TestNewCostumeSkillDoesNotCountAsSkillEnhancement(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeCostumeActiveSkillEnhanceByCount), ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	before := &store.UserState{}
	before.EnsureMaps()
	after := store.CloneUserState(*before)
	after.Costumes["new"] = store.CostumeState{UserCostumeUuid: "new", CostumeId: 100}
	after.CostumeActiveSkills["new"] = store.CostumeActiveSkillState{UserCostumeUuid: "new", Level: 1}

	Apply(catalogs, before, &after, nil, 100)
	if state := after.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("new costume skill advanced enhancement mission: %+v", state)
	}

	nextBefore := store.CloneUserState(after)
	skill := after.CostumeActiveSkills["new"]
	skill.Level++
	after.CostumeActiveSkills["new"] = skill
	Apply(catalogs, &nextBefore, &after, nil, 200)
	if state := after.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("real costume skill enhancement was not counted: %+v", state)
	}
}

func TestQuestClearByCountHonorsQuestTypeOptions(t *testing.T) {
	missions := []masterdata.EntityMMission{
		{MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount), MissionClearConditionOptionGroupId: questClearOptionMainQuest, ClearConditionValue: 100},
		{MissionId: 2, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount), MissionClearConditionOptionGroupId: questClearOptionSubquest, ClearConditionValue: 100},
		{MissionId: 3, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount), MissionClearConditionOptionGroupId: questClearOptionMainQuestHard, ClearConditionValue: 100},
		{MissionId: 4, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount), MissionClearConditionOptionGroupId: questClearOptionMainQuestHardOrVeryHard, ClearConditionValue: 100},
	}
	catalogs := testCatalog(missions...)
	catalogs.Quest = &masterdata.QuestCatalog{
		RouteIdByQuestId:                 map[int32]int32{8: 1, 100: 1, 200: 1},
		MainQuestDifficultyTypeByQuestId: map[int32]int32{8: 1, 100: 2, 200: 3},
		EventQuestTypeByChapterId:        map[int32]int32{300: eventQuestTypeMarathon},
		EventQuestIdsByChapterId:         map[int32][]int32{300: {400}},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Quests[8] = store.UserQuestState{QuestId: 8, ClearCount: 11}
	user.Quests[100] = store.UserQuestState{QuestId: 100, ClearCount: 2}
	user.Quests[200] = store.UserQuestState{QuestId: 200, ClearCount: 3}
	user.Quests[400] = store.UserQuestState{QuestId: 400, ClearCount: 20}

	Sync(catalogs, user, 100)
	for missionId, want := range map[int32]int32{1: 16, 2: 20, 3: 2, 4: 5} {
		if got := user.Missions[missionId].ProgressValue; got != want {
			t.Errorf("mission %d progress = %d, want %d", missionId, got, want)
		}
	}
}

func TestFateBoardMissionDoesNotCountCollidingGuerrillaQuestId(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: questClearOptionFateBoard, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	catalogs.Quest = &masterdata.QuestCatalog{
		EventQuestTypeByChapterId: map[int32]int32{7: eventQuestTypeGuerrilla, 12: eventQuestTypeLabyrinth},
		EventQuestIdsByChapterId:  map[int32][]int32{7: {questClearOptionFateBoard}, 12: {120001}},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Quests[questClearOptionFateBoard] = store.UserQuestState{QuestId: questClearOptionFateBoard, ClearCount: 1}

	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("Guerrilla quest with colliding ID advanced Fate Board mission: %+v", state)
	}

	user.Quests[120001] = store.UserQuestState{QuestId: 120001, ClearCount: 1}
	Sync(catalogs, user, 200)
	if state := user.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("Fate Board quest did not advance Fate Board mission: %+v", state)
	}
}

func TestLinkedEventQuestMissionRejectsCollidingMainQuestId(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: 344, MissionLinkId: 61, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	catalogs.Mission.LinkById = map[int32]masterdata.EntityMMissionLink{
		61: {MissionLinkId: 61, DestinationDomainType: missionLinkDestinationQuest, DestinationDomainId: 514},
	}
	catalogs.Quest = &masterdata.QuestCatalog{
		RouteIdByQuestId:         map[int32]int32{344: 1},
		EventQuestIdsByChapterId: map[int32][]int32{514: {51401}},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Quests[344] = store.UserQuestState{QuestId: 344, ClearCount: 1}

	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("colliding Main Quest advanced linked Event Quest mission: %+v", state)
	}

	user.Quests[51401] = store.UserQuestState{QuestId: 51401, ClearCount: 1}
	Sync(catalogs, user, 200)
	if state := user.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("linked Event Quest did not advance mission: %+v", state)
	}
}

func TestHardChapterMissionUsesChapterCompletionQuest(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: 10006, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	catalogs.Quest = &masterdata.QuestCatalog{
		MainQuestDifficultyTypeByQuestId: map[int32]int32{10006: mainQuestDifficultyHard, 10054: mainQuestDifficultyHard},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Quests[10006] = store.UserQuestState{QuestId: 10006, ClearCount: 1}

	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("unrelated Hard quest advanced chapter mission: %+v", state)
	}

	user.Quests[10054] = store.UserQuestState{QuestId: 10054, ClearCount: 1}
	Sync(catalogs, user, 200)
	if state := user.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("Hard chapter completion quest did not advance mission: %+v", state)
	}
}

func TestNormalChapterMissionRejectsCollidingQuestId(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: 101, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	catalogs.Quest = &masterdata.QuestCatalog{}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Quests[101] = store.UserQuestState{QuestId: 101, ClearCount: 1}

	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("colliding quest advanced chapter mission: %+v", state)
	}

	user.Quests[11] = store.UserQuestState{QuestId: 11, ClearCount: 1}
	Sync(catalogs, user, 200)
	if state := user.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("chapter completion quest did not advance mission: %+v", state)
	}
}

func TestEventDifficultyMissionRequiresFinalQuest(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: 343, MissionLinkId: 61, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	catalogs.Mission.LinkById = map[int32]masterdata.EntityMMissionLink{
		61: {MissionLinkId: 61, DestinationDomainType: missionLinkDestinationQuest, DestinationDomainId: 514},
	}
	catalogs.Quest = &masterdata.QuestCatalog{
		EventQuestIdsByChapterId: map[int32][]int32{514: {100, 101, 300, 301}},
		EventQuestIdsByChapterDifficulty: map[int32]map[int32][]int32{
			514: {eventQuestDifficultyNormal: {100, 101}, eventQuestDifficultyVeryHard: {300, 301}},
		},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Quests[100] = store.UserQuestState{QuestId: 100, ClearCount: 1}
	user.Quests[300] = store.UserQuestState{QuestId: 300, ClearCount: 1}

	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("wrong difficulty or non-final quest advanced mission: %+v", state)
	}

	user.Quests[301] = store.UserQuestState{QuestId: 301, ClearCount: 1}
	Sync(catalogs, user, 200)
	if state := user.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("final Very Hard quest did not advance mission: %+v", state)
	}
}

func TestWithoutSkipSpecificQuestRequiresSoloCharacter(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCountWithoutSkip),
		MissionClearConditionOptionDetailGroupId: 500056, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	catalogs.Quest = &masterdata.QuestCatalog{}
	user := &store.UserState{}
	user.EnsureMaps()

	Apply(catalogs, nil, user, []store.MissionEvent{{
		ConditionType: int32(model.MissionClearConditionTypeQuestClearByCountWithoutSkip),
		Count:         1,
		TargetId:      20034,
		DeckCharacterIds: []int32{
			1015, 1011,
		},
	}}, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("multi-character deck advanced solo mission: %+v", state)
	}

	Apply(catalogs, nil, user, []store.MissionEvent{{
		ConditionType:    int32(model.MissionClearConditionTypeQuestClearByCountWithoutSkip),
		Count:            1,
		TargetId:         20034,
		DeckCharacterIds: []int32{1015},
	}}, 200)
	if state := user.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("matching solo character clear did not advance mission: %+v", state)
	}
}

func TestDynastMemoriesFirstFloorRequiresSaryuClearEvent(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: 500004, ClearConditionValue: 1,
	}
	ordinary := masterdata.EntityMMission{
		MissionId: 2, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
		ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission, ordinary)
	catalogs.Quest = &masterdata.QuestCatalog{}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Quests[500004] = store.UserQuestState{QuestId: 500004, ClearCount: 1}
	user.Quests[210001] = store.UserQuestState{QuestId: 210001, ClearCount: 1}

	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("snapshot or colliding quest advanced deck-context mission: %+v", state)
	}

	Apply(catalogs, nil, user, []store.MissionEvent{{
		ConditionType:      int32(model.MissionClearConditionTypeQuestClearByCount),
		Count:              1,
		TargetId:           210001,
		DeckCharacterIds:   []int32{1015},
		QuestClearWithDeck: true,
	}}, 200)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("wrong character advanced Saryu mission: %+v", state)
	}
	if state := user.Missions[2]; state.ProgressValue != 2 {
		// The ordinary mission is reconciled from the two quest snapshots and
		// must not receive a third count from the deck-context event.
		t.Fatalf("context event changed ordinary clear-count mission: %+v", state)
	}

	Apply(catalogs, nil, user, []store.MissionEvent{{
		ConditionType:      int32(model.MissionClearConditionTypeQuestClearByCount),
		Count:              1,
		TargetId:           210001,
		DeckCharacterIds:   []int32{1022, 1015},
		QuestClearWithDeck: true,
	}}, 300)
	if state := user.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("matching Saryu clear did not advance mission: %+v", state)
	}
}

func TestLibraryElementCountIncludesHistoricalWeaponsAndMemories(t *testing.T) {
	user := &store.UserState{}
	user.EnsureMaps()
	user.Costumes["costume-a"] = store.CostumeState{CostumeId: 1}
	user.Costumes["costume-duplicate"] = store.CostumeState{CostumeId: 1}
	user.Weapons["weapon-current"] = store.WeaponState{WeaponId: 2}
	user.WeaponNotes[2] = store.WeaponNoteState{WeaponId: 2}
	user.WeaponNotes[3] = store.WeaponNoteState{WeaponId: 3}
	user.Companions["companion"] = store.CompanionState{CompanionId: 4}
	user.PartsGroupNotes[5] = store.PartsGroupNoteState{PartsGroupId: 5}
	user.PartsGroupNotes[6] = store.PartsGroupNoteState{PartsGroupId: 6}
	user.Thoughts["thought-a"] = store.ThoughtState{ThoughtId: 7}
	user.Thoughts["thought-duplicate"] = store.ThoughtState{ThoughtId: 7}

	if got := libraryElementCount(user); got != 7 {
		t.Fatalf("library element count = %d, want 7", got)
	}
}

func TestSpecificEarlyMainQuestOptionsDoNotCollideWithEventTypes(t *testing.T) {
	tests := []struct {
		option  int32
		questId int32
	}{
		{option: 6, questId: 32},
		{option: 10, questId: 1},
		{option: 11, questId: 13},
		{option: 12, questId: 31},
	}
	for _, test := range tests {
		mission := masterdata.EntityMMission{
			MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
			MissionClearConditionOptionGroupId: test.option, ClearConditionValue: 1,
		}
		catalogs := testCatalog(mission)
		catalogs.Quest = &masterdata.QuestCatalog{
			EventQuestTypeByChapterId: map[int32]int32{100: test.option},
			EventQuestIdsByChapterId:  map[int32][]int32{100: {999}},
		}
		user := &store.UserState{}
		user.EnsureMaps()
		user.Quests[999] = store.UserQuestState{QuestId: 999, ClearCount: 1}
		Sync(catalogs, user, 100)
		if state := user.Missions[1]; state.ProgressValue != 0 {
			t.Fatalf("option %d accepted colliding Event Quest type: %+v", test.option, state)
		}

		user.Quests[test.questId] = store.UserQuestState{QuestId: test.questId, ClearCount: 1}
		Sync(catalogs, user, 200)
		if state := user.Missions[1]; state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
			t.Fatalf("option %d did not accept quest %d: %+v", test.option, test.questId, state)
		}
	}
}

func TestActivityMissionUsesMissionGroupChapterWhenLinkHasNoChapter(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionGroupId: 90, MissionLinkId: 1,
		MissionClearConditionType:          int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: 196, ClearConditionValue: 10,
	}
	catalogs := testCatalog(mission)
	catalogs.Mission.GroupById[90] = masterdata.EntityMMissionGroup{MissionGroupId: 90, AssetId: 501}
	catalogs.Mission.LinkById = map[int32]masterdata.EntityMMissionLink{
		1: {MissionLinkId: 1, DestinationDomainType: missionLinkDestinationQuest},
	}
	catalogs.Quest = &masterdata.QuestCatalog{
		EventQuestTypeByChapterId: map[int32]int32{
			501: eventQuestTypeMarathon,
			502: eventQuestTypeMarathon,
			1:   eventQuestTypeDayOfTheWeek,
			2:   eventQuestTypeGuerrilla,
		},
		EventQuestIdsByChapterId: map[int32][]int32{
			501: {5011},
			502: {5021},
			1:   {4001},
			2:   {5001},
		},
	}

	if !questMissionMatches(catalogs, mission, 5011) {
		t.Fatal("quest from the mission group's activity chapter did not match")
	}
	for _, questId := range []int32{5021, 4001, 5001} {
		if questMissionMatches(catalogs, mission, questId) {
			t.Errorf("unrelated quest %d matched the activity-scoped mission", questId)
		}
	}

	user := &store.UserState{}
	user.EnsureMaps()
	user.Missions[mission.MissionId] = store.UserMissionState{
		MissionId: mission.MissionId, ProgressValue: 10,
		MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress),
	}
	user.Quests[5011] = store.UserQuestState{QuestId: 5011, ClearCount: 1}
	user.Quests[5021] = store.UserQuestState{QuestId: 5021, ClearCount: 2}
	user.Quests[4001] = store.UserQuestState{QuestId: 4001, ClearCount: 3}
	user.Quests[5001] = store.UserQuestState{QuestId: 5001, ClearCount: 4}
	Sync(catalogs, user, 100)
	if state := user.Missions[mission.MissionId]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("activity-scoped mission accumulated unrelated clears: %+v", state)
	}
}

func TestActivityMissionKeepsDeckContextProgress(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionGroupId: 90, MissionLinkId: 1,
		MissionClearConditionType:          int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: 900001, ClearConditionValue: 2,
	}
	catalogs := testCatalog(mission)
	catalogs.Mission.GroupById[90] = masterdata.EntityMMissionGroup{MissionGroupId: 90, AssetId: 501}
	catalogs.Mission.LinkById = map[int32]masterdata.EntityMMissionLink{
		1: {MissionLinkId: 1, DestinationDomainType: missionLinkDestinationQuest},
	}
	catalogs.Quest = &masterdata.QuestCatalog{
		EventQuestTypeByChapterId: map[int32]int32{501: eventQuestTypeMarathon},
		EventQuestIdsByChapterId:  map[int32][]int32{501: {5011}},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Missions[mission.MissionId] = store.UserMissionState{
		MissionId: mission.MissionId, ProgressValue: 1,
		MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress),
	}

	Sync(catalogs, user, 100)
	if state := user.Missions[mission.MissionId]; state.ProgressValue != 1 {
		t.Fatalf("activity deck-context progress was overwritten: %+v", state)
	}
}

func TestUnknownQuestConditionGroupDoesNotBecomeAQuestId(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: 101040301, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	catalogs.Quest = &masterdata.QuestCatalog{QuestById: map[int32]masterdata.EntityMQuest{
		101040301: {QuestId: 101040301},
	}}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Quests[101040301] = store.UserQuestState{QuestId: 101040301, ClearCount: 1}

	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 {
		t.Fatalf("unknown condition-group id was treated as a QuestId: %+v", state)
	}
}

func TestQuestClearLoadoutConditionsUseTransactionDeckContext(t *testing.T) {
	characterMission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: 479, ClearConditionValue: 1,
	}
	costumeMission := masterdata.EntityMMission{
		MissionId: 2, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount),
		MissionClearConditionOptionGroupId: 377, ClearConditionValue: 1,
	}
	catalogs := testCatalog(characterMission, costumeMission)
	catalogs.Quest = &masterdata.QuestCatalog{}
	user := &store.UserState{}
	user.EnsureMaps()
	user.Quests[10] = store.UserQuestState{QuestId: 10, ClearCount: 99}

	Sync(catalogs, user, 100)
	if user.Missions[1].ProgressValue != 0 || user.Missions[2].ProgressValue != 0 {
		t.Fatalf("historical clears advanced deck-context missions: %+v", user.Missions)
	}

	Apply(catalogs, nil, user, []store.MissionEvent{{
		ConditionType: int32(model.MissionClearConditionTypeQuestClearByCount), Count: 1, TargetId: 10,
		DeckCharacterIds: []int32{1008}, DeckCostumeIds: []int32{999}, QuestClearWithDeck: true,
	}}, 200)
	if user.Missions[1].MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) || user.Missions[2].ProgressValue != 0 {
		t.Fatalf("character-only context matched the wrong loadout missions: %+v", user.Missions)
	}

	Apply(catalogs, nil, user, []store.MissionEvent{{
		ConditionType: int32(model.MissionClearConditionTypeQuestClearByCount), Count: 1, TargetId: 10,
		DeckCharacterIds: []int32{1009}, DeckCostumeIds: []int32{35010}, QuestClearWithDeck: true,
	}}, 300)
	if user.Missions[2].MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("matching costume context did not advance mission: %+v", user.Missions[2])
	}
}

func TestQuestOptionMatchesKnownCategoryParameters(t *testing.T) {
	catalogs := &runtime.Catalogs{Quest: &masterdata.QuestCatalog{
		RouteIdByQuestId:                 map[int32]int32{101: 1, 102: 1, 103: 1},
		MainQuestDifficultyTypeByQuestId: map[int32]int32{101: 1, 102: mainQuestDifficultyHard, 103: mainQuestDifficultyVeryHard},
		EventQuestTypeByChapterId: map[int32]int32{
			100: eventQuestTypeMarathon, 200: eventQuestTypeHunt,
			300: eventQuestTypeDungeon, 400: eventQuestTypeDayOfTheWeek, 500: eventQuestTypeGuerrilla,
			600: eventQuestTypeCharacter, 700: eventQuestTypeCharacterQuest, 900: 9, 1000: eventQuestTypeTower,
			1100: eventQuestTypeLimitContent, 1200: eventQuestTypeLabyrinth,
		},
		EventQuestIdsByChapterId: map[int32][]int32{
			100: {1001}, 200: {2001},
			300: {3001}, 400: {4001}, 500: {5001}, 600: {6001}, 700: {7001}, 900: {9001},
			1000: {10001}, 1100: {11001}, 1200: {12001}, 777: {778},
		},
		EventDailyGroups: []masterdata.EventQuestDailyGroup{{ChapterIds: []int32{900}}},
		EventCharacterIdsByChapterId: map[int32]map[int32]bool{
			600:  {1020: true, 1024: true},
			1100: {1013: true},
		},
	}}

	tests := []struct {
		name    string
		option  int32
		questId int32
		want    bool
	}{
		{name: "Event Quest matches Marathon", option: questClearOptionSubquest, questId: 1001, want: true},
		{name: "Event Quest alias matches Hunt", option: questClearOptionSubquestAlt, questId: 2001, want: true},
		{name: "Event Quest rejects Dungeon", option: questClearOptionSubquest, questId: 3001},
		{name: "Event Quest alias rejects Dungeon", option: questClearOptionSubquestAlt, questId: 3001},
		{name: "Dark Memory direct type", option: eventQuestTypeCharacter, questId: 6001, want: true},
		{name: "Dark Memory alias", option: questClearOptionDarkMemory, questId: 6001, want: true},
		{name: "Dark Memory recurring option", option: 421, questId: 6001, want: true},
		{name: "Dark Memory panel option", option: 540, questId: 6001, want: true},
		{name: "Marie Dark Memory", option: 500144, questId: 6001, want: true},
		{name: "Hina Dark Memory", option: 500159, questId: 6001, want: true},
		{name: "Daily Challenge", option: questClearOptionDailyChallenge, questId: 9001, want: true},
		{name: "Daily Challenge rejects colliding quest ID", option: questClearOptionDailyChallenge, questId: questClearOptionDailyChallenge},
		{name: "Hard alias", option: questClearOptionMainQuestHardAlt, questId: 102, want: true},
		{name: "Hard alias rejects Very Hard", option: questClearOptionMainQuestHardAlt, questId: 103},
		{name: "Very Hard alias", option: questClearOptionMainQuestVeryHard, questId: 103, want: true},
		{name: "Abyss Tower alias", option: questClearOptionAbyssTower, questId: 10001, want: true},
		{name: "Fate Board alias", option: questClearOptionFateBoard, questId: 12001, want: true},
		{name: "Fate Board rejects Guerrilla", option: questClearOptionFateBoard, questId: 5001},
		{name: "daily quest alias", option: questClearOptionDailyQuest, questId: 4001, want: true},
		{name: "Guerrilla alias", option: questClearOptionGuerrilla, questId: 5001, want: true},
		{name: "dungeon alias", option: questClearOptionDungeon, questId: 3001, want: true},
		{name: "dungeon recurring option", option: 500072, questId: 3001, want: true},
		{name: "Tower direct type", option: eventQuestTypeTower, questId: 10001, want: true},
		{name: "limit content direct type", option: eventQuestTypeLimitContent, questId: 11001, want: true},
		{name: "Labyrinth direct type", option: eventQuestTypeLabyrinth, questId: 12001, want: true},
		{name: "Character Quest", option: 85, questId: 7001, want: true},
		{name: "Phantom Recollection", option: 86, questId: 11001, want: true},
		{name: "Lars Phantom Recollection", option: 11007, questId: 11001, want: true},
		{name: "event chapter", option: 777, questId: 778, want: true},
		{name: "event chapter rejects colliding quest ID", option: 777, questId: 777},
		{name: "unknown option rejects colliding quest ID", option: 888, questId: 888},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := questOptionMatches(catalogs, test.option, test.questId); got != test.want {
				t.Fatalf("questOptionMatches(%d, %d) = %t, want %t", test.option, test.questId, got, test.want)
			}
		})
	}
}

func TestCurrentMasterFateBoardOptionRejectsGuerrillaCollision(t *testing.T) {
	resolver := loadConditionResolver(t)
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	questCatalog, err := masterdata.LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	missionCatalog, err := masterdata.LoadMissionCatalog()
	if err != nil {
		t.Fatal(err)
	}
	mission := missionCatalog.MissionById[27701]
	if mission.MissionClearConditionOptionGroupId != questClearOptionFateBoard {
		t.Fatalf("Fate Board mission option = %d, want %d", mission.MissionClearConditionOptionGroupId, questClearOptionFateBoard)
	}
	catalogs := &runtime.Catalogs{Quest: questCatalog}
	if questOptionMatches(catalogs, mission.MissionClearConditionOptionGroupId, questClearOptionFateBoard) {
		t.Fatal("current Guerrilla quest 100025 matched the Fate Board category option")
	}
	for chapterId, eventQuestType := range questCatalog.EventQuestTypeByChapterId {
		if eventQuestType != eventQuestTypeLabyrinth || len(questCatalog.EventQuestIdsByChapterId[chapterId]) == 0 {
			continue
		}
		questId := questCatalog.EventQuestIdsByChapterId[chapterId][0]
		if !questOptionMatches(catalogs, mission.MissionClearConditionOptionGroupId, questId) {
			t.Fatalf("Fate Board quest %d did not match the Fate Board category option", questId)
		}
		return
	}
	t.Fatal("current master has no Fate Board quest to verify")
}

func TestCurrentMasterActivityAllQuestMissionsRejectDailyAndGuerrilla(t *testing.T) {
	resolver := loadConditionResolver(t)
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	questCatalog, err := masterdata.LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	missionCatalog, err := masterdata.LoadMissionCatalog()
	if err != nil {
		t.Fatal(err)
	}
	catalogs := &runtime.Catalogs{Mission: missionCatalog, Quest: questCatalog}

	var excludedQuestIds []int32
	for chapterId, eventQuestType := range questCatalog.EventQuestTypeByChapterId {
		if eventQuestType == eventQuestTypeDayOfTheWeek || eventQuestType == eventQuestTypeGuerrilla {
			excludedQuestIds = append(excludedQuestIds, questCatalog.EventQuestIdsByChapterId[chapterId]...)
		}
	}
	if len(excludedQuestIds) == 0 {
		t.Fatal("current master has no Day of the Week or Guerrilla quests to verify")
	}

	checked := 0
	for _, mission := range missionCatalog.OrderedMissions {
		if mission.MissionClearConditionType != int32(model.MissionClearConditionTypeQuestClearByCount) {
			continue
		}
		selector, ok := eventSelectorForOption(mission.MissionClearConditionOptionGroupId)
		if !ok || !selector.all {
			continue
		}
		chapterId := missionScopedEventQuestChapterId(catalogs, mission)
		eventQuestType := questCatalog.EventQuestTypeByChapterId[chapterId]
		if eventQuestType != eventQuestTypeMarathon && eventQuestType != eventQuestTypeHunt {
			continue
		}
		checked++
		for _, questId := range questCatalog.EventQuestIdsByChapterId[chapterId] {
			if !questMissionMatches(catalogs, mission, questId) {
				t.Errorf("mission %d did not match quest %d from its activity chapter %d", mission.MissionId, questId, chapterId)
			}
		}
		for _, questId := range excludedQuestIds {
			if questMissionMatches(catalogs, mission, questId) {
				t.Errorf("mission %d for activity chapter %d matched Day of the Week or Guerrilla quest %d", mission.MissionId, chapterId, questId)
			}
		}
	}
	if checked == 0 {
		t.Fatal("current master has no activity-scoped all-quest missions to verify")
	}
}

func TestCurrentMasterSpecificQuestTargets(t *testing.T) {
	resolver := loadConditionResolver(t)
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	questCatalog, err := masterdata.LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	missionCatalog, err := masterdata.LoadMissionCatalog()
	if err != nil {
		t.Fatal(err)
	}
	catalogs := &runtime.Catalogs{Mission: missionCatalog, Quest: questCatalog}

	eventMission := missionCatalog.MissionById[4404]
	if questMissionMatches(catalogs, eventMission, 344) {
		t.Fatal("colliding Main Quest 344 matched linked Event Quest mission 4404")
	}
	if questIds := questCatalog.EventQuestIdsByChapterId[514]; len(questIds) == 0 || !questMissionMatches(catalogs, eventMission, questIds[0]) {
		t.Fatal("mission 4404 did not match a quest from its linked Event Quest chapter")
	}
	difficultyMission := missionCatalog.MissionById[4403]
	if difficultyMission.MissionClearConditionOptionGroupId != 343 {
		t.Fatalf("mission 4403 option = %d, want 343", difficultyMission.MissionClearConditionOptionGroupId)
	}
	veryHardQuests := questCatalog.EventQuestIdsByChapterDifficulty[514][eventQuestDifficultyVeryHard]
	if len(veryHardQuests) < 2 {
		t.Fatal("event chapter 514 has too few Very Hard quests to verify")
	}
	if questMissionMatches(catalogs, difficultyMission, veryHardQuests[0]) {
		t.Fatal("non-final Very Hard quest matched the activity completion mission")
	}
	if !questMissionMatches(catalogs, difficultyMission, veryHardQuests[len(veryHardQuests)-1]) {
		t.Fatal("final Very Hard quest did not match the activity completion mission")
	}

	collidingEventMission := missionCatalog.MissionById[1411]
	if collidingEventMission.MissionClearConditionOptionGroupId != 200011 {
		t.Fatalf("mission 1411 option = %d, want 200011", collidingEventMission.MissionClearConditionOptionGroupId)
	}
	if questMissionMatches(catalogs, collidingEventMission, 200011) {
		t.Fatal("old Event Quest ID collision matched a different activity mission")
	}
	hardEventQuests := questCatalog.EventQuestIdsByChapterDifficulty[506][eventQuestDifficultyHard]
	if len(hardEventQuests) == 0 || !questMissionMatches(catalogs, collidingEventMission, hardEventQuests[len(hardEventQuests)-1]) {
		t.Fatal("final Hard quest did not match activity mission 1411")
	}

	hardMission := missionCatalog.MissionById[210014]
	if hardMission.MissionClearConditionOptionGroupId != 10006 {
		t.Fatalf("mission 210014 option = %d, want 10006", hardMission.MissionClearConditionOptionGroupId)
	}
	if questMissionMatches(catalogs, hardMission, 10006) {
		t.Fatal("unrelated Hard quest 10006 matched the sixth Hard chapter mission")
	}
	if !questMissionMatches(catalogs, hardMission, 10054) {
		t.Fatal("sixth Hard chapter completion quest 10054 did not match mission 210014")
	}

	characterMission := missionCatalog.MissionById[26309]
	if characterMission.MissionClearConditionOptionGroupId != 80 {
		t.Fatalf("mission 26309 option = %d, want 80", characterMission.MissionClearConditionOptionGroupId)
	}
	if questMissionMatches(catalogs, characterMission, 80) {
		t.Fatal("colliding Main Quest 80 matched Sarafa's Character Quest mission")
	}
	characterQuestIds := questCatalog.EventQuestIdsByChapterSortOrder[99019][1]
	if len(characterQuestIds) == 0 || !questMissionMatches(catalogs, characterMission, characterQuestIds[0]) {
		t.Fatal("Sarafa's first Character Quest did not match mission 26309")
	}

	darkLairMission := missionCatalog.MissionById[28907]
	if darkLairMission.MissionClearConditionOptionGroupId != 11001 {
		t.Fatalf("mission 28907 option = %d, want 11001", darkLairMission.MissionClearConditionOptionGroupId)
	}
	darkLairQuests := questCatalog.EventQuestIdsByChapterSortOrder[912][8]
	if len(darkLairQuests) == 0 || !questMissionMatches(catalogs, darkLairMission, darkLairQuests[0]) {
		t.Fatal("Levania's beginner Dark Lair quest did not match mission 28907")
	}

	phantomMission := missionCatalog.MissionById[31001]
	if phantomMission.MissionClearConditionOptionGroupId != 11007 {
		t.Fatalf("mission 31001 option = %d, want 11007", phantomMission.MissionClearConditionOptionGroupId)
	}
	var larsPhantomQuestId int32
	for chapterId, eventQuestType := range questCatalog.EventQuestTypeByChapterId {
		if eventQuestType == eventQuestTypeLimitContent && questCatalog.EventCharacterIdsByChapterId[chapterId][1013] &&
			len(questCatalog.EventQuestIdsByChapterId[chapterId]) != 0 {
			larsPhantomQuestId = questCatalog.EventQuestIdsByChapterId[chapterId][0]
			break
		}
	}
	if larsPhantomQuestId == 0 || !questMissionMatches(catalogs, phantomMission, larsPhantomQuestId) {
		t.Fatal("Lars's Phantom Recollection quest did not match mission 31001")
	}

	darkCoinMission := missionCatalog.MissionById[1011204]
	darkCoinQuestIds := questCatalog.EventQuestIdsByChapterSortOrder[905][1]
	if len(darkCoinQuestIds) == 0 || !questMissionMatches(catalogs, darkCoinMission, darkCoinQuestIds[0]) {
		t.Fatal("Argo's beginner Dark Coin quest did not match mission 1011204")
	}
	ticketMission := missionCatalog.MissionById[1011205]
	ticketQuestIds := questCatalog.EventQuestIdsByChapterSortOrder[905][4]
	if len(ticketQuestIds) == 0 || !questMissionMatches(catalogs, ticketMission, ticketQuestIds[0]) {
		t.Fatal("Argo's beginner EX ticket quest did not match mission 1011205")
	}

	dynastMission := missionCatalog.MissionById[500007]
	if dynastMission.MissionClearConditionOptionGroupId != 500004 {
		t.Fatalf("mission 500007 option = %d, want 500004", dynastMission.MissionClearConditionOptionGroupId)
	}
	if questMissionMatches(catalogs, dynastMission, 500004) {
		t.Fatal("colliding Event Quest chapter ID matched the Dynast's Memories mission")
	}
	if !questMissionMatches(catalogs, dynastMission, 210001) {
		t.Fatal("Dynast's Memories 1F quest did not match mission 500007")
	}
	if questMissionMatches(catalogs, dynastMission, 210010) {
		t.Fatal("Dynast's Memories 10F quest matched the 1F mission 500007")
	}

	for option, targetIds := range specificEventQuestTargetsByOption {
		for _, targetId := range targetIds {
			if _, ok := questCatalog.QuestById[targetId]; !ok {
				t.Errorf("specific Event Quest option %d targets missing quest %d", option, targetId)
			}
		}
	}

	for option, targetIds := range mainQuestTargetsByOption {
		for _, targetId := range targetIds {
			if _, ok := questCatalog.QuestById[targetId]; !ok {
				t.Errorf("specific Main Quest option %d targets missing quest %d", option, targetId)
			}
		}
	}
}

func TestCurrentMasterLinkedEventQuestMissionsHaveSelectors(t *testing.T) {
	resolver := loadConditionResolver(t)
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	questCatalog, err := masterdata.LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	missionCatalog, err := masterdata.LoadMissionCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, mission := range missionCatalog.OrderedMissions {
		if mission.MissionClearConditionType != int32(model.MissionClearConditionTypeQuestClearByCount) {
			continue
		}
		link, ok := missionCatalog.LinkById[mission.MissionLinkId]
		if !ok || link.DestinationDomainType != missionLinkDestinationQuest || len(questCatalog.EventQuestIdsByChapterId[link.DestinationDomainId]) == 0 {
			continue
		}
		if _, ok := eventSelectorForOption(mission.MissionClearConditionOptionGroupId); !ok {
			t.Errorf("mission %d has linked Event Quest chapter %d but no selector for option %d",
				mission.MissionId, link.DestinationDomainId, mission.MissionClearConditionOptionGroupId)
		}
	}
}

func TestChapterGachaMissionOnlyCountsChapterGacha(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeGachaDrawByCount),
		MissionClearConditionOptionGroupId: gachaOptionChapterSummon, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	catalogs.GachaEntries = []store.GachaCatalogEntry{
		{GachaId: 45, GachaLabelType: model.GachaLabelPremium},
		{GachaId: 200001, GachaLabelType: model.GachaLabelChapter},
	}
	before := &store.UserState{}
	before.EnsureMaps()
	after := store.CloneUserState(*before)
	after.Gacha.BannerStates[45] = store.GachaBannerState{GachaId: 45, DrawCount: 10}

	Apply(catalogs, before, &after, nil, 100)
	if state := after.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("ordinary Gacha advanced chapter Gacha mission: %+v", state)
	}

	nextBefore := store.CloneUserState(after)
	after.Gacha.BannerStates[200001] = store.GachaBannerState{GachaId: 200001, DrawCount: 5}
	Apply(catalogs, &nextBefore, &after, nil, 200)
	if state := after.Missions[1]; state.ProgressValue != 5 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("chapter Gacha did not advance its mission: %+v", state)
	}
}

func TestDailyGachaMissionRequiresDailyOption(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeGachaDrawByCount),
		MissionClearConditionOptionGroupId: gachaOptionDailySummon, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	user := &store.UserState{}
	user.EnsureMaps()
	before := store.CloneUserState(*user)

	Apply(catalogs, &before, user, []store.MissionEvent{{ConditionType: int32(model.MissionClearConditionTypeGachaDrawByCount), Count: 1}}, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 {
		t.Fatalf("uncategorized Gacha advanced Daily Gacha mission: %+v", state)
	}
	Apply(catalogs, &before, user, []store.MissionEvent{{ConditionType: int32(model.MissionClearConditionTypeGachaDrawByCount), Count: 1, OptionGroupId: gachaOptionDailySummon}}, 200)
	if state := user.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("Daily Gacha option did not advance its mission: %+v", state)
	}
}

func TestCampaignGachaMissionsUseMasterGachaIds(t *testing.T) {
	tests := []struct {
		option  int32
		gachaId int32
	}{
		{option: 600, gachaId: 552001},
		{option: 900002, gachaId: 9000073},
		{option: 101120601, gachaId: 209},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("option_%d", tt.option), func(t *testing.T) {
			mission := masterdata.EntityMMission{
				MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeGachaDrawByCount),
				MissionClearConditionOptionGroupId: tt.option, ClearConditionValue: 1,
			}
			catalogs := testCatalog(mission)
			user := &store.UserState{}
			user.EnsureMaps()
			before := store.CloneUserState(*user)

			Apply(catalogs, &before, user, []store.MissionEvent{{ConditionType: int32(model.MissionClearConditionTypeGachaDrawByCount), Count: 1, TargetId: tt.option}}, 100)
			if state := user.Missions[1]; state.ProgressValue != 0 {
				t.Fatalf("option id used as Gacha id advanced mission: %+v", state)
			}
			Apply(catalogs, &before, user, []store.MissionEvent{{ConditionType: int32(model.MissionClearConditionTypeGachaDrawByCount), Count: 1, TargetId: tt.gachaId}}, 200)
			if state := user.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
				t.Fatalf("master Gacha id did not advance mission: %+v", state)
			}
		})
	}
}

func TestItemShopMissionOnlyCountsItemShopPurchases(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeShopBuyByCount),
		MissionClearConditionOptionGroupId: shopOptionItemShop, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	catalogs.Shop = &masterdata.ShopCatalog{ItemShopPool: []int32{10}}
	before := &store.UserState{}
	before.EnsureMaps()
	after := store.CloneUserState(*before)
	after.ShopItems[20] = store.UserShopItemState{ShopItemId: 20, BoughtCount: 1}
	Apply(catalogs, before, &after, nil, 100)
	if state := after.Missions[1]; state.ProgressValue != 0 {
		t.Fatalf("non-item-shop purchase advanced item-shop mission: %+v", state)
	}

	nextBefore := store.CloneUserState(after)
	after.ShopItems[10] = store.UserShopItemState{ShopItemId: 10, BoughtCount: 1}
	Apply(catalogs, &nextBefore, &after, nil, 200)
	if state := after.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("item-shop purchase did not advance its mission: %+v", state)
	}
}

func TestUnknownOptionDoesNotAcceptUncategorizedEvent(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeTitleTransitionByCount),
		MissionClearConditionOptionGroupId: 999, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	user := &store.UserState{}
	user.EnsureMaps()
	before := store.CloneUserState(*user)
	event := store.MissionEvent{ConditionType: int32(model.MissionClearConditionTypeTitleTransitionByCount), Count: 1}

	Apply(catalogs, &before, user, []store.MissionEvent{event}, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 {
		t.Fatalf("uncategorized event advanced parameterized mission: %+v", state)
	}
	event.OptionGroupId = 999
	Apply(catalogs, &before, user, []store.MissionEvent{event}, 200)
	if state := user.Missions[1]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("matching option did not advance parameterized mission: %+v", state)
	}
}

func TestResourceDerivedEquipmentOptionTargets(t *testing.T) {
	targets, ok := knownOptionTargets(model.MissionClearConditionTypeWeaponEnhanceByCount, 205)
	if !ok || !containsTarget(targets, 340031) || !containsTarget(targets, 340032) || containsTarget(targets, 250141) {
		t.Fatalf("option 205 did not resolve to the Blackbird Dagger family: %v", targets)
	}

	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeWeaponMaxLevel),
		MissionClearConditionOptionGroupId: 205, ClearConditionValue: 10,
	}
	catalogs := testCatalog(mission)
	user := &store.UserState{}
	user.EnsureMaps()
	user.Weapons["target"] = store.WeaponState{UserWeaponUuid: "target", WeaponId: 340031, Level: 9}
	user.Weapons["other"] = store.WeaponState{UserWeaponUuid: "other", WeaponId: 250141, Level: 99}
	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 9 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("unrelated weapon satisfied a targeted max-level mission: %+v", state)
	}

	target := user.Weapons["target"]
	target.Level = 10
	user.Weapons["target"] = target
	Sync(catalogs, user, 200)
	if state := user.Missions[1]; state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("targeted weapon did not satisfy its max-level mission: %+v", state)
	}
}

func TestCostumeOptionsUseMasterCostumeIds(t *testing.T) {
	targets, ok := knownOptionTargets(model.MissionClearConditionTypeCostumeMaxLevel, 197)
	if !ok || !containsTarget(targets, 25002) || containsTarget(targets, 9005) {
		t.Fatalf("Guardian Hunter option did not resolve to its master CostumeId: %v", targets)
	}
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeCostumeMaxLevel),
		MissionClearConditionOptionGroupId: 197, ClearConditionValue: 60,
	}
	catalogs := testCatalog(mission)
	user := &store.UserState{}
	user.EnsureMaps()
	user.Costumes["target"] = store.CostumeState{UserCostumeUuid: "target", CostumeId: 25002, Level: 60}
	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("master CostumeId did not satisfy targeted mission: %+v", state)
	}
}

func TestBigHuntHighScoreHonorsTextDerivedDetailTarget(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeBigHuntHighScore),
		MissionClearConditionOptionDetailGroupId: 500045, ClearConditionValue: 100,
	}
	catalogs := testCatalog(mission)
	catalogs.BigHunt = &masterdata.BigHuntCatalog{
		// This numeric collision is intentional: the live quest table maps
		// 500045 to boss 5, while mission text says Windy Cursed God (boss 3).
		BossIdByQuestId: map[int32]int32{500045: 5},
		BossByBossId:    map[int32]masterdata.BigHuntBossRow{3: {BigHuntBossId: 3}, 5: {BigHuntBossId: 5}},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	user.BigHuntMaxScores[5] = store.BigHuntMaxScore{MaxScore: 999}
	user.BigHuntMaxScores[3] = store.BigHuntMaxScore{MaxScore: 999}
	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("historical score without deck context advanced mission: %+v", state)
	}
	Apply(catalogs, nil, user, []store.MissionEvent{{
		ConditionType: int32(model.MissionClearConditionTypeBigHuntHighScore), Value: 100, IsValue: true,
		TargetId: 3, DeckCharacterIds: []int32{1013}, BigHuntWithDeck: true,
	}}, 200)
	if state := user.Missions[1]; state.ProgressValue != 0 {
		t.Fatalf("incomplete Big Hunt loadout advanced mission: %+v", state)
	}
	Apply(catalogs, nil, user, []store.MissionEvent{{
		ConditionType: int32(model.MissionClearConditionTypeBigHuntHighScore), Value: 100, IsValue: true,
		TargetId: 3, DeckCharacterIds: []int32{1013, 1014}, BigHuntWithDeck: true,
	}}, 300)
	if state := user.Missions[1]; state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("matching Big Hunt boss and loadout did not advance mission: %+v", state)
	}
}

func TestSnapshotConditionsAreNotIncrementedTwiceByDerivedEvents(t *testing.T) {
	missions := []masterdata.EntityMMission{
		{MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeCostumeAwakenCount), ClearConditionValue: 2},
		{MissionId: 2, MissionClearConditionType: int32(model.MissionClearConditionTypeWeaponAwakenCount), ClearConditionValue: 2},
		{MissionId: 3, MissionClearConditionType: int32(model.MissionClearConditionTypeCostumeLotteryEffectSlotUnlockCount), ClearConditionValue: 2},
	}
	catalogs := testCatalog(missions...)
	before := store.UserState{}
	before.EnsureMaps()
	before.Costumes["costume"] = store.CostumeState{UserCostumeUuid: "costume", CostumeId: 1}
	before.Weapons["weapon"] = store.WeaponState{UserWeaponUuid: "weapon", WeaponId: 1}
	after := store.CloneUserState(before)
	costume := after.Costumes["costume"]
	costume.AwakenCount = 1
	costume.CostumeLotteryEffectUnlockedSlotCount = 1
	after.Costumes["costume"] = costume
	after.WeaponAwakens["weapon"] = store.WeaponAwakenState{UserWeaponUuid: "weapon"}

	Apply(catalogs, &before, &after, nil, 100)
	for _, mission := range missions {
		state := after.Missions[mission.MissionId]
		if state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
			t.Fatalf("condition %d was double-counted: %+v", mission.MissionClearConditionType, state)
		}
	}
}

func TestCharacterRebirthMissionUsesEncodedMasterSortOrder(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 521051, MissionClearConditionType: int32(model.MissionClearConditionTypeCharacterRebirthCount),
		MissionClearConditionOptionGroupId: 60016, ClearConditionValue: 2,
	}
	catalogs := testCatalog(mission)
	catalogs.CharacterRebirth = &masterdata.CharacterRebirthCatalog{
		StepGroupByCharacterId: map[int32]int32{1017: 1017, 1018: 1018},
		CharacterIdBySortOrder: map[int32]int32{105: 1018, 106: 1017},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	user.CharacterRebirths[1017] = store.CharacterRebirthState{CharacterId: 1017, RebirthCount: 5}
	user.CharacterRebirths[1018] = store.CharacterRebirthState{CharacterId: 1018, RebirthCount: 1}
	Sync(catalogs, user, 100)
	if state := user.Missions[mission.MissionId]; state.ProgressValue != 1 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("wrong character rebirth count was used: %+v", state)
	}
}

func TestCurrentMasterUsesOnlyImplementedEnums(t *testing.T) {
	loadConditionResolver(t)
	catalog, err := masterdata.LoadMissionCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, mission := range catalog.OrderedMissions {
		if !model.MissionClearConditionType(mission.MissionClearConditionType).IsKnown() {
			t.Fatalf("mission %d uses unsupported clear condition %d", mission.MissionId, mission.MissionClearConditionType)
		}
	}
	for id, condition := range catalog.UnlockById {
		if !model.MissionUnlockConditionType(condition.MissionUnlockConditionType).IsKnown() {
			t.Fatalf("unlock condition %d uses unsupported type %d", id, condition.MissionUnlockConditionType)
		}
	}
	boardCatalog, err := masterdata.LoadCharacterBoardCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if boardCatalog.MissionOptionByBoardId[100401] != 310001 || boardCatalog.MissionOptionByBoardId[100404] != 310002 {
		t.Fatalf("character-board mission option mapping was not reconstructed: %d, %d",
			boardCatalog.MissionOptionByBoardId[100401], boardCatalog.MissionOptionByBoardId[100404])
	}
	bigHuntCatalog := masterdata.LoadBigHuntCatalog()
	rebirthCatalog, err := masterdata.LoadCharacterRebirthCatalog()
	if err != nil {
		t.Fatal(err)
	}
	resourceCatalogs := &runtime.Catalogs{CharacterBoard: boardCatalog, CharacterRebirth: rebirthCatalog, BigHunt: bigHuntCatalog}
	if bossId := bigHuntBossId(resourceCatalogs, masterdata.EntityMMission{MissionClearConditionOptionDetailGroupId: 500045}); bossId != 3 {
		t.Fatalf("Big Hunt mission detail 500045 maps to boss %d, want text-derived boss 3", bossId)
	}
	boardOptions := make(map[int32]bool)
	for _, option := range boardCatalog.MissionOptionByBoardId {
		boardOptions[option] = true
	}
	for _, mission := range catalog.OrderedMissions {
		conditionType := model.MissionClearConditionType(mission.MissionClearConditionType)
		option := mission.MissionClearConditionOptionGroupId
		if option != 0 && isEquipmentTargetCondition(conditionType) {
			if _, ok := knownOptionTargets(conditionType, option); !ok {
				t.Fatalf("mission %d has an unmapped equipment option %d", mission.MissionId, option)
			}
		}
		switch conditionType {
		case model.MissionClearConditionTypeGachaDrawByCount:
			if option == 0 || option == gachaOptionChapterSummon || option == gachaOptionDailySummon {
				break
			}
			if _, ok := gachaTargetsByOption[option]; !ok {
				t.Fatalf("mission %d has an unmapped Gacha option %d", mission.MissionId, option)
			}
		case model.MissionClearConditionTypeShopBuyByCount:
			if option != 0 && option != shopOptionItemShop {
				t.Fatalf("mission %d has an unmapped shop option %d", mission.MissionId, option)
			}
		case model.MissionClearConditionTypeTitleTransitionByCount:
			if option != 0 && option != 395 {
				t.Fatalf("mission %d has an unmapped title-transition option %d", mission.MissionId, option)
			}
		case model.MissionClearConditionTypeCharacterBoardPanelReleaseByCount:
			if option != 0 && !boardOptions[option] {
				t.Fatalf("mission %d has an unmapped character-board option %d", mission.MissionId, option)
			}
		case model.MissionClearConditionTypeBigHuntHighScore:
			if (option != 0 || mission.MissionClearConditionOptionDetailGroupId != 0) && bigHuntBossId(resourceCatalogs, mission) == 0 {
				t.Fatalf("mission %d has an unmapped Big Hunt target", mission.MissionId)
			}
		case model.MissionClearConditionTypeCharacterRebirthCount:
			if option != 0 && characterRebirthMissionTarget(resourceCatalogs, mission) == 0 {
				t.Fatalf("mission %d has an unmapped character-rebirth option %d", mission.MissionId, option)
			}
		}
	}
}

func TestCurrentMasterQuestClearConditionsResolveOrAreDocumented(t *testing.T) {
	resolver := loadConditionResolver(t)
	missionCatalog, err := masterdata.LoadMissionCatalog()
	if err != nil {
		t.Fatal(err)
	}
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	questCatalog, err := masterdata.LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	catalogs := &runtime.Catalogs{Mission: missionCatalog, Quest: questCatalog}

	type conditionKey struct {
		option int32
		detail int32
	}
	// The Anecdote: Stars placeholder missions have only generic text and a
	// zero quest link, so neither the localized text nor master data identifies
	// a target. The seven ordinary event entries below do have implemented
	// text-derived selectors, but their expired chapter/difficulty rows are no
	// longer present in the current quest master snapshot.
	documented := map[conditionKey]bool{
		{detail: 101020301}: true, {detail: 101020401}: true, {detail: 101020501}: true,
		{detail: 101020801}: true, {detail: 101020901}: true, {detail: 101021001}: true,
		{detail: 101021301}: true, {detail: 101021401}: true, {detail: 101021501}: true,
		{option: 101020201}: true,
		{option: 101040301}: true, {option: 101040401}: true, {option: 101040501}: true,
		{option: 101040601}: true, {option: 101040801}: true, {option: 101040901}: true,
		{option: 101041001}: true, {option: 101041301}: true, {option: 101041401}: true,
		{option: 101041501}: true,
		{option: 370}:       true, {option: 371}: true,
		{option: 563}: true, {option: 564}: true,
		{option: 638}: true, {option: 639}: true, {option: 649}: true,
	}
	unresolved := make(map[conditionKey][]int32)
	for _, mission := range missionCatalog.OrderedMissions {
		if mission.MissionClearConditionType != int32(model.MissionClearConditionTypeQuestClearByCount) {
			continue
		}
		matched := false
		for questId := range questCatalog.QuestById {
			if questMissionMatches(catalogs, mission, questId) {
				matched = true
				break
			}
		}
		if !matched {
			key := conditionKey{mission.MissionClearConditionOptionGroupId, mission.MissionClearConditionOptionDetailGroupId}
			unresolved[key] = append(unresolved[key], mission.MissionId)
		}
	}
	for key, missionIds := range unresolved {
		if !documented[key] {
			t.Errorf("undocumented quest-clear condition %+v has no real quest target (missions %v)", key, missionIds)
		}
		delete(documented, key)
	}
	if len(documented) != 0 {
		t.Fatalf("documented unresolved quest conditions now resolve and should be reviewed: %v", documented)
	}
}

func testCatalog(missions ...masterdata.EntityMMission) *runtime.Catalogs {
	catalog := &masterdata.MissionCatalog{
		MissionById:                    make(map[int32]masterdata.EntityMMission),
		MissionIdsByType:               make(map[int32][]int32),
		TermById:                       make(map[int32]masterdata.EntityMMissionTerm),
		UnlockById:                     make(map[int32]masterdata.EntityMMissionUnlockCondition),
		GroupById:                      map[int32]masterdata.EntityMMissionGroup{1: {MissionGroupId: 1, MissionCategoryType: missionCategoryDaily}, 9: {MissionGroupId: 9, MissionCategoryType: missionCategoryMissionPassDaily}},
		CompletePossessionsByMissionId: make(map[int32][]masterdata.EntityMCompleteMissionGroup),
		WebviewPageNumberByPageId:      make(map[int32]int32),
	}
	for _, mission := range missions {
		catalog.MissionById[mission.MissionId] = mission
		catalog.MissionIdsByType[mission.MissionClearConditionType] = append(catalog.MissionIdsByType[mission.MissionClearConditionType], mission.MissionId)
	}
	return &runtime.Catalogs{Mission: catalog}
}

func clearedMission(id int32) store.UserMissionState {
	return store.UserMissionState{MissionId: id, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear), ClearDatetime: 1}
}

func explorationUnlockCatalog() *masterdata.ExploreCatalog {
	return &masterdata.ExploreCatalog{
		FirstExploreId: 1,
		Explores: map[int32]masterdata.EntityMExplore{
			1: {ExploreId: 1, ExploreUnlockConditionId: 1},
		},
		UnlockConditions: map[int32]masterdata.EntityMExploreUnlockCondition{
			1: {ExploreUnlockConditionId: 1, ExploreUnlockConditionType: 1, ConditionValue: 31},
		},
	}
}

func loadConditionResolver(t *testing.T) *masterdata.ConditionResolver {
	t.Helper()
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	resolver, err := masterdata.LoadConditionResolver()
	if err != nil {
		t.Fatal(err)
	}
	return resolver
}

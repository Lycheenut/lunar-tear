package missionprogress

import (
	"path/filepath"
	"testing"

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

	const nowMillis = int64(10 * 24 * 60 * 60 * 1000)
	Sync(catalogs, user, nowMillis)
	if state := user.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("old quest clears leaked into the new day: %+v", state)
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

func TestDailyGachaMissionOnlyCountsDailyGacha(t *testing.T) {
	mission := masterdata.EntityMMission{
		MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeGachaDrawByCount),
		MissionClearConditionOptionGroupId: 100001, ClearConditionValue: 1,
	}
	catalogs := testCatalog(mission)
	before := &store.UserState{}
	before.EnsureMaps()
	after := store.CloneUserState(*before)
	after.Gacha.BannerStates[45] = store.GachaBannerState{GachaId: 45, DrawCount: 10}

	Apply(catalogs, before, &after, nil, 100)
	if state := after.Missions[1]; state.ProgressValue != 0 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("ordinary Gacha advanced Daily Gacha mission: %+v", state)
	}

	nextBefore := store.CloneUserState(after)
	after.Gacha.BannerStates[201] = store.GachaBannerState{GachaId: 201, DrawCount: 5}
	Apply(catalogs, &nextBefore, &after, nil, 200)
	if state := after.Missions[1]; state.ProgressValue != 5 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeClear) {
		t.Fatalf("Daily Gacha did not advance its mission: %+v", state)
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
	user.BigHuntMaxScores[3] = store.BigHuntMaxScore{MaxScore: 99}
	Sync(catalogs, user, 100)
	if state := user.Missions[1]; state.ProgressValue != 99 || state.MissionProgressStatusType != int32(model.MissionProgressStatusTypeInProgress) {
		t.Fatalf("unrelated Big Hunt boss satisfied a detail-targeted mission: %+v", state)
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

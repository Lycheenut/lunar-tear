package missionprogress

import (
	"fmt"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

func TestCurrentMasterReplicantMissionsCountAllSubquests(t *testing.T) {
	resolver := loadConditionResolver(t)
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	quests, err := masterdata.LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	missions, err := masterdata.LoadMissionCatalog()
	if err != nil {
		t.Fatal(err)
	}
	catalogs := &runtime.Catalogs{Mission: missions, Quest: quests}
	for missionId := int32(1257); missionId <= 1270; missionId++ {
		mission := missions.MissionById[missionId]
		if mission.MissionClearConditionOptionGroupId != questClearOptionSubquest ||
			missions.GroupById[mission.MissionGroupId].AssetId != 502 {
			t.Fatalf("unexpected Replicant mission definition: %+v", mission)
		}
		for chapterId, questIds := range quests.EventQuestIdsByChapterId {
			for _, questId := range questIds {
				if !questMissionMatches(catalogs, mission, questId) {
					t.Fatalf("mission %d excluded subquest %d from chapter %d (type %d)",
						missionId, questId, chapterId, quests.EventQuestTypeByChapterId[chapterId])
				}
			}
		}
		for questId := range quests.RouteIdByQuestId {
			if questMissionMatches(catalogs, mission, questId) {
				t.Fatalf("mission %d counted main quest %d", missionId, questId)
			}
		}
	}
}

func TestSubquestMissionsReconcileAllTypesAndCountClears(t *testing.T) {
	for _, option := range []int32{questClearOptionSubquest, questClearOptionSubquestAlt} {
		for _, skip := range []bool{false, true} {
			t.Run(fmt.Sprintf("option_%d/skip_%t", option, skip), func(t *testing.T) {
				mission := masterdata.EntityMMission{
					MissionId: 1, MissionGroupId: 12, MissionLinkId: 1,
					MissionClearConditionType:          int32(model.MissionClearConditionTypeQuestClearByCount),
					MissionClearConditionOptionGroupId: option, ClearConditionValue: 100,
				}
				catalogs := testCatalog(mission)
				catalogs.Mission.GroupById[12] = masterdata.EntityMMissionGroup{MissionGroupId: 12, AssetId: 502}
				catalogs.Mission.LinkById = map[int32]masterdata.EntityMMissionLink{
					1: {MissionLinkId: 1, DestinationDomainType: missionLinkDestinationQuest},
				}
				catalogs.Quest = &masterdata.QuestCatalog{
					QuestById:                 map[int32]masterdata.EntityMQuest{},
					EventQuestTypeByChapterId: map[int32]int32{502: eventQuestTypeMarathon},
					EventQuestIdsByChapterId:  map[int32][]int32{502: {5021}},
				}
				user := &store.UserState{}
				user.EnsureMaps()
				user.Quests[5021] = store.UserQuestState{QuestId: 5021, ClearCount: 1}
				user.Quests[999] = store.UserQuestState{QuestId: 999, ClearCount: 50}
				user.Missions[1] = store.UserMissionState{
					MissionId: 1, ProgressValue: 1, MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress),
				}
				for eventType := int32(1); eventType <= 12; eventType++ {
					questId := eventType * 10
					catalogs.Quest.QuestById[questId] = masterdata.EntityMQuest{QuestId: questId, IsUsableSkipTicket: true}
					catalogs.Quest.EventQuestTypeByChapterId[eventType] = eventType
					catalogs.Quest.EventQuestIdsByChapterId[eventType] = []int32{questId}
					user.Quests[questId] = store.UserQuestState{
						QuestId: questId, ClearCount: 1, QuestStateType: model.UserQuestStateTypeCleared, IsRewardGranted: true,
					}
				}
				Sync(catalogs, user, 100)
				want := int32(13)
				if got := user.Missions[1].ProgressValue; got != want {
					t.Fatalf("reconciled progress = %d, want %d", got, want)
				}
				h := &questflow.QuestHandler{
					QuestCatalog: catalogs.Quest, Granter: &store.PossessionGranter{},
					Config: &masterdata.GameConfig{ConsumableItemIdForQuestSkipTicket: 7},
				}
				user.ConsumableItems[7] = 24
				for eventType := int32(1); eventType <= 12; eventType++ {
					questId := eventType * 10
					before := store.CloneUserState(*user)
					if skip {
						if _, err := h.HandleQuestSkip(user, questId, int32(model.QuestTypeEvent), eventType, 0, 2, 200); err != nil {
							t.Fatal(err)
						}
						want += 2
					} else {
						h.HandleQuestFinish(user, questId, false, false, 200)
						want++
					}
					events := user.PendingMissionEvents
					user.PendingMissionEvents = nil
					Apply(catalogs, &before, user, events, 200)
					Sync(catalogs, user, 300)
					if got := user.Missions[1].ProgressValue; got != want {
						t.Fatalf("progress after type %d clear = %d, want %d", eventType, got, want)
					}
				}
			})
		}
	}
}

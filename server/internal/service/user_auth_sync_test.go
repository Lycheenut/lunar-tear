package service

import (
	"encoding/json"
	"testing"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestBuildAuthMainQuestDiffDeletesCachedReplayStatus(t *testing.T) {
	user := store.UserState{UserId: 2}
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeMainFlow)
	user.PortalCageStatus.IsCurrentProgress = true

	diff := buildAuthMainQuestDiff(user)
	for _, table := range authMainQuestSyncTables {
		if _, ok := diff[table]; !ok {
			t.Fatalf("missing authoritative auth diff for %s", table)
		}
	}

	replay := diff["IUserMainQuestReplayFlowStatus"]
	if replay.UpdateRecordsJson != "[]" {
		t.Fatalf("replay updates = %s, want []", replay.UpdateRecordsJson)
	}
	var deletes []struct {
		UserId int64 `json:"userId"`
	}
	if err := json.Unmarshal([]byte(replay.DeleteKeysJson), &deletes); err != nil {
		t.Fatal(err)
	}
	if len(deletes) != 1 || deletes[0].UserId != user.UserId {
		t.Fatalf("replay deletes = %s, want userId %d", replay.DeleteKeysJson, user.UserId)
	}
}

func TestBuildAuthMainQuestDiffKeepsActiveReplayStatus(t *testing.T) {
	user := store.UserState{UserId: 2}
	user.MainQuest.CurrentQuestFlowType = int32(model.QuestFlowTypeReplayFlow)
	user.MainQuest.ReplayFlowCurrentQuestSceneId = 848
	user.MainQuest.ReplayFlowHeadQuestSceneId = 848

	replay := buildAuthMainQuestDiff(user)["IUserMainQuestReplayFlowStatus"]
	if replay.UpdateRecordsJson == "[]" {
		t.Fatal("active replay status was not included in auth diff")
	}
	if replay.DeleteKeysJson != "[]" {
		t.Fatalf("active replay deletes = %s, want []", replay.DeleteKeysJson)
	}
}

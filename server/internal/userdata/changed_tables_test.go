package userdata

import (
	"encoding/json"
	"testing"

	"lunar-tear/server/internal/store"
)

func TestComputeDeltaDeletesReplayFlowStatus(t *testing.T) {
	before := store.UserState{UserId: 2}
	before.MainQuest.ReplayFlowCurrentQuestSceneId = 848
	before.MainQuest.ReplayFlowHeadQuestSceneId = 848
	after := before
	after.MainQuest.ReplayFlowCurrentQuestSceneId = 0
	after.MainQuest.ReplayFlowHeadQuestSceneId = 0

	diff := ComputeDelta(&before, &after, []string{"IUserMainQuestReplayFlowStatus"})
	replay := diff["IUserMainQuestReplayFlowStatus"]
	if replay.UpdateRecordsJson != "[]" {
		t.Fatalf("replay updates = %s, want []", replay.UpdateRecordsJson)
	}
	var deletes []map[string]any
	if err := json.Unmarshal([]byte(replay.DeleteKeysJson), &deletes); err != nil {
		t.Fatal(err)
	}
	if len(deletes) != 1 || deletes[0]["userId"] != float64(2) {
		t.Fatalf("replay deletes = %s, want userId 2", replay.DeleteKeysJson)
	}
}

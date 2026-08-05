package questflow

import (
	"bytes"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestBattleCheckpointLifecycle(t *testing.T) {
	const questID int32 = 10
	handler := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{questID: {}},
	}}
	user := store.SeedUserState(1, "battle-checkpoint", 1, model.ClientPlatform{})
	checkpoint := []byte{0x10, 0x20, 0x30}
	user.BattleBinary = append([]byte(nil), checkpoint...)

	handler.HandleExtraQuestRestart(user, questID, 100)
	if !bytes.Equal(user.BattleBinary, checkpoint) {
		t.Fatalf("restart checkpoint = %x, want %x", user.BattleBinary, checkpoint)
	}

	if err := handler.HandleExtraQuestStart(user, questID, 1, 200); err != nil {
		t.Fatal(err)
	}
	if len(user.BattleBinary) != 0 {
		t.Fatalf("new quest retained stale checkpoint %x", user.BattleBinary)
	}

	user.BattleBinary = append([]byte(nil), checkpoint...)
	handler.HandleExtraQuestFinish(user, questID, true, false, 300)
	if len(user.BattleBinary) != 0 {
		t.Fatalf("finished quest retained checkpoint %x", user.BattleBinary)
	}
}

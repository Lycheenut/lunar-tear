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
	detail := store.BattleMissionDetailState{IsValid: true, CostumeSkillUseCount: 11, WeaponSkillUseCount: 6, CompanionSkillUseCount: 2}
	user.Battle = store.BattleState{LastFinishedAt: 50, MissionDetail: detail}
	user.Quests[questID] = store.UserQuestState{QuestId: questID, QuestStateType: model.UserQuestStateTypeActive, LatestStartDatetime: 1}

	if err := handler.HandleExtraQuestRestart(user, questID, 100); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(user.BattleBinary, checkpoint) {
		t.Fatalf("restart checkpoint = %x, want %x", user.BattleBinary, checkpoint)
	}
	if user.Battle.MissionDetail != detail {
		t.Fatal("resuming a quest discarded prior wave mission counts")
	}

	if err := handler.HandleExtraQuestStart(user, questID, 1, 200); err != nil {
		t.Fatal(err)
	}
	if len(user.BattleBinary) != 0 {
		t.Fatalf("new quest retained stale checkpoint %x", user.BattleBinary)
	}
	if user.Battle.MissionDetail != (store.BattleMissionDetailState{}) || user.Battle.LastFinishedAt != 0 {
		t.Fatal("new quest retained previous quest mission counts")
	}

	user.BattleBinary = append([]byte(nil), checkpoint...)
	user.Battle = store.BattleState{IsActive: true, LastFinishedAt: 250, MissionDetail: detail}
	handler.HandleExtraQuestFinish(user, questID, true, false, 300)
	if len(user.BattleBinary) != 0 {
		t.Fatalf("finished quest retained checkpoint %x", user.BattleBinary)
	}
	if user.Battle.MissionDetail != (store.BattleMissionDetailState{}) || user.Battle.LastFinishedAt != 0 || user.Battle.IsActive {
		t.Fatal("retired quest retained active battle mission counts")
	}
}

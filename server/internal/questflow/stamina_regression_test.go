package questflow

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestRepeatedEventQuestLevelUpPreservesOverflowStamina(t *testing.T) {
	const (
		chapterId = int32(7)
		questId   = int32(100026)
	)
	thresholds := make([]int32, 40)
	for level := 2; level <= 38; level++ {
		thresholds[level] = int32(level)
	}
	thresholds[39] = 480_625

	h := &QuestHandler{
		QuestCatalog: &masterdata.QuestCatalog{
			QuestById: map[int32]masterdata.EntityMQuest{
				questId: {QuestId: questId, Stamina: 10, UserExp: 288},
			},
			EventChapterById: map[int32]masterdata.EntityMEventQuestChapter{
				chapterId: {EventQuestChapterId: chapterId},
			},
			EventQuestIdsByChapterId: map[int32][]int32{chapterId: {questId}},
			MaxStaminaByLevel:        map[int32]int32{38: 87, 39: 88},
			UserExpThresholds:        thresholds,
		},
		Config:  &masterdata.GameConfig{},
		Granter: &store.PossessionGranter{},
	}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Status.Level = 38
	user.Status.Exp = 480_625 - 10*288
	user.Status.StaminaMilliValue = 968_000

	for run := 1; run <= 10; run++ {
		nowMillis := int64(run)
		if err := h.HandleEventQuestStart(user, chapterId, questId, false, 1, nowMillis); err != nil {
			t.Fatalf("start run %d: %v", run, err)
		}
		h.HandleEventQuestFinish(user, chapterId, questId, false, false, nowMillis)
	}

	if got := user.Status.Level; got != 39 {
		t.Fatalf("level = %d, want 39", got)
	}
	if got := user.Status.StaminaMilliValue; got != 868_000 {
		t.Fatalf("stamina = %d, want 868000 after ten 10-stamina runs", got)
	}
}

package masterdata

import (
	"path/filepath"
	"slices"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestSecretStoryFiveChainsProgressIndependently(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	resolver, err := LoadConditionResolver()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadGimmickCatalog(resolver, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The second story chain has separate Sun/Moon route sequences, with the
	// same story rewards. Keep the order from the shipped master data.
	chains := [][]int32{
		{900305, 901005, 900505, 900705, 900205, 900905, 901205, 900605, 901105, 900105, 900405, 900805, 902905},
		{901505, 901605, 901305, 901405, 902605, 901805, 902505, 901705},
		{902105, 902205, 901905, 902005, 902805, 902405, 902705, 902305},
	}
	user := store.SeedUserState(1, "story-chains", 1, model.ClientPlatform{})
	const nowMillis = int64(1800000000000)
	for _, chain := range chains {
		key := store.GimmickSequenceKey{GimmickSequenceScheduleId: chain[0], GimmickSequenceId: chain[0]}
		if catalog.SequenceAvailable(user, key.GimmickSequenceScheduleId, key.GimmickSequenceId, nowMillis) {
			t.Fatal("story chain opened before its chapter was cleared")
		}
		questId, ok := resolver.RequiredQuestId(catalog.scheduleByKey[key].ReleaseConditionId)
		if !ok {
			t.Fatalf("missing chapter condition for %d", chain[0])
		}
		user.Quests[questId] = store.UserQuestState{QuestId: questId, QuestStateType: model.UserQuestStateTypeCleared}
	}
	frontiers := make([]int, len(chains))
	check := func() {
		t.Helper()
		active := catalog.ActiveScheduleKeys(*user, nowMillis)
		for c, chain := range chains {
			for i, sequenceId := range chain {
				key := store.GimmickSequenceKey{GimmickSequenceScheduleId: chain[0], GimmickSequenceId: sequenceId}
				want := i <= frontiers[c]
				if got := catalog.SequenceAvailable(user, chain[0], sequenceId, nowMillis); got != want {
					t.Errorf("chain %d at step %d: sequence %d available = %v, want %v", chain[0], frontiers[c], sequenceId, got, want)
				}
				if got := slices.Contains(active, key); got != want {
					t.Errorf("chain %d at step %d: sequence %d active = %v, want %v", chain[0], frontiers[c], sequenceId, got, want)
				}
				for gimmickId := range catalog.gimmicksBySequence[sequenceId] {
					if got := catalog.GimmickUnlockAvailable(user, chain[0], sequenceId, gimmickId, nowMillis); got != want {
						t.Errorf("gimmick %d unlock available = %v, want %v", gimmickId, got, want)
					}
					if catalog.GimmickAvailable(user, chain[0], sequenceId, gimmickId, nowMillis) {
						t.Errorf("gimmick %d ignored its own mission requirements", gimmickId)
					}
				}
			}
		}
	}
	check()
	if t.Failed() {
		t.FailNow()
	}
	// Unlocking a marker and creating a sequence row do not complete its story.
	for _, chain := range chains {
		key := store.GimmickSequenceKey{GimmickSequenceScheduleId: chain[0], GimmickSequenceId: chain[0]}
		user.Gimmick.Sequences[key] = store.GimmickSequenceState{Key: key}
		for gimmickId := range catalog.gimmicksBySequence[chain[0]] {
			gimmickKey := store.GimmickKey{GimmickSequenceScheduleId: chain[0], GimmickSequenceId: chain[0], GimmickId: gimmickId}
			user.Gimmick.Unlocks[gimmickKey] = store.GimmickUnlockState{Key: gimmickKey, IsUnlocked: true}
		}
	}
	check()
	// Interleave all routes so advancing one must leave every other frontier alone.
	for step := 0; step < len(chains[0]); step++ {
		for c, chain := range chains {
			if step >= len(chain) {
				continue
			}
			key := store.GimmickSequenceKey{GimmickSequenceScheduleId: chain[0], GimmickSequenceId: chain[step]}
			user.Gimmick.Sequences[key] = store.GimmickSequenceState{Key: key, IsGimmickSequenceCleared: true}
			frontiers[c]++
			check()
		}
	}
}

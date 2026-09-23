package userdata

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func loadGimmickProjectionCatalog(t *testing.T) (*masterdata.GimmickCatalog, *masterdata.ConditionResolver) {
	t.Helper()
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	resolver, err := masterdata.LoadConditionResolver()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := masterdata.LoadGimmickCatalog(resolver, nil)
	if err != nil {
		t.Fatal(err)
	}
	previous := gimmickCatalog.Load()
	SetGimmickCatalog(catalog)
	t.Cleanup(func() { SetGimmickCatalog(previous) })
	return catalog, resolver
}

func hasGimmickSequenceJSON(payload string, scheduleId, sequenceId int32) bool {
	for _, row := range parseJSONRecords(payload) {
		if row["gimmickSequenceScheduleId"] == float64(scheduleId) && row["gimmickSequenceId"] == float64(sequenceId) {
			return true
		}
	}
	return false
}

func TestSecretStoryMapOnlyProjectsAvailableEntries(t *testing.T) {
	catalog, resolver := loadGimmickProjectionCatalog(t)
	chains := masterdata.LoadGimmickSequenceChains()
	refs := masterdata.LoadGimmickOrnamentRefs()
	schedules, err := memorydb.ReadTable[masterdata.EntityMGimmickSequenceSchedule]("m_gimmick_sequence_schedule")
	if err != nil {
		t.Fatal(err)
	}
	questBySchedule := map[int32]int32{}
	for _, s := range schedules {
		if questId, ok := resolver.RequiredQuestId(s.ReleaseEvaluateConditionId); ok {
			questBySchedule[s.GimmickSequenceScheduleId] = questId
		}
	}
	tables := []string{"IUserGimmick", "IUserGimmickOrnamentProgress", "IUserGimmickSequence", "IUserGimmickUnlock"}
	// Cover No. 1-4, No. 6-10, and both No. 5 chains including Sun/Moon routes.
	for _, root := range []int32{900101, 900106, 900305, 901505, 902105} {
		chain := chains[root]
		if len(chain) < 3 {
			t.Fatalf("short chain %d: %v", root, chain)
		}
		user := store.SeedUserState(1, "map-story", 1, model.ClientPlatform{})
		for _, sequenceId := range chain {
			seqKey := store.GimmickSequenceKey{GimmickSequenceScheduleId: root, GimmickSequenceId: sequenceId}
			user.Gimmick.Sequences[seqKey] = store.GimmickSequenceState{Key: seqKey}
			// Old placeholder records must not force locked markers onto the map.
			for _, ref := range refs[sequenceId] {
				key := store.GimmickKey{GimmickSequenceScheduleId: root, GimmickSequenceId: sequenceId, GimmickId: ref.GimmickId}
				user.Gimmick.Progress[key] = store.GimmickProgressState{Key: key}
				user.Gimmick.Unlocks[key] = store.GimmickUnlockState{Key: key}
				ornamentKey := store.GimmickOrnamentKey{GimmickSequenceScheduleId: root, GimmickSequenceId: sequenceId, GimmickId: ref.GimmickId, GimmickOrnamentIndex: ref.OrnamentIndex}
				user.Gimmick.OrnamentProgress[ornamentKey] = store.GimmickOrnamentProgressState{Key: ornamentKey}
			}
		}
		for _, table := range tables {
			if got := projectTable(table, *user); got != "[]" {
				t.Fatalf("%s exposed story chain %d before chapter unlock", table, root)
			}
		}
		questId, ok := questBySchedule[root]
		if !ok {
			t.Fatalf("missing chapter for %d", root)
		}
		beforeChapter := store.CloneUserState(*user)
		user.Quests[questId] = store.UserQuestState{QuestId: questId, QuestStateType: model.UserQuestStateTypeCleared}
		chapterDiff := ComputeDelta(&beforeChapter, user, ChangedTables(&beforeChapter, user))
		refresh := GimmickRefreshDiff(*user, *user)
		for _, table := range tables {
			if change := chapterDiff[table]; change == nil || !hasGimmickSequenceJSON(change.UpdateRecordsJson, root, root) {
				t.Errorf("%s did not reveal chain %d after chapter unlock", table, root)
			}
			payload := projectTable(table, *user)
			for i, sequenceId := range chain {
				if got := hasGimmickSequenceJSON(payload, root, sequenceId); got != (i == 0) {
					t.Errorf("%s chain %d sequence %d visible=%v, want %v", table, root, sequenceId, got, i == 0)
				}
				if got := hasGimmickSequenceJSON(refresh[table].DeleteKeysJson, root, sequenceId); got != (i > 0) {
					t.Errorf("%s cached sequence %d deleted=%v, want %v", table, sequenceId, got, i > 0)
				}
			}
		}
		if len(user.Gimmick.Sequences) != len(chain) || len(user.Gimmick.Unlocks) != len(chain) || len(user.Gimmick.Progress) != len(chain) {
			t.Fatal("map filtering changed the saved story records")
		}
		ref := refs[root][0]
		if !catalog.GimmickUnlockAvailable(user, root, root, ref.GimmickId, gametime.NowMillis()) || catalog.GimmickAvailable(user, root, root, ref.GimmickId, gametime.NowMillis()) {
			t.Fatal("fixture must be viewable while its own mission remains incomplete")
		}
		before := store.CloneUserState(*user)
		seqKey := store.GimmickSequenceKey{GimmickSequenceScheduleId: root, GimmickSequenceId: root}
		user.Gimmick.Sequences[seqKey] = store.GimmickSequenceState{Key: seqKey, IsGimmickSequenceCleared: true}
		diff := ComputeDelta(&before, user, ChangedTables(&before, user))
		for _, table := range tables {
			change := diff[table]
			if change == nil || !hasGimmickSequenceJSON(change.UpdateRecordsJson, root, chain[1]) {
				t.Errorf("%s did not immediately reveal chain %d's next entry", table, root)
			}
			if hasGimmickSequenceJSON(projectTable(table, *user), root, chain[2]) {
				t.Errorf("%s revealed a later entry in chain %d", table, root)
			}
		}
	}
}

func TestSecretStoryMapSyntheticAndLegacyEntries(t *testing.T) {
	catalog, resolver := loadGimmickProjectionCatalog(t)
	for _, tt := range []struct {
		name                             string
		clearRoot, unlockNext, clearNext bool
	}{
		{name: "fresh chain"},
		{name: "predecessor collected", clearRoot: true},
		{name: "legacy unlock", unlockNext: true},
		{name: "legacy progress without sequence", clearNext: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			user := store.SeedUserState(1, "map-story", 1, model.ClientPlatform{})
			for _, conditionId := range []int32{4104, 4204} {
				questId, ok := resolver.RequiredQuestId(conditionId)
				if !ok {
					t.Fatal("missing chapter condition")
				}
				user.Quests[questId] = store.UserQuestState{QuestId: questId, QuestStateType: model.UserQuestStateTypeCleared}
			}
			for _, root := range []int32{900305, 901505} {
				key := store.GimmickSequenceKey{GimmickSequenceScheduleId: root, GimmickSequenceId: root}
				user.Gimmick.Sequences[key] = store.GimmickSequenceState{Key: key}
			}
			first := store.GimmickKey{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 900305, GimmickId: 91034005}
			next := store.GimmickKey{GimmickSequenceScheduleId: 900305, GimmickSequenceId: 901005, GimmickId: 91104005}
			if tt.clearRoot {
				user.Gimmick.Progress[first] = store.GimmickProgressState{Key: first, IsGimmickCleared: true}
			}
			if tt.unlockNext {
				user.Gimmick.Unlocks[next] = store.GimmickUnlockState{Key: next, IsUnlocked: true}
			}
			if tt.clearNext {
				user.Gimmick.Progress[next] = store.GimmickProgressState{Key: next, IsGimmickCleared: true}
			}
			for _, table := range []string{"IUserGimmick", "IUserGimmickOrnamentProgress"} {
				payload := projectTable(table, *user)
				for _, expected := range []struct {
					scheduleId, sequenceId int32
					visible                bool
				}{
					{900305, 900305, true},
					{900305, 901005, tt.clearRoot || tt.unlockNext || tt.clearNext},
					{900305, 900505, tt.clearNext},
					{901505, 901505, true},
					{901505, 901605, false},
				} {
					if got := hasGimmickSequenceJSON(payload, expected.scheduleId, expected.sequenceId); got != expected.visible {
						t.Errorf("%s sequence %d visible=%v, want %v", table, expected.sequenceId, got, expected.visible)
					}
				}
				for _, row := range parseJSONRecords(payload) {
					if !catalog.GimmickUnlockAvailable(user, int32(row["gimmickSequenceScheduleId"].(float64)), int32(row["gimmickSequenceId"].(float64)), int32(row["gimmickId"].(float64)), gametime.NowMillis()) {
						t.Errorf("%s exposed an entry that the unlock endpoint rejects: %v", table, row)
					}
				}
			}
		})
	}
}

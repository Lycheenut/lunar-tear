package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/internal/userdata"
	"lunar-tear/server/migrations"
)

func TestLimitContentDeckRejectsReusedTargetsAndRecordsOnce(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Decks[store.DeckKey{DeckType: model.DeckTypeRestrictedLimitContentQuest, UserDeckNumber: 1}] = store.DeckState{UserDeckCharacterUuid01: "dc"}
	user.DeckCharacters["dc"] = store.DeckCharacterState{UserCostumeUuid: "costume", MainUserWeaponUuid: "weapon"}
	user.Costumes["costume"] = store.CostumeState{UserCostumeUuid: "costume"}
	user.Weapons["weapon"] = store.WeaponState{UserWeaponUuid: "weapon"}
	catalog := &masterdata.LimitContentCatalog{ContentsByChapter: map[int32][]masterdata.EntityMEventQuestLimitContent{500: {{StartDatetime: 1, EndDatetime: 100, EventQuestLimitContentDeckRestrictionId: 1}}}, RestrictionsById: map[int32][]masterdata.EntityMEventQuestLimitContentDeckRestriction{1: {{EventQuestLimitContentDeckRestrictionTargetId: 1, StartDatetime: 1, EndDatetime: 100}}}, TargetTypesById: map[int32][]int32{1: {1}}}
	if err := recordLimitContentDeck(user, catalog, 500, 10, 1, 50); err != nil {
		t.Fatal(err)
	}
	if len(user.DeckLimitContentRestricted) != 1 {
		t.Fatalf("restricted records = %d", len(user.DeckLimitContentRestricted))
	}
	if err := validateLimitContentDeck(user, catalog, 500, 1, 50); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("reuse status = %v", status.Code(err))
	}
	if err := recordLimitContentDeck(user, catalog, 500, 11, 1, 50); err != nil {
		t.Fatal(err)
	}
	if len(user.DeckLimitContentRestricted) != 1 {
		t.Fatalf("duplicate restricted records = %d", len(user.DeckLimitContentRestricted))
	}
}

func TestLimitContentDeckRequiresRestrictedDeck(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	catalog := &masterdata.LimitContentCatalog{ContentsByChapter: map[int32][]masterdata.EntityMEventQuestLimitContent{500: {{StartDatetime: 1, EndDatetime: 100, EventQuestLimitContentDeckRestrictionId: 1}}}, RestrictionsById: map[int32][]masterdata.EntityMEventQuestLimitContentDeckRestriction{1: {{EventQuestLimitContentDeckRestrictionTargetId: 1, StartDatetime: 1, EndDatetime: 100}}}, TargetTypesById: map[int32][]int32{1: {1}}}
	if err := validateLimitContentDeck(user, catalog, 500, 1, 50); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("missing deck status = %v", status.Code(err))
	}
}

func TestFinishEventQuestReleasesCompletedDifficultyDeck(t *testing.T) {
	masterData, err := os.ReadFile(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if err != nil {
		t.Fatal(err)
	}
	masterDataPath := filepath.Join(t.TempDir(), "master-data.bin.e")
	if err := os.WriteFile(masterDataPath, masterData, 0o600); err != nil {
		t.Fatal(err)
	}
	holder, err := runtime.NewHolder(masterDataPath)
	if err != nil {
		t.Fatal(err)
	}
	const chapterId = int32(500001)
	cat := holder.Get()

	for _, tt := range []struct {
		name          string
		difficulty    int32
		finishIndex   int
		incomplete    bool
		isRetired     bool
		isAnnihilated bool
		wantUnlocked  bool
	}{
		{name: "normal complete", difficulty: 1, finishIndex: 2, wantUnlocked: true},
		{name: "hard complete", difficulty: 2, finishIndex: 2, wantUnlocked: true},
		{name: "highest difficulty complete", difficulty: 3, finishIndex: 2, wantUnlocked: true},
		{name: "completed out of order", difficulty: 1, finishIndex: 0, wantUnlocked: true},
		{name: "another quest uncleared", difficulty: 1, finishIndex: 2, incomplete: true},
		{name: "retired", difficulty: 1, finishIndex: 2, isRetired: true},
		{name: "annihilated", difficulty: 1, finishIndex: 2, isAnnihilated: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := migrations.Up(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			repo := sqlite.New(db, nil)
			userId, err := repo.CreateUser("limit-difficulty", model.ClientPlatform{})
			if err != nil {
				t.Fatal(err)
			}
			questIds := cat.Quest.EventQuestIdsByChapterDifficulty[chapterId][tt.difficulty]
			if len(questIds) != 3 {
				t.Fatalf("difficulty quests = %v, want three quests", questIds)
			}
			questId := questIds[tt.finishIndex]
			otherDifficultyQuestId := cat.Quest.EventQuestIdsByChapterDifficulty[chapterId][tt.difficulty%3+1][0]
			if _, err := repo.UpdateUser(userId, func(user *store.UserState) {
				for _, id := range cat.Quest.EventUnlockQuestIdsForChapter(chapterId) {
					user.Quests[id] = store.UserQuestState{QuestId: id, QuestStateType: model.UserQuestStateTypeCleared}
				}
				for _, id := range questIds {
					user.Quests[id] = store.UserQuestState{QuestId: id, QuestStateType: model.UserQuestStateTypeCleared}
				}
				if tt.incomplete {
					delete(user.Quests, questIds[0])
				}
				user.Quests[questId] = store.UserQuestState{QuestId: questId, QuestStateType: model.UserQuestStateTypeActive, UserDeckNumber: 1}
				user.EventQuest = store.EventQuestState{CurrentEventQuestChapterId: chapterId, CurrentQuestId: questId}
				user.Decks[store.DeckKey{DeckType: model.DeckTypeRestrictedLimitContentQuest, UserDeckNumber: 1}] = store.DeckState{UserDeckCharacterUuid01: "dc"}
				user.DeckCharacters["dc"] = store.DeckCharacterState{UserDeckCharacterUuid: "dc", UserCostumeUuid: "costume", MainUserWeaponUuid: "weapon"}
				user.Costumes["costume"] = store.CostumeState{UserCostumeUuid: "costume"}
				user.Weapons["weapon"] = store.WeaponState{UserWeaponUuid: "weapon"}
				for _, row := range []store.DeckLimitContentRestrictedState{
					{DeckRestrictedUuid: "previous-costume", EventQuestChapterId: chapterId, QuestId: questIds[1], PossessionType: int32(model.PossessionTypeCostume), TargetUuid: "previous-costume"},
					{DeckRestrictedUuid: "previous-weapon", EventQuestChapterId: chapterId, QuestId: questIds[1], PossessionType: int32(model.PossessionTypeWeapon), TargetUuid: "previous-weapon"},
					{DeckRestrictedUuid: "other-difficulty", EventQuestChapterId: chapterId, QuestId: otherDifficultyQuestId, PossessionType: int32(model.PossessionTypeCostume), TargetUuid: "other-costume"},
					{DeckRestrictedUuid: "other-chapter", EventQuestChapterId: chapterId + 1, QuestId: questIds[1], PossessionType: int32(model.PossessionTypeCostume), TargetUuid: "costume"},
				} {
					user.DeckLimitContentRestricted[row.DeckRestrictedUuid] = row
				}
			}); err != nil {
				t.Fatal(err)
			}
			before, err := repo.LoadUser(userId)
			if err != nil {
				t.Fatal(err)
			}
			server := NewQuestServiceServer(repo, repo, holder)
			if _, err := server.FinishEventQuest(context.Background(), &pb.FinishEventQuestRequest{
				EventQuestChapterId: chapterId, QuestId: questId, IsRetired: tt.isRetired, IsAnnihilated: tt.isAnnihilated,
			}); err != nil {
				t.Fatal(err)
			}
			after, err := repo.LoadUser(userId)
			if err != nil {
				t.Fatal(err)
			}
			var restrictions int
			for _, row := range after.DeckLimitContentRestricted {
				if row.EventQuestChapterId == chapterId && slices.Contains(questIds, row.QuestId) {
					restrictions++
				}
			}
			wantRestrictions := 0
			if !tt.wantUnlocked {
				wantRestrictions = 2
				if !tt.isRetired && !tt.isAnnihilated {
					wantRestrictions++
				}
			}
			if restrictions != wantRestrictions {
				t.Fatalf("restrictions immediately after finish = %d, want %d", restrictions, wantRestrictions)
			}
			for _, id := range []string{"other-difficulty", "other-chapter"} {
				if after.DeckLimitContentRestricted[id] != before.DeckLimitContentRestricted[id] {
					t.Fatalf("unrelated restriction %s changed", id)
				}
			}
			if tt.wantUnlocked {
				delta := userdata.ComputeDelta(&before, &after, userdata.ChangedTables(&before, &after))["IUserDeckLimitContentRestricted"]
				if delta == nil {
					t.Fatal("finish did not synchronize unlocked decks")
				}
				var deletes []struct {
					DeckRestrictedUuid string `json:"deckRestrictedUuid"`
				}
				if err := json.Unmarshal([]byte(delta.DeleteKeysJson), &deletes); err != nil {
					t.Fatal(err)
				}
				if len(deletes) != 2 || delta.UpdateRecordsJson != "[]" {
					t.Fatalf("deck restriction delta = %+v, want only the two previous restrictions deleted", delta)
				}
				for _, questId := range cat.Quest.EventQuestIdsByChapterDifficulty[chapterId][tt.difficulty%3+1] {
					if _, err := server.StartEventQuest(context.Background(), &pb.StartEventQuestRequest{
						EventQuestChapterId: chapterId, QuestId: questId, UserDeckNumber: 1,
					}); err != nil {
						t.Fatalf("reuse last deck for quest %d: %v", questId, err)
					}
				}
			}
		})
	}
}

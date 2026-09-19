package service

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/internal/userdata"
	"lunar-tear/server/migrations"

	"google.golang.org/protobuf/types/known/emptypb"
)

func TestLoginBonusUnlockRewardTutorialTransition(t *testing.T) {
	masterData, err := os.ReadFile(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if err != nil {
		t.Fatal(err)
	}
	masterPath := filepath.Join(t.TempDir(), "master.bin.e")
	if err := os.WriteFile(masterPath, masterData, 0o600); err != nil {
		t.Fatal(err)
	}
	holder, err := runtime.NewHolder(masterPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, withDeck := range []bool{false, true} {
		name := "progress"
		if withDeck {
			name = "progress and deck"
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name              string
				before, next, typ int32
				wantReward        bool
			}{
				{"unlock boundary", 19, 20, 3, true},
				{"skip to complete", 0, 99999, 3, true},
				{"still locked", 0, 19, 3, false},
				{"different tutorial", 0, 20, 1, false},
				{"already unlocked", 20, 20, 3, false},
				{"completed old account", 99999, 20, 3, false},
				{"stale request", 20, 19, 3, false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
					if err != nil {
						t.Fatal(err)
					}
					defer db.Close()
					if err := migrations.Up(context.Background(), db); err != nil {
						t.Fatal(err)
					}
					repo := sqlite.New(db, nil)
					id, err := repo.CreateUser("unlock-reward", model.ClientPlatform{})
					if err != nil {
						t.Fatal(err)
					}
					before, err := repo.UpdateUser(id, func(user *store.UserState) {
						user.Tutorials[3] = store.TutorialProgressState{TutorialType: 3, ProgressPhase: tc.before}
						// Chapter completion alone must not bypass the tutorial gate.
						user.Quests[11] = store.UserQuestState{QuestId: 11, ClearCount: 1}
					})
					if err != nil {
						t.Fatal(err)
					}
					tutorial := NewTutorialServiceServer(repo, repo, holder)
					ctx := context.Background()
					trigger := func() error {
						if withDeck {
							_, err := tutorial.SetTutorialProgressAndReplaceDeck(ctx, &pb.SetTutorialProgressAndReplaceDeckRequest{TutorialType: tc.typ, ProgressPhase: tc.next})
							return err
						}
						_, err := tutorial.SetTutorialProgress(ctx, &pb.SetTutorialProgressRequest{TutorialType: tc.typ, ProgressPhase: tc.next})
						return err
					}
					// Concurrent retries at the unlock boundary must produce one bundle.
					var wg sync.WaitGroup
					for range 4 {
						wg.Go(func() {
							if err := trigger(); err != nil {
								t.Error(err)
							}
						})
					}
					wg.Wait()
					after, err := repo.LoadUser(id)
					if err != nil {
						t.Fatal(err)
					}
					if !tc.wantReward {
						if len(after.Weapons) != len(before.Weapons) || len(after.Costumes) != len(before.Costumes) || len(after.Materials) != len(before.Materials) || len(after.Companions) != len(before.Companions) {
							t.Fatal("request without an unlock transition granted the bundle")
						}
						return
					}
					if len(after.Costumes) != len(before.Costumes)+1 || len(after.Weapons) != len(before.Weapons)+1 {
						t.Fatal("expected one costume and one weapon")
					}
					if len(after.Companions) != 23 {
						t.Fatalf("expected 23 companions before the companion tutorial, got %d", len(after.Companions))
					}
					owned := make(map[int32]bool)
					for _, companion := range after.Companions {
						wantLevel := int32(1)
						if companion.CompanionId >= 49 && companion.CompanionId <= 51 {
							wantLevel = 50
						}
						if companion.Level != wantLevel || owned[companion.CompanionId] {
							t.Fatalf("incorrect or duplicate companion: %+v", companion)
						}
						owned[companion.CompanionId] = true
					}
					for id := int32(31); id <= 53; id++ {
						if !owned[id] {
							t.Fatalf("missing non-main companion %d", id)
						}
					}
					// The client unlocks companion deck slots from this tutorial,
					// independently of inventory. Early gifting must not advance it.
					if after.Tutorials[8] != before.Tutorials[8] {
						t.Fatal("login reward advanced the companion tutorial")
					}
					if len(after.Gifts.NotReceived) != len(before.Gifts.NotReceived) {
						t.Fatal("unlock reward entered the gift box")
					}
					for id, count := range map[int32]int32{311211: 40, 313197: 5, 312011: 4} {
						if after.Materials[id]-before.Materials[id] != count {
							t.Errorf("material %d: expected delta %d", id, count)
						}
					}
					diff := userdata.ComputeDelta(&before, &after, userdata.ChangedTables(&before, &after))
					for _, table := range []string{"IUserCostume", "IUserCostumeActiveSkill", "IUserWeapon", "IUserWeaponSkill", "IUserWeaponAbility", "IUserMaterial", "IUserCompanion"} {
						if diff[table] == nil {
							t.Errorf("inventory missing from client diff: %s", table)
						}
					}
					consumed, err := repo.UpdateUser(id, func(user *store.UserState) {
						for uuid, weapon := range user.Weapons {
							if weapon.WeaponId == 240271 {
								delete(user.Weapons, uuid)
								delete(user.WeaponSkills, uuid)
								delete(user.WeaponAbilities, uuid)
							}
						}
						for _, id := range []int32{311211, 313197, 312011} {
							delete(user.Materials, id)
						}
						for uuid, companion := range user.Companions {
							if companion.CompanionId == 31 {
								delete(user.Companions, uuid)
							}
						}
					})
					if err != nil {
						t.Fatal(err)
					}
					if err := trigger(); err != nil {
						t.Fatal(err)
					}
					if _, err := NewUserServiceServer(repo, repo, holder, "", false).Auth(ctx, &pb.AuthUserRequest{Uuid: "unlock-reward"}); err != nil {
						t.Fatal(err)
					}
					if _, err := NewLoginBonusServiceServer(repo, repo, holder).ReceiveStamp(ctx, &emptypb.Empty{}); err != nil {
						t.Fatal(err)
					}
					replayed, err := repo.LoadUser(id)
					if err != nil {
						t.Fatal(err)
					}
					if len(replayed.Weapons) != len(consumed.Weapons) || len(replayed.Materials) != len(consumed.Materials) || len(replayed.Companions) != len(consumed.Companions) {
						t.Fatal("replay, login or stamp receipt regranted the consumed bundle")
					}
				})
			}
		})
	}
}

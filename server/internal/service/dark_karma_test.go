package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func TestDarkKarmaAcquisitionAndLoginRepair(t *testing.T) {
	const stone = 315002
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	id, err := repo.CreateUser("dark-karma", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	holder := newGachaResponseTestHolder(t)
	granter := holder.Get().QuestHandler.Granter
	server := NewUserServiceServer(repo, repo, holder, "", false)
	update := func(f func(*store.UserState)) store.UserState {
		t.Helper()
		user, err := repo.UpdateUser(id, f)
		if err != nil {
			t.Fatal(err)
		}
		return user
	}
	login := func(want int32) {
		t.Helper()
		resp, err := server.Auth(ctx, &pb.AuthUserRequest{Uuid: "dark-karma"})
		if err != nil {
			t.Fatal(err)
		}
		user, err := repo.LoadUser(id)
		if err != nil {
			t.Fatal(err)
		}
		if got, exists := user.Materials[stone]; got != want || exists != (want > 0) {
			t.Fatalf("stored dark karma = %d (exists %t), want %d", got, exists, want)
		}
		diff := resp.DiffUserData["IUserMaterial"]
		payload := diff.GetUpdateRecordsJson()
		if want == 0 {
			payload = diff.GetDeleteKeysJson()
		}
		var rows []struct{ MaterialId, Count int32 }
		if err := json.Unmarshal([]byte(payload), &rows); err != nil || len(rows) != 1 || rows[0].MaterialId != stone || rows[0].Count != want {
			t.Fatalf("client material diff = %v, want %d", diff, want)
		}
	}

	user := update(func(u *store.UserState) {
		u.Materials[stone] = 7
		granter.GrantCostume(u, 31000, 1) // ordinary Karma Stones
		granter.GrantFull(u, model.PossessionTypeCostume, 41000, 2, 1)
		granter.GrantCostume(u, 44004, 1)
	})
	if user.Materials[stone] != 67 {
		t.Fatalf("two distinct dark costumes should add 60 stones, got inventory %d", user.Materials[stone])
	}
	update(func(u *store.UserState) {
		for uuid, costume := range u.Costumes {
			switch costume.CostumeId {
			case 41000:
				costume.CostumeLotteryEffectUnlockedSlotCount = 1
				u.Costumes[uuid] = costume
			case 44004:
				// A legacy snapshot may have the slot rows but a stale count.
				for slot := int32(1); slot <= 3; slot++ {
					key := store.CostumeLotteryEffectKey{UserCostumeUuid: uuid, SlotNumber: slot}
					u.CostumeLotteryEffects[key] = store.CostumeLotteryEffectState{UserCostumeUuid: uuid, SlotNumber: slot}
				}
			}
		}
	})
	login(20) // remove surplus
	update(func(u *store.UserState) { delete(u.Materials, stone) })
	login(20) // backfill deficit
	login(20) // retry
	config := holder.Get().GameConfig
	update(func(u *store.UserState) {
		u.ConsumableItems[config.ConsumableItemIdForGold] = 2 * config.CostumeLotteryEffectUnlockSlotConsumeGold
	})
	costumes := NewCostumeServiceServer(repo, repo, holder)
	for uuid, costume := range user.Costumes {
		if costume.CostumeId == 41000 {
			for slot := int32(2); slot <= 3; slot++ {
				_, err := costumes.UnlockLotteryEffectSlot(ctx, &pb.UnlockLotteryEffectSlotRequest{UserCostumeUuid: uuid, SlotNumber: slot})
				if err != nil {
					t.Fatal(err)
				}
				login((3 - slot) * 10)
			}
		}
	}
}

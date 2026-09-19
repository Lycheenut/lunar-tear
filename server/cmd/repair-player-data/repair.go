package main

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
)

var errPreview = errors.New("read-only preview")

type inventoryChange struct {
	Type, ID             int32
	Before, Delta, After int32
}

type playerChange struct {
	UserID, PlayerID int64
	Inventory        []inventoryChange
	CompanionLevels  []companionLevelChange `json:",omitempty"`
}

type companionLevelChange struct {
	ID, Before, After int32
}

type repairReport struct {
	Applied        bool
	ScannedPlayers int
	Players        []playerChange
}

// repair is intentionally a one-time operation: it does not persist a receipt
// and running it with apply=true again grants the login bundle again.
func repair(db *sql.DB, granter *store.PossessionGranter, apply bool, now int64) (repairReport, error) {
	report := repairReport{}
	if _, ok := granter.CostumeById[24008]; !ok {
		return report, fmt.Errorf("master data is missing costume 24008")
	}
	if _, ok := granter.WeaponById[240271]; !ok {
		return report, fmt.Errorf("master data is missing weapon 240271")
	}
	repo := sqlite.New(db, nil)
	ids, err := repo.ListUserIds()
	if err != nil {
		return report, err
	}
	report.ScannedPlayers = len(ids)
	_, err = repo.UpdateUsers(ids, func(users map[int64]*store.UserState) error {
		for _, id := range ids {
			user := users[id]
			if !store.IsLoginBonusUnlocked(user) {
				continue
			}
			before := rewardInventory(user)
			beforeLevels := make(map[int32]int32, len(user.Companions))
			for _, companion := range user.Companions {
				beforeLevels[companion.CompanionId] = companion.Level
			}
			store.GrantLoginBonusUnlockReward(user, granter, now)
			player := playerChange{UserID: id, PlayerID: user.PlayerId}
			for _, companion := range user.Companions {
				if beforeLevels[companion.CompanionId] != companion.Level {
					player.CompanionLevels = append(player.CompanionLevels, companionLevelChange{
						ID: companion.CompanionId, Before: beforeLevels[companion.CompanionId], After: companion.Level,
					})
				}
			}
			sort.Slice(player.CompanionLevels, func(i, j int) bool { return player.CompanionLevels[i].ID < player.CompanionLevels[j].ID })
			for key, count := range rewardInventory(user) {
				if count < before[key] {
					return fmt.Errorf("user %d: inventory type=%d id=%d would overflow", id, key[0], key[1])
				}
				if count != before[key] {
					player.Inventory = append(player.Inventory, inventoryChange{
						Type: key[0], ID: key[1], Before: before[key], Delta: count - before[key], After: count,
					})
				}
			}
			sort.Slice(player.Inventory, func(i, j int) bool {
				a, b := player.Inventory[i], player.Inventory[j]
				if a.Type != b.Type {
					return a.Type < b.Type
				}
				return a.ID < b.ID
			})
			if len(player.Inventory) > 0 || len(player.CompanionLevels) > 0 {
				report.Players = append(report.Players, player)
			}
		}
		if !apply {
			// Abort before UpdateUsers opens a write transaction. This also works
			// against a database opened with mode=ro.
			return errPreview
		}
		return nil
	})
	if errors.Is(err, errPreview) {
		return report, nil
	}
	if err != nil {
		return report, err
	}
	report.Applied = true
	return report, nil
}

func rewardInventory(user *store.UserState) map[[2]int32]int32 {
	counts := make(map[[2]int32]int32)
	for _, costume := range user.Costumes {
		if costume.CostumeId == 24008 {
			counts[[2]int32{int32(model.PossessionTypeCostume), costume.CostumeId}]++
		}
	}
	for _, weapon := range user.Weapons {
		if weapon.WeaponId == 240271 {
			counts[[2]int32{int32(model.PossessionTypeWeapon), weapon.WeaponId}]++
		}
	}
	for _, companion := range user.Companions {
		counts[[2]int32{int32(model.PossessionTypeCompanion), companion.CompanionId}]++
	}
	// Include duplicate-costume conversion materials in the preview too.
	for id, count := range user.Materials {
		counts[[2]int32{int32(model.PossessionTypeMaterial), id}] = count
	}
	return counts
}

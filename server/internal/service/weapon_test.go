package service

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestCanSellWeapon(t *testing.T) {
	for _, tc := range []struct {
		name        string
		weapon      store.WeaponState
		master      masterdata.EntityMWeapon
		wantCanSell bool
	}{
		{name: "eligible", wantCanSell: true},
		{name: "protected", weapon: store.WeaponState{IsProtected: true}},
		{name: "restricted by master data", master: masterdata.EntityMWeapon{IsRestrictDiscard: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := canSellWeapon(tc.weapon, tc.master); got != tc.wantCanSell {
				t.Fatalf("canSellWeapon() = %v, want %v", got, tc.wantCanSell)
			}
		})
	}
}

func TestValidateMaterialWeaponsRejectsUnsafeWeapons(t *testing.T) {
	catalog := &masterdata.WeaponCatalog{}
	newUser := func() *store.UserState {
		user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
		user.Weapons["target"] = store.WeaponState{UserWeaponUuid: "target", WeaponId: 10}
		user.Weapons["same"] = store.WeaponState{UserWeaponUuid: "same", WeaponId: 10}
		user.Weapons["other"] = store.WeaponState{UserWeaponUuid: "other", WeaponId: 20}
		user.Weapons["protected"] = store.WeaponState{UserWeaponUuid: "protected", WeaponId: 10, IsProtected: true}
		user.Weapons["main"] = store.WeaponState{UserWeaponUuid: "main", WeaponId: 10}
		user.Weapons["sub"] = store.WeaponState{UserWeaponUuid: "sub", WeaponId: 10}
		user.DeckCharacters["deck-character"] = store.DeckCharacterState{
			UserDeckCharacterUuid: "deck-character",
			MainUserWeaponUuid:    "main",
		}
		user.DeckSubWeapons["deck-character"] = []string{"sub"}
		return user
	}

	for _, tc := range []struct {
		name  string
		uuids []string
		code  codes.Code
		max   int32
	}{
		{name: "target", uuids: []string{"target"}, code: codes.InvalidArgument},
		{name: "duplicate", uuids: []string{"same", "same"}, code: codes.InvalidArgument},
		{name: "protected", uuids: []string{"protected"}, code: codes.FailedPrecondition},
		{name: "main weapon", uuids: []string{"main"}, code: codes.FailedPrecondition},
		{name: "sub weapon", uuids: []string{"sub"}, code: codes.FailedPrecondition},
		{name: "different weapon", uuids: []string{"other"}, code: codes.FailedPrecondition},
		{name: "too many", uuids: []string{"same", "other"}, code: codes.FailedPrecondition, max: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateMaterialWeapons(catalog, newUser(), "target", tc.uuids, true, tc.max)
			if status.Code(err) != tc.code {
				t.Fatalf("status = %v, want %v (err=%v)", status.Code(err), tc.code, err)
			}
		})
	}
}

func TestValidateMaterialWeaponsAllowsEligibleCopies(t *testing.T) {
	catalog := &masterdata.WeaponCatalog{}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Weapons["target"] = store.WeaponState{UserWeaponUuid: "target", WeaponId: 10}
	user.Weapons["copy"] = store.WeaponState{UserWeaponUuid: "copy", WeaponId: 10}

	validated, err := validateMaterialWeapons(catalog, user, "target", []string{"copy"}, true, 1)
	if err != nil || len(validated) != 1 || validated[0] != "copy" {
		t.Fatalf("validated = %v, err=%v", validated, err)
	}
}

func TestValidateMaterialWeaponsAllowsUnawakenedCopyOfAwakenedTarget(t *testing.T) {
	catalog := &masterdata.WeaponCatalog{
		EvolutionGroupByWeaponId: map[int32]int32{
			310001: 31000,
			310002: 31000,
		},
	}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Weapons["target"] = store.WeaponState{UserWeaponUuid: "target", WeaponId: 310002}
	user.Weapons["copy"] = store.WeaponState{UserWeaponUuid: "copy", WeaponId: 310001}
	user.WeaponAwakens["target"] = store.WeaponAwakenState{UserWeaponUuid: "target"}

	validated, err := validateMaterialWeapons(catalog, user, "target", []string{"copy"}, true, 1)
	if err != nil || len(validated) != 1 || validated[0] != "copy" {
		t.Fatalf("validated = %v, err=%v", validated, err)
	}
}

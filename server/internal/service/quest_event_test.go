package service

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
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

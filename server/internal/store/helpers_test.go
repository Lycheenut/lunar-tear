package store

import (
	"testing"

	"lunar-tear/server/internal/model"
)

func TestPossessionGranterGrantFullHonorsEquipmentCountAndDuplicates(t *testing.T) {
	user := SeedUserState(1, "test", 1, model.ClientPlatform{})
	granter := &PossessionGranter{
		CostumeDupExchange: map[int32][]model.DupExchangeEntry{
			101: {{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 501, Count: 10}},
		},
		CompanionDupExchange: map[int32][]model.DupExchangeEntry{
			201: {{PossessionType: int32(model.PossessionTypeConsumableItem), PossessionId: 601, Count: 100}},
		},
	}

	granter.GrantFull(user, model.PossessionTypeCostume, 101, 3, 1000)
	granter.GrantFull(user, model.PossessionTypeCompanion, 201, 2, 1000)
	granter.GrantFull(user, model.PossessionTypeWeapon, 301, 3, 1000)
	granter.GrantFull(user, model.PossessionTypeParts, 401, 2, 1000)
	granter.GrantFull(user, model.PossessionTypeThought, 701, 2, 1000)

	if got := len(user.Costumes); got != 1 {
		t.Fatalf("costumes = %d, want 1 unique costume", got)
	}
	if got := user.Materials[501]; got != 20 {
		t.Fatalf("costume duplicate material = %d, want 20", got)
	}
	if got := len(user.Companions); got != 1 {
		t.Fatalf("companions = %d, want 1 unique companion", got)
	}
	if got := user.ConsumableItems[601]; got != 100 {
		t.Fatalf("companion duplicate item = %d, want 100", got)
	}
	if got := len(user.Weapons); got != 3 {
		t.Fatalf("weapons = %d, want 3", got)
	}
	if got := len(user.Parts); got != 2 {
		t.Fatalf("parts = %d, want 2", got)
	}
	if got := len(user.Thoughts); got != 2 {
		t.Fatalf("thoughts = %d, want 2", got)
	}
}

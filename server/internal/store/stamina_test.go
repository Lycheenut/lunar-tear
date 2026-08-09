package store

import "testing"

func TestReplenishStaminaFillsToMaximum(t *testing.T) {
	user := &UserState{}
	user.Status.StaminaMilliValue = 40_000

	ReplenishStamina(user, 88_000, 123)

	if got := user.Status.StaminaMilliValue; got != 88_000 {
		t.Fatalf("stamina = %d, want 88000", got)
	}
	if got := user.Status.StaminaUpdateDatetime; got != 123 {
		t.Fatalf("update datetime = %d, want 123", got)
	}
}

func TestReplenishStaminaPreservesOverflow(t *testing.T) {
	user := &UserState{}
	user.Status.StaminaMilliValue = 868_000

	ReplenishStamina(user, 88_000, 123)

	if got := user.Status.StaminaMilliValue; got != 868_000 {
		t.Fatalf("overflow stamina = %d, want 868000", got)
	}
	if got := user.Status.StaminaUpdateDatetime; got != 123 {
		t.Fatalf("update datetime = %d, want 123", got)
	}
}

package store

import "testing"

func TestRecoverStaminaAddsNewMaximumToOverflow(t *testing.T) {
	user := &UserState{}
	user.Status.StaminaMilliValue = 912_000

	RecoverStamina(user, 88_000, 88_000, 123)

	if got := user.Status.StaminaMilliValue; got != 1_000_000 {
		t.Fatalf("stamina = %d, want 1000000", got)
	}
	if got := user.Status.StaminaUpdateDatetime; got != 123 {
		t.Fatalf("update datetime = %d, want 123", got)
	}
}

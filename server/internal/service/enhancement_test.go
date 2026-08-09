package service

import (
	"math"
	"testing"
)

func TestFinalizeEnhancementExpGreatSuccess(t *testing.T) {
	exp, great, err := finalizeEnhancementExp(100, 60, 59)
	if err != nil || !great || exp != 200 {
		t.Fatalf("exp=%d great=%v err=%v, want 200, true, nil", exp, great, err)
	}

	exp, great, err = finalizeEnhancementExp(100, 60, 60)
	if err != nil || great || exp != 100 {
		t.Fatalf("exp=%d great=%v err=%v, want 100, false, nil", exp, great, err)
	}
}

func TestFinalizeEnhancementExpRejectsOverflow(t *testing.T) {
	if _, _, err := finalizeEnhancementExp(math.MaxInt32, 1000, 0); err == nil {
		t.Fatal("expected doubled experience overflow to fail")
	}
}

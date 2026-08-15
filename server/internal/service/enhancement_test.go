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

func TestConsumeEnhancementSelectionsStopsAtLevelCap(t *testing.T) {
	consumed, exp, count := consumeEnhancementSelections([]enhancementSelection{
		{count: 3, expPerUnit: 400},
		{count: 2, expPerUnit: 100},
	}, 900, greatSuccessExpMultiplier)
	if consumed[0] != 2 || consumed[1] != 0 || exp != 1600 || count != 2 {
		t.Fatalf("consumed=%v exp=%d count=%d, want [2 0], 1600, 2", consumed, exp, count)
	}
}

func TestConsumeEnhancementSelectionsUsesAllWhenBelowLevelCap(t *testing.T) {
	consumed, exp, count := consumeEnhancementSelections([]enhancementSelection{
		{count: 2, expPerUnit: 400},
		{count: 1, expPerUnit: 100},
	}, 2000, greatSuccessExpMultiplier)
	if consumed[0] != 2 || consumed[1] != 1 || exp != 1800 || count != 3 {
		t.Fatalf("consumed=%v exp=%d count=%d, want [2 1], 1800, 3", consumed, exp, count)
	}
}

func TestConsumeEnhancementMaterialsReturnsLowerValueSurplus(t *testing.T) {
	selections := []enhancementMaterialSelection{
		{materialId: 30, enhancementSelection: enhancementSelection{count: 1, expPerUnit: 100}},
		{materialId: 20, enhancementSelection: enhancementSelection{count: 1, expPerUnit: 400}},
		{materialId: 10, enhancementSelection: enhancementSelection{count: 1, expPerUnit: 400}},
	}
	exp, count, surplus := consumeEnhancementMaterials(selections, 700, greatSuccessExpMultiplier)
	if exp != 800 || count != 1 || selections[0].materialId != 10 || selections[0].count != 1 || surplus[20] != 1 || surplus[30] != 1 {
		t.Fatalf("selections=%v exp=%d count=%d surplus=%v", selections, exp, count, surplus)
	}
}

func TestEnhancementExpCapUsesInstanceLevelLimit(t *testing.T) {
	thresholds := []int32{0, 100, 300, 600}
	if cap, ok := enhancementExpCap(thresholds, 2); !ok || cap != 300 {
		t.Fatalf("cap=%d ok=%v, want 300, true", cap, ok)
	}
	if cap, ok := enhancementExpCap(thresholds, 0); !ok || cap != 600 {
		t.Fatalf("uncapped cap=%d ok=%v, want 600, true", cap, ok)
	}
}

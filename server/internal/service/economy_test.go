package service

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"lunar-tear/server/internal/masterdata"
)

func TestSelectedMaterialCostsSupportsDifferentAlternativeRates(t *testing.T) {
	options := []masterdata.MaterialOption{
		{MaterialId: 10, Count: 10},
		{MaterialId: 20, Count: 1},
	}
	costs, steps, err := selectedMaterialCosts(map[int32]int32{10: 10, 20: 1}, options)
	if err != nil {
		t.Fatal(err)
	}
	if steps != 2 || len(costs) != 2 {
		t.Fatalf("selection = %d steps and %d costs, want 2 and 2", steps, len(costs))
	}
	if _, steps, err := selectedMaterialCosts(map[int32]int32{10: 5, 30: 5}, []masterdata.MaterialOption{
		{MaterialId: 10, Count: 10},
		{MaterialId: 30, Count: 10},
	}); err != nil || steps != 1 {
		t.Fatalf("mixed equal-rate materials = %d steps, err=%v; want 1 step", steps, err)
	}

	if _, _, err := selectedMaterialCosts(map[int32]int32{10: 5}, options); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("partial material step status = %v, want InvalidArgument", status.Code(err))
	}
	if _, _, err := selectedMaterialCosts(map[int32]int32{30: 1}, options); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unknown material status = %v, want InvalidArgument", status.Code(err))
	}
}

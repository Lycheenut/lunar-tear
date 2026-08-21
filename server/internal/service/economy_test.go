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

func TestSelectedMaterialCostsAcceptsUniversalLimitBreakMaterials(t *testing.T) {
	for _, tc := range []struct {
		name      string
		materials map[int32]int32
		options   []masterdata.MaterialOption
	}{
		{
			name:      "costume handbook",
			materials: map[int32]int32{311003: 10},
			options: []masterdata.MaterialOption{
				{MaterialId: 311100, Count: 10},
				{MaterialId: 311003, Count: 10},
			},
		},
		{
			name:      "weapon pearl",
			materials: map[int32]int32{312001: 1},
			options: []masterdata.MaterialOption{
				{MaterialId: 312100, Count: 1},
				{MaterialId: 312001, Count: 1},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			costs, steps, err := selectedMaterialCosts(tc.materials, tc.options)
			if err != nil || steps != 1 || len(costs) != 1 {
				t.Fatalf("selection = %d steps and %d costs, err=%v; want one step and one cost", steps, len(costs), err)
			}
		})
	}
}

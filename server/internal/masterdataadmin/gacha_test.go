package masterdataadmin

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
)

func TestCostumeIconPathUsesCostumeAssetNaming(t *testing.T) {
	costume := masterdata.EntityMCostume{ActorSkeletonId: 8, AssetVariationId: 13}
	if got, want := costumeIconPath(costume), "costume/ch008013/ch008013_standard.png"; got != want {
		t.Fatalf("costumeIconPath() = %q, want %q", got, want)
	}
}

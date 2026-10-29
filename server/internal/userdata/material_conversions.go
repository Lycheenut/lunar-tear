package userdata

import (
	"maps"
	"slices"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

func init() {
	// The client observes changes to this table to open its conversion dialog.
	// Full synchronization must not replay historical conversion notifications.
	registerStatic("IUserPossessionAutoConvert")
}

// MaterialConversionDiff projects request-local counts, not the stored totals:
// CalculatorPossession's change callback passes FromCount/ToCount straight to
// DataPossessionAutoConvert, without subtracting the previous client row.
func MaterialConversionDiff(before, after *store.UserState) *pb.DiffData {
	var records []map[string]any
	for _, id := range slices.Sorted(maps.Keys(after.MaterialConversions)) {
		conversion := after.MaterialConversions[id]
		count := conversion.ConvertedCount - before.MaterialConversions[id].ConvertedCount
		if count <= 0 {
			continue
		}
		records = append(records, map[string]any{
			"userId":           after.UserId,
			"possessionType":   int32(model.PossessionTypeMaterial),
			"possessionId":     id,
			"fromCount":        count * 10,
			"toPossessionType": int32(model.PossessionTypeMaterial),
			"toPossessionId":   conversion.ToMaterialId,
			"toCount":          count,
			"convertDatetime":  conversion.ConvertDatetime,
			"latestVersion":    conversion.ConvertedCount,
		})
	}
	if len(records) == 0 {
		return nil
	}
	updates, _ := utils.EncodeJSONMaps(records...)
	return &pb.DiffData{UpdateRecordsJson: updates, DeleteKeysJson: "[]"}
}

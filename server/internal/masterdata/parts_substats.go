package masterdata

import "lunar-tear/server/internal/model"

// buildPartsStatusSub returns independent maps for each catalog snapshot.
// Values are reconstructed game rules.
func buildPartsStatusSub() (map[int32]model.PartsStatusSubDef, map[int32][]int32) {
	definitions := map[int32]model.PartsStatusSubDef{
		1:  {StatusKindType: 2, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 8, Max: 15, Step: 1}, Growth: model.PartsSubStatusRange{Min: 8, Max: 18, Step: 1}},
		2:  {StatusKindType: 2, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 16, Max: 25, Step: 1}, Growth: model.PartsSubStatusRange{Min: 15, Max: 25, Step: 1}},
		3:  {StatusKindType: 2, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 24, Max: 35, Step: 1}, Growth: model.PartsSubStatusRange{Min: 20, Max: 35, Step: 1}},
		4:  {StatusKindType: 2, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 30, Max: 45, Step: 1}, Growth: model.PartsSubStatusRange{Min: 25, Max: 45, Step: 1}},
		5:  {StatusKindType: 7, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 8, Max: 15, Step: 1}, Growth: model.PartsSubStatusRange{Min: 8, Max: 18, Step: 1}},
		6:  {StatusKindType: 7, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 16, Max: 25, Step: 1}, Growth: model.PartsSubStatusRange{Min: 15, Max: 25, Step: 1}},
		7:  {StatusKindType: 7, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 24, Max: 35, Step: 1}, Growth: model.PartsSubStatusRange{Min: 20, Max: 35, Step: 1}},
		8:  {StatusKindType: 7, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 30, Max: 45, Step: 1}, Growth: model.PartsSubStatusRange{Min: 25, Max: 45, Step: 1}},
		9:  {StatusKindType: 2, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 5, Max: 15, Step: 5}, Growth: model.PartsSubStatusRange{Min: 5, Max: 10, Step: 5}},
		10: {StatusKindType: 2, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}, Growth: model.PartsSubStatusRange{Min: 5, Max: 15, Step: 5}},
		11: {StatusKindType: 2, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}, Growth: model.PartsSubStatusRange{Min: 10, Max: 15, Step: 5}},
		12: {StatusKindType: 2, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 15, Max: 25, Step: 5}, Growth: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}},
		13: {StatusKindType: 7, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 5, Max: 15, Step: 5}, Growth: model.PartsSubStatusRange{Min: 5, Max: 10, Step: 5}},
		14: {StatusKindType: 7, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}, Growth: model.PartsSubStatusRange{Min: 5, Max: 15, Step: 5}},
		15: {StatusKindType: 7, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}, Growth: model.PartsSubStatusRange{Min: 10, Max: 15, Step: 5}},
		16: {StatusKindType: 7, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 15, Max: 25, Step: 5}, Growth: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}},
		17: {StatusKindType: 6, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 5, Max: 15, Step: 5}, Growth: model.PartsSubStatusRange{Min: 5, Max: 10, Step: 5}},
		18: {StatusKindType: 6, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}, Growth: model.PartsSubStatusRange{Min: 5, Max: 15, Step: 5}},
		19: {StatusKindType: 6, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}, Growth: model.PartsSubStatusRange{Min: 10, Max: 15, Step: 5}},
		20: {StatusKindType: 6, StatusCalculationType: 2, Initial: model.PartsSubStatusRange{Min: 15, Max: 25, Step: 5}, Growth: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}},
		21: {StatusKindType: 6, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 200, Max: 300, Step: 5}, Growth: model.PartsSubStatusRange{Min: 200, Max: 300, Step: 5}},
		22: {StatusKindType: 6, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 350, Max: 450, Step: 5}, Growth: model.PartsSubStatusRange{Min: 350, Max: 450, Step: 5}},
		23: {StatusKindType: 6, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 500, Max: 600, Step: 5}, Growth: model.PartsSubStatusRange{Min: 500, Max: 600, Step: 5}},
		24: {StatusKindType: 6, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 650, Max: 750, Step: 5}, Growth: model.PartsSubStatusRange{Min: 650, Max: 750, Step: 5}},
		25: {StatusKindType: 4, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}, Growth: model.PartsSubStatusRange{Min: 5, Max: 15, Step: 5}},
		26: {StatusKindType: 4, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 15, Max: 30, Step: 5}, Growth: model.PartsSubStatusRange{Min: 10, Max: 20, Step: 5}},
		27: {StatusKindType: 4, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 20, Max: 40, Step: 5}, Growth: model.PartsSubStatusRange{Min: 15, Max: 25, Step: 5}},
		28: {StatusKindType: 4, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 30, Max: 50, Step: 5}, Growth: model.PartsSubStatusRange{Min: 20, Max: 30, Step: 5}},
		29: {StatusKindType: 3, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 20, Max: 30, Step: 5}, Growth: model.PartsSubStatusRange{Min: 20, Max: 35, Step: 5}},
		30: {StatusKindType: 3, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 25, Max: 40, Step: 5}, Growth: model.PartsSubStatusRange{Min: 25, Max: 45, Step: 5}},
		31: {StatusKindType: 3, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 30, Max: 50, Step: 5}, Growth: model.PartsSubStatusRange{Min: 30, Max: 55, Step: 5}},
		32: {StatusKindType: 3, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 40, Max: 60, Step: 5}, Growth: model.PartsSubStatusRange{Min: 40, Max: 60, Step: 5}},
		33: {StatusKindType: 1, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 3, Max: 6, Step: 1}, Growth: model.PartsSubStatusRange{Min: 3, Max: 6, Step: 1}},
		34: {StatusKindType: 1, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 6, Max: 8, Step: 1}, Growth: model.PartsSubStatusRange{Min: 6, Max: 8, Step: 1}},
		35: {StatusKindType: 1, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 8, Max: 10, Step: 1}, Growth: model.PartsSubStatusRange{Min: 8, Max: 10, Step: 1}},
		36: {StatusKindType: 1, StatusCalculationType: 1, Initial: model.PartsSubStatusRange{Min: 9, Max: 12, Step: 1}, Growth: model.PartsSubStatusRange{Min: 9, Max: 12, Step: 1}},
	}
	pools := map[int32][]int32{
		1:  {1, 5, 9, 13, 17, 21, 25, 29, 33},
		2:  {2, 6, 10, 14, 18, 22, 26, 30, 34},
		3:  {3, 7, 11, 15, 19, 23, 27, 31, 35},
		4:  {4, 8, 12, 16, 20, 24, 28, 32, 36},
		11: {1, 5, 9, 13, 17, 21, 25, 29, 33},
		12: {2, 6, 10, 14, 18, 22, 26, 30, 34},
	}
	return definitions, pools
}

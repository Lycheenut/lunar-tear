package model

import "math/rand"

const (
	PartsMaxLevel                     int32 = 15
	PartsMaxSubStatusCount            int32 = 4
	PartsSubStatusEnhancementInterval int32 = 3
)

func IsPartsSubStatusEnhancementLevel(level int32) bool {
	return level >= PartsSubStatusEnhancementInterval && level <= PartsMaxLevel && level%PartsSubStatusEnhancementInterval == 0
}

// PartsSubStatusRange uses permil for percentages (including critical stats),
// and whole attribute points otherwise. Every step is equally likely.
type PartsSubStatusRange struct {
	Min  int32 `json:"min"`
	Max  int32 `json:"max"`
	Step int32 `json:"step"`
}

func (r PartsSubStatusRange) Roll() int32 {
	return r.Min + rand.Int31n((r.Max-r.Min)/r.Step+1)*r.Step
}

type PartsStatusSubDef struct {
	StatusKindType        int32               `json:"status_kind_type"`
	StatusCalculationType int32               `json:"status_calculation_type"`
	Initial               PartsSubStatusRange `json:"initial"`
	Growth                PartsSubStatusRange `json:"growth"`
}

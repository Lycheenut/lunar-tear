package masterdata

import (
	"log"
	"sort"

	"lunar-tear/server/internal/utils"
)

type loginBonusStampKey struct {
	LoginBonusId    int32
	LowerPageNumber int32
	StampNumber     int32
}

type LoginBonusReward struct {
	PossessionType int32
	PossessionId   int32
	Count          int32
}

type LoginBonusTerm struct {
	StartDatetime           int64
	EndDatetime             int64
	StampReceiveEndDatetime int64
}

type LoginBonusCatalog struct {
	stamps     map[loginBonusStampKey]LoginBonusReward
	bonusPages map[int32][]int32
	totalPages map[int32]int32
	terms      map[int32]LoginBonusTerm
}

func (c *LoginBonusCatalog) LookupStampReward(loginBonusId, pageNumber, stampNumber int32) (LoginBonusReward, bool) {
	pages := c.bonusPages[loginBonusId]
	lower := int32(-1)
	for _, p := range pages {
		if p <= pageNumber {
			lower = p
		} else {
			break
		}
	}
	if lower < 0 {
		return LoginBonusReward{}, false
	}
	entry, ok := c.stamps[loginBonusStampKey{loginBonusId, lower, stampNumber}]
	return entry, ok
}

func (c *LoginBonusCatalog) TotalPageCount(loginBonusId int32) int32 {
	return c.totalPages[loginBonusId]
}

func (c *LoginBonusCatalog) LookupTerm(loginBonusId int32) (LoginBonusTerm, bool) {
	term, ok := c.terms[loginBonusId]
	return term, ok
}

func LoadLoginBonusCatalog() *LoginBonusCatalog {
	stamps, err := utils.ReadTable[EntityMLoginBonusStamp]("m_login_bonus_stamp")
	if err != nil {
		log.Fatalf("load login bonus stamp table: %v", err)
	}

	bonuses, err := utils.ReadTable[EntityMLoginBonus]("m_login_bonus")
	if err != nil {
		log.Fatalf("load login bonus table: %v", err)
	}

	cat := &LoginBonusCatalog{
		stamps:     make(map[loginBonusStampKey]LoginBonusReward, len(stamps)),
		bonusPages: make(map[int32][]int32),
		totalPages: make(map[int32]int32, len(bonuses)),
		terms:      make(map[int32]LoginBonusTerm, len(bonuses)),
	}

	for _, b := range bonuses {
		cat.totalPages[b.LoginBonusId] = b.TotalPageCount
		cat.terms[b.LoginBonusId] = LoginBonusTerm{
			StartDatetime:           b.StartDatetime,
			EndDatetime:             b.EndDatetime,
			StampReceiveEndDatetime: b.StampReceiveEndDatetime,
		}
	}

	seenPages := make(map[loginBonusStampKey]struct{})
	for _, s := range stamps {
		cat.stamps[loginBonusStampKey{s.LoginBonusId, s.LowerPageNumber, s.StampNumber}] = LoginBonusReward{
			PossessionType: s.RewardPossessionType,
			PossessionId:   s.RewardPossessionId,
			Count:          s.RewardCount,
		}
		dedup := loginBonusStampKey{LoginBonusId: s.LoginBonusId, LowerPageNumber: s.LowerPageNumber}
		if _, exists := seenPages[dedup]; !exists {
			seenPages[dedup] = struct{}{}
			cat.bonusPages[s.LoginBonusId] = append(cat.bonusPages[s.LoginBonusId], s.LowerPageNumber)
		}
	}

	for id := range cat.bonusPages {
		sort.Slice(cat.bonusPages[id], func(i, j int) bool {
			return cat.bonusPages[id][i] < cat.bonusPages[id][j]
		})
	}

	return cat
}

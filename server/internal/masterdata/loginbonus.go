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

type LoginBonusDefinition struct {
	LoginBonusId               int32
	SortOrder                  int32
	LoginBonusStartConditionId int32
	Term                       LoginBonusTerm
}

type LoginBonusCatalog struct {
	stamps      map[loginBonusStampKey]LoginBonusReward
	bonusPages  map[int32][]int32
	totalPages  map[int32]int32
	definitions []LoginBonusDefinition
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

func (c *LoginBonusCatalog) ActiveDefinitions(nowMillis int64) []LoginBonusDefinition {
	active := make([]LoginBonusDefinition, 0, len(c.definitions))
	for _, definition := range c.definitions {
		term := definition.Term
		if term.StartDatetime != 0 && nowMillis < term.StartDatetime {
			continue
		}
		if term.EndDatetime != 0 && nowMillis >= term.EndDatetime {
			continue
		}
		if term.StampReceiveEndDatetime != 0 && nowMillis >= term.StampReceiveEndDatetime {
			continue
		}
		active = append(active, definition)
	}
	return active
}

func (c *LoginBonusCatalog) Definitions() []LoginBonusDefinition {
	return append([]LoginBonusDefinition(nil), c.definitions...)
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
		stamps:      make(map[loginBonusStampKey]LoginBonusReward, len(stamps)),
		bonusPages:  make(map[int32][]int32),
		totalPages:  make(map[int32]int32, len(bonuses)),
		definitions: make([]LoginBonusDefinition, 0, len(bonuses)),
	}

	for _, b := range bonuses {
		cat.totalPages[b.LoginBonusId] = b.TotalPageCount
		term := LoginBonusTerm{
			StartDatetime:           b.StartDatetime,
			EndDatetime:             b.EndDatetime,
			StampReceiveEndDatetime: b.StampReceiveEndDatetime,
		}
		cat.definitions = append(cat.definitions, LoginBonusDefinition{
			LoginBonusId:               b.LoginBonusId,
			SortOrder:                  b.SortOrder,
			LoginBonusStartConditionId: b.LoginBonusStartConditionId,
			Term:                       term,
		})
	}
	sort.Slice(cat.definitions, func(i, j int) bool {
		if cat.definitions[i].SortOrder == cat.definitions[j].SortOrder {
			return cat.definitions[i].LoginBonusId < cat.definitions[j].LoginBonusId
		}
		return cat.definitions[i].SortOrder < cat.definitions[j].SortOrder
	})

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

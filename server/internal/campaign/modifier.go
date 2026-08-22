package campaign

type RateBonus struct {
	override    int32
	bonusPermil int32
}

func (b RateBonus) Apply(basePermil int32) int32 {
	best := basePermil
	if b.override > best {
		best = b.override
	}
	if additional := int64(basePermil) + int64(b.bonusPermil); additional > int64(best) {
		if additional > 1000 {
			return 1000
		}
		best = int32(additional)
	}
	return clampPermil(best)
}

type StaminaMul struct {
	permil int32
}

func (m StaminaMul) Apply(base int32) int32 {
	if m.permil == 1000 {
		return base
	}
	return int32(int64(base) * int64(m.permil) / 1000)
}

type DropRateMul struct {
	bonusPermil int32
}

func (m DropRateMul) WithBonusPermil(bonusPermil int32) DropRateMul {
	m.bonusPermil += bonusPermil
	return m
}

func (m DropRateMul) Apply(base int32) int32 {
	return int32((int64(base)*int64(1000+m.bonusPermil) + 999) / 1000)
}

type DropCountMul struct {
	bonusPermil int32
}

func (m DropCountMul) WithBonusPermil(bonusPermil int32) DropCountMul {
	m.bonusPermil += bonusPermil
	return m
}

func (m DropCountMul) Apply(base int32) int32 {
	return int32(int64(base) * int64(1000+m.bonusPermil) / 1000)
}

type GoldMul struct {
	bonusPermil int32
}

func (m GoldMul) Apply(base int32) int32 {
	return int32(int64(base) * int64(1000+m.bonusPermil) / 1000)
}

type BonusDrop struct {
	PossessionType int32
	PossessionId   int32
	Count          int32
}

func clampPermil(v int32) int32 {
	if v < 0 {
		return 0
	}
	if v > 1000 {
		return 1000
	}
	return v
}

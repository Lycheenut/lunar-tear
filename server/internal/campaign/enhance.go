package campaign

func (c *Catalog) PartsRateBonus(t PartsTarget, f Filter) RateBonus {
	var out RateBonus
	for _, r := range c.enhance {
		if !r.isActive(f) {
			continue
		}
		if !matchesParts(r.targets, t) {
			continue
		}
		out = applyEnhanceEffect(out, r)
	}
	return out
}

func (c *Catalog) CostumeRateBonus(t CostumeTarget, f Filter) RateBonus {
	var out RateBonus
	for _, r := range c.enhance {
		if !r.isActive(f) {
			continue
		}
		if matchesCostume(r.targets, t) {
			out = applyEnhanceEffect(out, r)
		}
	}
	return out
}

func (c *Catalog) WeaponRateBonus(t WeaponTarget, f Filter) RateBonus {
	var out RateBonus
	for _, r := range c.enhance {
		if !r.isActive(f) {
			continue
		}
		if matchesWeapon(r.targets, t) {
			out = applyEnhanceEffect(out, r)
		}
	}
	return out
}

func applyEnhanceEffect(b RateBonus, r enhanceRow) RateBonus {
	switch r.effectType {
	case EnhanceEffectProbability:
		if r.effectValue > b.override {
			b.override = r.effectValue
		}
	case EnhanceEffectAdditionalPerm:
		if r.effectValue > b.bonusPermil {
			b.bonusPermil = r.effectValue
		}
	}
	return b
}

func matchesParts(targets []enhanceMatch, t PartsTarget) bool {
	for _, m := range targets {
		switch m.t {
		case EnhanceTargetPartsAll:
			return true
		case EnhanceTargetPartsSeriesId:
			if m.v == t.PartsGroupId {
				return true
			}
		case EnhanceTargetPartsId:
			if m.v == t.PartsId {
				return true
			}
		}
	}
	return false
}

func matchesCostume(targets []enhanceMatch, t CostumeTarget) bool {
	for _, m := range targets {
		switch m.t {
		case EnhanceTargetCostumeAll:
			return true
		case EnhanceTargetCostumeCharacterId:
			if m.v == t.CharacterId {
				return true
			}
		case EnhanceTargetCostumeSkillfulWeapon:
			if m.v == t.SkillfulWeaponType {
				return true
			}
		case EnhanceTargetCostumeId:
			if m.v == t.CostumeId {
				return true
			}
		}
	}
	return false
}

func matchesWeapon(targets []enhanceMatch, t WeaponTarget) bool {
	for _, m := range targets {
		switch m.t {
		case EnhanceTargetWeaponAll:
			return true
		case EnhanceTargetWeaponTypeId:
			if m.v == t.WeaponType {
				return true
			}
		case EnhanceTargetWeaponAttributeTypeId:
			if m.v == t.AttributeType {
				return true
			}
		case EnhanceTargetWeaponId:
			if m.v == t.WeaponId {
				return true
			}
		}
	}
	return false
}

package store

import (
	"reflect"
	"testing"

	"lunar-tear/server/internal/model"
)

func TestEnhancedRewardsRejectUnknownTemplates(t *testing.T) {
	for _, possessionType := range []model.PossessionType{model.PossessionTypeWeaponEnhanced, model.PossessionTypeCompanionEnhanced, model.PossessionTypePartsEnhanced} {
		user := SeedUserState(1, "test", 1, model.ClientPlatform{})
		result := (&PossessionGranter{}).GrantFull(user, possessionType, 9001, 2, 1000)
		if result.Status != GrantStatusInvalid || len(user.Weapons)+len(user.Companions)+len(user.Parts) != 0 {
			t.Fatalf("type %d: missing template produced result %+v and inventory %v/%v/%v", possessionType, result, user.Weapons, user.Companions, user.Parts)
		}
	}
}

func TestEnhancedWeaponGrantsTemplateStatsSkillsAndNotes(t *testing.T) {
	user := SeedUserState(1, "test", 1, model.ClientPlatform{})
	g := &PossessionGranter{
		WeaponById:       map[int32]WeaponRef{101: {WeaponSkillGroupId: 10, WeaponAbilityGroupId: 20, WeaponStoryReleaseConditionGroupId: 30}},
		WeaponSkillSlots: map[int32][]int32{10: {2, 1}}, WeaponAbilitySlots: map[int32][]int32{20: {1, 2}},
		WeaponEnhancedById: map[int32]WeaponEnhancedRef{9001: {
			WeaponId: 101, Level: 70, Exp: 5000, LimitBreakCount: 3,
			SkillLevels: map[int32]int32{2: 8}, AbilityLevels: map[int32]int32{1: 10},
		}},
		ReleaseConditions: map[int32][]WeaponStoryReleaseCond{30: {{StoryIndex: 1, WeaponStoryReleaseConditionType: model.WeaponStoryReleaseConditionTypeAcquisition}}},
	}
	g.GrantWeapon(user, 101, 500)
	// The collection note must improve even if this weapon was already owned.
	result := g.GrantFull(user, model.PossessionTypeWeaponEnhanced, 9001, 2, 1000)
	if result.Status != GrantStatusGranted || len(user.Weapons) != 3 {
		t.Fatalf("result=%+v weapons=%v", result, user.Weapons)
	}
	for key, weapon := range user.Weapons {
		if weapon.AcquisitionDatetime != 1000 {
			continue
		}
		if weapon.WeaponId != 101 || weapon.Level != 70 || weapon.Exp != 5000 || weapon.LimitBreakCount != 3 {
			t.Fatalf("weapon=%+v", weapon)
		}
		for _, skill := range user.WeaponSkills[key] {
			want := int32(1)
			if skill.SlotNumber == 2 {
				want = 8
			}
			if skill.Level != want {
				t.Fatalf("skill=%+v", skill)
			}
		}
		for _, ability := range user.WeaponAbilities[key] {
			want := int32(1)
			if ability.SlotNumber == 1 {
				want = 10
			}
			if ability.Level != want {
				t.Fatalf("ability=%+v", ability)
			}
		}
	}
	note := user.WeaponNotes[101]
	if note.MaxLevel != 70 || note.MaxLimitBreakCount != 3 || note.FirstAcquisitionDatetime != 500 || note.LatestVersion != 1000 {
		t.Fatalf("note=%+v", note)
	}
	g.GrantWeapon(user, 101, 2000)
	if user.WeaponNotes[101] != note {
		t.Fatal("ordinary reward downgraded the collection note")
	}
	fresh := SeedUserState(2, "test", 1, model.ClientPlatform{})
	result = g.GrantFull(fresh, model.PossessionTypeWeaponEnhanced, 9001, 1, 1000)
	if !reflect.DeepEqual(result.ChangedStoryWeaponIds, []int32{101}) {
		t.Fatalf("story unlocks=%v", result.ChangedStoryWeaponIds)
	}
}

func enhancedPartsTestGranter() *PossessionGranter {
	return &PossessionGranter{
		PartsById:                  map[int32]PartsRef{101: {PartsGroupId: 10, RarityType: 40, PartsInitialLotteryId: 3, PartsStatusSubLotteryGroupId: 1}},
		PartsVariantsByGroupRarity: map[int32]map[int32][]int32{10: {40: {102, 103, 104, 105, 106}}},
		PartsEnhancedById: map[int32]PartsEnhancedRef{9001: {
			PartsId: 101, PartsStatusMainId: 8, Level: 15, SubStatusCount: 3,
			SubStatuses: []PartsStatusSubState{{StatusIndex: 2, PartsStatusSubLotteryId: 1, Level: 7, StatusKindType: 6, StatusCalculationType: 2, StatusChangeValue: 777}},
		}},
		PartsSubStatusPool: map[int32][]int32{1: {1, 2, 3, 4}},
		PartsSubStatusDefs: map[int32]PartsStatusSubDef{
			1: {StatusKindType: 6, StatusCalculationType: 2, StatusChangeInitialValue: 5},
			2: {StatusKindType: 2, StatusCalculationType: 1, StatusFunc: func(level int32) int32 { return level * 10 }},
			3: {StatusKindType: 7, StatusCalculationType: 1, StatusChangeInitialValue: 25},
			4: {StatusKindType: 1, StatusCalculationType: 1, StatusChangeInitialValue: 35},
		},
	}
}

func TestEnhancedPartsPreserveFixedStatsAndUniqueRandomSlots(t *testing.T) {
	user := SeedUserState(1, "test", 1, model.ClientPlatform{})
	g := enhancedPartsTestGranter()
	result := g.GrantFull(user, model.PossessionTypePartsEnhanced, 9001, 2, 1000)
	if result.Status != GrantStatusGranted || len(user.Parts) != 2 || len(user.PartsStatusSubs) != 6 {
		t.Fatalf("result=%+v parts=%v subs=%v", result, user.Parts, user.PartsStatusSubs)
	}
	for key, part := range user.Parts {
		if part.PartsId != 101 || part.Level != 15 || part.PartsStatusMainId != 8 || part.AcquisitionDatetime != 1000 {
			t.Fatalf("part=%+v", part)
		}
		used := map[int32]bool{}
		for slot := int32(1); slot <= 3; slot++ {
			sub := user.PartsStatusSubs[PartsStatusSubKey{UserPartsUuid: key, StatusIndex: slot}]
			if used[sub.PartsStatusSubLotteryId] || sub.LatestVersion != 1000 {
				t.Fatalf("sub=%+v used=%v", sub, used)
			}
			used[sub.PartsStatusSubLotteryId] = true
			if slot == 2 {
				if sub.StatusChangeValue != 777 || sub.Level != 7 || sub.StatusKindType != 6 || sub.StatusCalculationType != 2 {
					t.Fatalf("fixed sub=%+v", sub)
				}
			} else if sub.Level != 15 || sub.PartsStatusSubLotteryId == 1 {
				t.Fatalf("random sub=%+v", sub)
			} else if sub.PartsStatusSubLotteryId == 2 && sub.StatusChangeValue != 150 {
				t.Fatalf("calculated sub=%+v", sub)
			}
		}
	}
	if user.PartsGroupNotes[10].FirstAcquisitionDatetime != 1000 {
		t.Fatal("missing collection note")
	}
	if g.PartsEnhancedById[9001].SubStatuses[0].UserPartsUuid != "" {
		t.Fatal("grant mutated the template")
	}
}

func TestEnhancedPartsRandomCountPreservesTemplateAndFixedSlots(t *testing.T) {
	g := enhancedPartsTestGranter()
	for id := int32(102); id <= 106; id++ {
		g.PartsById[id] = PartsRef{PartsInitialLotteryId: 5}
	}
	enhanced := g.PartsEnhancedById[9001]
	enhanced.IsRandomSubStatusCount = true
	g.PartsEnhancedById[9001] = enhanced
	user := SeedUserState(1, "test", 1, model.ClientPlatform{})
	if result := g.GrantFull(user, model.PossessionTypePartsEnhanced, 9001, 1, 1000); result.Status != GrantStatusGranted {
		t.Fatal(result)
	}
	if len(user.PartsStatusSubs) != 4 {
		t.Fatalf("random rank count=%d, want 4", len(user.PartsStatusSubs))
	}
	for _, part := range user.Parts {
		if part.PartsId != 101 {
			t.Fatalf("template replaced with random variant: %+v", part)
		}
	}
}

func TestEnhancedPartsInvalidDefinitionDoesNotPartiallyGrant(t *testing.T) {
	for _, mutate := range []func(*PossessionGranter){
		func(g *PossessionGranter) { delete(g.PartsById, 101) },
		func(g *PossessionGranter) { g.PartsSubStatusPool[1] = []int32{1} },
		func(g *PossessionGranter) { delete(g.PartsSubStatusDefs, 2) },
		func(g *PossessionGranter) {
			row := g.PartsEnhancedById[9001]
			row.SubStatuses = append(row.SubStatuses, row.SubStatuses[0])
			g.PartsEnhancedById[9001] = row
		},
	} {
		g := enhancedPartsTestGranter()
		mutate(g)
		user := SeedUserState(1, "test", 1, model.ClientPlatform{})
		result := g.GrantFull(user, model.PossessionTypePartsEnhanced, 9001, 2, 1000)
		if result.Status != GrantStatusInvalid || len(user.Parts)+len(user.PartsStatusSubs)+len(user.PartsGroupNotes) != 0 {
			t.Fatalf("invalid reward mutated inventory: %+v", result)
		}
	}
}

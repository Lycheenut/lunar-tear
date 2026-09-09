package masterdata

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
	"lunar-tear/server/internal/masterdata/memorydb"
)

// Use the real binary table reader, including the enhanced tables that are empty in the installed snapshot.
func initEnhancedRewardTables(t *testing.T, tables map[string]any) {
	t.Helper()
	var body []byte
	header := map[string][2]int{}
	for name, rows := range tables {
		raw, err := msgpack.Marshal(rows)
		if err != nil {
			t.Fatal(err)
		}
		header[name] = [2]int{len(body), len(raw)}
		body = append(body, raw...)
	}
	raw, err := msgpack.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, body...)
	padding := aes.BlockSize - len(raw)%aes.BlockSize
	raw = append(raw, bytes.Repeat([]byte{byte(padding)}, padding)...)
	key, _ := hex.DecodeString("36436230313332314545356536624265")
	iv, _ := hex.DecodeString("45666341656634434165356536446141")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(raw, raw)
	file := filepath.Join(t.TempDir(), "rewards.bin.e")
	if err := os.WriteFile(file, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := memorydb.Init(file); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		installed := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
		if _, err := os.Stat(installed); err == nil {
			if err := memorydb.Init(installed); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func TestLoadEnhancedWeaponTemplatesResolvesCurvesAndSlots(t *testing.T) {
	tables := map[string]any{
		"m_weapon_enhanced": []EntityMWeaponEnhanced{
			{WeaponEnhancedId: 9001, WeaponId: 101, Level: 15, LimitBreakCount: 3},
			{WeaponEnhancedId: 9002, WeaponId: 102, Level: 15, LimitBreakCount: 1},
		},
		"m_weapon_enhanced_skill":   []EntityMWeaponEnhancedSkill{{WeaponEnhancedId: 9001, SkillId: 501, Level: 7}},
		"m_weapon_enhanced_ability": []EntityMWeaponEnhancedAbility{{WeaponEnhancedId: 9001, AbilityId: 601, Level: 8}},
		"m_weapon_specific_enhance": []EntityMWeaponSpecificEnhance{{WeaponSpecificEnhanceId: 10, RequiredExpForLevelUpNumericalParameterMapId: 100}},
		"m_weapon_rarity":           []EntityMWeaponRarity{{RarityType: 40, RequiredExpForLevelUpNumericalParameterMapId: 200}},
	}
	initEnhancedRewardTables(t, tables)
	weapons := map[int32]EntityMWeapon{
		101: {WeaponId: 101, WeaponSpecificEnhanceId: 10, WeaponSkillGroupId: 20, WeaponAbilityGroupId: 30},
		102: {WeaponId: 102, RarityType: 40},
	}
	skills := []EntityMWeaponSkillGroup{{WeaponSkillGroupId: 20, SlotNumber: 2, SkillId: 501}, {WeaponSkillGroupId: 21, SlotNumber: 1, SkillId: 501}}
	abilities := []EntityMWeaponAbilityGroup{{WeaponAbilityGroupId: 30, SlotNumber: 3, AbilityId: 601}}
	parameters := []EntityMNumericalParameterMap{{NumericalParameterMapId: 100, ParameterKey: 15, ParameterValue: 1234}, {NumericalParameterMapId: 200, ParameterKey: 15, ParameterValue: 5678}}
	got, err := loadWeaponEnhancedRewards(weapons, skills, abilities, parameters)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[9001].WeaponId != 101 || got[9001].Exp != 1234 || got[9002].Exp != 5678 || got[9001].LimitBreakCount != 3 {
		t.Fatalf("templates=%+v", got)
	}
	if len(got[9001].SkillLevels) != 1 || got[9001].SkillLevels[2] != 7 || got[9001].AbilityLevels[3] != 8 {
		t.Fatalf("slot mapping=%+v", got[9001])
	}
	delete(weapons, 101)
	if _, err := loadWeaponEnhancedRewards(weapons, skills, abilities, parameters); err == nil {
		t.Fatal("accepted missing base weapon")
	}
	weapons[101] = EntityMWeapon{WeaponId: 101, WeaponSpecificEnhanceId: 10}
	if _, err := loadWeaponEnhancedRewards(weapons, nil, abilities, parameters); err == nil {
		t.Fatal("accepted missing skill slot")
	}
}

func TestLoadEnhancedPartsTemplatesAndFixedSubStatuses(t *testing.T) {
	initEnhancedRewardTables(t, map[string]any{
		"m_parts":                              []EntityMParts{{PartsId: 101, PartsGroupId: 10, RarityType: 40}},
		"m_parts_rarity":                       []EntityMPartsRarity{},
		"m_parts_level_up_rate_group":          []EntityMPartsLevelUpRateGroup{},
		"m_parts_level_up_price_group":         []EntityMPartsLevelUpPriceGroup{},
		"m_numerical_function":                 []EntityMNumericalFunction{},
		"m_numerical_function_parameter_group": []EntityMNumericalFunctionParameterGroup{},
		"m_parts_enhanced":                     []EntityMPartsEnhanced{{PartsEnhancedId: 9001, PartsId: 101, Level: 15, PartsStatusMainId: 8, SubStatusCount: 4}},
		"m_parts_enhanced_sub_status":          []EntityMPartsEnhancedSubStatus{{PartsEnhancedId: 9001, StatusIndex: 2, PartsStatusSubLotteryId: 1, Level: 7, StatusKindType: 6, StatusCalculationType: 2, FixedStatusChangeValue: 777}},
	})
	catalog, err := LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	row := catalog.EnhancedById[9001]
	if row.PartsId != 101 || row.Level != 15 || row.PartsStatusMainId != 8 || row.SubStatusCount != 4 {
		t.Fatalf("template=%+v", row)
	}
	subs := catalog.EnhancedSubStatuses[9001]
	if len(subs) != 1 || subs[0].StatusIndex != 2 || subs[0].FixedStatusChangeValue != 777 || subs[0].Level != 7 {
		t.Fatalf("sub statuses=%+v", subs)
	}
}

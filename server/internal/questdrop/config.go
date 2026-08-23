package questdrop

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"lunar-tear/server/internal/masterdata"
)

const (
	ConfigVersion = 1
	MaxWeight     = int32(1<<31 - 1)
)

type Reward struct {
	BattleDropRewardID int32 `json:"battleDropRewardId"`
	Weight             int32 `json:"weight"`
	Guaranteed         bool  `json:"guaranteed,omitempty"`
}

type QuestConfig struct {
	Rewards []Reward `json:"rewards"`
}

type Config struct {
	Version              int                   `json:"version"`
	SourceMasterDataHash string                `json:"sourceMasterDataHash"`
	Quests               map[int32]QuestConfig `json:"quests"`
}

type BuildOptions struct {
	RequireCurrentMasterData bool
	CurrentMasterDataHash    string
}

func DefaultConfig() *Config {
	return &Config{Version: ConfigVersion, Quests: make(map[int32]QuestConfig)}
}

func ReadConfig(path string) (*Config, string, bool, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultConfig(), ContentHash(nil), false, nil
	}
	if err != nil {
		return nil, "", false, fmt.Errorf("read quest drop config: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return nil, "", true, fmt.Errorf("decode quest drop config: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, "", true, err
	}
	normalizeConfig(&config)
	return &config, ContentHash(raw), true, nil
}

func EncodeConfig(config *Config) ([]byte, string, error) {
	if config == nil {
		return nil, "", fmt.Errorf("quest drop config is nil")
	}
	normalizeConfig(config)
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("encode quest drop config: %w", err)
	}
	raw = append(raw, '\n')
	return raw, ContentHash(raw), nil
}

func ContentHash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// BuildOverrides validates and returns only explicitly configured quest pools.
// Quests absent from the result must continue through the legacy master-data
// pickup-row lottery without normalization.
func BuildOverrides(config *Config, catalog *masterdata.QuestCatalog, options BuildOptions) (map[int32][]Reward, error) {
	if config == nil {
		return nil, fmt.Errorf("quest drop config is nil")
	}
	if catalog == nil {
		return nil, fmt.Errorf("quest catalog is nil")
	}
	normalizeConfig(config)
	if config.Version != ConfigVersion {
		return nil, fmt.Errorf("unsupported quest drop config version %d", config.Version)
	}
	if options.RequireCurrentMasterData && options.CurrentMasterDataHash != "" && config.SourceMasterDataHash != options.CurrentMasterDataHash {
		return nil, fmt.Errorf("quest drop config was built for master data %q, current master data is %q", config.SourceMasterDataHash, options.CurrentMasterDataHash)
	}

	overrides := make(map[int32][]Reward, len(config.Quests))
	for questID, questConfig := range config.Quests {
		if _, exists := catalog.QuestById[questID]; !exists {
			return nil, fmt.Errorf("QuestId %d does not exist", questID)
		}
		if len(questConfig.Rewards) > 200 {
			return nil, fmt.Errorf("QuestId %d has too many drop rewards", questID)
		}
		type rewardKey struct {
			id         int32
			guaranteed bool
		}
		seen := make(map[rewardKey]bool, len(questConfig.Rewards))
		var total int64
		for _, reward := range questConfig.Rewards {
			if _, exists := catalog.BattleDropRewardById[reward.BattleDropRewardID]; !exists {
				return nil, fmt.Errorf("QuestId %d references unknown BattleDropRewardId %d", questID, reward.BattleDropRewardID)
			}
			key := rewardKey{id: reward.BattleDropRewardID, guaranteed: reward.Guaranteed}
			if seen[key] {
				return nil, fmt.Errorf("QuestId %d contains duplicate BattleDropRewardId %d in guaranteed=%t group", questID, reward.BattleDropRewardID, reward.Guaranteed)
			}
			seen[key] = true
			if reward.Weight < 1 || reward.Weight > MaxWeight {
				return nil, fmt.Errorf("QuestId %d BattleDropRewardId %d weight must be between 1 and %d", questID, reward.BattleDropRewardID, MaxWeight)
			}
			total += int64(reward.Weight)
		}
		overrides[questID] = append([]Reward(nil), questConfig.Rewards...)
	}
	return overrides, nil
}

func normalizeConfig(config *Config) {
	if config.Quests == nil {
		config.Quests = make(map[int32]QuestConfig)
	}
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra interface{}
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode quest drop config: %w", err)
	}
	return fmt.Errorf("decode quest drop config: multiple JSON values")
}

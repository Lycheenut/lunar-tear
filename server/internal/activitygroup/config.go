package activitygroup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"lunar-tear/server/internal/gacha"
)

const (
	ConfigVersion = 2
	TypePremium   = 1
	TypeRecord    = 2
	TypeVariation = 3
)

// Config is persisted independently in activity-groups.json.
type Config struct {
	Version int             `json:"version"`
	Units   []ActivityUnit  `json:"units"`
	Groups  []ActivityGroup `json:"groups"`
}

type ActivityMember struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
}

type ActivityUnit struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Type    int              `json:"type"`
	Members []ActivityMember `json:"members"`
}

type ActivityGroup struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	UnitIDs []string `json:"unitIds"`
}

func ReadConfig(path string) (*Config, string, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, gacha.ContentHash(nil), nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("read activity group config: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return nil, "", fmt.Errorf("decode activity group config: %w", err)
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		return nil, "", fmt.Errorf("activity group config must contain one JSON object")
	}
	if err := validateConfig(&config); err != nil {
		return nil, "", err
	}
	return &config, gacha.ContentHash(raw), nil
}

func EncodeConfig(config *Config) ([]byte, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func validateConfig(config *Config) error {
	if config == nil || (config.Version != 1 && config.Version != ConfigVersion) {
		return fmt.Errorf("unsupported activity group config version")
	}
	return nil
}

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DirName           = ".kairos"
	ConfigFileName    = "config.json"
	DefaultMaxHistory = 5
	MinMaxHistory     = 1
	MaxHistoryCap     = 500
)

// file is persisted as .kairos/config.json
type file struct {
	MaxHistory int `json:"max_history,omitempty"`
}

func configPath(repoTop string) string {
	return filepath.Join(repoTop, DirName, ConfigFileName)
}

// Read returns the effective max history (clamped), and whether max_history was
// set to a non-zero value in the config file.
func Read(repoTop string) (max int, explicitInFile bool, err error) {
	path := configPath(repoTop)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultMaxHistory, false, nil
		}
		return 0, false, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return 0, false, fmt.Errorf("parse config: %w", err)
	}
	if f.MaxHistory == 0 {
		return DefaultMaxHistory, false, nil
	}
	m, err := ValidateMaxHistory(f.MaxHistory)
	if err != nil {
		return 0, false, err
	}
	return m, true, nil
}

// EffectiveMaxHistory is the window size used for trimming checkpoints.
func EffectiveMaxHistory(repoTop string) (int, error) {
	max, _, err := Read(repoTop)
	return max, err
}

// ValidateMaxHistory enforces MinMaxHistory..MaxHistoryCap.
func ValidateMaxHistory(n int) (int, error) {
	if n < MinMaxHistory || n > MaxHistoryCap {
		return 0, fmt.Errorf("max_history must be between %d and %d", MinMaxHistory, MaxHistoryCap)
	}
	return n, nil
}

// SetMaxHistory writes max_history to .kairos/config.json, merging with existing keys.
func SetMaxHistory(repoTop string, n int) error {
	n, err := ValidateMaxHistory(n)
	if err != nil {
		return err
	}
	dir := filepath.Join(repoTop, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := configPath(repoTop)
	f := file{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &f)
	}
	f.MaxHistory = n
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

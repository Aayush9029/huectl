package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Aayush9029/huectl/internal/api"
)

type Config struct {
	BridgeIP  string            `json:"bridge_ip"`
	AppKey    string            `json:"app_key"`
	Lights    []api.CachedLight `json:"lights,omitempty"`
	UpdatedAt time.Time         `json:"updated_at,omitempty"`
}

func Dir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "huectl")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "huectl")
}

func Path() string {
	return filepath.Join(Dir(), "config.json")
}

func LegacyPath() string {
	return filepath.Join(Dir(), "config")
}

func EnsureDir() error {
	return os.MkdirAll(Dir(), 0o700)
}

func Load() (Config, bool, error) {
	var cfg Config
	loadedLegacy := false

	if data, err := os.ReadFile(Path()); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, false, fmt.Errorf("parse %s: %w", Path(), err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, false, err
	} else if legacy, err := loadLegacy(); err == nil {
		cfg = legacy
		loadedLegacy = true
	}

	if env := os.Getenv("HUE_BRIDGE_IP"); env != "" {
		cfg.BridgeIP = env
	}
	if env := os.Getenv("HUE_APP_KEY"); env != "" {
		cfg.AppKey = env
	}

	return cfg, loadedLegacy, nil
}

func Save(cfg Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}

	cfg.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := Path() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, Path()); err != nil {
		return err
	}
	return os.Chmod(Path(), 0o600)
}

func UpdateLights(cfg Config, lights []api.Light) Config {
	now := time.Now()
	cfg.Lights = cfg.Lights[:0]
	for _, light := range lights {
		cfg.Lights = append(cfg.Lights, light.CacheEntry(now))
	}
	cfg.UpdatedAt = now
	return cfg
}

func loadLegacy() (Config, error) {
	file, err := os.Open(LegacyPath())
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	cfg := Config{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"'`)
		switch key {
		case "BRIDGE_IP", "HUE_BRIDGE_IP":
			cfg.BridgeIP = value
		case "HUE_APP_KEY":
			cfg.AppKey = value
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	if cfg.BridgeIP == "" && cfg.AppKey == "" {
		return Config{}, os.ErrNotExist
	}
	return cfg, nil
}

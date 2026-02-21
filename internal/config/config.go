package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type Config struct {
	Backend BackendConfig         `yaml:"backend" mapstructure:"backend"`
	Pools   map[string]PoolConfig `yaml:"pools"   mapstructure:"pools"`
}

type BackendConfig struct {
	Lock    LockBackendConfig   `yaml:"lock"    mapstructure:"lock"`
	Secrets SecretBackendConfig `yaml:"secrets" mapstructure:"secrets"`
}

type LockBackendConfig struct {
	Type       string `yaml:"type"       mapstructure:"type"`
	Project    string `yaml:"project"    mapstructure:"project"`
	Collection string `yaml:"collection" mapstructure:"collection"`
}

type SecretBackendConfig struct {
	Type    string `yaml:"type"    mapstructure:"type"`
	Project string `yaml:"project" mapstructure:"project"`
}

type PoolConfig struct {
	Slots []SlotConfig  `yaml:"slots" mapstructure:"slots"`
	Keys  []string      `yaml:"keys"  mapstructure:"keys"`
	TTL   time.Duration `yaml:"ttl"   mapstructure:"ttl"`
}

type SlotConfig struct {
	Name string `yaml:"name" mapstructure:"name"`
}

// SlotNames returns the ordered list of slot names in the pool.
func (p *PoolConfig) SlotNames() []string {
	names := make([]string, len(p.Slots))
	for i, s := range p.Slots {
		names[i] = s.Name
	}
	return names
}

// SecretName derives the GCP Secret Manager secret name for a given slot and key.
// Convention: {slot-name}-{kebab-case-key}, e.g. "app-alpha" + "SHOPIFY_API_SECRET" → "app-alpha-shopify-api-secret".
func SecretName(slotName, key string) string {
	kebab := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
	return slotName + "-" + kebab
}

// SecretsForSlot returns a map of env var key → derived secret name for the given slot.
func (p *PoolConfig) SecretsForSlot(slotName string) map[string]string {
	m := make(map[string]string, len(p.Keys))
	for _, key := range p.Keys {
		m[key] = SecretName(slotName, key)
	}
	return m
}

func Load(v *viper.Viper) (*Config, error) {
	var cfg Config
	err := v.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Backend.Lock.Type == "" {
		return fmt.Errorf("backend.lock.type is required")
	}
	if cfg.Backend.Secrets.Type == "" {
		return fmt.Errorf("backend.secrets.type is required")
	}
	if len(cfg.Pools) == 0 {
		return fmt.Errorf("at least one pool must be defined")
	}
	for name, pool := range cfg.Pools {
		if len(pool.Slots) == 0 {
			return fmt.Errorf("pool %q: at least one slot is required", name)
		}
		if len(pool.Keys) == 0 {
			return fmt.Errorf("pool %q: at least one key is required", name)
		}
		if pool.TTL <= 0 {
			return fmt.Errorf("pool %q: ttl must be > 0", name)
		}
		seen := make(map[string]bool)
		for i, slot := range pool.Slots {
			if slot.Name == "" {
				return fmt.Errorf("pool %q: slot %d: name is required", name, i)
			}
			if seen[slot.Name] {
				return fmt.Errorf("pool %q: duplicate slot name %q", name, slot.Name)
			}
			seen[slot.Name] = true
		}
	}
	return nil
}

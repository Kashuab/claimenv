package config

import (
	"fmt"
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
	TTL   time.Duration `yaml:"ttl"   mapstructure:"ttl"`
}

type SlotConfig struct {
	Name   string `yaml:"name"   mapstructure:"name"`
	Secret string `yaml:"secret" mapstructure:"secret"`
}

// SlotNames returns the ordered list of slot names in the pool.
func (p *PoolConfig) SlotNames() []string {
	names := make([]string, len(p.Slots))
	for i, s := range p.Slots {
		names[i] = s.Name
	}
	return names
}

// SecretForSlot returns the secret name for the given slot name.
func (p *PoolConfig) SecretForSlot(slotName string) (string, bool) {
	for _, s := range p.Slots {
		if s.Name == slotName {
			return s.Secret, true
		}
	}
	return "", false
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
		if pool.TTL <= 0 {
			return fmt.Errorf("pool %q: ttl must be > 0", name)
		}
		seen := make(map[string]bool)
		for i, slot := range pool.Slots {
			if slot.Name == "" {
				return fmt.Errorf("pool %q: slot %d: name is required", name, i)
			}
			if slot.Secret == "" {
				return fmt.Errorf("pool %q: slot %q: secret is required", name, slot.Name)
			}
			if seen[slot.Name] {
				return fmt.Errorf("pool %q: duplicate slot name %q", name, slot.Name)
			}
			seen[slot.Name] = true
		}
	}
	return nil
}

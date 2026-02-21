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
	Slots        int           `yaml:"slots"         mapstructure:"slots"`
	TTL          time.Duration `yaml:"ttl"           mapstructure:"ttl"`
	SecretPrefix string        `yaml:"secret_prefix" mapstructure:"secret_prefix"`
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
		if pool.Slots <= 0 {
			return fmt.Errorf("pool %q: slots must be > 0", name)
		}
		if pool.TTL <= 0 {
			return fmt.Errorf("pool %q: ttl must be > 0", name)
		}
		if pool.SecretPrefix == "" {
			return fmt.Errorf("pool %q: secret_prefix is required", name)
		}
	}
	return nil
}

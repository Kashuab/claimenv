package cmd

import (
	"fmt"
	"os"

	"github.com/Kashuab/claimenv/internal/config"
	"github.com/Kashuab/claimenv/internal/engine"
	"github.com/Kashuab/claimenv/internal/identity"
	"github.com/Kashuab/claimenv/internal/lockstore"
	lockmem "github.com/Kashuab/claimenv/internal/lockstore/memory"
	"github.com/Kashuab/claimenv/internal/secretstore"
	secretmem "github.com/Kashuab/claimenv/internal/secretstore/memory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	leaseFile string
	eng       *engine.Engine
)

var rootCmd = &cobra.Command{
	Use:   "claimenv",
	Short: "Claim exclusive environment variable sets from a shared pool",
	Long: `claimenv manages a pool of environment variable groups with exclusive locking.
Use it in CI/CD to claim a set of credentials (e.g. Shopify app keys) for
branch preview deployments, ensuring no two environments share the same credentials.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip engine init for help commands
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}

		v := viper.New()

		if cfgFile != "" {
			v.SetConfigFile(cfgFile)
		} else if envCfg := os.Getenv("CLAIMENV_CONFIG"); envCfg != "" {
			v.SetConfigFile(envCfg)
		} else {
			v.SetConfigName("claimenv")
			v.SetConfigType("yaml")
			v.AddConfigPath(".")
			home, _ := os.UserHomeDir()
			if home != "" {
				v.AddConfigPath(home + "/.config/claimenv")
			}
		}

		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}

		cfg, err := config.Load(v)
		if err != nil {
			return err
		}

		ls, err := newLockStore(cfg.Backend.Lock)
		if err != nil {
			return fmt.Errorf("failed to create lock store: %w", err)
		}

		ss, err := newSecretStore(cfg.Backend.Secrets)
		if err != nil {
			return fmt.Errorf("failed to create secret store: %w", err)
		}

		// Resolve lease file path
		lf := leaseFile
		if lf == "" {
			if envLF := os.Getenv("CLAIMENV_LEASE_FILE"); envLF != "" {
				lf = envLF
			} else {
				lf = ".claimenv"
			}
		}

		eng = &engine.Engine{
			Cfg:         cfg,
			LockStore:   ls,
			SecretStore: ss,
			Identity:    identity.Resolve(),
			LeaseFile:   lf,
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if eng != nil {
			return eng.Close()
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: ./claimenv.yaml)")
	rootCmd.PersistentFlags().StringVar(&leaseFile, "lease-file", "", "lease file path (default: .claimenv)")
}

func newLockStore(cfg config.LockBackendConfig) (lockstore.LockStore, error) {
	switch cfg.Type {
	case "memory":
		return lockmem.New(), nil
	case "firestore":
		return newFirestoreLockStore(cfg)
	default:
		return nil, fmt.Errorf("unknown lock backend type: %q", cfg.Type)
	}
}

func newSecretStore(cfg config.SecretBackendConfig) (secretstore.SecretStore, error) {
	switch cfg.Type {
	case "memory":
		return secretmem.New(), nil
	case "gcp-secret-manager":
		return newGCPSecretStore(cfg)
	default:
		return nil, fmt.Errorf("unknown secret backend type: %q", cfg.Type)
	}
}

package engine

import (
	"context"
	"fmt"

	"github.com/Kashuab/claimenv/internal/config"
	"github.com/Kashuab/claimenv/internal/lease"
	"github.com/Kashuab/claimenv/internal/lockstore"
	"github.com/Kashuab/claimenv/internal/secretstore"
)

type Engine struct {
	Cfg         *config.Config
	LockStore   lockstore.LockStore
	SecretStore secretstore.SecretStore
	Identity    string
	LeaseFile   string
}

func (e *Engine) poolConfig(poolName string) (*config.PoolConfig, error) {
	pool, ok := e.Cfg.Pools[poolName]
	if !ok {
		return nil, fmt.Errorf("pool %q not found in config", poolName)
	}
	return &pool, nil
}

// Claim acquires a free slot in the named pool and returns a LeaseFile.
func (e *Engine) Claim(ctx context.Context, poolName string) (*lease.LeaseFile, error) {
	pool, err := e.poolConfig(poolName)
	if err != nil {
		return nil, err
	}

	claim, err := e.LockStore.Claim(ctx, poolName, pool.SlotNames(), e.Identity, pool.TTL)
	if err != nil {
		return nil, err
	}

	return &lease.LeaseFile{
		Pool:      claim.Pool,
		SlotName:  claim.SlotName,
		LeaseID:   claim.LeaseID,
		Secrets:   pool.SecretsForSlot(claim.SlotName),
		Holder:    claim.Holder,
		ClaimedAt: claim.ClaimedAt,
		ExpiresAt: claim.ExpiresAt,
	}, nil
}

// Release releases the claim described by the lease file.
func (e *Engine) Release(ctx context.Context, lf *lease.LeaseFile) error {
	if _, err := e.LockStore.ValidateLease(ctx, lf.Pool, lf.LeaseID); err != nil {
		return fmt.Errorf("lease validation failed: %w", err)
	}

	return e.LockStore.Release(ctx, lf.Pool, lf.LeaseID)
}

// ReadKey reads a single env var value from the claimed slot.
func (e *Engine) ReadKey(ctx context.Context, lf *lease.LeaseFile, key string) (string, error) {
	secretName, ok := lf.Secrets[key]
	if !ok {
		return "", fmt.Errorf("key %q is not defined in this slot's secrets", key)
	}

	if _, err := e.LockStore.ValidateLease(ctx, lf.Pool, lf.LeaseID); err != nil {
		return "", fmt.Errorf("lease validation failed: %w", err)
	}

	return e.SecretStore.Read(ctx, secretName)
}

// ReadAll reads all env var values from the claimed slot.
func (e *Engine) ReadAll(ctx context.Context, lf *lease.LeaseFile) (map[string]string, error) {
	if _, err := e.LockStore.ValidateLease(ctx, lf.Pool, lf.LeaseID); err != nil {
		return nil, fmt.Errorf("lease validation failed: %w", err)
	}

	result := make(map[string]string, len(lf.Secrets))
	for key, secretName := range lf.Secrets {
		val, err := e.SecretStore.Read(ctx, secretName)
		if err != nil {
			return nil, fmt.Errorf("failed to read secret for key %q: %w", key, err)
		}
		result[key] = val
	}
	return result, nil
}

// WriteKey writes a single env var to the claimed slot.
func (e *Engine) WriteKey(ctx context.Context, lf *lease.LeaseFile, key, value string) error {
	secretName, ok := lf.Secrets[key]
	if !ok {
		return fmt.Errorf("key %q is not defined in this slot's secrets", key)
	}

	if _, err := e.LockStore.ValidateLease(ctx, lf.Pool, lf.LeaseID); err != nil {
		return fmt.Errorf("lease validation failed: %w", err)
	}

	return e.SecretStore.Write(ctx, secretName, value)
}

// SecretName returns the GCP Secret Manager secret name for a key without reading the value.
func (e *Engine) SecretName(lf *lease.LeaseFile, key string) (string, error) {
	secretName, ok := lf.Secrets[key]
	if !ok {
		return "", fmt.Errorf("key %q is not defined in this slot's secrets", key)
	}
	return secretName, nil
}

// Renew extends the TTL on the current claim and returns updated lease info.
func (e *Engine) Renew(ctx context.Context, lf *lease.LeaseFile) (*lease.LeaseFile, error) {
	pool, err := e.poolConfig(lf.Pool)
	if err != nil {
		return nil, err
	}

	claim, err := e.LockStore.Renew(ctx, lf.Pool, lf.LeaseID, pool.TTL)
	if err != nil {
		return nil, err
	}

	return &lease.LeaseFile{
		Pool:      claim.Pool,
		SlotName:  claim.SlotName,
		LeaseID:   claim.LeaseID,
		Secrets:   lf.Secrets,
		Holder:    claim.Holder,
		ClaimedAt: claim.ClaimedAt,
		ExpiresAt: claim.ExpiresAt,
	}, nil
}

// Status returns the status of all slots in the named pool.
func (e *Engine) Status(ctx context.Context, poolName string) ([]lockstore.SlotStatus, error) {
	pool, err := e.poolConfig(poolName)
	if err != nil {
		return nil, err
	}

	return e.LockStore.Status(ctx, poolName, pool.SlotNames())
}

// Close releases resources held by both stores.
func (e *Engine) Close() error {
	var errs []error
	if err := e.LockStore.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := e.SecretStore.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

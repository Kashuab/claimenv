package secretstore

import (
	"context"
	"errors"
)

var (
	ErrSecretNotFound = errors.New("claimenv: secret not found")
	ErrKeyNotFound    = errors.New("claimenv: key not found in secret")
)

// SecretStore manages the credential values within slots.
type SecretStore interface {
	// ReadAll returns all key-value pairs for the given secret name.
	ReadAll(ctx context.Context, secretName string) (map[string]string, error)

	// ReadKey returns a single value from the given secret.
	// Returns ErrKeyNotFound if the key does not exist.
	ReadKey(ctx context.Context, secretName string, key string) (string, error)

	// WriteKey sets a single key-value pair within the given secret.
	// Existing keys are preserved; the target key is created or updated.
	WriteKey(ctx context.Context, secretName string, key string, value string) error

	// Close releases any resources held by the store.
	Close() error
}

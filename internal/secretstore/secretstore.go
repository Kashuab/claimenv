package secretstore

import (
	"context"
	"errors"
)

var (
	ErrSecretNotFound = errors.New("claimenv: secret not found")
)

// SecretStore manages individual secret values.
// Each secret holds a single string value.
type SecretStore interface {
	// Read returns the value of the given secret.
	// Returns ErrSecretNotFound if the secret does not exist.
	Read(ctx context.Context, secretName string) (string, error)

	// Write sets the value of the given secret, creating it if necessary.
	Write(ctx context.Context, secretName string, value string) error

	// Close releases any resources held by the store.
	Close() error
}

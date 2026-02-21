package memory

import (
	"context"
	"sync"

	"github.com/Kashuab/claimenv/internal/secretstore"
)

// Store is a thread-safe in-memory secret store for testing and local development.
type Store struct {
	mu      sync.Mutex
	secrets map[string]string // secret name → value
}

func New() *Store {
	return &Store{
		secrets: make(map[string]string),
	}
}

// Seed pre-populates a secret with a value (useful for testing).
func (s *Store) Seed(secretName string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[secretName] = value
}

func (s *Store) Read(_ context.Context, secretName string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, ok := s.secrets[secretName]
	if !ok {
		return "", secretstore.ErrSecretNotFound
	}
	return val, nil
}

func (s *Store) Write(_ context.Context, secretName string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[secretName] = value
	return nil
}

func (s *Store) Close() error {
	return nil
}

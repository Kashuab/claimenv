package memory

import (
	"context"
	"sync"

	"github.com/Kashuab/claimenv/internal/secretstore"
)

// Store is a thread-safe in-memory secret store for testing and local development.
type Store struct {
	mu      sync.Mutex
	secrets map[string]map[string]string // key: secret name, value: key-value pairs
}

func New() *Store {
	return &Store{
		secrets: make(map[string]map[string]string),
	}
}

// Seed pre-populates a secret with key-value pairs (useful for testing).
func (s *Store) Seed(secretName string, data map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.secrets[secretName] = make(map[string]string, len(data))
	for k, v := range data {
		s.secrets[secretName][k] = v
	}
}

func (s *Store) ReadAll(_ context.Context, secretName string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, ok := s.secrets[secretName]
	if !ok {
		return nil, secretstore.ErrSecretNotFound
	}

	// Return a copy to prevent mutation
	result := make(map[string]string, len(data))
	for k, v := range data {
		result[k] = v
	}
	return result, nil
}

func (s *Store) ReadKey(_ context.Context, secretName string, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, ok := s.secrets[secretName]
	if !ok {
		return "", secretstore.ErrSecretNotFound
	}

	val, ok := data[key]
	if !ok {
		return "", secretstore.ErrKeyNotFound
	}

	return val, nil
}

func (s *Store) WriteKey(_ context.Context, secretName string, key string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.secrets[secretName]; !ok {
		s.secrets[secretName] = make(map[string]string)
	}

	s.secrets[secretName][key] = value
	return nil
}

func (s *Store) Close() error {
	return nil
}

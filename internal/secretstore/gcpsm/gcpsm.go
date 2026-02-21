package gcpsm

import (
	"context"
	"encoding/json"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/Kashuab/claimenv/internal/secretstore"
)

// Store implements secretstore.SecretStore using GCP Secret Manager.
// Each slot is a single secret whose payload is a JSON-encoded map[string]string.
type Store struct {
	client  *secretmanager.Client
	project string
}

func New(ctx context.Context, project string) (*Store, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret manager client: %w", err)
	}
	return &Store{client: client, project: project}, nil
}

func (s *Store) secretResource(secretName string) string {
	return fmt.Sprintf("projects/%s/secrets/%s/versions/latest", s.project, secretName)
}

func (s *Store) parentResource(secretName string) string {
	return fmt.Sprintf("projects/%s/secrets/%s", s.project, secretName)
}

func (s *Store) readPayload(ctx context.Context, secretName string) (map[string]string, error) {
	result, err := s.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: s.secretResource(secretName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access secret %q: %w", secretName, err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Payload.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse secret %q payload as JSON: %w", secretName, err)
	}

	return data, nil
}

func (s *Store) ReadAll(ctx context.Context, secretName string) (map[string]string, error) {
	data, err := s.readPayload(ctx, secretName)
	if err != nil {
		return nil, err
	}

	// Return a copy
	result := make(map[string]string, len(data))
	for k, v := range data {
		result[k] = v
	}
	return result, nil
}

func (s *Store) ReadKey(ctx context.Context, secretName string, key string) (string, error) {
	data, err := s.readPayload(ctx, secretName)
	if err != nil {
		return "", err
	}

	val, ok := data[key]
	if !ok {
		return "", secretstore.ErrKeyNotFound
	}
	return val, nil
}

func (s *Store) WriteKey(ctx context.Context, secretName string, key string, value string) error {
	// Read-modify-write: safe because the lock store guarantees exclusive access
	data, err := s.readPayload(ctx, secretName)
	if err != nil {
		// If the secret has no versions yet, start with an empty map
		data = make(map[string]string)
	}

	data[key] = value

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal secret data: %w", err)
	}

	_, err = s.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: s.parentResource(secretName),
		Payload: &secretmanagerpb.SecretPayload{
			Data: payload,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to write secret version for %q: %w", secretName, err)
	}

	return nil
}

func (s *Store) Close() error {
	return s.client.Close()
}

package gcpsm

import (
	"context"
	"encoding/json"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/Kashuab/claimenv/internal/secretstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (s *Store) projectResource() string {
	return fmt.Sprintf("projects/%s", s.project)
}

func (s *Store) readPayload(ctx context.Context, secretName string) (map[string]string, error) {
	result, err := s.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: s.secretResource(secretName),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, secretstore.ErrSecretNotFound
		}
		return nil, fmt.Errorf("failed to access secret %q: %w", secretName, err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Payload.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse secret %q payload as JSON: %w", secretName, err)
	}

	return data, nil
}

// ensureSecret creates the secret if it doesn't exist. Returns nil if already exists.
func (s *Store) ensureSecret(ctx context.Context, secretName string) error {
	_, err := s.client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   s.projectResource(),
		SecretId: secretName,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil
		}
		return fmt.Errorf("failed to create secret %q: %w", secretName, err)
	}
	return nil
}

func (s *Store) ReadAll(ctx context.Context, secretName string) (map[string]string, error) {
	data, err := s.readPayload(ctx, secretName)
	if err != nil {
		return nil, err
	}

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
	// Read current data, starting fresh if the secret has no versions
	data, err := s.readPayload(ctx, secretName)
	if err != nil {
		data = make(map[string]string)
	}

	data[key] = value

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal secret data: %w", err)
	}

	// Try to add a version; if the secret doesn't exist, create it first
	_, err = s.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: s.parentResource(secretName),
		Payload: &secretmanagerpb.SecretPayload{
			Data: payload,
		},
	})
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return fmt.Errorf("failed to write secret version for %q: %w", secretName, err)
		}

		// Secret doesn't exist — create it and retry
		if createErr := s.ensureSecret(ctx, secretName); createErr != nil {
			return createErr
		}

		_, err = s.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
			Parent: s.parentResource(secretName),
			Payload: &secretmanagerpb.SecretPayload{
				Data: payload,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to write secret version for %q after creating: %w", secretName, err)
		}
	}

	return nil
}

func (s *Store) Close() error {
	return s.client.Close()
}

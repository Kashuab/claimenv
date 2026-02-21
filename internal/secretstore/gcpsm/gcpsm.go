package gcpsm

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/Kashuab/claimenv/internal/secretstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Store implements secretstore.SecretStore using GCP Secret Manager.
// Each secret holds a single string value.
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

func (s *Store) Read(ctx context.Context, secretName string) (string, error) {
	result, err := s.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: s.secretResource(secretName),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", secretstore.ErrSecretNotFound
		}
		return "", fmt.Errorf("failed to access secret %q: %w", secretName, err)
	}

	return string(result.Payload.Data), nil
}

func (s *Store) Write(ctx context.Context, secretName string, value string) error {
	payload := []byte(value)

	// Try to add a version; if the secret doesn't exist, create it first
	_, err := s.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
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

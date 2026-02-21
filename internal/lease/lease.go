package lease

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type LeaseFile struct {
	Pool       string    `json:"pool"`
	SlotName   string    `json:"slot_name"`
	LeaseID    string    `json:"lease_id"`
	SecretName string    `json:"secret_name"`
	Holder     string    `json:"holder"`
	ClaimedAt  time.Time `json:"claimed_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func Load(path string) (*LeaseFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no active lease (file not found: %s)", path)
		}
		return nil, fmt.Errorf("failed to read lease file: %w", err)
	}

	var lf LeaseFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("failed to parse lease file: %w", err)
	}

	return &lf, nil
}

func Save(path string, lf *LeaseFile) error {
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lease file: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write lease file: %w", err)
	}

	return nil
}

func Delete(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete lease file: %w", err)
	}
	return nil
}

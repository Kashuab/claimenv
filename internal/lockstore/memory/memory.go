package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Kashuab/claimenv/internal/lockstore"
	"github.com/google/uuid"
)

// Store is a thread-safe in-memory lock store for testing and local development.
type Store struct {
	mu    sync.Mutex
	slots map[string]*lockstore.Claim // key: "{pool}-{slotName}"
}

func New() *Store {
	return &Store{
		slots: make(map[string]*lockstore.Claim),
	}
}

func slotKey(pool string, slotName string) string {
	return fmt.Sprintf("%s-%s", pool, slotName)
}

func (s *Store) Claim(_ context.Context, pool string, slotNames []string, holder string, ttl time.Duration) (*lockstore.Claim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Check if this holder already has an active claim in the pool
	for _, name := range slotNames {
		key := slotKey(pool, name)
		existing := s.slots[key]
		if existing != nil && existing.Holder == holder && now.Before(existing.ExpiresAt) {
			return existing, nil
		}
	}

	// Otherwise find a free slot
	for _, name := range slotNames {
		key := slotKey(pool, name)
		existing := s.slots[key]

		if existing == nil || now.After(existing.ExpiresAt) {
			claim := &lockstore.Claim{
				Pool:      pool,
				SlotName:  name,
				LeaseID:   uuid.New().String(),
				Holder:    holder,
				ClaimedAt: now,
				ExpiresAt: now.Add(ttl),
			}
			s.slots[key] = claim
			return claim, nil
		}
	}

	return nil, lockstore.ErrPoolExhausted
}

func (s *Store) Release(_ context.Context, pool string, leaseID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, claim := range s.slots {
		if claim.Pool == pool && claim.LeaseID == leaseID {
			delete(s.slots, key)
			return nil
		}
	}

	return lockstore.ErrLeaseNotFound
}

func (s *Store) ReleaseByHolder(_ context.Context, pool string, holder string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for key, claim := range s.slots {
		if claim.Pool == pool && claim.Holder == holder && now.Before(claim.ExpiresAt) {
			delete(s.slots, key)
			return nil
		}
	}

	return lockstore.ErrLeaseNotFound
}

func (s *Store) Renew(_ context.Context, pool string, leaseID string, ttl time.Duration) (*lockstore.Claim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for _, claim := range s.slots {
		if claim.Pool == pool && claim.LeaseID == leaseID {
			if now.After(claim.ExpiresAt) {
				return nil, lockstore.ErrLeaseExpired
			}
			claim.ExpiresAt = now.Add(ttl)
			return claim, nil
		}
	}

	return nil, lockstore.ErrLeaseNotFound
}

func (s *Store) Status(_ context.Context, pool string, slotNames []string) ([]lockstore.SlotStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	statuses := make([]lockstore.SlotStatus, len(slotNames))

	for i, name := range slotNames {
		key := slotKey(pool, name)
		statuses[i] = lockstore.SlotStatus{SlotName: name}

		if claim, ok := s.slots[key]; ok && now.Before(claim.ExpiresAt) {
			statuses[i].Claimed = true
			statuses[i].Claim = claim
		}
	}

	return statuses, nil
}

func (s *Store) ValidateLease(_ context.Context, pool string, leaseID string) (*lockstore.Claim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for _, claim := range s.slots {
		if claim.Pool == pool && claim.LeaseID == leaseID {
			if now.After(claim.ExpiresAt) {
				return nil, lockstore.ErrLeaseExpired
			}
			return claim, nil
		}
	}

	return nil, lockstore.ErrLeaseNotFound
}

func (s *Store) Close() error {
	return nil
}

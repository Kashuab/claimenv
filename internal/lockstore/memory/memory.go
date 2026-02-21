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
	slots map[string]*lockstore.Claim // key: "{pool}-slot-{index}"
}

func New() *Store {
	return &Store{
		slots: make(map[string]*lockstore.Claim),
	}
}

func slotKey(pool string, index int) string {
	return fmt.Sprintf("%s-slot-%d", pool, index)
}

func (s *Store) Claim(_ context.Context, pool string, slots int, holder string, ttl time.Duration) (*lockstore.Claim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for i := 0; i < slots; i++ {
		key := slotKey(pool, i)
		existing := s.slots[key]

		// Slot is free if it doesn't exist or has expired
		if existing == nil || now.After(existing.ExpiresAt) {
			claim := &lockstore.Claim{
				Pool:      pool,
				SlotIndex: i,
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

func (s *Store) Status(_ context.Context, pool string, slots int) ([]lockstore.SlotStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	statuses := make([]lockstore.SlotStatus, slots)

	for i := 0; i < slots; i++ {
		key := slotKey(pool, i)
		statuses[i] = lockstore.SlotStatus{SlotIndex: i}

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

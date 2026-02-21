package lockstore

import (
	"context"
	"errors"
	"time"
)

var (
	ErrPoolExhausted = errors.New("claimenv: all slots in pool are currently claimed")
	ErrLeaseNotFound = errors.New("claimenv: lease not found")
	ErrLeaseExpired  = errors.New("claimenv: lease has expired")
)

// Claim represents an active lease on a slot.
type Claim struct {
	Pool      string    `json:"pool"`
	SlotName  string    `json:"slot_name"`
	LeaseID   string    `json:"lease_id"`
	Holder    string    `json:"holder"`
	ClaimedAt time.Time `json:"claimed_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SlotStatus represents the state of a single slot.
type SlotStatus struct {
	SlotName string `json:"slot_name"`
	Claimed  bool   `json:"claimed"`
	Claim    *Claim `json:"claim,omitempty"`
}

// LockStore manages exclusive leases on pool slots.
type LockStore interface {
	// Claim atomically acquires a free slot in the named pool.
	// slotNames is the list of valid slot names in the pool.
	// Returns ErrPoolExhausted if no slots are available.
	Claim(ctx context.Context, pool string, slotNames []string, holder string, ttl time.Duration) (*Claim, error)

	// Release releases the claim identified by leaseID.
	// Returns ErrLeaseNotFound if the lease does not exist.
	Release(ctx context.Context, pool string, leaseID string) error

	// ReleaseByHolder releases the claim held by the given holder in the pool.
	// Returns ErrLeaseNotFound if no active claim is found for the holder.
	ReleaseByHolder(ctx context.Context, pool string, holder string) error

	// Renew extends the TTL of an existing claim.
	// Returns ErrLeaseNotFound or ErrLeaseExpired as appropriate.
	Renew(ctx context.Context, pool string, leaseID string, ttl time.Duration) (*Claim, error)

	// Status returns the status of all slots in the named pool.
	Status(ctx context.Context, pool string, slotNames []string) ([]SlotStatus, error)

	// ValidateLease checks that a lease is still valid (exists and not expired).
	ValidateLease(ctx context.Context, pool string, leaseID string) (*Claim, error)

	// Close releases any resources held by the store.
	Close() error
}

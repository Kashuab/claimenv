package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/Kashuab/claimenv/internal/lockstore"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Store implements lockstore.LockStore using Google Cloud Firestore.
type Store struct {
	client     *firestore.Client
	collection string
}

// slotDoc is the Firestore document schema for a slot.
type slotDoc struct {
	Pool      string    `firestore:"pool"`
	SlotIndex int       `firestore:"slot_index"`
	LeaseID   string    `firestore:"lease_id"`
	Holder    string    `firestore:"holder"`
	ClaimedAt time.Time `firestore:"claimed_at"`
	ExpiresAt time.Time `firestore:"expires_at"`
}

func New(ctx context.Context, project, collection string) (*Store, error) {
	client, err := firestore.NewClient(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}
	return &Store{client: client, collection: collection}, nil
}

func (s *Store) docID(pool string, index int) string {
	return fmt.Sprintf("%s-slot-%d", pool, index)
}

func (s *Store) docRef(pool string, index int) *firestore.DocumentRef {
	return s.client.Collection(s.collection).Doc(s.docID(pool, index))
}

func (s *Store) Claim(ctx context.Context, pool string, slots int, holder string, ttl time.Duration) (*lockstore.Claim, error) {
	var result *lockstore.Claim

	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		now := time.Now()

		// Read all slot documents
		for i := 0; i < slots; i++ {
			ref := s.docRef(pool, i)
			doc, err := tx.Get(ref)

			isFree := false

			if err != nil {
				if status.Code(err) == codes.NotFound {
					isFree = true
				} else {
					return fmt.Errorf("failed to read slot %d: %w", i, err)
				}
			} else {
				var sd slotDoc
				if err := doc.DataTo(&sd); err != nil {
					return fmt.Errorf("failed to parse slot %d: %w", i, err)
				}
				// Slot is free if lease_id is empty or lease has expired
				if sd.LeaseID == "" || now.After(sd.ExpiresAt) {
					isFree = true
				}
			}

			if isFree {
				claim := &lockstore.Claim{
					Pool:      pool,
					SlotIndex: i,
					LeaseID:   uuid.New().String(),
					Holder:    holder,
					ClaimedAt: now,
					ExpiresAt: now.Add(ttl),
				}

				sd := slotDoc{
					Pool:      claim.Pool,
					SlotIndex: claim.SlotIndex,
					LeaseID:   claim.LeaseID,
					Holder:    claim.Holder,
					ClaimedAt: claim.ClaimedAt,
					ExpiresAt: claim.ExpiresAt,
				}

				if err := tx.Set(ref, sd); err != nil {
					return fmt.Errorf("failed to write slot %d: %w", i, err)
				}

				result = claim
				return nil
			}
		}

		return lockstore.ErrPoolExhausted
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) Release(ctx context.Context, pool string, leaseID string) error {
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Find the slot with this lease
		iter := tx.Documents(s.client.Collection(s.collection).Where("pool", "==", pool).Where("lease_id", "==", leaseID))
		docs, err := iter.GetAll()
		if err != nil {
			return fmt.Errorf("failed to query for lease: %w", err)
		}

		if len(docs) == 0 {
			return lockstore.ErrLeaseNotFound
		}

		// Clear the claim by setting lease_id to empty
		return tx.Update(docs[0].Ref, []firestore.Update{
			{Path: "lease_id", Value: ""},
			{Path: "holder", Value: ""},
		})
	})
}

func (s *Store) Renew(ctx context.Context, pool string, leaseID string, ttl time.Duration) (*lockstore.Claim, error) {
	var result *lockstore.Claim

	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		now := time.Now()

		iter := tx.Documents(s.client.Collection(s.collection).Where("pool", "==", pool).Where("lease_id", "==", leaseID))
		docs, err := iter.GetAll()
		if err != nil {
			return fmt.Errorf("failed to query for lease: %w", err)
		}

		if len(docs) == 0 {
			return lockstore.ErrLeaseNotFound
		}

		var sd slotDoc
		if err := docs[0].DataTo(&sd); err != nil {
			return fmt.Errorf("failed to parse slot: %w", err)
		}

		if now.After(sd.ExpiresAt) {
			return lockstore.ErrLeaseExpired
		}

		newExpiry := now.Add(ttl)
		if err := tx.Update(docs[0].Ref, []firestore.Update{
			{Path: "expires_at", Value: newExpiry},
		}); err != nil {
			return err
		}

		result = &lockstore.Claim{
			Pool:      sd.Pool,
			SlotIndex: sd.SlotIndex,
			LeaseID:   sd.LeaseID,
			Holder:    sd.Holder,
			ClaimedAt: sd.ClaimedAt,
			ExpiresAt: newExpiry,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) Status(ctx context.Context, pool string, slots int) ([]lockstore.SlotStatus, error) {
	now := time.Now()
	statuses := make([]lockstore.SlotStatus, slots)

	for i := 0; i < slots; i++ {
		statuses[i] = lockstore.SlotStatus{SlotIndex: i}

		doc, err := s.docRef(pool, i).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			return nil, fmt.Errorf("failed to read slot %d: %w", i, err)
		}

		var sd slotDoc
		if err := doc.DataTo(&sd); err != nil {
			return nil, fmt.Errorf("failed to parse slot %d: %w", i, err)
		}

		if sd.LeaseID != "" && now.Before(sd.ExpiresAt) {
			statuses[i].Claimed = true
			statuses[i].Claim = &lockstore.Claim{
				Pool:      sd.Pool,
				SlotIndex: sd.SlotIndex,
				LeaseID:   sd.LeaseID,
				Holder:    sd.Holder,
				ClaimedAt: sd.ClaimedAt,
				ExpiresAt: sd.ExpiresAt,
			}
		}
	}

	return statuses, nil
}

func (s *Store) ValidateLease(ctx context.Context, pool string, leaseID string) (*lockstore.Claim, error) {
	now := time.Now()

	iter := s.client.Collection(s.collection).Where("pool", "==", pool).Where("lease_id", "==", leaseID).Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to query for lease: %w", err)
	}

	if len(docs) == 0 {
		return nil, lockstore.ErrLeaseNotFound
	}

	var sd slotDoc
	if err := docs[0].DataTo(&sd); err != nil {
		return nil, fmt.Errorf("failed to parse slot: %w", err)
	}

	if now.After(sd.ExpiresAt) {
		return nil, lockstore.ErrLeaseExpired
	}

	return &lockstore.Claim{
		Pool:      sd.Pool,
		SlotIndex: sd.SlotIndex,
		LeaseID:   sd.LeaseID,
		Holder:    sd.Holder,
		ClaimedAt: sd.ClaimedAt,
		ExpiresAt: sd.ExpiresAt,
	}, nil
}

func (s *Store) Close() error {
	return s.client.Close()
}

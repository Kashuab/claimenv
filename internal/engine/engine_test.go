package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/Kashuab/claimenv/internal/config"
	"github.com/Kashuab/claimenv/internal/engine"
	"github.com/Kashuab/claimenv/internal/lockstore"
	lockmem "github.com/Kashuab/claimenv/internal/lockstore/memory"
	secretmem "github.com/Kashuab/claimenv/internal/secretstore/memory"
)

func testEngine() (*engine.Engine, *lockmem.Store, *secretmem.Store) {
	ls := lockmem.New()
	ss := secretmem.New()

	cfg := &config.Config{
		Pools: map[string]config.PoolConfig{
			"testpool": {
				Keys: []string{"SHOPIFY_API_KEY", "APP_URL"},
				Slots: []config.SlotConfig{
					{Name: "alpha"},
					{Name: "beta"},
				},
				TTL: 1 * time.Hour,
			},
		},
	}

	e := &engine.Engine{
		Cfg:         cfg,
		LockStore:   ls,
		SecretStore: ss,
		Identity:    "test-holder",
		LeaseFile:   "/tmp/test-claimenv",
	}

	return e, ls, ss
}

func TestClaim(t *testing.T) {
	e, _, _ := testEngine()
	ctx := context.Background()

	lf, err := e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	if lf.Pool != "testpool" {
		t.Errorf("expected pool 'testpool', got %q", lf.Pool)
	}
	if lf.SlotName != "alpha" {
		t.Errorf("expected slot 'alpha', got %q", lf.SlotName)
	}
	if lf.Secrets["SHOPIFY_API_KEY"] != "alpha-shopify-api-key" {
		t.Errorf("expected secret name 'alpha-shopify-api-key', got %q", lf.Secrets["SHOPIFY_API_KEY"])
	}
	if lf.Secrets["APP_URL"] != "alpha-app-url" {
		t.Errorf("expected secret name 'alpha-app-url', got %q", lf.Secrets["APP_URL"])
	}
	if lf.Holder != "test-holder" {
		t.Errorf("expected holder 'test-holder', got %q", lf.Holder)
	}
	if lf.LeaseID == "" {
		t.Error("expected non-empty lease ID")
	}
}

func TestClaimExhaustsPool(t *testing.T) {
	e, _, _ := testEngine()
	ctx := context.Background()

	// Claim both slots
	_, err := e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("first claim failed: %v", err)
	}
	_, err = e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("second claim failed: %v", err)
	}

	// Third claim should fail
	_, err = e.Claim(ctx, "testpool")
	if err != lockstore.ErrPoolExhausted {
		t.Errorf("expected ErrPoolExhausted, got %v", err)
	}
}

func TestClaimInvalidPool(t *testing.T) {
	e, _, _ := testEngine()
	ctx := context.Background()

	_, err := e.Claim(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent pool")
	}
}

func TestRelease(t *testing.T) {
	e, _, _ := testEngine()
	ctx := context.Background()

	lf, err := e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	err = e.Release(ctx, lf)
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Should be able to claim again
	lf2, err := e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("re-Claim failed: %v", err)
	}
	if lf2.SlotName != "alpha" {
		t.Errorf("expected slot 'alpha' to be reclaimed, got %q", lf2.SlotName)
	}
}

func TestReadWriteKey(t *testing.T) {
	e, _, ss := testEngine()
	ctx := context.Background()

	// Seed the secret store with a value for the derived secret name
	ss.Seed("alpha-shopify-api-key", "test-key-123")

	lf, err := e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Read existing key
	val, err := e.ReadKey(ctx, lf, "SHOPIFY_API_KEY")
	if err != nil {
		t.Fatalf("ReadKey failed: %v", err)
	}
	if val != "test-key-123" {
		t.Errorf("expected 'test-key-123', got %q", val)
	}

	// Write to a key
	err = e.WriteKey(ctx, lf, "APP_URL", "https://preview.example.com")
	if err != nil {
		t.Fatalf("WriteKey failed: %v", err)
	}

	// Read it back
	val, err = e.ReadKey(ctx, lf, "APP_URL")
	if err != nil {
		t.Fatalf("ReadKey after write failed: %v", err)
	}
	if val != "https://preview.example.com" {
		t.Errorf("expected 'https://preview.example.com', got %q", val)
	}
}

func TestReadWriteKeyUndefinedKey(t *testing.T) {
	e, _, _ := testEngine()
	ctx := context.Background()

	lf, err := e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Read undefined key should error
	_, err = e.ReadKey(ctx, lf, "UNDEFINED_KEY")
	if err == nil {
		t.Error("expected error for undefined key")
	}

	// Write undefined key should error
	err = e.WriteKey(ctx, lf, "UNDEFINED_KEY", "value")
	if err == nil {
		t.Error("expected error for undefined key")
	}
}

func TestReadAll(t *testing.T) {
	e, _, ss := testEngine()
	ctx := context.Background()

	// Seed both derived secret names
	ss.Seed("alpha-shopify-api-key", "val_a")
	ss.Seed("alpha-app-url", "val_b")

	lf, err := e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	all, err := e.ReadAll(ctx, lf)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("expected 2 keys, got %d", len(all))
	}
	if all["SHOPIFY_API_KEY"] != "val_a" {
		t.Errorf("expected SHOPIFY_API_KEY='val_a', got %q", all["SHOPIFY_API_KEY"])
	}
	if all["APP_URL"] != "val_b" {
		t.Errorf("expected APP_URL='val_b', got %q", all["APP_URL"])
	}
}

func TestSecretName(t *testing.T) {
	e, _, _ := testEngine()
	ctx := context.Background()

	lf, err := e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	name, err := e.SecretName(lf, "SHOPIFY_API_KEY")
	if err != nil {
		t.Fatalf("SecretName failed: %v", err)
	}
	if name != "alpha-shopify-api-key" {
		t.Errorf("expected 'alpha-shopify-api-key', got %q", name)
	}

	_, err = e.SecretName(lf, "NONEXISTENT")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}

func TestRenew(t *testing.T) {
	e, _, _ := testEngine()
	ctx := context.Background()

	lf, err := e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	originalExpiry := lf.ExpiresAt

	// Small sleep to ensure time advances
	time.Sleep(10 * time.Millisecond)

	renewed, err := e.Renew(ctx, lf)
	if err != nil {
		t.Fatalf("Renew failed: %v", err)
	}

	if !renewed.ExpiresAt.After(originalExpiry) {
		t.Error("expected renewed expiry to be after original")
	}

	// Secrets map should be preserved
	if renewed.Secrets["SHOPIFY_API_KEY"] != lf.Secrets["SHOPIFY_API_KEY"] {
		t.Error("expected secrets map to be preserved after renew")
	}
}

func TestStatus(t *testing.T) {
	e, _, _ := testEngine()
	ctx := context.Background()

	// Before any claims
	statuses, err := e.Status(ctx, "testpool")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(statuses))
	}
	if statuses[0].Claimed || statuses[1].Claimed {
		t.Error("expected both slots to be free")
	}
	if statuses[0].SlotName != "alpha" {
		t.Errorf("expected slot name 'alpha', got %q", statuses[0].SlotName)
	}

	// After one claim
	_, err = e.Claim(ctx, "testpool")
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	statuses, err = e.Status(ctx, "testpool")
	if err != nil {
		t.Fatalf("Status after claim failed: %v", err)
	}
	if !statuses[0].Claimed {
		t.Error("expected slot 'alpha' to be claimed")
	}
	if statuses[1].Claimed {
		t.Error("expected slot 'beta' to be free")
	}
}

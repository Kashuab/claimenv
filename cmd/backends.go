package cmd

import (
	"context"

	"github.com/Kashuab/claimenv/internal/config"
	"github.com/Kashuab/claimenv/internal/lockstore"
	firestorelock "github.com/Kashuab/claimenv/internal/lockstore/firestore"
	"github.com/Kashuab/claimenv/internal/secretstore"
	"github.com/Kashuab/claimenv/internal/secretstore/gcpsm"
)

func newFirestoreLockStore(cfg config.LockBackendConfig) (lockstore.LockStore, error) {
	return firestorelock.New(context.Background(), cfg.Project, cfg.Collection)
}

func newGCPSecretStore(cfg config.SecretBackendConfig) (secretstore.SecretStore, error) {
	return gcpsm.New(context.Background(), cfg.Project)
}

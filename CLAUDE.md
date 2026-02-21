# claimenv

CLI tool for claiming exclusive environment variable sets from a shared pool. Built for CI/CD branch preview deployments where each preview needs its own set of credentials (e.g. Shopify app keys).

## Project Structure

```
main.go                              # Entry point
cmd/                                 # Cobra CLI commands
  root.go                            # Config loading, backend factory, engine wiring
  backends.go                        # GCP backend constructors
  claim.go release.go read.go        # Core commands
  env.go write.go renew.go status.go
internal/
  config/config.go                   # YAML config types + Viper loading
  engine/engine.go                   # Core orchestration (composes lock + secret stores)
  identity/identity.go               # Resolve holder ID from CI env vars
  lease/lease.go                     # Local .claimenv lease file CRUD
  lockstore/lockstore.go             # LockStore interface
  lockstore/firestore/firestore.go   # Firestore implementation (atomic transactions)
  lockstore/memory/memory.go         # In-memory implementation
  secretstore/secretstore.go         # SecretStore interface
  secretstore/gcpsm/gcpsm.go         # GCP Secret Manager implementation
  secretstore/memory/memory.go       # In-memory implementation
```

## Architecture

Two pluggable backend interfaces:
- **LockStore** (`internal/lockstore/lockstore.go`): manages exclusive leases on named pool slots
- **SecretStore** (`internal/secretstore/secretstore.go`): reads/writes individual secret values

The **Engine** (`internal/engine/engine.go`) orchestrates both stores. CLI commands are thin wrappers that delegate to the engine.

Slots are named (not numbered). Env var keys are defined at the pool level. Each key gets its own GCP Secret Manager secret, with names derived by convention: `{slot-name}-{kebab-key}` (e.g. slot `app-alpha` + key `SHOPIFY_API_SECRET` → secret `app-alpha-shopify-api-secret`). The GCP SM backend auto-creates secrets on first write if they don't exist.

## Build & Test

```bash
go build -o claimenv .
go test ./...
go vet ./...
```

## Config

Config file at `./claimenv.yaml`, `~/.config/claimenv/config.yaml`, or `CLAIMENV_CONFIG` env var. See `claimenv.example.yaml`.

## Adding a New Backend

1. Create a new package under `internal/lockstore/` or `internal/secretstore/`
2. Implement the interface from `lockstore.go` or `secretstore.go`
3. Add a case to the factory function in `cmd/root.go`

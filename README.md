# claimenv

A CLI tool for claiming exclusive sets of environment variables from a shared pool. Designed for CI/CD pipelines where branch preview deployments each need their own credentials.

## The Problem

You have a team working on a Shopify app with branch preview environments. Each preview needs its own Shopify app credentials (API key, secret, etc.), but you can't share them across previews without conflicts. You need a pool of pre-provisioned credential sets that deploy jobs can claim exclusively.

## How It Works

```
Pool "onboard"
  app-alpha  { SHOPIFY_API_KEY=aaa, SHOPIFY_API_SECRET=bbb }  <- claimed by MR !423
  app-beta   { SHOPIFY_API_KEY=ccc, SHOPIFY_API_SECRET=ddd }  <- free
  app-gamma  { SHOPIFY_API_KEY=eee, SHOPIFY_API_SECRET=fff }  <- claimed by MR !518
  app-delta  { ... }                                           <- free
```

`claimenv` atomically claims a free slot, gives you access to its credentials, and releases it when you're done. Leases have a TTL so crashed/cancelled jobs don't hold slots forever.

## Install

```bash
go install github.com/Kashuab/claimenv@latest
```

Or build from source:

```bash
git clone https://github.com/Kashuab/claimenv.git
cd claimenv
go build -o claimenv .
```

## Usage

```bash
# Claim a slot from the "onboard" pool
claimenv claim onboard

# Source all credentials into your shell
eval $(claimenv env)

# Read a single value
claimenv read SHOPIFY_API_KEY

# Write a value back (e.g. set the preview URL)
claimenv write APP_URL https://mr-423.preview.example.com

# Check pool status
claimenv status onboard

# Extend your lease
claimenv renew

# Release when done
claimenv release
```

## Configuration

Create a `claimenv.yaml` in your project root (or set `CLAIMENV_CONFIG`):

```yaml
backend:
  lock:
    type: firestore
    project: my-gcp-project
    collection: claimenv-locks
  secrets:
    type: gcp-secret-manager
    project: my-gcp-project

pools:
  onboard:
    ttl: 4h
    slots:
      - name: app-alpha
        secret: onboard-app-alpha
      - name: app-beta
        secret: onboard-app-beta
      - name: app-gamma
        secret: onboard-app-gamma
      - name: app-delta
        secret: onboard-app-delta
```

Each slot has a `name` (shown in status output and logs) and a `secret` (the GCP Secret Manager secret name holding that slot's credentials as a JSON object).

Config file lookup order:
1. `CLAIMENV_CONFIG` env var
2. `--config` flag
3. `./claimenv.yaml`
4. `~/.config/claimenv/config.yaml`

## Backends

### Lock Store (claim coordination)

| Backend | Config `type` | Description |
|---------|--------------|-------------|
| Firestore | `firestore` | Atomic transactions, TTL support. Recommended for production. |
| Memory | `memory` | Ephemeral, per-process. For development and testing only. |

### Secret Store (credential storage)

| Backend | Config `type` | Description |
|---------|--------------|-------------|
| GCP Secret Manager | `gcp-secret-manager` | Each slot is a secret with a JSON payload of key-value pairs. |
| Memory | `memory` | Ephemeral, per-process. For development and testing only. |

## GCP Setup

### Prerequisites

- A GCP project with Firestore and Secret Manager APIs enabled
- Application Default Credentials configured (`gcloud auth application-default login`)

### Provisioning the Pool

Secrets are auto-created by `claimenv write` if they don't exist. You can also pre-provision them:

```bash
# Pre-create secrets with initial credentials
for name in app-alpha app-beta app-gamma app-delta; do
  gcloud secrets create "onboard-${name}" --project=my-gcp-project

  echo '{"SHOPIFY_API_KEY":"key-'${name}'","SHOPIFY_API_SECRET":"secret-'${name}'"}' | \
    gcloud secrets versions add "onboard-${name}" --data-file=- --project=my-gcp-project
done
```

No Firestore setup is needed -- documents are created automatically on first claim.

## GitLab CI Example

```yaml
deploy_preview:
  stage: deploy
  script:
    - claimenv claim onboard
    - eval $(claimenv env)
    - claimenv write APP_URL "https://${CI_MERGE_REQUEST_IID}.preview.example.com"
    - ./deploy.sh
  after_script:
    - claimenv release
  environment:
    name: preview/$CI_MERGE_REQUEST_IID
    on_stop: stop_preview

stop_preview:
  stage: deploy
  when: manual
  script:
    - claimenv release
```

## Lease Management

- Claims are identified by a UUID lease ID stored in a local `.claimenv` file
- The holder identity is auto-detected from CI environment variables (`CI_JOB_ID`, `GITHUB_RUN_ID`, etc.) or falls back to the hostname
- Expired leases are automatically treated as free slots during claiming (lazy cleanup)
- Override the lease file location with `--lease-file` or `CLAIMENV_LEASE_FILE`

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Pool exhausted / general error |

## License

MIT

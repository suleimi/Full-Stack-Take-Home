# EIAE -- Emissions Ingestion and Analysis Engine

Backend service for a multi-tenant emissions monitoring platform. Ingests emission readings from field devices, sensors, and satellites; rolls them up into per-period aggregates; and detects when an organization, site, or asset exceeds its configured emission policy.

## Prerequisites

- Go 1.25+
- PostgreSQL 16+
- Redis 7+ (optional but recommended)
- Docker and Docker Compose (for local development)

## Quick Start

```bash
# From the repo root, start all services (Postgres, Redis, backend, frontend)
docker compose up --build

# Or run the backend locally against an existing Postgres/Redis
cd backend
go run .
```

The server starts on the port specified by the `PORT` environment variable (default `8080`).

## File Structure

```
backend/
├── main.go                          # Entrypoint: wires dependencies, registers routes, starts server
├── Dockerfile                       # Multi-stage build (golang:1.25-alpine -> alpine)
├── go.mod / go.sum                  # Go module definition (module: eiae)
├── ARCHITECTURE.md                  # System design decisions and production topology
├── DATABASE.md                      # Schema diagrams and rollup/violation engine docs
│
├── internal/                        # Private application code (Go convention)
│   ├── config/
│   │   └── config.go                # Viper-based env var loading into Config struct
│   │
│   ├── store/                       # Data access layer (PostgreSQL)
│   │   ├── db.go                    # IStore interface, Store struct, NewStore constructor
│   │   ├── models.go                # All domain model structs + typed enums
│   │   ├── organizations.go         # Org CRUD + summary + emission trend queries
│   │   ├── sites.go                 # Site CRUD + emission trend queries
│   │   ├── assets.go                # Asset CRUD + emission trend queries
│   │   ├── asset_types.go           # Asset type lookup queries
│   │   ├── users.go                 # User CRUD queries
│   │   ├── recorder_devices.go      # Device CRUD + user assignment/revocation
│   │   ├── readings.go              # Batch insert (plain + transactional) + read queries
│   │   ├── policies.go              # Policy CRUD + retire + entity/org listing
│   │   ├── rollups.go               # Rollup trend queries + total-to-date + RefreshEmissions
│   │   ├── violations.go            # Violation listing + acknowledge + counts
│   │   └── store_integration_test.go
│   │
│   ├── handlers/v1/                 # HTTP handlers (Gin), versioned under /v1
│   │   ├── app.go                   # AppV1 struct: holds store, cache, job client, logger, config
│   │   ├── helpers.go               # Shared response helpers
│   │   ├── orgs.go                  # Organization endpoints
│   │   ├── sites.go                 # Site endpoints
│   │   ├── assets.go                # Asset endpoints
│   │   ├── devices.go               # Recorder device endpoints
│   │   ├── readings.go              # Reading ingestion endpoint
│   │   ├── emissions.go             # Emission trend + total + refresh endpoints
│   │   ├── policies.go              # Policy CRUD + retire endpoints
│   │   ├── violations.go            # Violation list + acknowledge endpoints
│   │   ├── users.go                 # User endpoints
│   │   └── handlers_integration_test.go
│   │
│   ├── jobs/                        # Background job definitions (River)
│   │   ├── job.go                   # JoClient interface and RiverJobClient implementation
│   │   ├── jobs.go                  # Job arg types (CalculateEmissionArgs) + uniqueness opts
│   │   └── workers.go               # CalculateEmissionWorker: calls refresh_emissions()
│   │
│   ├── cache/
│   │   └── cahce.go                 # Cache interface + Redis implementation
│   │
│   ├── logger/
│   │   └── logger.go                # Structured JSON logger (slog backend, *log.Logger output)
│   │
│   └── metrics/
│       └── metrics.go               # Prometheus metrics registry + handler
│
└── migration/                       # Database migrations
    ├── migration.go                 # Goose migrator: embeds SQL, runs on startup
    ├── river.go                     # River schema migration (Go-based, version 6)
    └── sql/
        ├── 002_create_fountation.sql  # Full schema: tables, indexes, stored procedure, views
        ├── 003_seed_asset_types.sql   # Seeds 10 standard asset types
        ├── 004_seed_recorder_devices.sql  # Seeds 3 global field devices
        └── 005_seed_sample_data.sql   # Sample org with sites, assets, readings, policies, violations
```

Note: Migration version 1 is the River schema (Go-based, registered in `migration.go`). SQL migrations start at version 2.

## Conventions

### Code Organisation

- All application code lives under `internal/` -- Go enforces that nothing outside this module can import it.
- The **store layer** (`internal/store/`) owns all database access. Every query function is a method on `*Store` and is declared in the `IStore` interface (`db.go`) so handlers depend on an interface, not the concrete type.
- Store files are split by domain entity (one file per table group), not by operation type. CRUD for `sites` lives in `sites.go`, not in a generic `queries.go`.
- **Handlers** (`internal/handlers/v1/`) receive an `*AppV1` receiver and call store methods. They do not contain SQL.
- Domain types use Go typed enums (`EntityType`, `DeviceType`, `Period`, `EmissionUnit`, `Grain`) defined in `models.go`. Handlers convert incoming strings to these types at the boundary.

### Testing

- Use [testify](https://github.com/stretchr/testify) (`v1.11`) for assertions and mocks. It is already a dependency in `go.mod`.
- Test files sit next to the code they test: `store/store_integration_test.go` tests store functions.
- Name test functions `Test<Function>_<scenario>`, e.g. `TestCreateOrganization_DuplicateEmail`.
- Store tests should run against a real PostgreSQL instance (integration tests), not mocks. Use a test-scoped transaction that rolls back after each test to keep the database clean.
- Handler tests may mock the `IStore` interface for unit tests.

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific package
go test -v ./internal/store/...
```

### Commits

Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):

```
<type>(<scope>): <short summary>

<optional body>
```

| Type | When to use |
| --- | --- |
| `feat` | New feature or endpoint |
| `fix` | Bug fix |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test` | Adding or updating tests |
| `docs` | Documentation only |
| `chore` | Build config, CI, dependencies |
| `perf` | Performance improvement |

Examples:

```
feat(store): add batch reading insertion with transaction support
fix(rollups): correct daily bucket truncation in refresh_emissions
docs: add DATABASE.md with schema diagrams
chore(deps): bump river to v0.37.1
```

### Migrations

- Migrations are managed by [Goose](https://github.com/pressly/goose) and live in `migration/sql/`.
- File naming: `NNN_short_description.sql` where `NNN` is a zero-padded sequence number (e.g. `006_add_yearly_rollup.sql`).
- Each file must contain `-- +goose Up` and `-- +goose Down` annotations. If a migration is not reversible, add `-- +goose Down` with a comment explaining why.
- PL/pgSQL blocks containing `$` dollar-quoting or internal semicolons must be wrapped in `-- +goose StatementBegin` / `-- +goose StatementEnd`.
- Migrations run automatically on application startup (via `migrator.Up()` in `main.go`).
- Go-based migrations (like the River schema setup in `migration/river.go`) are registered in the Goose provider alongside the SQL files. The River migration occupies version 1; SQL migrations start at version 2.
- SQL files are embedded into the binary at compile time via `//go:embed sql/*.sql` in `migration/migration.go`.
- **Never** edit a migration that has already been applied to a shared environment. Write a new migration instead.

```bash
# Create a new migration
goose -dir migration/sql create add_yearly_rollup sql

# Apply pending migrations manually (if not relying on startup)
goose -dir migration/sql postgres "$DATABASE_URL" up

# Roll back the last migration
goose -dir migration/sql postgres "$DATABASE_URL" down
```

### Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `DEBUG` | Log level: DEBUG, INFO, WARN, ERROR |
| `POSTGRES_HOST` | -- | PostgreSQL hostname |
| `POSTGRES_PORT` | -- | PostgreSQL port |
| `POSTGRES_DB` | -- | Database name |
| `POSTGRES_USER` | -- | Database user |
| `POSTGRES_PASSWORD` | -- | Database password |
| `REDIS_HOST` | -- | Redis hostname |
| `REDIS_PORT` | -- | Redis port |

### API Endpoints

All endpoints are under `/v1` except health and metrics.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/healthz` | Health check (returns `{"status": "up"}`) |
| `GET` | `/metric` | Prometheus metrics |
| `GET` | `/v1/orgs` | List organizations |
| `POST` | `/v1/orgs` | Create organization |
| `GET` | `/v1/orgs/:org_id` | Get organization by ID |
| `GET` | `/v1/orgs/:org_id/summary` | Get org summary (site/asset counts) |
| `GET` | `/v1/orgs/:org_id/sites` | List sites for an org |
| `POST` | `/v1/orgs/:org_id/sites` | Create site |
| `GET` | `/v1/asset-types` | List asset types |
| `POST` | `/v1/asset-types` | Create asset type |
| `GET` | `/v1/orgs/:org_id/sites/:site_id/assets` | List assets for a site |
| `POST` | `/v1/orgs/:org_id/sites/:site_id/assets` | Create asset |
| `POST` | `/v1/devices` | Create recorder device |
| `GET` | `/v1/devices/field` | List field devices (org-independent) |
| `GET` | `/v1/orgs/:org_id/devices` | List org-scoped devices |
| `POST` | `/v1/orgs/:org_id/sites/:site_id/readings` | Ingest batch of emission readings |
| `GET` | `/v1/orgs/:org_id/emissions/trend` | Get emission trend (rollup time series) |
| `GET` | `/v1/orgs/:org_id/emissions/total` | Get emission total to date |
| `POST` | `/v1/emissions/refresh` | Trigger manual rollup refresh |
| `GET` | `/v1/orgs/:org_id/policies` | List emission policies |
| `POST` | `/v1/orgs/:org_id/policies` | Create emission policy |
| `POST` | `/v1/policies/:policy_id/retire` | Retire a policy |
| `GET` | `/v1/orgs/:org_id/violations` | List emission violations |
| `POST` | `/v1/violations/:violation_id/acknowledge` | Acknowledge a violation |

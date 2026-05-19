# Architecture

The goal here is not to build a fully fledged production-ready system but to highlight and build out the foundational elements that make a system resilient and extensible for a production-grade, multi-tenant platform capable of facilitating asset/site/org-level emission data management.

## Language Decision

Go was chosen as the backend language for two reasons:

1. **Simplicity** -- Go's syntax is minimal and opinionated, which makes onboarding fast and code reviews straightforward.
2. **Concurrency as a first-class citizen** -- Go handles concurrency with goroutines, which are lightweight, user-space threads multiplexed onto OS threads by the Go runtime. This is further leveraged by Go's `net/http` library, which spawns a goroutine per incoming request automatically, enabling high throughput at minimal memory cost. By contrast, Node.js uses a single-threaded event loop; while non-blocking I/O keeps it responsive, CPU-bound work blocks the loop unless offloaded to worker threads.

## PostgreSQL

PostgreSQL was chosen as the primary datastore for four reasons:

1. **Relational fit** -- the data being modelled (organizations, sites, assets, readings) is intrinsically relational, with strict hierarchical foreign-key relationships.
2. **Developer familiarity** -- SQL is by far the most widely known database query language, lowering the barrier for new contributors.
3. **Transactional guarantees** -- PostgreSQL's ACID transactions let us atomically insert a batch of readings and enqueue a background job in the same commit, avoiding inconsistencies that would arise with eventually-consistent stores.
4. **Scalability tooling** -- features like declarative partitioning (used on `asset_emission_readings` and `emission_rollup_hourly`), partial indexes, exclusion constraints, and autovacuum let PostgreSQL scale with the system without requiring an architecture change.

The schema is defined in `migration/sql/002_create_fountation.sql` and applied via Goose (v3.27). Migration version 1 is reserved for the River job queue schema (Go-based).

## Redis

Redis (or Valkey) is included as an optional but important addition to keep the system resilient under load. It is used for:

- **Caching** -- reducing queries against PostgreSQL (e.g. duplicate data emission).
- **Distributed rate limiting (future)** -- storing rate-limit counters that are shared across multiple application instances.

The cache layer is abstracted behind a `Cache` interface (`internal/cache/`) so the application can run without Redis if needed (degraded mode).

## Packages and Libraries

| Library | Purpose |
|---|---|
| **[River](https://riverqueue.com)** (v0.37) | A PostgreSQL-native job queue. Jobs are enqueued inside the same database transaction as the data they depend on, guaranteeing delivery. Used to run `refresh_emissions()` on two schedules: every 1 minute (3-hour window) and every hour (48-hour window). Both run on start. |
| **[Goose](https://github.com/pressly/goose)** (v3.27) | Migration engine responsible for applying DDL changes. SQL migrations are embedded in the binary via `//go:embed` and run at startup. River's own schema is also managed through Goose via a Go-based migration. |
| **[Gin](https://github.com/gin-gonic/gin)** (v1.12) | HTTP framework built on top of Go's `net/http`, providing routing, middleware, and request binding with minimal overhead. |
| **[pgx](https://github.com/jackc/pgx)** (v5.9, transitive) | PostgreSQL driver used under `database/sql`. |
| **[go-redis](https://github.com/redis/go-redis)** (v9.19) | Redis client for caching and rate limiting. |
| **[Prometheus client](https://github.com/prometheus/client_golang)** (v1.23) | Exposes application metrics at `/metric` (request counters, Go runtime stats, process stats). |
| **[Viper](https://github.com/spf13/viper)** (v1.21) | Configuration management, binding environment variables to the `Config` struct. |

## Application Startup Flow

1. Load configuration from environment variables via Viper.
2. Initialize the `AppV1` struct, which wires together the store (PostgreSQL), cache (Redis), job client (River), and a structured JSON logger (slog backend).
3. Run Goose migrations (River schema at version 1, then SQL DDL + seed data at versions 2-5).
4. Start the River job client (begins polling for periodic emission-refresh jobs).
5. Register all HTTP routes under `/v1` with CORS middleware.
6. Start the Gin HTTP server on the configured port.

On shutdown, the application stops the River client, closes the database connection, and closes the Redis connection (in `AppV1.Shutdown`).

## Database Schema Overview

The schema follows a strict hierarchy: **Organization -> Site -> Asset**. Composite foreign keys enforce that a child can never reference a parent belonging to a different org or site.

| Layer | Tables | Notes |
|---|---|---|
| Core hierarchy | `organizations`, `sites`, `assets`, `asset_types` | Soft-deleted via `deleted_at`. Composite unique constraints enable hierarchical FK chains. |
| People and devices | `users`, `recorder_devices`, `user_recorder_devices` | Users are global (not org-scoped). Device assignments use `TSTZRANGE` with an exclusion constraint to prevent overlapping ownership. |
| Readings | `asset_emission_readings`, `ingested_batches` | Append-only, partitioned monthly by `recorded_at`. Highest-volume table. `ingested_batches` acts as a unique-constraint guard for batch-level idempotency. |
| Policies | `emission_policies` | Time-windowed emission limits per entity and period. Exclusion constraint prevents overlapping policies for the same entity/period. |
| Rollups | `emission_rollup_hourly`, `_daily`, `_monthly` | Pre-computed aggregates. Hourly is partitioned; daily and monthly are not (row counts are small). |
| Violations | `emission_violations` | Machine-written breach records. Upserted by `refresh_emissions()` so re-runs are idempotent. |

The `refresh_emissions(window)` stored procedure cascades rollups (asset-hourly from raw readings, then site-hourly, org-hourly, daily, monthly) and scans for policy violations in a single call.

## Seed Data

Migrations 003-005 seed the database with demo data:

- **003**: 10 standard asset types (compressor, tank, valve, flare, pump, generator, etc.)
- **004**: 3 global field recorder devices
- **005**: A complete sample organization (Acme Energy Corp) with 3 sites, 9 assets, 3 org-scoped sensors, 3 emission policies, ~60 days of generated readings with realistic variation patterns, and pre-computed rollups and violations via `refresh_emissions()`

The seed migration dynamically creates any missing monthly partitions for the date range it covers.

## Containerization

The application uses a multi-stage Docker build:

1. **Builder stage** (`golang:1.25-alpine`) -- downloads dependencies, compiles a statically linked binary (`CGO_ENABLED=0`).
2. **Runtime stage** (`alpine:latest`) -- copies only the binary, resulting in a minimal image.

The frontend uses a three-stage Docker build:

1. **Deps stage** (`node:22-alpine`) -- installs npm dependencies.
2. **Builder stage** -- copies deps, builds the Next.js app in standalone mode. `NEXT_PUBLIC_API_BASE` is injected as a build arg.
3. **Runner stage** (`node:22-alpine`) -- runs the standalone Node.js server as a non-root user.

`docker-compose.yml` orchestrates four services:

| Service | Image | Notes |
|---|---|---|
| `backend` (eiae) | Built from `./backend` | Depends on healthy Postgres and Redis. |
| `frontend` (emissions_dashboard) | Built from `./frontend/monitoring-dashboard` | Depends on healthy backend. `NEXT_PUBLIC_API_BASE` baked at build time. |
| `emissions_postgres` | `postgres:16-alpine` | Persistent volume, health-checked with `pg_isready`. |
| `emissions_redis` | `redis:7-alpine` | Ephemeral (append-only disabled), health-checked with `redis-cli ping`. |

An optional `pgadmin4` service is available under the `tools` profile for database inspection.

## Suggested Production Deployment Topology

### Infrastructure

Cloud-native deployment (AWS EKS, GKE, or similar) with regional coverage:

- **PostgreSQL Cluster** -- 1 primary, 2+ replicas with automatic failover (consider CloudNativePG for Kubernetes):
  - Replicas serve read requests (rollup trends, violation lists) to reduce load on the primary.
  - Automatic failover promotes a replica when the primary fails.
- **Redis Cluster** -- 1 primary, 1 replica for failover. Redis is not a hard dependency; the application degrades gracefully without it.

### CI/CD

- **CI pipeline** -- runs tests, lints, and builds the container image on every push.
- **CD pipeline** -- progressively promotes deployments: staging -> production canaries -> full production rollout (e.g. Argo CD with progressive delivery).

### Observability and Alerting

- **Prometheus** -- scrapes the `/metric` endpoint for application and runtime metrics.
- **Grafana** -- dashboards and configurable alerts for platform failures, latency spikes, and violation counts.
- **OpenTelemetry** (optional) -- distributed tracing to visualize request lifecycles across services.

### Auth Backend

- Auth backend responsible for:
  - JWT token issuance and revocation.
  - User identity management (separate from the `users` table, which represents field engineers/contractors).

### API Gateway

- Single entry point to the application, handling:
  - Rate limiting (backed by Redis counters).
  - JWT validation before routing requests to the backend.
  - TLS termination.

### TLS

- Encrypted connections between all components: application to PostgreSQL, application to Redis, recording devces to application, and gateway to application.

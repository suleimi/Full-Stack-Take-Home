# Getting Started

This guide walks you through running the full Emissions Ingestion and Analysis Engine (EIAE) locally.

## Prerequisites

- **Docker** and **Docker Compose** (v2+)
- Ports available: `3000` (backend), `5123` (frontend), `5432` (Postgres), `6379` (Redis)
- ~2 GB of free disk space for Docker images

No Go, Node.js, or PostgreSQL installation is required -- everything runs in containers.

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/HW-Emissions/Full-Stack-Take-Home.git

git checkout sahmed-submission

cd Full-Stack-Take-Home

# 2. Create the environment file
#    Review and adjust values if needed (ports, passwords, etc.)
cp .env.example .env

# 3. Start all services
docker compose up --build
```

This builds and starts four services:

| Service | URL | Description |
| --- | --- | --- |
| Backend (eiae) | `http://localhost:3000` | Go API server |
| Frontend | `http://localhost:5123` | Next.js monitoring dashboard |
| PostgreSQL | `localhost:5432` | Database (persistent volume) |
| Redis | `localhost:6379` | Cache (ephemeral) |

The first build takes a few minutes. On subsequent runs, Docker layer caching makes it much faster.

## What Happens on Startup

1. **PostgreSQL** starts and becomes healthy.
2. **Redis** starts and becomes healthy.
3. **Backend** starts, runs database migrations automatically:
   - Creates the River job queue schema (migration 1)
   - Creates all tables, indexes, stored procedures, and views (migration 2)
   - Seeds 10 standard asset types (migration 3)
   - Seeds 3 global field recorder devices (migration 4)
   - Seeds a sample organization (Acme Energy Corp) with 3 sites, 9 assets, ~60 days of emission readings, 3 policies, and pre-computed rollups and violations (migration 5)
4. **Backend** starts the River background worker, which immediately runs `refresh_emissions()` to ensure rollups and violation detection are current.
5. **Frontend** starts and waits for the backend health check to pass.

## Using the Dashboard

Open `http://localhost:5123` in your browser. The seeded "Acme Energy Corp" organization is pre-selected.

### Dashboard Tab
- View summary cards: site count, asset count, open violations, total emissions to date
- View the org-level emission trend chart (toggle between hour/day/month granularity)

### Sites & Assets Tab
- Browse sites and their emission totals
- Click a site to see its assets and their individual emission totals
- Click an asset to see its emission detail chart
- Create new sites and assets using the inline forms

### Violations Tab
- View open policy violations with measured vs. limit values
- Acknowledge violations to clear them from the open list

### Policies Tab
- View active emission policies at org, site, or asset level
- Create new policies with configurable limits and periods
- Retire policies that are no longer needed

### Emission Simulator
- Always visible at the bottom of the page
- Submit manual emission readings for any site/asset/device combination
- Click "Refresh Rollups" to trigger a rollup recomputation and see updated charts and violations

## Configuration

All configuration is via environment variables in the `.env` file:

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `3000` | Backend API port |
| `FRONTEND_PORT` | `5123` | Frontend dashboard port |
| `POSTGRES_HOST` | `postgres` | PostgreSQL hostname (Docker service name) |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_DB` | `emissions` | Database name |
| `POSTGRES_USER` | `emissions` | Database user |
| `POSTGRES_PASSWORD` | `emissions` | Database password |
| `REDIS_HOST` | `redis` | Redis hostname (Docker service name) |
| `REDIS_PORT` | `6379` | Redis port |
| `PGADMIN_EMAIL` | `admin@local.dev` | pgAdmin login email (optional) |
| `PGADMIN_PASSWORD` | `admin` | pgAdmin login password (optional) |
| `PGADMIN_PORT` | `5050` | pgAdmin port (optional) |

## Optional: Database Admin UI

To inspect the database with pgAdmin:

```bash
docker compose --profile tools up -d
```

Open `http://localhost:5050`, log in with the pgAdmin credentials from `.env`, and add a server connection:
- Host: `postgres`
- Port: `5432`
- Database / User / Password: as configured in `.env`

## Local Development (without Docker)

### Backend

Requires Go 1.25+ and a running PostgreSQL instance.

```bash
cd backend

# Set environment variables (point to your local Postgres/Redis)
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_DB=emissions
export POSTGRES_USER=emissions
export POSTGRES_PASSWORD=emissions
export REDIS_HOST=localhost
export REDIS_PORT=6379

go run .
```

The backend runs on port `8080` by default (override with `PORT`).

### Frontend

Requires Node.js 22+.

```bash
cd frontend/monitoring-dashboard
npm install
npm run dev
```

Opens at `http://localhost:3000`. Set `NEXT_PUBLIC_API_BASE` to point to your backend (defaults to `http://localhost:8080/v1`).

## Stopping and Cleaning Up

```bash
# Stop all services
docker compose down

# Stop and remove the database volume (full reset)
docker compose down -v
```

## Project Structure

```
Full-Stack-Take-Home/
├── .env                    # Environment variables (create from .env.example)
├── docker-compose.yml      # Orchestrates all services
├── GETTING_STARTED.md      # This file
├── README.md               # Challenge brief
│
├── backend/                # Go API server
│   ├── README.md           # Backend docs, conventions, full endpoint reference
│   ├── ARCHITECTURE.md     # System design decisions and production topology
│   ├── DATABASE.md         # Schema diagrams, rollup engine, constraint docs
│   ├── main.go             # Entrypoint
│   ├── Dockerfile          # Multi-stage Go build
│   ├── internal/           # Application code (handlers, store, jobs, cache, etc.)
│   └── migration/          # Goose migrations (embedded in binary)
│
└── frontend/
    └── monitoring-dashboard/   # Next.js dashboard
        ├── README.md           # Frontend docs
        ├── Dockerfile          # Three-stage Next.js build
        ├── app/                # Next.js App Router pages
        ├── components/         # React components
        └── lib/                # API client
```

## Further Reading

- [Backend README](backend/README.md) -- conventions, full API endpoint reference, migration guide
- [Architecture](backend/ARCHITECTURE.md) -- design decisions, library choices, production topology
- [Database Schema](backend/DATABASE.md) -- ER diagram, rollup engine, partitioning strategy
- [Frontend README](frontend/monitoring-dashboard/README.md) -- component overview, environment variables

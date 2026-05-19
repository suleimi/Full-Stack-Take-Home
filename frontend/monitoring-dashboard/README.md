# Emissions Monitoring Dashboard

Next.js frontend for the EIAE (Emissions Ingestion and Analysis Engine). Provides a multi-tab dashboard for managing organizations, sites, assets, emission policies, and violations.

## Tech Stack

- **Next.js** 16.2 (App Router, standalone output mode)
- **React** 19
- **TypeScript** 5
- **Tailwind CSS** 4
- **Recharts** 2.15 (line charts for emission trends)

## Getting Started

### Via Docker (recommended)

The frontend is included in the root `docker-compose.yml` and starts automatically alongside the backend:

```bash
# From the repo root
docker compose up --build
```

The dashboard is available at `http://localhost:5123` (configurable via `FRONTEND_PORT` in `.env`).

### Local Development

```bash
npm install
npm run dev
```

Opens at `http://localhost:3000`. Expects the backend API at `http://localhost:8080/v1` (override with `NEXT_PUBLIC_API_BASE` env var).

## Project Structure

```
app/
  layout.tsx         # Root layout (HTML shell, global CSS)
  page.tsx           # Main page: org selection, tab routing, data fetching

components/
  OrgSelector.tsx    # Top bar: org dropdown + create org form
  DashboardSummary.tsx  # Summary cards (sites, assets, violations, total emissions) + trend chart
  SitesAssets.tsx     # Site/asset lists with inline emission totals + detail charts
  ViolationsPanel.tsx # Violation list with acknowledge action
  PoliciesPanel.tsx   # Policy list + create/retire actions
  EmissionSimulator.tsx  # Manual reading submission + rollup refresh trigger

lib/
  api.ts             # API client: typed fetch wrappers for all backend endpoints
```

## Features

- **Dashboard tab**: Summary cards (site count, asset count, open violations, total emissions to date) and an org-level emission trend chart with hour/day/month granularity.
- **Sites & Assets tab**: Create and browse sites and assets. Each list item shows its cumulative emissions to date. Clicking a site or asset reveals a detail chart with its own granularity controls.
- **Violations tab**: Lists open policy violations with entity info, measured vs. limit values, overage, and detection time. Supports acknowledging violations.
- **Policies tab**: Lists active emission policies. Supports creating new policies (org/site/asset level, configurable period and limit) and retiring existing ones.
- **Emission Simulator**: Always-visible panel at the bottom. Submit manual readings for any site/asset/device combination and trigger a rollup refresh to see the effects immediately.

## Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `NEXT_PUBLIC_API_BASE` | `http://localhost:8080/v1` | Backend API base URL. Baked at build time in Docker (passed as build arg). |
| `PORT` | `3000` | Server port (overridden to `5123` in docker-compose) |

## Docker Build

The Dockerfile uses a three-stage build:

1. **deps** -- `npm ci` to install dependencies
2. **builder** -- builds the Next.js app with `output: "standalone"`. `NEXT_PUBLIC_API_BASE` is injected as a build arg since `NEXT_PUBLIC_` env vars are inlined at build time.
3. **runner** -- copies the standalone output and runs as a non-root `nextjs` user

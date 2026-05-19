# Database Schema

Full schema is defined in [`migration/sql/002_create_fountation.sql`](migration/sql/002_create_fountation.sql). Migration version 1 is the River job queue schema (Go-based).

## Entity Relationship Diagram

```mermaid
erDiagram
    organizations ||--o{ sites : "has many"
    organizations ||--o{ recorder_devices : "owns (sensor/satellite)"
    organizations ||--o{ emission_policies : "governs"
    organizations ||--o{ emission_violations : "scoped to"

    sites ||--o{ assets : "has many"

    asset_types ||--o{ assets : "classifies"

    assets ||--o{ asset_emission_readings : "produces"

    ingested_batches ||--o{ asset_emission_readings : "guards"
    organizations ||--o{ ingested_batches : "scoped to"

    recorder_devices ||--o{ asset_emission_readings : "sources"
    recorder_devices ||--o{ user_recorder_devices : "assigned via"

    users ||--o{ user_recorder_devices : "holds"

    emission_policies ||--o{ emission_violations : "breached by"

    organizations {
        uuid id PK
        text name
        text description
        text office_location
        text email UK
        timestamptz created_at
        timestamptz updated_at
        timestamptz deleted_at
    }

    sites {
        uuid id PK
        uuid org_id FK
        text name
        text location
        timestamptz created_at
        timestamptz updated_at
        timestamptz deleted_at
    }

    asset_types {
        uuid id PK
        text name UK
        text description
        timestamptz created_at
        timestamptz updated_at
    }

    assets {
        uuid id PK
        uuid org_id FK
        uuid site_id FK
        uuid asset_type_id FK
        text name
        timestamptz created_at
        timestamptz updated_at
        timestamptz deleted_at
    }

    users {
        uuid id PK
        text first_name
        text last_name
        text email UK
        timestamptz created_at
        timestamptz updated_at
        timestamptz deleted_at
    }

    recorder_devices {
        uuid id PK
        uuid org_id FK
        text type "field_device | sensor | satellite"
        jsonb device_metadata
        timestamptz created_at
        timestamptz updated_at
        timestamptz deleted_at
    }

    user_recorder_devices {
        uuid id PK
        uuid user_id FK
        uuid recorder_device_id FK
        tstzrange assigned_during "EXCLUDE overlap"
        timestamptz created_at
    }

    asset_emission_readings {
        uuid id PK
        uuid org_id FK
        uuid site_id FK
        uuid asset_id FK
        uuid recorder_device_id FK
        uuid batch_id FK
        numeric reading
        text unit "default kg_co2e"
        timestamptz recorded_at "partition key"
        timestamptz created_at
    }

    ingested_batches {
        uuid batch_id PK
        uuid org_id FK
        uuid site_id
        timestamptz created_at
    }

    emission_policies {
        uuid id PK
        uuid org_id FK
        text entity_type "org | site | asset"
        uuid entity_id
        numeric emission_limit
        text unit "default kg_co2e"
        text period "hour | day | month | year"
        timestamptz effective_from
        timestamptz effective_to
        timestamptz created_at
    }

    emission_violations {
        uuid id PK
        uuid org_id FK
        text entity_type "org | site | asset"
        uuid entity_id
        uuid policy_id FK
        text period
        numeric limit_value
        text unit
        timestamptz period_start
        numeric measured_value
        numeric overage "generated"
        timestamptz detected_at
        timestamptz last_evaluated_at
        timestamptz acknowledged_at
    }

    emission_rollup_hourly {
        text entity_type PK "org | site | asset"
        uuid entity_id PK
        timestamptz bucket_start PK "partition key"
        numeric total_emission
        integer reading_count
        numeric min_reading
        numeric max_reading
        text unit
        timestamptz computed_at
    }

    emission_rollup_daily {
        text entity_type PK
        uuid entity_id PK
        timestamptz bucket_start PK
        numeric total_emission
        integer reading_count
        numeric min_reading
        numeric max_reading
        text unit
        timestamptz computed_at
    }

    emission_rollup_monthly {
        text entity_type PK
        uuid entity_id PK
        timestamptz bucket_start PK
        numeric total_emission
        integer reading_count
        numeric min_reading
        numeric max_reading
        text unit
        timestamptz computed_at
    }
```

## Key Constraints

| Constraint | Table | Purpose |
| --- | --- | --- |
| Composite FK `(org_id, id)` | `sites` | Assets reference `(org_id, site_id)` as a pair, so a site can never be paired with the wrong org. |
| Composite FK `(org_id, site_id, id)` | `assets` | Readings reference `(org_id, site_id, asset_id)` as a triple, transitively guaranteeing hierarchy integrity. |
| `EXCLUDE USING GIST` | `user_recorder_devices` | No two assignments for the same device may overlap in time. |
| `EXCLUDE USING GIST` | `emission_policies` | No two policies for the same `(entity_type, entity_id, period)` may overlap in time, so the violation scan always finds at most one governing policy. |
| Composite PK `(id, recorded_at)` | `asset_emission_readings` | The partition key (`recorded_at`) must be part of every unique constraint in a partitioned table. |
| PK on `batch_id` | `ingested_batches` | Prevents the TOCTOU race on duplicate-batch detection. Concurrent transactions block on the INSERT rather than both passing a SELECT EXISTS check. |
| Ownership check | `recorder_devices` | `field_device` must have `org_id IS NULL`; `sensor` and `satellite` must have `org_id IS NOT NULL`. |

## Partitioning

| Table | Partition scheme | Reason |
| --- | --- | --- |
| `asset_emission_readings` | Monthly by `recorded_at` | Highest-volume table. Time-windowed queries prune to recent partitions; old data can be dropped instantly. |
| `emission_rollup_hourly` | Monthly by `bucket_start` | ~9.6M rows/year at ~1,100 entities. Partitioning keeps scans fast for recent data. |
| `emission_rollup_daily` | Not partitioned | ~400k rows/year -- small enough to remain a single table. |
| `emission_rollup_monthly` | Not partitioned | ~13k rows/year -- trivially small. |

The foundation migration (002) pre-creates partitions for 2026-05 through 2026-07. The seed data migration (005) dynamically creates additional partitions as needed to cover its 60-day lookback window. In production, use `pg_partman` to create partitions ahead of time automatically.

## Emission Rollup Engine

The `refresh_emissions(window)` stored procedure recomputes rollup aggregates and detects policy violations. It is called by River background jobs on two schedules: every 1 minute with a 3-hour window and every hour with a 48-hour window.

The procedure is **stateless and idempotent**: every write is an upsert, so re-running or overlapping executions are harmless.

### Rollup Cascade (Stage 1)

Readings are scanned exactly once. Every other bucket is derived by summing the level below.

```mermaid
flowchart TD
    subgraph "Raw Data"
        R[("asset_emission_readings<br/>(append-only, partitioned)")]
    end

    subgraph "Stage 1 -- Rollup Cascade"
        direction TB
        AH["Step 1: Asset-Hourly<br/>GROUP BY asset_id, hour<br/>SUM(reading)"]
        SH["Step 2: Site-Hourly<br/>GROUP BY site_id, hour<br/>SUM(asset-hourly)"]
        OH["Step 3: Org-Hourly<br/>GROUP BY org_id, hour<br/>SUM(site-hourly)"]

        AD["Step 4: Daily (all levels)<br/>GROUP BY entity, day<br/>SUM(hourly)"]
        AM["Step 5: Monthly (all levels)<br/>GROUP BY entity, month<br/>SUM(daily)"]
    end

    R -->|"recorded_at >= now() - window"| AH
    AH -->|"JOIN assets"| SH
    SH -->|"JOIN sites"| OH

    AH & SH & OH -->|"date_trunc(day)"| AD
    AD -->|"date_trunc(month)"| AM

    AH -->|"UPSERT"| HT[("emission_rollup_hourly")]
    SH -->|"UPSERT"| HT
    OH -->|"UPSERT"| HT
    AD -->|"UPSERT"| DT[("emission_rollup_daily")]
    AM -->|"UPSERT"| MT[("emission_rollup_monthly")]

    style R fill:#e8f4fd,stroke:#1a73e8
    style HT fill:#fce8e6,stroke:#d93025
    style DT fill:#fce8e6,stroke:#d93025
    style MT fill:#fce8e6,stroke:#d93025
```

**Why the cascade works:**

- A site total equals the sum of its assets; an org total equals the sum of its sites. A day equals 24 hours; a month equals its days.
- There is one base computation (asset-hourly from raw readings). Everything else sums the level below.
- The `window` parameter (default 48 hours) controls how far back to recompute, accommodating late-arriving data from field engineers who sync with up to a 24-hour delay.

### Policy Violation Scan (Stage 2)

After rollups are current, the procedure checks each rollup bucket against its governing policy.

```mermaid
flowchart TD
    subgraph "Inputs"
        HT[("emission_rollup_hourly")]
        DT[("emission_rollup_daily")]
        MT[("emission_rollup_monthly")]
        POL[("emission_policies")]
    end

    subgraph "Stage 2 -- Violation Scan"
        direction TB
        JH["Hourly: JOIN policies<br/>WHERE period = 'hour'<br/>AND effective_from <= bucket_start<br/>AND (effective_to IS NULL OR > bucket_start)"]
        JD["Daily: JOIN policies<br/>WHERE period = 'day'<br/>AND same effective window"]
        JM["Monthly: JOIN policies<br/>WHERE period = 'month'<br/>AND same effective window"]

        CHK{"total_emission<br/>> emission_limit?"}
    end

    HT --> JH
    DT --> JD
    MT --> JM
    POL --> JH & JD & JM

    JH & JD & JM --> CHK

    CHK -->|"Yes"| UPS["UPSERT into emission_violations<br/>- First detection: INSERT (detected_at frozen)<br/>- Later runs: UPDATE measured_value,<br/>  last_evaluated_at only"]
    CHK -->|"No"| SKIP["No action"]

    UPS --> VT[("emission_violations")]

    style VT fill:#fff3e0,stroke:#e65100
    style CHK fill:#e8f5e9,stroke:#1b5e20
```

**Policy matching rules:**

1. **Entity match** -- the policy's `(entity_type, entity_id)` must equal the rollup row's entity.
2. **Grain match** -- the policy's `period` (hour/day/month) must match the rollup table's grain.
3. **Time match** -- the policy must have been in effect at the bucket's start time: `effective_from <= bucket_start` and `effective_to IS NULL OR effective_to > bucket_start`.

The `bucket_start` rule means a mid-period policy change takes effect at the **next** period boundary, not mid-period.

**Violation upsert behavior:**

- The conflict target is `(entity_type, entity_id, policy_id, period_start)` -- one violation per breached bucket.
- On first detection: `detected_at` and the policy snapshot (`limit_value`, `unit`, `period`) are frozen.
- On subsequent runs: only `measured_value` and `last_evaluated_at` are updated. Because readings are append-only, a breach is permanent -- totals only grow.
- `acknowledged_at` is set by a human operator via the dashboard; the scan never touches it.

**Note:** `period = 'year'` policies are not evaluated. There is no yearly rollup table. To support yearly policies, either add `emission_rollup_yearly` with a sixth step or sum the 12 monthly buckets at query time.

## Views

| View | Source | Purpose |
| --- | --- | --- |
| `emission_total_to_date` | `emission_rollup_monthly` | Lifetime total emission per entity. Uses the monthly table because it is the smallest (~13k rows/year) and the current month's bucket already contains everything rolled up so far. |

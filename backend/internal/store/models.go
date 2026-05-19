package store

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Enum types -- these mirror the CHECK constraints in the database schema.
// Using typed constants instead of bare strings gives compile-time safety
// and makes valid values discoverable via autocomplete.
// ---------------------------------------------------------------------------

// EntityType identifies the level in the org > site > asset hierarchy.
type EntityType string

const (
	EntityTypeOrg   EntityType = "org"
	EntityTypeSite  EntityType = "site"
	EntityTypeAsset EntityType = "asset"
)

// DeviceType classifies a recorder device.
type DeviceType string

const (
	DeviceTypeFieldDevice DeviceType = "field_device"
	DeviceTypeSensor      DeviceType = "sensor"
	DeviceTypeSatellite   DeviceType = "satellite"
)

// Period represents the time grain for a policy or rollup.
type Period string

const (
	PeriodHour  Period = "hour"
	PeriodDay   Period = "day"
	PeriodMonth Period = "month"
	PeriodYear  Period = "year"
)

// EmissionUnit is the measurement unit for emission readings and policies.
type EmissionUnit string

const (
	UnitKgCO2e EmissionUnit = "kg_co2e"
)

// Grain selects which rollup table to query.
type Grain string

const (
	GrainHour  Grain = "hour"
	GrainDay   Grain = "day"
	GrainMonth Grain = "month"
)

// ---------------------------------------------------------------------------
// Core hierarchy
// ---------------------------------------------------------------------------

// Organization is the top-level tenant. All sites, assets, policies, and
// violations are scoped to an organization.
type Organization struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Description    *string    `json:"description"`
	OfficeLocation *string    `json:"office_location"`
	Email          string     `json:"email"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// Site is a physical location belonging to an organization (e.g. a plant, well pad).
type Site struct {
	ID        uuid.UUID  `json:"id"`
	OrgID     uuid.UUID  `json:"org_id"`
	Name      string     `json:"name"`
	Location  *string    `json:"location"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// AssetType is a static lookup for classifying assets (e.g. "compressor", "tank").
type AssetType struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Asset is a piece of equipment at a site that produces emission readings.
type Asset struct {
	ID          uuid.UUID  `json:"id"`
	OrgID       uuid.UUID  `json:"org_id"`
	SiteID      uuid.UUID  `json:"site_id"`
	AssetTypeID *uuid.UUID `json:"asset_type_id"`
	Name        string     `json:"name"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// OrgSummary contains high-level dashboard stats for an organization:
// total sites, total assets, and the timestamp of the latest rollup.
type OrgSummary struct {
	OrgID      uuid.UUID  `json:"org_id"`
	SiteCount  int        `json:"site_count"`
	AssetCount int        `json:"asset_count"`
	LastUpdate *time.Time `json:"last_update"`
}

// ---------------------------------------------------------------------------
// People & devices
// ---------------------------------------------------------------------------

// User represents a contractor engineer. Users are global (not org-scoped);
// their association with an org is derived through the readings they produce.
type User struct {
	ID        uuid.UUID  `json:"id"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Email     string     `json:"email"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// RecorderDevice is the source of every reading. Three types exist:
//   - field_device: global, carried by engineers (org_id is NULL)
//   - sensor: org-scoped automated source
//   - satellite: org-scoped automated source
type RecorderDevice struct {
	ID             uuid.UUID  `json:"id"`
	OrgID          *uuid.UUID `json:"org_id"`
	Type           DeviceType `json:"type"`
	DeviceMetadata *string    `json:"device_metadata"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// UserRecorderDevice records which engineer held which field device, and when.
// The assigned range is [AssignedFrom, AssignedTo) -- a nil AssignedTo means
// the assignment is still open.
type UserRecorderDevice struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RecorderDeviceID uuid.UUID  `json:"recorder_device_id"`
	AssignedFrom     time.Time  `json:"assigned_from"`
	AssignedTo       *time.Time `json:"assigned_to"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Readings
// ---------------------------------------------------------------------------

// AssetEmissionReading is a single emission measurement from a recorder device.
// Readings are append-only; they are never updated or deleted.
type AssetEmissionReading struct {
	ID               uuid.UUID    `json:"id"`
	OrgID            uuid.UUID    `json:"org_id"`
	SiteID           uuid.UUID    `json:"site_id"`
	AssetID          uuid.UUID    `json:"asset_id"`
	RecorderDeviceID uuid.UUID    `json:"recorder_device_id"`
	BatchID          uuid.UUID    `json:"batch_id"`
	Reading          float64      `json:"reading"`
	Unit             EmissionUnit `json:"unit"`
	RecordedAt       time.Time    `json:"recorded_at"`
	CreatedAt        time.Time    `json:"created_at"`
}

// ReadingInput is the client-supplied payload for a single reading in a batch.
type ReadingInput struct {
	AssetID          uuid.UUID `json:"asset_id"`
	RecorderDeviceID uuid.UUID `json:"recorder_device_id"`
	Reading          float64   `json:"reading"`
	Unit             string    `json:"unit"`
	RecordedAt       time.Time `json:"recorded_at"`
}

// ---------------------------------------------------------------------------
// Policies
// ---------------------------------------------------------------------------

// EmissionPolicy is a time-windowed rule: "this entity may not emit more than
// EmissionLimit per Period". Policies are never mutated -- they are retired
// (EffectiveTo is set) and replaced.
type EmissionPolicy struct {
	ID            uuid.UUID    `json:"id"`
	OrgID         uuid.UUID    `json:"org_id"`
	EntityType    EntityType   `json:"entity_type"`
	EntityID      uuid.UUID    `json:"entity_id"`
	EmissionLimit float64      `json:"emission_limit"`
	Unit          EmissionUnit `json:"unit"`
	Period        Period       `json:"period"`
	EffectiveFrom time.Time    `json:"effective_from"`
	EffectiveTo   *time.Time   `json:"effective_to"`
	CreatedAt     time.Time    `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Rollups
// ---------------------------------------------------------------------------

// EmissionRollup is a precomputed aggregate: one row per entity per time bucket.
// The time grain is implied by which table the row lives in (hourly, daily, monthly).
type EmissionRollup struct {
	EntityType    EntityType   `json:"entity_type"`
	EntityID      uuid.UUID    `json:"entity_id"`
	BucketStart   time.Time    `json:"bucket_start"`
	TotalEmission float64      `json:"total_emission"`
	ReadingCount  int          `json:"reading_count"`
	MinReading    float64      `json:"min_reading"`
	MaxReading    float64      `json:"max_reading"`
	Unit          EmissionUnit `json:"unit"`
	ComputedAt    time.Time    `json:"computed_at"`
}

// EmissionTotalToDate is the lifetime cumulative emission for an entity,
// computed from the emission_total_to_date view over monthly rollups.
type EmissionTotalToDate struct {
	EntityType   EntityType `json:"entity_type"`
	EntityID     uuid.UUID  `json:"entity_id"`
	TotalToDate  float64    `json:"total_to_date"`
	ReadingCount int        `json:"reading_count"`
	AsOf         time.Time  `json:"as_of"`
}

// ---------------------------------------------------------------------------
// Violations
// ---------------------------------------------------------------------------

// EmissionViolation is a machine-written observation: a rollup bucket exceeded
// its governing policy. One row per (entity, policy, period bucket). The
// policy snapshot (LimitValue, Unit, Period) is frozen at first detection.
type EmissionViolation struct {
	ID              uuid.UUID    `json:"id"`
	OrgID           uuid.UUID    `json:"org_id"`
	EntityType      EntityType   `json:"entity_type"`
	EntityID        uuid.UUID    `json:"entity_id"`
	PolicyID        uuid.UUID    `json:"policy_id"`
	Period          Period       `json:"period"`
	LimitValue      float64      `json:"limit_value"`
	Unit            EmissionUnit `json:"unit"`
	PeriodStart     time.Time    `json:"period_start"`
	MeasuredValue   float64      `json:"measured_value"`
	Overage         float64      `json:"overage"`
	DetectedAt      time.Time    `json:"detected_at"`
	LastEvaluatedAt time.Time    `json:"last_evaluated_at"`
	AcknowledgedAt  *time.Time   `json:"acknowledged_at"`
}

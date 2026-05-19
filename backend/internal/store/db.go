package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type IStore interface {
	DB() *sql.DB

	// Organizations
	CreateOrganization(ctx context.Context, name, email string, description, officeLocation *string) (*Organization, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error)
	GetOrganizationByEmail(ctx context.Context, email string) (*Organization, error)
	ListOrganizations(ctx context.Context, limit, offset int) ([]Organization, error)
	UpdateOrganization(ctx context.Context, id uuid.UUID, name, email string, description, officeLocation *string) (*Organization, error)
	DeleteOrganization(ctx context.Context, id uuid.UUID) error
	CountOrganizations(ctx context.Context) (int, error)
	GetOrganizationSummary(ctx context.Context, orgID uuid.UUID) (*OrgSummary, error)
	GetOrganizationEmissionTrend(ctx context.Context, orgID uuid.UUID, grain Grain, from, to time.Time) ([]EmissionRollup, error)

	// Sites
	CreateSite(ctx context.Context, orgID uuid.UUID, name string, location *string) (*Site, error)
	GetSite(ctx context.Context, orgID, siteID uuid.UUID) (*Site, error)
	ListSitesByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]Site, error)
	UpdateSite(ctx context.Context, orgID, siteID uuid.UUID, name string, location *string) (*Site, error)
	DeleteSite(ctx context.Context, orgID, siteID uuid.UUID) error
	CountSitesByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
	GetSiteEmissionTrend(ctx context.Context, siteID uuid.UUID, grain Grain, from, to time.Time) ([]EmissionRollup, error)

	// Assets
	CreateAsset(ctx context.Context, orgID, siteID uuid.UUID, name string, assetTypeID *uuid.UUID) (*Asset, error)
	GetAsset(ctx context.Context, orgID, siteID, assetID uuid.UUID) (*Asset, error)
	ListAssetsBySite(ctx context.Context, orgID, siteID uuid.UUID, limit, offset int) ([]Asset, error)
	ListAssetsByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]Asset, error)
	UpdateAsset(ctx context.Context, orgID, siteID, assetID uuid.UUID, name string, assetTypeID *uuid.UUID) (*Asset, error)
	DeleteAsset(ctx context.Context, orgID, siteID, assetID uuid.UUID) error
	CountAssetsBySite(ctx context.Context, orgID, siteID uuid.UUID) (int, error)
	CountAssetsByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
	GetAssetEmissionTrend(ctx context.Context, assetID uuid.UUID, grain Grain, from, to time.Time) ([]EmissionRollup, error)

	// Asset Types
	CreateAssetType(ctx context.Context, name string, description *string) (*AssetType, error)
	GetAssetType(ctx context.Context, id uuid.UUID) (*AssetType, error)
	ListAssetTypes(ctx context.Context) ([]AssetType, error)

	// Users
	CreateUser(ctx context.Context, firstName, lastName, email string) (*User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	ListUsers(ctx context.Context, limit, offset int) ([]User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, firstName, lastName, email string) (*User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	CountUsers(ctx context.Context) (int, error)

	// Recorder Devices
	CreateRecorderDevice(ctx context.Context, deviceType DeviceType, orgID *uuid.UUID, deviceMetadata *string) (*RecorderDevice, error)
	GetRecorderDevice(ctx context.Context, id uuid.UUID) (*RecorderDevice, error)
	ListRecorderDevicesByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]RecorderDevice, error)
	ListFieldDevices(ctx context.Context, limit, offset int) ([]RecorderDevice, error)
	DeleteRecorderDevice(ctx context.Context, id uuid.UUID) error
	AssignDeviceToUser(ctx context.Context, userID, deviceID uuid.UUID, from time.Time) (*UserRecorderDevice, error)
	RevokeDeviceFromUser(ctx context.Context, userID, deviceID uuid.UUID, revokedAt time.Time) error
	ListDeviceAssignmentsByUser(ctx context.Context, userID uuid.UUID) ([]UserRecorderDevice, error)

	// Readings
	InsertReadings(ctx context.Context, orgID, siteID, batchID uuid.UUID, inputs []ReadingInput) error
	InsertReadingsTx(ctx context.Context, tx *sql.Tx, orgID, siteID, batchID uuid.UUID, inputs []ReadingInput) error
	ListReadingsByAsset(ctx context.Context, orgID, siteID, assetID uuid.UUID, from, to time.Time, limit, offset int) ([]AssetEmissionReading, error)
	ListReadingsBySite(ctx context.Context, orgID, siteID uuid.UUID, from, to time.Time, limit, offset int) ([]AssetEmissionReading, error)
	ListReadingsByDevice(ctx context.Context, deviceID uuid.UUID, from, to time.Time, limit, offset int) ([]AssetEmissionReading, error)
	CountReadingsByAsset(ctx context.Context, orgID, siteID, assetID uuid.UUID, from, to time.Time) (int, error)

	// Policies
	CreateEmissionPolicy(ctx context.Context, orgID, entityID uuid.UUID, entityType EntityType, period Period, emissionLimit float64, unit EmissionUnit, effectiveFrom time.Time) (*EmissionPolicy, error)
	GetEmissionPolicy(ctx context.Context, id uuid.UUID) (*EmissionPolicy, error)
	ListPoliciesByOrg(ctx context.Context, orgID uuid.UUID, activeOnly bool, limit, offset int) ([]EmissionPolicy, error)
	ListPoliciesByEntity(ctx context.Context, entityType EntityType, entityID uuid.UUID, activeOnly bool) ([]EmissionPolicy, error)
	RetireEmissionPolicy(ctx context.Context, id uuid.UUID, effectiveTo time.Time) error
	CountPoliciesByOrg(ctx context.Context, orgID uuid.UUID, activeOnly bool) (int, error)

	// Rollups
	GetEmissionTotalToDate(ctx context.Context, entityType EntityType, entityID uuid.UUID) (*EmissionTotalToDate, error)
	ListRollupsByOrg(ctx context.Context, orgID uuid.UUID, entityType EntityType, grain Grain, from, to time.Time) ([]EmissionRollup, error)
	RefreshEmissions(ctx context.Context, window string) error

	// Violations
	ListOpenViolationsByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]EmissionViolation, error)
	ListViolationsByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]EmissionViolation, error)
	ListViolationsByEntity(ctx context.Context, entityType EntityType, entityID uuid.UUID, limit, offset int) ([]EmissionViolation, error)
	GetViolation(ctx context.Context, id uuid.UUID) (*EmissionViolation, error)
	AcknowledgeViolation(ctx context.Context, id uuid.UUID) error
	CountOpenViolationsByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
	CountViolationsByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
}

// Store wraps a sql.DB connection and implements the IStore interface.
type Store struct {
	db *sql.DB
}

// DB returns the underlying sql.DB connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// NewStore opens a PostgreSQL connection and returns a ready-to-use Store.
func NewStore(host, port, db, user, password string) (IStore, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, db)

	sqldb, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	err = sqldb.Ping()
	if err != nil {
		return nil, err
	}
	return &Store{sqldb}, err
}

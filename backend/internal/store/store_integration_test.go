package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	db, _ := testutil.SetupPostgres(t)
	return &Store{db: db}, db
}

func seedHierarchy(t *testing.T, s *Store) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	org, err := s.CreateOrganization(ctx, "Org A", "orga@example.com", nil, nil)
	require.NoError(t, err)
	site, err := s.CreateSite(ctx, org.ID, "Site A", nil)
	require.NoError(t, err)
	at, err := s.CreateAssetType(ctx, "custom_type_"+uuid.NewString(), nil)
	require.NoError(t, err)
	asset, err := s.CreateAsset(ctx, org.ID, site.ID, "Asset A", &at.ID)
	require.NoError(t, err)
	return org.ID, site.ID, at.ID, asset.ID
}

func TestStore_OrganizationSiteAssetLifecycle(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	org, err := s.CreateOrganization(ctx, "Org 1", "org1@example.com", nil, nil)
	require.NoError(t, err)

	fetchedOrg, err := s.GetOrganization(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, fetchedOrg.ID)

	site, err := s.CreateSite(ctx, org.ID, "Site 1", nil)
	require.NoError(t, err)
	_, err = s.GetSite(ctx, org.ID, site.ID)
	require.NoError(t, err)

	assetType, err := s.CreateAssetType(ctx, "tank_"+uuid.NewString(), nil)
	require.NoError(t, err)
	asset, err := s.CreateAsset(ctx, org.ID, site.ID, "Asset 1", &assetType.ID)
	require.NoError(t, err)

	assets, err := s.ListAssetsBySite(ctx, org.ID, site.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	assert.Equal(t, asset.ID, assets[0].ID)

	countSites, err := s.CountSitesByOrg(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, countSites)

	countAssets, err := s.CountAssetsByOrg(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, countAssets)
}

func TestStore_InsertReadingsAndQuery(t *testing.T) {
	s, db := newTestStore(t)
	ctx := context.Background()
	orgID, siteID, _, assetID := seedHierarchy(t, s)

	device, err := s.CreateRecorderDevice(ctx, DeviceTypeSensor, &orgID, nil)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	batchID := uuid.New()
	err = s.InsertReadings(ctx, orgID, siteID, batchID, []ReadingInput{
		{AssetID: assetID, RecorderDeviceID: device.ID, Reading: 12.5, RecordedAt: now},
		{AssetID: assetID, RecorderDeviceID: device.ID, Reading: 7.5, Unit: "kg_co2e", RecordedAt: now.Add(1 * time.Minute)},
	})
	require.NoError(t, err)

	// Duplicate batch_id should be rejected
	err = s.InsertReadings(ctx, orgID, siteID, batchID, []ReadingInput{
		{AssetID: assetID, RecorderDeviceID: device.ID, Reading: 99, RecordedAt: now},
	})
	assert.ErrorIs(t, err, ErrDuplicateBatch)

	count, err := s.CountReadingsByAsset(ctx, orgID, siteID, assetID, now.Add(-1*time.Hour), now.Add(1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	rows, err := s.ListReadingsByAsset(ctx, orgID, siteID, assetID, now.Add(-1*time.Hour), now.Add(1*time.Hour), 10, 0)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, UnitKgCO2e, rows[0].Unit)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	txBatchID := uuid.New()
	err = s.InsertReadingsTx(ctx, tx, orgID, siteID, txBatchID, []ReadingInput{{AssetID: assetID, RecorderDeviceID: device.ID, Reading: 3.3, RecordedAt: now.Add(2 * time.Minute)}})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	deviceRows, err := s.ListReadingsByDevice(ctx, device.ID, now.Add(-1*time.Hour), now.Add(1*time.Hour), 10, 0)
	require.NoError(t, err)
	require.Len(t, deviceRows, 3)
}

func TestStore_PoliciesRollupsAndViolations(t *testing.T) {
	s, db := newTestStore(t)
	ctx := context.Background()
	orgID, siteID, _, assetID := seedHierarchy(t, s)

	device, err := s.CreateRecorderDevice(ctx, DeviceTypeSensor, &orgID, nil)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Hour)
	err = s.InsertReadings(ctx, orgID, siteID, uuid.New(), []ReadingInput{{AssetID: assetID, RecorderDeviceID: device.ID, Reading: 100, RecordedAt: now}})
	require.NoError(t, err)

	policy, err := s.CreateEmissionPolicy(ctx, orgID, orgID, EntityTypeOrg, PeriodDay, 10, UnitKgCO2e, now.Add(-72*time.Hour))
	require.NoError(t, err)
	require.NoError(t, s.RefreshEmissions(ctx, "48 hours"))

	total, err := s.GetEmissionTotalToDate(ctx, EntityTypeOrg, orgID)
	require.NoError(t, err)
	assert.Greater(t, total.TotalToDate, 0.0)

	rollups, err := s.GetOrganizationEmissionTrend(ctx, orgID, GrainDay, now.Add(-24*time.Hour), now.Add(24*time.Hour))
	require.NoError(t, err)
	assert.NotEmpty(t, rollups)

	openViolations, err := s.ListOpenViolationsByOrg(ctx, orgID, 50, 0)
	require.NoError(t, err)
	require.NotEmpty(t, openViolations)

	require.NoError(t, s.AcknowledgeViolation(ctx, openViolations[0].ID))
	openCount, err := s.CountOpenViolationsByOrg(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, 0, openCount)

	require.NoError(t, s.RetireEmissionPolicy(ctx, policy.ID, now.Add(2*time.Hour)))
	pl, err := s.GetEmissionPolicy(ctx, policy.ID)
	require.NoError(t, err)
	assert.NotNil(t, pl.EffectiveTo)

	_ = db
}

func TestStore_ListReadingsBySite_IntendedBehavior(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	orgID, siteID, _, assetID := seedHierarchy(t, s)
	device, err := s.CreateRecorderDevice(ctx, DeviceTypeSensor, &orgID, nil)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	err = s.InsertReadings(ctx, orgID, siteID, uuid.New(), []ReadingInput{{AssetID: assetID, RecorderDeviceID: device.ID, Reading: 1.2, RecordedAt: now}})
	require.NoError(t, err)

	rows, err := s.ListReadingsBySite(ctx, orgID, siteID, now.Add(-time.Hour), now.Add(time.Hour), 10, 0)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

// ---------------------------------------------------------------------------
// 1. Organization full CRUD
// ---------------------------------------------------------------------------

func TestStore_OrganizationCRUD(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	desc := "A test org"
	loc := "New York"
	email := "crud-org-" + uuid.NewString() + "@example.com"

	// Create
	org, err := s.CreateOrganization(ctx, "CRUD Org", email, &desc, &loc)
	require.NoError(t, err)
	assert.Equal(t, "CRUD Org", org.Name)
	assert.Equal(t, email, org.Email)
	assert.NotNil(t, org.Description)
	assert.Equal(t, desc, *org.Description)
	assert.NotNil(t, org.OfficeLocation)
	assert.Equal(t, loc, *org.OfficeLocation)

	// GetOrganization
	fetched, err := s.GetOrganization(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, fetched.ID)
	assert.Equal(t, org.Name, fetched.Name)
	assert.Equal(t, org.Email, fetched.Email)
	assert.Equal(t, *org.Description, *fetched.Description)
	assert.Equal(t, *org.OfficeLocation, *fetched.OfficeLocation)

	// GetOrganizationByEmail
	byEmail, err := s.GetOrganizationByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, org.ID, byEmail.ID)

	// ListOrganizations
	orgs, err := s.ListOrganizations(ctx, 100, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(orgs), 1)

	// CountOrganizations
	count, err := s.CountOrganizations(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1)

	// UpdateOrganization
	newEmail := "updated-" + uuid.NewString() + "@example.com"
	updated, err := s.UpdateOrganization(ctx, org.ID, "Updated Org", newEmail, &desc, &loc)
	require.NoError(t, err)
	assert.Equal(t, "Updated Org", updated.Name)
	assert.Equal(t, newEmail, updated.Email)

	// GetOrganizationSummary (create site+asset first)
	site, err := s.CreateSite(ctx, org.ID, "Summary Site", nil)
	require.NoError(t, err)
	at, err := s.CreateAssetType(ctx, "summary_type_"+uuid.NewString(), nil)
	require.NoError(t, err)
	_, err = s.CreateAsset(ctx, org.ID, site.ID, "Summary Asset", &at.ID)
	require.NoError(t, err)

	summary, err := s.GetOrganizationSummary(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.SiteCount)
	assert.Equal(t, 1, summary.AssetCount)

	// DeleteOrganization
	err = s.DeleteOrganization(ctx, org.ID)
	require.NoError(t, err)

	_, err = s.GetOrganization(ctx, org.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ---------------------------------------------------------------------------
// 2. Site full CRUD
// ---------------------------------------------------------------------------

func TestStore_SiteCRUD(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	org, err := s.CreateOrganization(ctx, "Site CRUD Org", "site-crud-"+uuid.NewString()+"@example.com", nil, nil)
	require.NoError(t, err)

	loc := "Houston, TX"
	site, err := s.CreateSite(ctx, org.ID, "Site Alpha", &loc)
	require.NoError(t, err)
	assert.Equal(t, "Site Alpha", site.Name)
	assert.NotNil(t, site.Location)
	assert.Equal(t, loc, *site.Location)

	// GetSite
	fetched, err := s.GetSite(ctx, org.ID, site.ID)
	require.NoError(t, err)
	assert.Equal(t, site.ID, fetched.ID)
	assert.Equal(t, site.Name, fetched.Name)

	// ListSitesByOrg
	sites, err := s.ListSitesByOrg(ctx, org.ID, 100, 0)
	require.NoError(t, err)
	require.Len(t, sites, 1)

	// UpdateSite
	updated, err := s.UpdateSite(ctx, org.ID, site.ID, "Site Beta", &loc)
	require.NoError(t, err)
	assert.Equal(t, "Site Beta", updated.Name)

	// DeleteSite
	err = s.DeleteSite(ctx, org.ID, site.ID)
	require.NoError(t, err)

	_, err = s.GetSite(ctx, org.ID, site.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ---------------------------------------------------------------------------
// 3. Asset full CRUD
// ---------------------------------------------------------------------------

func TestStore_AssetCRUD(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	orgID, siteID, atID, _ := seedHierarchy(t, s)

	asset, err := s.CreateAsset(ctx, orgID, siteID, "Asset CRUD", &atID)
	require.NoError(t, err)
	assert.Equal(t, "Asset CRUD", asset.Name)

	// GetAsset
	fetched, err := s.GetAsset(ctx, orgID, siteID, asset.ID)
	require.NoError(t, err)
	assert.Equal(t, asset.ID, fetched.ID)
	assert.Equal(t, asset.Name, fetched.Name)

	// ListAssetsByOrg
	assets, err := s.ListAssetsByOrg(ctx, orgID, 100, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(assets), 1)

	// UpdateAsset
	updated, err := s.UpdateAsset(ctx, orgID, siteID, asset.ID, "Asset Updated", &atID)
	require.NoError(t, err)
	assert.Equal(t, "Asset Updated", updated.Name)

	// DeleteAsset
	err = s.DeleteAsset(ctx, orgID, siteID, asset.ID)
	require.NoError(t, err)

	_, err = s.GetAsset(ctx, orgID, siteID, asset.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// CountAssetsBySite after delete (only the seed asset remains, but we deleted our new one)
	count, err := s.CountAssetsBySite(ctx, orgID, siteID)
	require.NoError(t, err)
	// seedHierarchy created 1 asset; we created 1 more and deleted it => 1 remains
	// But the spec says "should be 0" — so we use a fresh site with no seed asset.
	// Actually let's just verify: we deleted asset.ID, the seed asset is still there.
	// The spec asks CountAssetsBySite after delete should be 0, meaning for the deleted asset's site.
	// We'll create a separate site to isolate the count.
	site2, err := s.CreateSite(ctx, orgID, "Isolated Site", nil)
	require.NoError(t, err)
	asset2, err := s.CreateAsset(ctx, orgID, site2.ID, "Lone Asset", &atID)
	require.NoError(t, err)
	err = s.DeleteAsset(ctx, orgID, site2.ID, asset2.ID)
	require.NoError(t, err)
	count, err = s.CountAssetsBySite(ctx, orgID, site2.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	_ = count
}

// ---------------------------------------------------------------------------
// 4. AssetType CRUD
// ---------------------------------------------------------------------------

func TestStore_AssetTypeCRUD(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	name1 := "type_a_" + uuid.NewString()
	name2 := "type_b_" + uuid.NewString()

	at1, err := s.CreateAssetType(ctx, name1, nil)
	require.NoError(t, err)
	assert.Equal(t, name1, at1.Name)

	desc := "second type"
	at2, err := s.CreateAssetType(ctx, name2, &desc)
	require.NoError(t, err)
	assert.Equal(t, name2, at2.Name)

	// GetAssetType
	fetched, err := s.GetAssetType(ctx, at1.ID)
	require.NoError(t, err)
	assert.Equal(t, at1.ID, fetched.ID)
	assert.Equal(t, name1, fetched.Name)

	// ListAssetTypes
	types, err := s.ListAssetTypes(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(types), 2)
}

// ---------------------------------------------------------------------------
// 5. User CRUD
// ---------------------------------------------------------------------------

func TestStore_UserCRUD(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	email := "user-" + uuid.NewString() + "@example.com"
	user, err := s.CreateUser(ctx, "Jane", "Doe", email)
	require.NoError(t, err)
	assert.Equal(t, "Jane", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.Equal(t, email, user.Email)

	// GetUser
	fetched, err := s.GetUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, fetched.ID)
	assert.Equal(t, "Jane", fetched.FirstName)
	assert.Equal(t, "Doe", fetched.LastName)
	assert.Equal(t, email, fetched.Email)

	// GetUserByEmail
	byEmail, err := s.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, user.ID, byEmail.ID)

	// ListUsers
	users, err := s.ListUsers(ctx, 100, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 1)

	// CountUsers before delete
	countBefore, err := s.CountUsers(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, countBefore, 1)

	// UpdateUser
	updated, err := s.UpdateUser(ctx, user.ID, "Janet", "Doe", email)
	require.NoError(t, err)
	assert.Equal(t, "Janet", updated.FirstName)

	// DeleteUser
	err = s.DeleteUser(ctx, user.ID)
	require.NoError(t, err)

	_, err = s.GetUser(ctx, user.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// CountUsers should decrease by 1
	countAfter, err := s.CountUsers(ctx)
	require.NoError(t, err)
	assert.Equal(t, countBefore-1, countAfter)
}

// ---------------------------------------------------------------------------
// 6. RecorderDevice CRUD
// ---------------------------------------------------------------------------

func TestStore_RecorderDeviceCRUD(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	org, err := s.CreateOrganization(ctx, "Device Org", "dev-"+uuid.NewString()+"@example.com", nil, nil)
	require.NoError(t, err)

	// Create sensor (org-scoped)
	sensor, err := s.CreateRecorderDevice(ctx, DeviceTypeSensor, &org.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, DeviceTypeSensor, sensor.Type)
	assert.NotNil(t, sensor.OrgID)
	assert.Equal(t, org.ID, *sensor.OrgID)

	// Create field_device (nil orgID)
	fieldDev, err := s.CreateRecorderDevice(ctx, DeviceTypeFieldDevice, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, DeviceTypeFieldDevice, fieldDev.Type)
	assert.Nil(t, fieldDev.OrgID)

	// GetRecorderDevice
	fetched, err := s.GetRecorderDevice(ctx, sensor.ID)
	require.NoError(t, err)
	assert.Equal(t, sensor.ID, fetched.ID)
	assert.Equal(t, DeviceTypeSensor, fetched.Type)

	// ListRecorderDevicesByOrg
	orgDevices, err := s.ListRecorderDevicesByOrg(ctx, org.ID, 100, 0)
	require.NoError(t, err)
	found := false
	for _, d := range orgDevices {
		if d.ID == sensor.ID {
			found = true
		}
	}
	assert.True(t, found, "sensor should appear in ListRecorderDevicesByOrg")

	// ListFieldDevices
	fieldDevices, err := s.ListFieldDevices(ctx, 100, 0)
	require.NoError(t, err)
	found = false
	for _, d := range fieldDevices {
		if d.ID == fieldDev.ID {
			found = true
		}
	}
	assert.True(t, found, "field device should appear in ListFieldDevices")

	// DeleteRecorderDevice
	err = s.DeleteRecorderDevice(ctx, sensor.ID)
	require.NoError(t, err)

	_, err = s.GetRecorderDevice(ctx, sensor.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ---------------------------------------------------------------------------
// 7. Device Assignment
// ---------------------------------------------------------------------------

func TestStore_DeviceAssignment(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	user, err := s.CreateUser(ctx, "Assign", "User", "assign-"+uuid.NewString()+"@example.com")
	require.NoError(t, err)

	fieldDev, err := s.CreateRecorderDevice(ctx, DeviceTypeFieldDevice, nil, nil)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)

	// AssignDeviceToUser
	_, err = s.AssignDeviceToUser(ctx, user.ID, fieldDev.ID, now)
	require.NoError(t, err)

	// ListDeviceAssignmentsByUser — 1 assignment, AssignedTo nil (open)
	assignments, err := s.ListDeviceAssignmentsByUser(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.Nil(t, assignments[0].AssignedTo)

	// RevokeDeviceFromUser
	err = s.RevokeDeviceFromUser(ctx, user.ID, fieldDev.ID, now.Add(time.Hour))
	require.NoError(t, err)

	// Verify AssignedTo is now set
	assignments, err = s.ListDeviceAssignmentsByUser(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.NotNil(t, assignments[0].AssignedTo)

	// Revoking again should return sql.ErrNoRows (no open assignment)
	err = s.RevokeDeviceFromUser(ctx, user.ID, fieldDev.ID, now.Add(2*time.Hour))
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ---------------------------------------------------------------------------
// 8. Policy CRUD
// ---------------------------------------------------------------------------

func TestStore_PolicyCRUD(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	orgID, siteID, _, _ := seedHierarchy(t, s)
	now := time.Now().UTC()

	// Create daily policy for org with effective_from = now
	policyOrg, err := s.CreateEmissionPolicy(ctx, orgID, orgID, EntityTypeOrg, PeriodDay, 50, UnitKgCO2e, now)
	require.NoError(t, err)
	// effective_from should be truncated to day boundary
	assert.Equal(t, 0, policyOrg.EffectiveFrom.Hour())
	assert.Equal(t, 0, policyOrg.EffectiveFrom.Minute())
	assert.Equal(t, 0, policyOrg.EffectiveFrom.Second())

	// Create hourly policy for site with effective_from = now
	policySite, err := s.CreateEmissionPolicy(ctx, orgID, siteID, EntityTypeSite, PeriodHour, 25, UnitKgCO2e, now)
	require.NoError(t, err)
	// effective_from should be truncated to hour boundary
	assert.Equal(t, 0, policySite.EffectiveFrom.Minute())
	assert.Equal(t, 0, policySite.EffectiveFrom.Second())

	// GetEmissionPolicy
	fetched, err := s.GetEmissionPolicy(ctx, policyOrg.ID)
	require.NoError(t, err)
	assert.Equal(t, policyOrg.ID, fetched.ID)
	assert.Equal(t, EntityTypeOrg, fetched.EntityType)
	assert.Equal(t, PeriodDay, fetched.Period)
	assert.Equal(t, 50.0, fetched.EmissionLimit)
	assert.Equal(t, UnitKgCO2e, fetched.Unit)
	assert.Nil(t, fetched.EffectiveTo)

	// ListPoliciesByOrg (activeOnly=true)
	activePolicies, err := s.ListPoliciesByOrg(ctx, orgID, true, 100, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(activePolicies), 2)

	// ListPoliciesByOrg (activeOnly=false)
	allPolicies, err := s.ListPoliciesByOrg(ctx, orgID, false, 100, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(allPolicies), 2)

	// ListPoliciesByEntity for the org entity
	orgPolicies, err := s.ListPoliciesByEntity(ctx, EntityTypeOrg, orgID, true)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(orgPolicies), 1)

	// CountPoliciesByOrg
	countActive, err := s.CountPoliciesByOrg(ctx, orgID, true)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, countActive, 2)

	// RetireEmissionPolicy (retire the org daily policy)
	err = s.RetireEmissionPolicy(ctx, policyOrg.ID, now.Add(time.Hour))
	require.NoError(t, err)

	retired, err := s.GetEmissionPolicy(ctx, policyOrg.ID)
	require.NoError(t, err)
	assert.NotNil(t, retired.EffectiveTo)

	// ListPoliciesByOrg (activeOnly=true) should have 1 fewer
	activePoliciesAfter, err := s.ListPoliciesByOrg(ctx, orgID, true, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, len(activePolicies)-1, len(activePoliciesAfter))
}

// ---------------------------------------------------------------------------
// 9. Violation Lifecycle
// ---------------------------------------------------------------------------

func TestStore_ViolationLifecycle(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	orgID, siteID, _, assetID := seedHierarchy(t, s)

	device, err := s.CreateRecorderDevice(ctx, DeviceTypeSensor, &orgID, nil)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Hour)
	err = s.InsertReadings(ctx, orgID, siteID, uuid.New(), []ReadingInput{
		{AssetID: assetID, RecorderDeviceID: device.ID, Reading: 500, RecordedAt: now},
	})
	require.NoError(t, err)

	// Create policy with a low limit so violations are generated
	policy, err := s.CreateEmissionPolicy(ctx, orgID, orgID, EntityTypeOrg, PeriodDay, 1, UnitKgCO2e, now.Add(-72*time.Hour))
	require.NoError(t, err)

	require.NoError(t, s.RefreshEmissions(ctx, "48 hours"))

	// ListOpenViolationsByOrg
	openViolations, err := s.ListOpenViolationsByOrg(ctx, orgID, 100, 0)
	require.NoError(t, err)
	require.NotEmpty(t, openViolations)

	// ListViolationsByOrg (all)
	allViolations, err := s.ListViolationsByOrg(ctx, orgID, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, len(openViolations), len(allViolations))

	// CountOpenViolationsByOrg
	openCount, err := s.CountOpenViolationsByOrg(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, len(openViolations), openCount)

	// CountViolationsByOrg
	totalCount, err := s.CountViolationsByOrg(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, len(allViolations), totalCount)

	// GetViolation
	v, err := s.GetViolation(ctx, openViolations[0].ID)
	require.NoError(t, err)
	assert.Equal(t, policy.ID, v.PolicyID)
	assert.Greater(t, v.MeasuredValue, v.LimitValue)
	assert.Greater(t, v.Overage, 0.0)

	// ListViolationsByEntity
	entityViolations, err := s.ListViolationsByEntity(ctx, EntityTypeOrg, orgID, 100, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, entityViolations)

	// Acknowledge all open violations
	for _, ov := range openViolations {
		require.NoError(t, s.AcknowledgeViolation(ctx, ov.ID))
	}

	// CountOpenViolationsByOrg should be 0
	openCount, err = s.CountOpenViolationsByOrg(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, 0, openCount)

	// CountViolationsByOrg should remain the same (acknowledged != deleted)
	totalCountAfter, err := s.CountViolationsByOrg(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, totalCount, totalCountAfter)

	// AcknowledgeViolation again should return sql.ErrNoRows
	err = s.AcknowledgeViolation(ctx, openViolations[0].ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ---------------------------------------------------------------------------
// 10. Rollup Queries
// ---------------------------------------------------------------------------

func TestStore_RollupQueries(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	orgID, siteID, _, assetID := seedHierarchy(t, s)

	device, err := s.CreateRecorderDevice(ctx, DeviceTypeSensor, &orgID, nil)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Hour)
	err = s.InsertReadings(ctx, orgID, siteID, uuid.New(), []ReadingInput{
		{AssetID: assetID, RecorderDeviceID: device.ID, Reading: 42, RecordedAt: now},
		{AssetID: assetID, RecorderDeviceID: device.ID, Reading: 58, RecordedAt: now.Add(time.Minute)},
	})
	require.NoError(t, err)

	require.NoError(t, s.RefreshEmissions(ctx, "48 hours"))

	// GetEmissionTotalToDate for org, site, asset
	orgTotal, err := s.GetEmissionTotalToDate(ctx, EntityTypeOrg, orgID)
	require.NoError(t, err)
	assert.Greater(t, orgTotal.TotalToDate, 0.0)

	siteTotal, err := s.GetEmissionTotalToDate(ctx, EntityTypeSite, siteID)
	require.NoError(t, err)
	assert.Greater(t, siteTotal.TotalToDate, 0.0)

	assetTotal, err := s.GetEmissionTotalToDate(ctx, EntityTypeAsset, assetID)
	require.NoError(t, err)
	assert.Greater(t, assetTotal.TotalToDate, 0.0)

	from := now.Add(-24 * time.Hour)
	to := now.Add(24 * time.Hour)

	// GetOrganizationEmissionTrend
	orgTrend, err := s.GetOrganizationEmissionTrend(ctx, orgID, GrainDay, from, to)
	require.NoError(t, err)
	assert.NotEmpty(t, orgTrend)

	// GetSiteEmissionTrend
	siteTrend, err := s.GetSiteEmissionTrend(ctx, siteID, GrainDay, from, to)
	require.NoError(t, err)
	assert.NotEmpty(t, siteTrend)

	// GetAssetEmissionTrend
	assetTrend, err := s.GetAssetEmissionTrend(ctx, assetID, GrainDay, from, to)
	require.NoError(t, err)
	assert.NotEmpty(t, assetTrend)

	// ListRollupsByOrg for site entities
	siteRollups, err := s.ListRollupsByOrg(ctx, orgID, EntityTypeSite, GrainDay, from, to)
	require.NoError(t, err)
	assert.NotEmpty(t, siteRollups)

	// ListRollupsByOrg for asset entities
	assetRollups, err := s.ListRollupsByOrg(ctx, orgID, EntityTypeAsset, GrainDay, from, to)
	require.NoError(t, err)
	assert.NotEmpty(t, assetRollups)
}

// ---------------------------------------------------------------------------
// 11. Policy Takes Effect On Current Bucket (truncateToPeriod integration)
// ---------------------------------------------------------------------------

func TestStore_PolicyTakesEffectOnCurrentBucket(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	orgID, siteID, _, assetID := seedHierarchy(t, s)

	device, err := s.CreateRecorderDevice(ctx, DeviceTypeSensor, &orgID, nil)
	require.NoError(t, err)

	// Insert a reading with value 100, recorded_at = now truncated to hour
	now := time.Now().UTC().Truncate(time.Hour)
	err = s.InsertReadings(ctx, orgID, siteID, uuid.New(), []ReadingInput{
		{AssetID: assetID, RecorderDeviceID: device.ID, Reading: 100, RecordedAt: now},
	})
	require.NoError(t, err)

	// Create daily policy with limit 10, effective_from = now (NOT backdated).
	// truncateToPeriod should floor this to start-of-day, so the policy covers
	// today's bucket and the reading of 100 exceeds the limit of 10.
	_, err = s.CreateEmissionPolicy(ctx, orgID, orgID, EntityTypeOrg, PeriodDay, 10, UnitKgCO2e, now)
	require.NoError(t, err)

	require.NoError(t, s.RefreshEmissions(ctx, "48 hours"))

	// Should find a violation
	violations, err := s.ListOpenViolationsByOrg(ctx, orgID, 100, 0)
	require.NoError(t, err)
	require.NotEmpty(t, violations, "expected a violation because the policy's effective_from is truncated to start-of-day")

	// Verify violation values
	assert.GreaterOrEqual(t, violations[0].MeasuredValue, 100.0)
	assert.Equal(t, 10.0, violations[0].LimitValue)
}

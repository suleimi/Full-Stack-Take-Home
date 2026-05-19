package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/cache"
	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/store"
	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/testutil"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginate_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		expectLimit  int
		expectOffset int
	}{
		{"defaults", "", 50, 0},
		{"valid", "?limit=10&offset=20", 10, 20},
		{"limit clamped", "?limit=300", 50, 0},
		{"negative offset", "?offset=-1", 50, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			req := httptest.NewRequest(http.MethodGet, "/"+tc.query, nil)
			c.Request = req
			limit, offset := paginate(c)
			assert.Equal(t, tc.expectLimit, limit)
			assert.Equal(t, tc.expectOffset, offset)
		})
	}
}

func TestHandlers_EndToEndHappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, dsn := testutil.SetupPostgres(t)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	password, _ := u.User.Password()
	dbName := strings.TrimPrefix(u.Path, "/")
	storeImpl, err := store.NewStore(u.Hostname(), u.Port(), dbName, u.User.Username(), password)
	require.NoError(t, err)
	a := &AppV1{Store: storeImpl}

	r := gin.New()
	v1g := r.Group("/v1")
	v1g.POST("/orgs", a.CreateOrganization())
	v1g.POST("/orgs/:org_id/sites", a.CreateSite())
	v1g.POST("/asset-types", a.CreateAssetType())
	v1g.POST("/orgs/:org_id/sites/:site_id/assets", a.CreateAsset())
	v1g.POST("/devices", a.CreateRecorderDevice())
	v1g.POST("/orgs/:org_id/sites/:site_id/readings", a.IngestReadings())
	v1g.POST("/orgs/:org_id/policies", a.CreatePolicy())
	v1g.POST("/emissions/refresh", a.RefreshEmissions())
	v1g.GET("/orgs/:org_id/violations", a.ListViolations())

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "Org", "email": "org@test.com"})
	siteID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites", map[string]any{"name": "Site"})
	assetTypeID := postAndExtractID(t, r, "/v1/asset-types", map[string]any{"name": "compressor_" + uuid.NewString()})
	assetID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites/"+siteID+"/assets", map[string]any{"name": "Asset", "asset_type_id": assetTypeID})
	deviceID := postAndExtractID(t, r, "/v1/devices", map[string]any{"type": "sensor", "org_id": orgID})

	_ = postJSON(t, r, "/v1/orgs/"+orgID+"/sites/"+siteID+"/readings", map[string]any{
		"readings": []map[string]any{{"asset_id": assetID, "recorder_device_id": deviceID, "reading": 99.9, "recorded_at": time.Now().UTC().Format(time.RFC3339)}},
	}, http.StatusCreated)

	_ = postAndExtractID(t, r, "/v1/orgs/"+orgID+"/policies", map[string]any{
		"entity_type": "org", "entity_id": orgID, "emission_limit": 10, "period": "day", "effective_from": time.Now().UTC().Format(time.RFC3339),
	})

	postJSON(t, r, "/v1/emissions/refresh", map[string]any{"window": "48 hours"}, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+orgID+"/violations?open_only=true", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "data")
}

func postAndExtractID(t *testing.T, r *gin.Engine, path string, body map[string]any) string {
	t.Helper()
	rr := postJSON(t, r, path, body, http.StatusCreated)
	var out map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	id, ok := out["id"].(string)
	require.True(t, ok)
	_, err := uuid.Parse(id)
	require.NoError(t, err)
	return id
}

func postJSON(t *testing.T, r *gin.Engine, path string, body map[string]any, expectedStatus int) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, expectedStatus, rr.Code, rr.Body.String())
	return rr
}

// ---------------------------------------------------------------------------
// Shared helpers for new tests
// ---------------------------------------------------------------------------

func setupTestRouter(t *testing.T) (*gin.Engine, *AppV1) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	_, dsn := testutil.SetupPostgres(t)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	password, _ := u.User.Password()
	dbName := strings.TrimPrefix(u.Path, "/")
	storeImpl, err := store.NewStore(u.Hostname(), u.Port(), dbName, u.User.Username(), password)
	require.NoError(t, err)
	a := &AppV1{Store: storeImpl}

	r := gin.New()
	v1g := r.Group("/v1")

	// orgs
	v1g.GET("/orgs", a.ListOrganizations())
	v1g.POST("/orgs", a.CreateOrganization())
	v1g.GET("/orgs/:org_id", a.GetOrganization())
	v1g.GET("/orgs/:org_id/summary", a.GetOrganizationSummary())

	// sites
	v1g.GET("/orgs/:org_id/sites", a.ListSites())
	v1g.POST("/orgs/:org_id/sites", a.CreateSite())

	// asset types
	v1g.GET("/asset-types", a.ListAssetTypes())
	v1g.POST("/asset-types", a.CreateAssetType())

	// assets
	v1g.GET("/orgs/:org_id/sites/:site_id/assets", a.ListAssets())
	v1g.POST("/orgs/:org_id/sites/:site_id/assets", a.CreateAsset())

	// devices
	v1g.POST("/devices", a.CreateRecorderDevice())
	v1g.GET("/devices/field", a.ListFieldDevices())
	v1g.GET("/orgs/:org_id/devices", a.ListOrgDevices())

	// readings
	v1g.POST("/orgs/:org_id/sites/:site_id/readings", a.IngestReadings())

	// emissions
	v1g.GET("/orgs/:org_id/emissions/trend", a.GetEmissionTrend())
	v1g.GET("/orgs/:org_id/emissions/total", a.GetEmissionTotal())
	v1g.POST("/emissions/refresh", a.RefreshEmissions())

	// policies
	v1g.GET("/orgs/:org_id/policies", a.ListPolicies())
	v1g.POST("/orgs/:org_id/policies", a.CreatePolicy())
	v1g.POST("/policies/:policy_id/retire", a.RetirePolicy())

	// violations
	v1g.GET("/orgs/:org_id/violations", a.ListViolations())
	v1g.POST("/violations/:violation_id/acknowledge", a.AcknowledgeViolation())

	return r, a
}

func getJSON(t *testing.T, r *gin.Engine, path string, expectedStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, expectedStatus, rr.Code, rr.Body.String())
	return rr
}

// ---------------------------------------------------------------------------
// 1. TestHandlers_ListOrganizations
// ---------------------------------------------------------------------------

func TestHandlers_ListOrganizations(t *testing.T) {
	r, _ := setupTestRouter(t)

	postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "OrgA", "email": "a@test.com"})
	postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "OrgB", "email": "b@test.com"})

	rr := getJSON(t, r, "/v1/orgs", http.StatusOK)
	var out map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))

	data, ok := out["data"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(data), 2)

	total, ok := out["total"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, int(total), 2)
}

// ---------------------------------------------------------------------------
// 2. TestHandlers_GetOrganization
// ---------------------------------------------------------------------------

func TestHandlers_GetOrganization(t *testing.T) {
	r, _ := setupTestRouter(t)

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "TestOrg", "email": "test@test.com"})

	// happy path
	rr := getJSON(t, r, "/v1/orgs/"+orgID, http.StatusOK)
	var out map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	assert.Equal(t, "TestOrg", out["name"])

	// invalid uuid -> 400
	getJSON(t, r, "/v1/orgs/invalid-uuid", http.StatusBadRequest)

	// random valid uuid -> 404
	getJSON(t, r, "/v1/orgs/"+uuid.NewString(), http.StatusNotFound)
}

// ---------------------------------------------------------------------------
// 3. TestHandlers_GetOrganizationSummary
// ---------------------------------------------------------------------------

func TestHandlers_GetOrganizationSummary(t *testing.T) {
	r, _ := setupTestRouter(t)

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "SumOrg", "email": "sum@test.com"})
	siteID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites", map[string]any{"name": "Site1"})
	postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites/"+siteID+"/assets", map[string]any{"name": "Asset1"})

	rr := getJSON(t, r, "/v1/orgs/"+orgID+"/summary", http.StatusOK)
	var out map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	assert.EqualValues(t, 1, out["site_count"])
	assert.EqualValues(t, 1, out["asset_count"])
}

// ---------------------------------------------------------------------------
// 4. TestHandlers_ListSitesAndAssets
// ---------------------------------------------------------------------------

func TestHandlers_ListSitesAndAssets(t *testing.T) {
	r, _ := setupTestRouter(t)

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "SAOrg", "email": "sa@test.com"})
	site1 := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites", map[string]any{"name": "SiteA"})
	site2 := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites", map[string]any{"name": "SiteB"})

	postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites/"+site1+"/assets", map[string]any{"name": "A1"})
	postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites/"+site1+"/assets", map[string]any{"name": "A2"})

	// list sites
	rr := getJSON(t, r, "/v1/orgs/"+orgID+"/sites", http.StatusOK)
	var sitesOut map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &sitesOut))
	assert.EqualValues(t, 2, sitesOut["total"])

	// list assets for site1 -> 2
	rr = getJSON(t, r, "/v1/orgs/"+orgID+"/sites/"+site1+"/assets", http.StatusOK)
	var assets1 map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &assets1))
	assert.EqualValues(t, 2, assets1["total"])

	// list assets for site2 -> 0
	rr = getJSON(t, r, "/v1/orgs/"+orgID+"/sites/"+site2+"/assets", http.StatusOK)
	var assets2 map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &assets2))
	assert.EqualValues(t, 0, assets2["total"])
}

// ---------------------------------------------------------------------------
// 5. TestHandlers_ListAssetTypes
// ---------------------------------------------------------------------------

func TestHandlers_ListAssetTypes(t *testing.T) {
	r, _ := setupTestRouter(t)

	postAndExtractID(t, r, "/v1/asset-types", map[string]any{"name": "type_" + uuid.NewString()})
	postAndExtractID(t, r, "/v1/asset-types", map[string]any{"name": "type_" + uuid.NewString()})

	rr := getJSON(t, r, "/v1/asset-types", http.StatusOK)
	var out map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	data, ok := out["data"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(data), 2)
}

// ---------------------------------------------------------------------------
// 6. TestHandlers_DeviceEndpoints
// ---------------------------------------------------------------------------

func TestHandlers_DeviceEndpoints(t *testing.T) {
	r, _ := setupTestRouter(t)

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "DevOrg", "email": "dev@test.com"})

	// create sensor (with org_id)
	postAndExtractID(t, r, "/v1/devices", map[string]any{"type": "sensor", "org_id": orgID})
	// create field_device (no org_id)
	postAndExtractID(t, r, "/v1/devices", map[string]any{"type": "field_device"})

	// list org devices -> sensor should appear
	rr := getJSON(t, r, "/v1/orgs/"+orgID+"/devices", http.StatusOK)
	var orgDevices map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &orgDevices))
	data, ok := orgDevices["data"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(data), 1)

	// list field devices -> field_device should appear
	rr = getJSON(t, r, "/v1/devices/field", http.StatusOK)
	var fieldDevices map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &fieldDevices))
	fdata, ok := fieldDevices["data"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(fdata), 1)
}

// ---------------------------------------------------------------------------
// 7. TestHandlers_IngestReadings_DuplicateBatch
// ---------------------------------------------------------------------------

func TestHandlers_IngestReadings_DuplicateBatch(t *testing.T) {
	r, _ := setupTestRouter(t)

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "DupOrg", "email": "dup@test.com"})
	siteID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites", map[string]any{"name": "DupSite"})
	assetID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites/"+siteID+"/assets", map[string]any{"name": "DupAsset"})
	deviceID := postAndExtractID(t, r, "/v1/devices", map[string]any{"type": "sensor", "org_id": orgID})

	batchID := uuid.NewString()
	readingsPath := "/v1/orgs/" + orgID + "/sites/" + siteID + "/readings"
	body := map[string]any{
		"batch_id": batchID,
		"readings": []map[string]any{{
			"asset_id":           assetID,
			"recorder_device_id": deviceID,
			"reading":            42.0,
			"recorded_at":        time.Now().UTC().Format(time.RFC3339),
		}},
	}

	// first ingest -> 201
	postJSON(t, r, readingsPath, body, http.StatusCreated)

	// duplicate -> 409
	rr := postJSON(t, r, readingsPath, body, http.StatusConflict)
	assert.Contains(t, rr.Body.String(), "duplicate")
}

// ---------------------------------------------------------------------------
// 8. TestHandlers_IngestReadings_BadInput
// ---------------------------------------------------------------------------

func TestHandlers_IngestReadings_BadInput(t *testing.T) {
	r, _ := setupTestRouter(t)

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "BadOrg", "email": "bad@test.com"})
	siteID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites", map[string]any{"name": "BadSite"})

	readingsPath := "/v1/orgs/" + orgID + "/sites/" + siteID + "/readings"

	// missing batch_id -> 400
	postJSON(t, r, readingsPath, map[string]any{
		"readings": []map[string]any{{"reading": 1.0, "recorded_at": time.Now().UTC().Format(time.RFC3339)}},
	}, http.StatusBadRequest)

	// invalid org_id in URL -> 400
	postJSON(t, r, "/v1/orgs/not-a-uuid/sites/"+siteID+"/readings", map[string]any{
		"batch_id": uuid.NewString(),
		"readings": []map[string]any{{"reading": 1.0, "recorded_at": time.Now().UTC().Format(time.RFC3339)}},
	}, http.StatusBadRequest)

	// empty readings array -> 400
	postJSON(t, r, readingsPath, map[string]any{
		"batch_id": uuid.NewString(),
		"readings": []map[string]any{},
	}, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// 9. TestHandlers_PolicyLifecycle
// ---------------------------------------------------------------------------

func TestHandlers_PolicyLifecycle(t *testing.T) {
	r, _ := setupTestRouter(t)

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "PolOrg", "email": "pol@test.com"})

	// create policy
	policyID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/policies", map[string]any{
		"entity_type":    "org",
		"entity_id":      orgID,
		"emission_limit": 100,
		"period":         "day",
		"effective_from": time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339),
	})

	// list policies -> total=1
	rr := getJSON(t, r, "/v1/orgs/"+orgID+"/policies", http.StatusOK)
	var list1 map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list1))
	assert.EqualValues(t, 1, list1["total"])

	// retire policy
	postJSON(t, r, "/v1/policies/"+policyID+"/retire", map[string]any{}, http.StatusOK)

	// active_only=true -> total=0
	rr = getJSON(t, r, "/v1/orgs/"+orgID+"/policies?active_only=true", http.StatusOK)
	var list2 map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list2))
	assert.EqualValues(t, 0, list2["total"])

	// active_only=false -> total=1
	rr = getJSON(t, r, "/v1/orgs/"+orgID+"/policies?active_only=false", http.StatusOK)
	var list3 map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list3))
	assert.EqualValues(t, 1, list3["total"])
}

// ---------------------------------------------------------------------------
// 10. TestHandlers_ViolationLifecycle
// ---------------------------------------------------------------------------

func TestHandlers_ViolationLifecycle(t *testing.T) {
	r, _ := setupTestRouter(t)

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "VioOrg", "email": "vio@test.com"})
	siteID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites", map[string]any{"name": "VioSite"})
	assetTypeID := postAndExtractID(t, r, "/v1/asset-types", map[string]any{"name": "vio_type_" + uuid.NewString()})
	assetID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites/"+siteID+"/assets", map[string]any{"name": "VioAsset", "asset_type_id": assetTypeID})
	deviceID := postAndExtractID(t, r, "/v1/devices", map[string]any{"type": "sensor", "org_id": orgID})

	// ingest a high reading
	postJSON(t, r, "/v1/orgs/"+orgID+"/sites/"+siteID+"/readings", map[string]any{
		"batch_id": uuid.NewString(),
		"readings": []map[string]any{{
			"asset_id":           assetID,
			"recorder_device_id": deviceID,
			"reading":            100.0,
			"recorded_at":        time.Now().UTC().Format(time.RFC3339),
		}},
	}, http.StatusCreated)

	// create policy with a low limit
	postAndExtractID(t, r, "/v1/orgs/"+orgID+"/policies", map[string]any{
		"entity_type":    "org",
		"entity_id":      orgID,
		"emission_limit": 10,
		"period":         "day",
		"effective_from": time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339),
	})

	// refresh emissions
	postJSON(t, r, "/v1/emissions/refresh", map[string]any{"window": "48 hours"}, http.StatusOK)

	// list open violations
	rr := getJSON(t, r, "/v1/orgs/"+orgID+"/violations?open_only=true", http.StatusOK)
	var vioList map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &vioList))

	total := int(vioList["total"].(float64))
	require.Greater(t, total, 0, "expected at least one violation")

	data := vioList["data"].([]any)
	firstVio := data[0].(map[string]any)
	violationID := firstVio["id"].(string)

	// acknowledge violation
	postJSON(t, r, "/v1/violations/"+violationID+"/acknowledge", map[string]any{}, http.StatusOK)

	// open violations should now be 0
	rr = getJSON(t, r, "/v1/orgs/"+orgID+"/violations?open_only=true", http.StatusOK)
	var vioList2 map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &vioList2))
	assert.EqualValues(t, 0, vioList2["total"])
}

// ---------------------------------------------------------------------------
// 11. TestHandlers_EmissionEndpoints
// ---------------------------------------------------------------------------

func TestHandlers_EmissionEndpoints(t *testing.T) {
	r, _ := setupTestRouter(t)

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "EmOrg", "email": "em@test.com"})
	siteID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites", map[string]any{"name": "EmSite"})
	assetTypeID := postAndExtractID(t, r, "/v1/asset-types", map[string]any{"name": "em_type_" + uuid.NewString()})
	assetID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites/"+siteID+"/assets", map[string]any{"name": "EmAsset", "asset_type_id": assetTypeID})
	deviceID := postAndExtractID(t, r, "/v1/devices", map[string]any{"type": "sensor", "org_id": orgID})

	now := time.Now().UTC()
	postJSON(t, r, "/v1/orgs/"+orgID+"/sites/"+siteID+"/readings", map[string]any{
		"batch_id": uuid.NewString(),
		"readings": []map[string]any{{
			"asset_id":           assetID,
			"recorder_device_id": deviceID,
			"reading":            55.5,
			"recorded_at":        now.Format(time.RFC3339),
		}},
	}, http.StatusCreated)

	// refresh emissions
	postJSON(t, r, "/v1/emissions/refresh", map[string]any{"window": "48 hours"}, http.StatusOK)

	from := now.Add(-24 * time.Hour).Format(time.RFC3339)
	to := now.Add(24 * time.Hour).Format(time.RFC3339)

	// GET trend -> 200
	trendPath := "/v1/orgs/" + orgID + "/emissions/trend?entity_type=org&entity_id=" + orgID + "&grain=day&from=" + from + "&to=" + to
	rr := getJSON(t, r, trendPath, http.StatusOK)
	var trendOut map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &trendOut))
	_, ok := trendOut["data"]
	assert.True(t, ok, "expected data field in trend response")

	// GET total -> 200
	totalPath := "/v1/orgs/" + orgID + "/emissions/total?entity_type=org&entity_id=" + orgID
	getJSON(t, r, totalPath, http.StatusOK)

	// GET trend with missing params -> 400
	getJSON(t, r, "/v1/orgs/"+orgID+"/emissions/trend?entity_type=org", http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// 12. TestHandlers_IngestReadings_CacheDuplicate
// ---------------------------------------------------------------------------

func TestHandlers_IngestReadings_CacheDuplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, dsn := testutil.SetupPostgres(t)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	password, _ := u.User.Password()
	dbName := strings.TrimPrefix(u.Path, "/")
	storeImpl, err := store.NewStore(u.Hostname(), u.Port(), dbName, u.User.Username(), password)
	require.NoError(t, err)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	c, err := cache.NewCache(mr.Host(), mr.Port())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	a := &AppV1{Store: storeImpl, Cache: c}

	r := gin.New()
	v1g := r.Group("/v1")
	v1g.POST("/orgs", a.CreateOrganization())
	v1g.POST("/orgs/:org_id/sites", a.CreateSite())
	v1g.POST("/orgs/:org_id/sites/:site_id/assets", a.CreateAsset())
	v1g.POST("/devices", a.CreateRecorderDevice())
	v1g.POST("/orgs/:org_id/sites/:site_id/readings", a.IngestReadings())

	orgID := postAndExtractID(t, r, "/v1/orgs", map[string]any{"name": "CacheOrg", "email": "cache@test.com"})
	siteID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites", map[string]any{"name": "CacheSite"})
	_ = postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites/"+siteID+"/assets", map[string]any{"name": "CacheAsset"})
	deviceID := postAndExtractID(t, r, "/v1/devices", map[string]any{"type": "sensor", "org_id": orgID})
	assetID := postAndExtractID(t, r, "/v1/orgs/"+orgID+"/sites/"+siteID+"/assets", map[string]any{"name": "CacheAsset2"})

	batchID := uuid.NewString()
	readingsPath := "/v1/orgs/" + orgID + "/sites/" + siteID + "/readings"
	body := map[string]any{
		"batch_id": batchID,
		"readings": []map[string]any{{
			"asset_id":           assetID,
			"recorder_device_id": deviceID,
			"reading":            55.5,
			"recorded_at":        time.Now().UTC().Format(time.RFC3339),
		}},
	}

	// First ingest -> 201, batch_id should be cached
	postJSON(t, r, readingsPath, body, http.StatusCreated)

	// Verify cache was populated
	val, err := c.GetKeyVal(context.Background(), "batch:"+batchID)
	require.NoError(t, err)
	assert.Equal(t, "1", val)

	// Second ingest -> 409 (caught by cache, never hits DB)
	rr := postJSON(t, r, readingsPath, body, http.StatusConflict)
	assert.Contains(t, rr.Body.String(), "duplicate")

	// Third ingest with a NEW batch_id -> 201 (cache miss, goes to DB)
	body2 := map[string]any{
		"batch_id": uuid.NewString(),
		"readings": []map[string]any{{
			"asset_id":           assetID,
			"recorder_device_id": deviceID,
			"reading":            33.3,
			"recorded_at":        time.Now().UTC().Format(time.RFC3339),
		}},
	}
	postJSON(t, r, readingsPath, body2, http.StatusCreated)
}

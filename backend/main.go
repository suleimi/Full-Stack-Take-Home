package main

import (
	"context"
	"net/http"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/config"
	v1 "github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/handlers/v1"
	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/metrics"
	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/migration"
	"github.com/gin-gonic/gin"
)

func main() {

	app, err := v1.NewApp(config.LoadConfig())
	if err != nil {
		panic(err)
	}
	defer app.Shutdown(context.Background())

	migrator, err := migration.NewSchemaMigrator(app.Store.DB())
	if err != nil {
		panic(err)
	}

	err = migrator.Up(context.Background())
	if err != nil {
		panic(err)
	}

	app.StartBackgroundJobWorker(context.Background())

	srv, err := newServer(app)
	if err != nil {
		panic(err)
	}

	err = srv.ListenAndServe()
	if err != nil {
		app.Logger.Print(err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func requestMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics.IncreaseRequestCounter()
		c.Next()
	}
}

func newServer(a *v1.AppV1) (*http.Server, error) {
	r := gin.Default()
	r.Use(corsMiddleware())
	r.Use(requestMetricsMiddleware())

	r.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "up"})
	})

	h := metrics.NewAppMetricHandler()
	r.GET("/metric", func(ctx *gin.Context) {
		h.ServeHTTP(ctx.Writer, ctx.Request)
	})

	v1g := r.Group("/v1")

	// Organizations
	v1g.GET("/orgs", a.ListOrganizations())
	v1g.POST("/orgs", a.CreateOrganization())
	v1g.GET("/orgs/:org_id", a.GetOrganization())
	v1g.GET("/orgs/:org_id/summary", a.GetOrganizationSummary())

	// Sites
	v1g.GET("/orgs/:org_id/sites", a.ListSites())
	v1g.POST("/orgs/:org_id/sites", a.CreateSite())

	// Asset types
	v1g.GET("/asset-types", a.ListAssetTypes())
	v1g.POST("/asset-types", a.CreateAssetType())

	// Assets
	v1g.GET("/orgs/:org_id/sites/:site_id/assets", a.ListAssets())
	v1g.POST("/orgs/:org_id/sites/:site_id/assets", a.CreateAsset())

	// Recorder devices
	v1g.POST("/devices", a.CreateRecorderDevice())
	v1g.GET("/devices/field", a.ListFieldDevices())
	v1g.GET("/orgs/:org_id/devices", a.ListOrgDevices())

	// Readings
	v1g.POST("/orgs/:org_id/sites/:site_id/readings", a.IngestReadings())

	// Emissions (rollups)
	v1g.GET("/orgs/:org_id/emissions/trend", a.GetEmissionTrend())
	v1g.GET("/orgs/:org_id/emissions/total", a.GetEmissionTotal())
	v1g.POST("/emissions/refresh", a.RefreshEmissions())

	// Policies
	v1g.GET("/orgs/:org_id/policies", a.ListPolicies())
	v1g.POST("/orgs/:org_id/policies", a.CreatePolicy())
	v1g.POST("/policies/:policy_id/retire", a.RetirePolicy())

	// Violations
	v1g.GET("/orgs/:org_id/violations", a.ListViolations())
	v1g.POST("/violations/:violation_id/acknowledge", a.AcknowledgeViolation())

	return &http.Server{
		Addr:    ":" + a.Config.Port,
		Handler: r,
	}, nil
}

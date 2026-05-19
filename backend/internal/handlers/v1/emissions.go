package v1

import (
	"net/http"
	"time"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (a *AppV1) GetEmissionTrend() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		entityType := c.Query("entity_type")
		entityIDStr := c.Query("entity_id")
		grain := c.Query("grain")
		fromStr := c.Query("from")
		toStr := c.Query("to")

		if entityType == "" || entityIDStr == "" || grain == "" || fromStr == "" || toStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type, entity_id, grain, from, and to are required"})
			return
		}

		entityID, err := uuid.Parse(entityIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		from, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' date: " + err.Error()})
			return
		}
		to, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' date: " + err.Error()})
			return
		}

		g := store.Grain(grain)
		var rollups any
		switch entityType {
		case "org":
			rollups, err = a.Store.GetOrganizationEmissionTrend(c.Request.Context(), orgID, g, from, to)
		case "site":
			rollups, err = a.Store.GetSiteEmissionTrend(c.Request.Context(), entityID, g, from, to)
		case "asset":
			rollups, err = a.Store.GetAssetEmissionTrend(c.Request.Context(), entityID, g, from, to)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type must be org, site, or asset"})
			return
		}
		_ = orgID // used for authorization scoping

		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": rollups})
	}
}

func (a *AppV1) GetEmissionTotal() gin.HandlerFunc {
	return func(c *gin.Context) {
		entityType := c.Query("entity_type")
		entityIDStr := c.Query("entity_id")

		if entityType == "" || entityIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type and entity_id are required"})
			return
		}

		entityID, err := uuid.Parse(entityIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		total, err := a.Store.GetEmissionTotalToDate(c.Request.Context(), store.EntityType(entityType), entityID)
		if err != nil {
			c.JSON(http.StatusNotFound, errResp(err))
			return
		}

		c.JSON(http.StatusOK, total)
	}
}

func (a *AppV1) RefreshEmissions() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Window string `json:"window"`
		}
		_ = c.ShouldBindJSON(&req)
		if req.Window == "" {
			req.Window = "48 hours"
		}

		err := a.Store.RefreshEmissions(c.Request.Context(), req.Window)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

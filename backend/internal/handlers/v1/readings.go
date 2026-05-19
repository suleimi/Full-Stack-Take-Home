package v1

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const batchCacheTTL = 24 * time.Hour

type ingestReadingsRequest struct {
	BatchID  string              `json:"batch_id" binding:"required"`
	Readings []store.ReadingInput `json:"readings" binding:"required,min=1"`
}

func (a *AppV1) IngestReadings() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}
		siteID, err := uuid.Parse(c.Param("site_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		var req ingestReadingsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		batchID, err := uuid.Parse(req.BatchID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "batch_id must be a valid UUID"})
			return
		}

		ctx := c.Request.Context()
		cacheKey := fmt.Sprintf("batch:%s", batchID)

		// Fast-path: check Redis cache before hitting the database.
		// A cache miss is safe — the DB-level check in InsertReadings
		// remains the authoritative duplicate guard.
		if a.Cache != nil {
			if v, _ := a.Cache.GetKeyVal(ctx, cacheKey); v != "" {
				if a.Logger != nil {
					a.Logger.Printf("readings: duplicate batch_id rejected from cache batch_id=%s", batchID)
				}
				c.JSON(http.StatusConflict, gin.H{
					"error":    "duplicate batch_id: readings already ingested",
					"batch_id": batchID,
				})
				return
			}
		}

		err = a.Store.InsertReadings(ctx, orgID, siteID, batchID, req.Readings)
		if err != nil {
			if errors.Is(err, store.ErrDuplicateBatch) {
				// Backfill cache so subsequent retries are caught fast.
				if a.Cache != nil {
					_ = a.Cache.SetKeyValWithTTL(ctx, cacheKey, "1", batchCacheTTL)
				}
				if a.Logger != nil {
					a.Logger.Printf("readings: duplicate batch_id rejected from database batch_id=%s", batchID)
				}
				c.JSON(http.StatusConflict, gin.H{
					"error":    "duplicate batch_id: readings already ingested",
					"batch_id": batchID,
				})
				return
			}
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		// Success: cache the batch_id so future duplicates are rejected from Redis.
		if a.Cache != nil {
			_ = a.Cache.SetKeyValWithTTL(ctx, cacheKey, "1", batchCacheTTL)
		}

		c.JSON(http.StatusCreated, gin.H{"batch_id": batchID, "count": len(req.Readings)})
	}
}

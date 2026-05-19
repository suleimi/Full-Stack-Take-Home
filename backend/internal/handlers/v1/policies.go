package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createPolicyRequest struct {
	EntityType    string    `json:"entity_type" binding:"required,oneof=org site asset"`
	EntityID      uuid.UUID `json:"entity_id" binding:"required"`
	EmissionLimit float64   `json:"emission_limit" binding:"required"`
	Unit          string    `json:"unit"`
	Period        string    `json:"period" binding:"required,oneof=hour day month"`
	EffectiveFrom *string   `json:"effective_from"`
}

func (a *AppV1) ListPolicies() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		activeOnly, _ := strconv.ParseBool(c.DefaultQuery("active_only", "true"))
		limit, offset := paginate(c)

		policies, err := a.Store.ListPoliciesByOrg(c.Request.Context(), orgID, activeOnly, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		total, err := a.Store.CountPoliciesByOrg(c.Request.Context(), orgID, activeOnly)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": policies, "total": total})
	}
}

func (a *AppV1) CreatePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		var req createPolicyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		unit := req.Unit
		if unit == "" {
			unit = "kg_co2e"
		}

		effectiveFrom := time.Now()
		if req.EffectiveFrom != nil {
			parsed, err := time.Parse(time.RFC3339, *req.EffectiveFrom)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid effective_from: " + err.Error()})
				return
			}
			effectiveFrom = parsed
		}

		policy, err := a.Store.CreateEmissionPolicy(c.Request.Context(), orgID, req.EntityID, store.EntityType(req.EntityType), store.Period(req.Period), req.EmissionLimit, store.EmissionUnit(unit), effectiveFrom)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusCreated, policy)
	}
}

func (a *AppV1) RetirePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		policyID, err := uuid.Parse(c.Param("policy_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		var req struct {
			EffectiveTo *string `json:"effective_to"`
		}
		_ = c.ShouldBindJSON(&req)

		effectiveTo := time.Now()
		if req.EffectiveTo != nil {
			parsed, err := time.Parse(time.RFC3339, *req.EffectiveTo)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid effective_to: " + err.Error()})
				return
			}
			effectiveTo = parsed
		}

		err = a.Store.RetireEmissionPolicy(c.Request.Context(), policyID, effectiveTo)
		if err != nil {
			c.JSON(http.StatusNotFound, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

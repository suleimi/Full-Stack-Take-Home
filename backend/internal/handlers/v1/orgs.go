package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createOrgRequest struct {
	Name           string  `json:"name" binding:"required"`
	Email          string  `json:"email" binding:"required,email"`
	Description    *string `json:"description"`
	OfficeLocation *string `json:"office_location"`
}

func (a *AppV1) ListOrganizations() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset := paginate(c)

		orgs, err := a.Store.ListOrganizations(c.Request.Context(), limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		total, err := a.Store.CountOrganizations(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": orgs, "total": total})
	}
}

func (a *AppV1) CreateOrganization() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createOrgRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		org, err := a.Store.CreateOrganization(c.Request.Context(), req.Name, req.Email, req.Description, req.OfficeLocation)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusCreated, org)
	}
}

func (a *AppV1) GetOrganization() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		org, err := a.Store.GetOrganization(c.Request.Context(), orgID)
		if err != nil {
			c.JSON(http.StatusNotFound, errResp(err))
			return
		}

		c.JSON(http.StatusOK, org)
	}
}

func (a *AppV1) GetOrganizationSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		summary, err := a.Store.GetOrganizationSummary(c.Request.Context(), orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusOK, summary)
	}
}

package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createSiteRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location"`
}

func (a *AppV1) ListSites() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		limit, offset := paginate(c)

		sites, err := a.Store.ListSitesByOrg(c.Request.Context(), orgID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		total, err := a.Store.CountSitesByOrg(c.Request.Context(), orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": sites, "total": total})
	}
}

func (a *AppV1) CreateSite() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		var req createSiteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		site, err := a.Store.CreateSite(c.Request.Context(), orgID, req.Name, req.Location)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusCreated, site)
	}
}

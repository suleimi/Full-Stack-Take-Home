package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createAssetRequest struct {
	Name        string     `json:"name" binding:"required"`
	AssetTypeID *uuid.UUID `json:"asset_type_id"`
}

func (a *AppV1) ListAssets() gin.HandlerFunc {
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

		limit, offset := paginate(c)

		assets, err := a.Store.ListAssetsBySite(c.Request.Context(), orgID, siteID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		total, err := a.Store.CountAssetsBySite(c.Request.Context(), orgID, siteID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": assets, "total": total})
	}
}

func (a *AppV1) CreateAsset() gin.HandlerFunc {
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

		var req createAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		asset, err := a.Store.CreateAsset(c.Request.Context(), orgID, siteID, req.Name, req.AssetTypeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusCreated, asset)
	}
}

func (a *AppV1) ListAssetTypes() gin.HandlerFunc {
	return func(c *gin.Context) {
		types, err := a.Store.ListAssetTypes(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": types})
	}
}

func (a *AppV1) CreateAssetType() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string  `json:"name" binding:"required"`
			Description *string `json:"description"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		at, err := a.Store.CreateAssetType(c.Request.Context(), req.Name, req.Description)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusCreated, at)
	}
}

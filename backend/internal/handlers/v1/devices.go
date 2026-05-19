package v1

import (
	"encoding/json"
	"net/http"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createDeviceRequest struct {
	Type           string          `json:"type" binding:"required,oneof=field_device sensor satellite"`
	OrgID          *uuid.UUID      `json:"org_id"`
	DeviceMetadata json.RawMessage `json:"device_metadata"`
}

func (a *AppV1) CreateRecorderDevice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createDeviceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		var metaStr *string
		if len(req.DeviceMetadata) > 0 && string(req.DeviceMetadata) != "null" {
			s := string(req.DeviceMetadata)
			metaStr = &s
		}

		device, err := a.Store.CreateRecorderDevice(c.Request.Context(), store.DeviceType(req.Type), req.OrgID, metaStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusCreated, device)
	}
}

func (a *AppV1) ListOrgDevices() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		limit, offset := paginate(c)

		devices, err := a.Store.ListRecorderDevicesByOrg(c.Request.Context(), orgID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": devices})
	}
}

func (a *AppV1) ListFieldDevices() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset := paginate(c)

		devices, err := a.Store.ListFieldDevices(c.Request.Context(), limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": devices})
	}
}

package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (a *AppV1) ListViolations() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		openOnly, _ := strconv.ParseBool(c.DefaultQuery("open_only", "true"))
		limit, offset := paginate(c)

		var total int
		var violations any

		if openOnly {
			violations, err = a.Store.ListOpenViolationsByOrg(c.Request.Context(), orgID, limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errResp(err))
				return
			}
			total, err = a.Store.CountOpenViolationsByOrg(c.Request.Context(), orgID)
		} else {
			violations, err = a.Store.ListViolationsByOrg(c.Request.Context(), orgID, limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errResp(err))
				return
			}
			total, err = a.Store.CountViolationsByOrg(c.Request.Context(), orgID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": violations, "total": total})
	}
}

func (a *AppV1) AcknowledgeViolation() gin.HandlerFunc {
	return func(c *gin.Context) {
		violationID, err := uuid.Parse(c.Param("violation_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp(err))
			return
		}

		err = a.Store.AcknowledgeViolation(c.Request.Context(), violationID)
		if err != nil {
			c.JSON(http.StatusNotFound, errResp(err))
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

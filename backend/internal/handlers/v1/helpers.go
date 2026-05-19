package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func errResp(err error) gin.H {
	return gin.H{"error": err.Error()}
}

func paginate(c *gin.Context) (int, int) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

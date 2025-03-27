package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common/ipblack"
)

func IPBlock(c *gin.Context) {
	ip := c.ClientIP()
	isBlock, err := ipblack.GetIPIsBlock(c.Request.Context(), ip)
	if err != nil {
		c.Next()
		return
	}
	if isBlock {
		AbortLogWithMessage(c, http.StatusForbidden, "please try again later")
		c.Abort()
		return
	}
	c.Next()
}

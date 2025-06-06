package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
)

func GenRequestID(t time.Time) string {
	return strconv.FormatInt(t.UnixMicro(), 10)
}

const (
	RequestIDHeader = "X-Request-Id"
)

func SetRequestID(c *gin.Context, id string) {
	c.Set(RequestID, id)
	c.Header(RequestIDHeader, id)
	log := common.GetLogger(c)
	SetLogRequestIDField(log.Data, id)
}

func GetRequestID(c *gin.Context) string {
	return c.GetString(RequestID)
}

func RequestIDMiddleware(c *gin.Context) {
	now := time.Now()
	id := GenRequestID(now)
	SetRequestID(c, id)
	SetRequestAt(c, now)
}

func SetRequestAt(c *gin.Context, requestAt time.Time) {
	c.Set(RequestAt, requestAt)
}

func GetRequestAt(c *gin.Context) time.Time {
	return c.GetTime(RequestAt)
}

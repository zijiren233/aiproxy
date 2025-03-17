package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GenRequestID() string {
	return strconv.FormatInt(time.Now().UnixMicro(), 10)
}

func SetRequestID(c *gin.Context, id string) {
	c.Set(RequestID, id)
	c.Header(RequestID, id)
	log := GetLogger(c)
	SetLogRequestIDField(log.Data, id)
}

func GetRequestID(c *gin.Context) string {
	return c.GetString(RequestID)
}

func RequestIDMiddleware(c *gin.Context) {
	id := GenRequestID()
	SetRequestID(c, id)
}

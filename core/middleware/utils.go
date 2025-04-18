package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/relay/model"
)

const (
	ErrorTypeAIPROXY = "aiproxy_error"
)

func MessageWithRequestID(c *gin.Context, message string) string {
	return fmt.Sprintf("%s (aiproxy: %s)", message, GetRequestID(c))
}

func AbortLogWithMessage(c *gin.Context, statusCode int, message string, fields ...*ErrorField) {
	GetLogger(c).Error(message)
	AbortWithMessage(c, statusCode, message, fields...)
}

type ErrorField struct {
	Type string `json:"type"`
	Code any    `json:"code"`
}

func AbortWithMessage(c *gin.Context, statusCode int, message string, fields ...*ErrorField) {
	typeName := ErrorTypeAIPROXY
	var code any
	if len(fields) > 0 {
		if fields[0].Type != "" {
			typeName = fields[0].Type
		}
		code = fields[0].Code
	}
	c.JSON(statusCode, gin.H{
		"error": &model.Error{
			Message: MessageWithRequestID(c, message),
			Type:    typeName,
			Code:    code,
		},
	})
	c.Abort()
}

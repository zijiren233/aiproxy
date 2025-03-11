package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/relay/meta"
)

func RelayParsePdfHelper(meta *meta.Meta, c *gin.Context) *HandleResult {
	return Handle(meta, c, func() (*PreCheckGroupBalanceReq, error) {
		return &PreCheckGroupBalanceReq{}, nil
	})
}

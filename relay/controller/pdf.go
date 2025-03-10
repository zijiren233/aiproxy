package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/relay/meta"
	relaymodel "github.com/labring/aiproxy/relay/model"
)

func RelayParsePdfHelper(meta *meta.Meta, c *gin.Context) *relaymodel.ErrorWithStatusCode {
	return Handle(meta, c, func() (*PreCheckGroupBalanceReq, error) {
		return &PreCheckGroupBalanceReq{}, nil
	})
}

package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/config"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
)

func Handle(
	innerAdaptor adaptor.Adaptor,
	c *gin.Context,
	meta *meta.Meta,
	store adaptor.Store,
) *HandleResult {
	log := common.GetLogger(c)

	usageResult, detail, respErr := DoHelper(innerAdaptor, c, meta, store)

	// Extract usage from UsageResult
	var usage model.Usage
	if usageResult != nil {
		usage = usageResult.Usage()
	}

	if respErr != nil {
		var logDetail *RequestDetail
		if detail != nil && config.DebugEnabled {
			logDetail = detail
			log.Errorf(
				"handle failed: %+v\nrequest detail:\n%s\nresponse detail:\n%s",
				respErr,
				logDetail.RequestBody,
				logDetail.ResponseBody,
			)
		} else {
			log.Errorf("handle failed: %+v", respErr)
		}

		return &HandleResult{
			Error:       respErr,
			Usage:       usage,
			Detail:      detail,
			UsageResult: usageResult,
		}
	}

	return &HandleResult{
		Usage:       usage,
		Detail:      detail,
		UsageResult: usageResult,
	}
}

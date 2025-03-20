package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common/config"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/channeltype"
	"github.com/labring/aiproxy/relay/meta"
	relaymodel "github.com/labring/aiproxy/relay/model"
)

// HandleResult contains all the information needed for consumption recording
type HandleResult struct {
	Error  *relaymodel.ErrorWithStatusCode
	Usage  *relaymodel.Usage
	Detail *model.RequestDetail
}

func Handle(meta *meta.Meta, c *gin.Context) *HandleResult {
	log := middleware.GetLogger(c)

	// 1. Get adaptor
	adaptor, ok := channeltype.GetAdaptor(meta.Channel.Type)
	if !ok {
		log.Errorf("invalid (%s[%d]) channel type: %d", meta.Channel.Name, meta.Channel.ID, meta.Channel.Type)
		return &HandleResult{
			Error: openai.ErrorWrapperWithMessage(
				"invalid channel error", "invalid_channel_type", http.StatusInternalServerError),
		}
	}

	// 5. Do request
	usage, detail, respErr := DoHelper(adaptor, c, meta)
	if respErr != nil {
		var logDetail *model.RequestDetail
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
			Error:  respErr,
			Usage:  usage,
			Detail: detail,
		}
	}

	return &HandleResult{
		Usage:  usage,
		Detail: detail,
	}
}

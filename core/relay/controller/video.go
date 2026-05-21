package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

func GetVideoGenerationJobRequestUsage(c *gin.Context, _ model.ModelConfig) (model.Usage, error) {
	contentType := c.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := common.ParseMultipartFormWithLimit(c.Request); err != nil {
			return model.Usage{}, err
		}

		return model.Usage{}, nil
	}

	_, err := utils.UnmarshalVideoGenerationJobRequest(c.Request)
	if err != nil {
		return model.Usage{}, err
	}

	return model.Usage{}, nil
}

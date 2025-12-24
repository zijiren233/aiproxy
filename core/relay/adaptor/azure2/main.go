package azure2

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/azure"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
)

type Adaptor struct {
	azure.Adaptor
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	// Use parent's ConvertRequest
	result, err := a.Adaptor.ConvertRequest(meta, store, c, req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	// Override URL with Azure-specific URL (replaceDot = false for azure2)
	method, fullURL, err := azure.GetRequestURL(meta, false)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	result.Method = method
	result.URL = fullURL

	return result, nil
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme: fmt.Sprintf(
			"Model names can contain '.' character\nAPI version is optional, default is '%s'\nGemini support",
			azure.DefaultAPIVersion,
		),
		KeyHelp: "key or key|api-version",
		Models:  openai.ModelList,
	}
}

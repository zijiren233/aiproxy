package textembeddingsinference

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
)

func EmbeddingsHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (model.Usage, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return model.Usage{}, EmbeddingsErrorHanlder(resp)
	}

	result, err := openai.DoResponse(meta, store, c, resp)
	if err != nil {
		return model.Usage{}, err
	}

	return result.Usage(), nil
}

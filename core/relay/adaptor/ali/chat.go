package ali

import (
	"fmt"
	"net/http"

	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func getEnableSearch(node *ast.Node) bool {
	enableSearch, _ := node.Get("enable_search").Bool()
	return enableSearch
}

func ChatHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (model.Usage, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return model.Usage{}, ErrorHanlder(resp)
	}

	node, err := common.UnmarshalBody2Node(c.Request)
	if err != nil {
		return model.Usage{}, relaymodel.WrapperOpenAIErrorWithMessage(
			fmt.Sprintf("get request body failed: %s", err),
			"get_request_body_failed",
			http.StatusInternalServerError,
		)
	}
	u, e := openai.DoResponse(meta, store, c, resp)
	if e != nil {
		return model.Usage{}, e
	}
	if getEnableSearch(&node) {
		u.WebSearchCount++
	}
	return u, nil
}

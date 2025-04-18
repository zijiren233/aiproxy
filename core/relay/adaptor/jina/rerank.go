package jina

import (
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func RerankHandler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	node, err := sonic.Get(responseBody)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	var usage relaymodel.Usage
	usageNode := node.Get("usage")
	usageStr, err := usageNode.Raw()
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_usage_failed", http.StatusInternalServerError)
	}
	err = sonic.UnmarshalString(usageStr, &usage)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_usage_failed", http.StatusInternalServerError)
	}
	if usage.PromptTokens == 0 && usage.TotalTokens != 0 {
		usage.PromptTokens = usage.TotalTokens
	} else if usage.PromptTokens == 0 {
		usage.PromptTokens = meta.InputTokens
		usage.TotalTokens = meta.InputTokens
	}
	modelUsage := usage.ToModelUsage()
	node.SetAny("meta", map[string]any{
		"tokens": modelUsage,
	})
	_, err = node.Unset("usage")
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_usage_failed", http.StatusInternalServerError)
	}
	_, err = node.Set("model", ast.NewString(meta.OriginModel))
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_usage_failed", http.StatusInternalServerError)
	}
	c.Writer.WriteHeader(resp.StatusCode)
	respData, err := node.MarshalJSON()
	if err != nil {
		return nil, openai.ErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
	}
	_, err = c.Writer.Write(respData)
	if err != nil {
		log.Warnf("write response body failed: %v", err)
	}
	return modelUsage, nil
}

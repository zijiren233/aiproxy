package openai

import (
	"bytes"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func ConvertRerankRequest(
	meta *meta.Meta,
	req *http.Request,
) (*adaptor.ConvertRequestResult, error) {
	node, err := common.UnmarshalBody2Node(req)
	if err != nil {
		return nil, err
	}

	_, err = node.Set("model", ast.NewString(meta.ActualModel))
	if err != nil {
		return nil, err
	}

	jsonData, err := node.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return &adaptor.ConvertRequestResult{
		Method: http.MethodPost,
		Header: nil,
		Body:   bytes.NewReader(jsonData),
	}, nil
}

func RerankHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (*model.Usage, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, relaymodel.WrapperOpenAIError(
			err,
			"read_response_body_failed",
			http.StatusInternalServerError,
		)
	}
	var rerankResponse relaymodel.SlimRerankResponse
	err = sonic.Unmarshal(responseBody, &rerankResponse)
	if err != nil {
		return nil, relaymodel.WrapperOpenAIError(
			err,
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	c.Writer.WriteHeader(resp.StatusCode)

	_, err = c.Writer.Write(responseBody)
	if err != nil {
		log.Warnf("write response body failed: %v", err)
	}

	if rerankResponse.Meta.Tokens == nil {
		return &model.Usage{
			InputTokens: meta.RequestUsage.InputTokens,
			TotalTokens: meta.RequestUsage.InputTokens,
		}, nil
	}
	if rerankResponse.Meta.Tokens.InputTokens <= 0 {
		rerankResponse.Meta.Tokens.InputTokens = int64(meta.RequestUsage.InputTokens)
	}
	return &model.Usage{
		InputTokens:  model.ZeroNullInt64(rerankResponse.Meta.Tokens.InputTokens),
		OutputTokens: model.ZeroNullInt64(rerankResponse.Meta.Tokens.OutputTokens),
		TotalTokens: model.ZeroNullInt64(
			rerankResponse.Meta.Tokens.InputTokens + rerankResponse.Meta.Tokens.OutputTokens,
		),
	}, nil
}

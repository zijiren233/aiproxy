package ali

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	model "github.com/labring/aiproxy/core/relay/model"
)

// Deprecated: Use openai.ConvertRequest instead
// /api/v1/services/embeddings/text-embedding/text-embedding

func ConvertEmbeddingsRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	var reqMap map[string]any
	err := common.UnmarshalBodyReusable(req, &reqMap)
	if err != nil {
		return "", nil, nil, err
	}
	reqMap["model"] = meta.ActualModel
	input, ok := reqMap["input"]
	if !ok {
		return "", nil, nil, errors.New("input is required")
	}
	switch v := input.(type) {
	case string:
		reqMap["input"] = map[string]any{
			"texts": []string{v},
		}
	case []any:
		reqMap["input"] = map[string]any{
			"texts": v,
		}
	}
	parameters := make(map[string]any)
	for k, v := range reqMap {
		if k == "model" || k == "input" {
			continue
		}
		parameters[k] = v
		delete(reqMap, k)
	}
	reqMap["parameters"] = parameters
	jsonData, err := sonic.Marshal(reqMap)
	if err != nil {
		return "", nil, nil, err
	}
	return http.MethodPost, nil, bytes.NewReader(jsonData), nil
}

func embeddingResponse2OpenAI(meta *meta.Meta, response *EmbeddingResponse) *model.EmbeddingResponse {
	openAIEmbeddingResponse := model.EmbeddingResponse{
		Object: "list",
		Data:   make([]*model.EmbeddingResponseItem, 0, 1),
		Model:  meta.OriginModel,
		Usage:  response.Usage,
	}

	for i, embedding := range response.Output.Embeddings {
		openAIEmbeddingResponse.Data = append(openAIEmbeddingResponse.Data, &model.EmbeddingResponseItem{
			Object:    "embedding",
			Index:     i,
			Embedding: embedding.Embedding,
		})
	}
	return &openAIEmbeddingResponse
}

func EmbeddingsHandler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "read_response_body_failed", resp.StatusCode)
	}
	var respBody EmbeddingResponse
	err = sonic.Unmarshal(responseBody, &respBody)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_response_body_failed", resp.StatusCode)
	}
	if respBody.Usage.PromptTokens == 0 {
		respBody.Usage.PromptTokens = respBody.Usage.TotalTokens
	}
	openaiResponse := embeddingResponse2OpenAI(meta, &respBody)
	data, err := sonic.Marshal(openaiResponse)
	if err != nil {
		return &respBody.Usage, openai.ErrorWrapper(err, "marshal_response_body_failed", resp.StatusCode)
	}
	_, err = c.Writer.Write(data)
	if err != nil {
		log.Warnf("write response body failed: %v", err)
	}
	return &openaiResponse.Usage, nil
}

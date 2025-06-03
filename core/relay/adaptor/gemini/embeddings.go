package gemini

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

func ConvertEmbeddingRequest(
	meta *meta.Meta,
	req *http.Request,
) (*adaptor.ConvertRequestResult, error) {
	request, err := utils.UnmarshalGeneralOpenAIRequest(req)
	if err != nil {
		return nil, err
	}
	request.Model = meta.ActualModel

	inputs := request.ParseInput()
	requests := make([]EmbeddingRequest, len(inputs))
	model := "models/" + request.Model

	for i, input := range inputs {
		requests[i] = EmbeddingRequest{
			Model: model,
			Content: ChatContent{
				Parts: []*Part{
					{
						Text: input,
					},
				},
			},
		}
	}

	data, err := sonic.Marshal(BatchEmbeddingRequest{
		Requests: requests,
	})
	if err != nil {
		return nil, err
	}
	return &adaptor.ConvertRequestResult{
		Header: nil,
		Body:   bytes.NewReader(data),
	}, nil
}

func EmbeddingHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (*model.Usage, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return nil, openai.ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var geminiEmbeddingResponse EmbeddingResponse
	err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&geminiEmbeddingResponse)
	if err != nil {
		return nil, relaymodel.WrapperOpenAIError(
			err,
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}
	if geminiEmbeddingResponse.Error != nil {
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			geminiEmbeddingResponse.Error.Message,
			geminiEmbeddingResponse.Error.Code,
			resp.StatusCode,
		)
	}
	fullTextResponse := embeddingResponse2OpenAI(meta, &geminiEmbeddingResponse)
	jsonResponse, err := sonic.Marshal(fullTextResponse)
	if err != nil {
		return nil, relaymodel.WrapperOpenAIError(
			err,
			"marshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(jsonResponse)))
	_, _ = c.Writer.Write(jsonResponse)
	return fullTextResponse.ToModelUsage(), nil
}

func embeddingResponse2OpenAI(
	meta *meta.Meta,
	response *EmbeddingResponse,
) *relaymodel.EmbeddingResponse {
	openAIEmbeddingResponse := relaymodel.EmbeddingResponse{
		Object: "list",
		Data:   make([]*relaymodel.EmbeddingResponseItem, 0, len(response.Embeddings)),
		Model:  meta.OriginModel,
		Usage: relaymodel.Usage{
			TotalTokens:  int64(meta.RequestUsage.InputTokens),
			PromptTokens: int64(meta.RequestUsage.InputTokens),
		},
	}
	for _, item := range response.Embeddings {
		openAIEmbeddingResponse.Data = append(
			openAIEmbeddingResponse.Data,
			&relaymodel.EmbeddingResponseItem{
				Object:    `embedding`,
				Index:     0,
				Embedding: item.Values,
			},
		)
	}
	return &openAIEmbeddingResponse
}

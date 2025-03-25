package baidu

import (
	"bufio"
	"bytes"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common"
	"github.com/labring/aiproxy/common/conv"
	"github.com/labring/aiproxy/common/render"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/model"
	"github.com/labring/aiproxy/relay/utils"
)

// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/flfmc9do2

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Temperature     *float64         `json:"temperature,omitempty"`
	TopP            *float64         `json:"top_p,omitempty"`
	PenaltyScore    *float64         `json:"penalty_score,omitempty"`
	System          string           `json:"system,omitempty"`
	UserID          string           `json:"user_id,omitempty"`
	Messages        []*model.Message `json:"messages"`
	MaxOutputTokens int              `json:"max_output_tokens,omitempty"`
	Stream          bool             `json:"stream,omitempty"`
	DisableSearch   bool             `json:"disable_search,omitempty"`
	EnableCitation  bool             `json:"enable_citation,omitempty"`
}

func ConvertRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	request, err := utils.UnmarshalGeneralOpenAIRequest(req)
	if err != nil {
		return "", nil, nil, err
	}
	request.Model = meta.ActualModel
	baiduRequest := ChatRequest{
		Messages:        request.Messages,
		Temperature:     request.Temperature,
		TopP:            request.TopP,
		Stream:          request.Stream,
		DisableSearch:   false,
		EnableCitation:  false,
		MaxOutputTokens: request.MaxTokens,
		UserID:          request.User,
	}
	// Convert frequency penalty to penalty score range [1.0, 2.0]
	if request.FrequencyPenalty != nil {
		penaltyScore := *request.FrequencyPenalty
		if penaltyScore < -2.0 {
			penaltyScore = -2.0
		}
		if penaltyScore > 2.0 {
			penaltyScore = 2.0
		}
		// Map [-2.0, 2.0] to [1.0, 2.0]
		mappedScore := (penaltyScore+2.0)/4.0 + 1.0
		baiduRequest.PenaltyScore = &mappedScore
	}

	for i, message := range request.Messages {
		if message.Role == "system" {
			baiduRequest.System = message.StringContent()
			request.Messages = append(request.Messages[:i], request.Messages[i+1:]...)
			break
		}
	}

	data, err := sonic.Marshal(baiduRequest)
	if err != nil {
		return "", nil, nil, err
	}
	return http.MethodPost, nil, bytes.NewReader(data), nil
}

func response2OpenAI(meta *meta.Meta, response *ChatResponse) *model.TextResponse {
	choice := model.TextResponseChoice{
		Index: 0,
		Message: model.Message{
			Role:    "assistant",
			Content: response.Result,
		},
		FinishReason: model.StopFinishReason,
	}
	fullTextResponse := model.TextResponse{
		ID:      response.ID,
		Object:  model.ChatCompletion,
		Created: response.Created,
		Model:   meta.OriginModel,
		Choices: []*model.TextResponseChoice{&choice},
	}
	if response.Usage != nil {
		fullTextResponse.Usage = *response.Usage
	}
	return &fullTextResponse
}

func streamResponse2OpenAI(meta *meta.Meta, baiduResponse *ChatStreamResponse) *model.ChatCompletionsStreamResponse {
	var choice model.ChatCompletionsStreamResponseChoice
	choice.Delta.Content = baiduResponse.Result
	if baiduResponse.IsEnd {
		finishReason := model.StopFinishReason
		choice.FinishReason = &finishReason
	}
	response := model.ChatCompletionsStreamResponse{
		ID:      baiduResponse.ID,
		Object:  model.ChatCompletionChunk,
		Created: baiduResponse.Created,
		Model:   meta.OriginModel,
		Choices: []*model.ChatCompletionsStreamResponseChoice{&choice},
		Usage:   baiduResponse.Usage,
	}
	return &response
}

func StreamHandler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	var usage model.Usage
	scanner := bufio.NewScanner(resp.Body)
	buf := openai.GetScannerBuffer()
	defer openai.PutScannerBuffer(buf)
	scanner.Buffer(*buf, cap(*buf))

	common.SetEventStreamHeaders(c)

	for scanner.Scan() {
		data := scanner.Bytes()
		if len(data) < 6 || conv.BytesToString(data[:6]) != "data: " {
			continue
		}
		data = data[6:]

		if conv.BytesToString(data) == "[DONE]" {
			break
		}

		var baiduResponse ChatStreamResponse
		err := sonic.Unmarshal(data, &baiduResponse)
		if err != nil {
			log.Error("error unmarshalling stream response: " + err.Error())
			continue
		}
		response := streamResponse2OpenAI(meta, &baiduResponse)
		if response.Usage != nil {
			usage = *response.Usage
		}
		_ = render.ObjectData(c, response)
	}

	if err := scanner.Err(); err != nil {
		log.Error("error reading stream: " + err.Error())
	}

	render.Done(c)

	return nil, &usage
}

func Handler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
	defer resp.Body.Close()

	var baiduResponse ChatResponse
	err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&baiduResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if baiduResponse.Error != nil && baiduResponse.Error.ErrorCode != 0 {
		return nil, ErrorHandler(baiduResponse.Error)
	}
	fullTextResponse := response2OpenAI(meta, &baiduResponse)
	jsonResponse, err := sonic.Marshal(fullTextResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(jsonResponse)
	return &fullTextResponse.Usage, nil
}

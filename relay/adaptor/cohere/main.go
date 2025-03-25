package cohere

import (
	"bufio"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common"
	"github.com/labring/aiproxy/common/conv"
	"github.com/labring/aiproxy/common/render"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/model"
)

var WebSearchConnector = Connector{ID: "web-search"}

func stopReasonCohere2OpenAI(reason *string) string {
	if reason == nil {
		return ""
	}
	switch *reason {
	case "COMPLETE":
		return model.StopFinishReason
	default:
		return *reason
	}
}

func ConvertRequest(textRequest *model.GeneralOpenAIRequest) *Request {
	cohereRequest := Request{
		Model:            textRequest.Model,
		Message:          "",
		MaxTokens:        textRequest.MaxTokens,
		Temperature:      textRequest.Temperature,
		P:                textRequest.TopP,
		K:                textRequest.TopK,
		Stream:           textRequest.Stream,
		FrequencyPenalty: textRequest.FrequencyPenalty,
		PresencePenalty:  textRequest.PresencePenalty,
		Seed:             int(textRequest.Seed),
	}
	if cohereRequest.Model == "" {
		cohereRequest.Model = "command-r"
	}
	if strings.HasSuffix(cohereRequest.Model, "-internet") {
		cohereRequest.Model = strings.TrimSuffix(cohereRequest.Model, "-internet")
		cohereRequest.Connectors = append(cohereRequest.Connectors, WebSearchConnector)
	}
	for _, message := range textRequest.Messages {
		if message.Role == "user" {
			cohereRequest.Message = message.Content.(string)
		} else {
			var role string
			switch message.Role {
			case "assistant":
				role = "CHATBOT"
			case "system":
				role = "SYSTEM"
			default:
				role = "USER"
			}
			cohereRequest.ChatHistory = append(cohereRequest.ChatHistory, ChatMessage{
				Role:    role,
				Message: message.Content.(string),
			})
		}
	}
	return &cohereRequest
}

func StreamResponse2OpenAI(meta *meta.Meta, cohereResponse *StreamResponse) *model.ChatCompletionsStreamResponse {
	var response *Response
	var responseText string
	var finishReason string

	switch cohereResponse.EventType {
	case "stream-start":
		return nil
	case "text-generation":
		responseText += cohereResponse.Text
	case "stream-end":
		usage := cohereResponse.Response.Meta.Tokens
		response = &Response{
			Meta: Meta{
				Tokens: Usage{
					InputTokens:  usage.InputTokens,
					OutputTokens: usage.OutputTokens,
				},
			},
		}
		finishReason = *cohereResponse.Response.FinishReason
	default:
		return nil
	}

	var choice model.ChatCompletionsStreamResponseChoice
	choice.Delta.Content = responseText
	choice.Delta.Role = "assistant"
	if finishReason != "" {
		choice.FinishReason = &finishReason
	}
	openaiResponse := model.ChatCompletionsStreamResponse{
		ID:      "chatcmpl-" + cohereResponse.GenerationID,
		Model:   meta.OriginModel,
		Created: time.Now().Unix(),
		Object:  model.ChatCompletionChunk,
		Choices: []*model.ChatCompletionsStreamResponseChoice{&choice},
	}
	if response != nil {
		openaiResponse.Usage = &model.Usage{
			PromptTokens:     response.Meta.Tokens.InputTokens,
			CompletionTokens: response.Meta.Tokens.OutputTokens,
			TotalTokens:      response.Meta.Tokens.InputTokens + response.Meta.Tokens.OutputTokens,
		}
	}
	return &openaiResponse
}

func Response2OpenAI(meta *meta.Meta, cohereResponse *Response) *model.TextResponse {
	choice := model.TextResponseChoice{
		Index: 0,
		Message: model.Message{
			Role:    "assistant",
			Content: cohereResponse.Text,
			Name:    nil,
		},
		FinishReason: stopReasonCohere2OpenAI(cohereResponse.FinishReason),
	}
	fullTextResponse := model.TextResponse{
		ID:      openai.ChatCompletionID(),
		Model:   meta.OriginModel,
		Object:  model.ChatCompletion,
		Created: time.Now().Unix(),
		Choices: []*model.TextResponseChoice{&choice},
		Usage: model.Usage{
			PromptTokens:     cohereResponse.Meta.Tokens.InputTokens,
			CompletionTokens: cohereResponse.Meta.Tokens.OutputTokens,
			TotalTokens:      cohereResponse.Meta.Tokens.InputTokens + cohereResponse.Meta.Tokens.OutputTokens,
		},
	}
	return &fullTextResponse
}

func StreamHandler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, openai.ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	scanner := bufio.NewScanner(resp.Body)
	buf := openai.GetScannerBuffer()
	defer openai.PutScannerBuffer(buf)
	scanner.Buffer(*buf, cap(*buf))

	common.SetEventStreamHeaders(c)
	var usage model.Usage

	for scanner.Scan() {
		data := scanner.Text()
		data = strings.TrimSuffix(data, "\r")

		var cohereResponse StreamResponse
		err := sonic.Unmarshal(conv.StringToBytes(data), &cohereResponse)
		if err != nil {
			log.Error("error unmarshalling stream response: " + err.Error())
			continue
		}

		response := StreamResponse2OpenAI(meta, &cohereResponse)
		if response.Usage != nil {
			usage = *response.Usage
		}

		_ = render.ObjectData(c, response)
	}

	if err := scanner.Err(); err != nil {
		log.Error("error reading stream: " + err.Error())
	}

	render.Done(c)

	return &usage, nil
}

func Handler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, openai.ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	var cohereResponse Response
	err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&cohereResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if cohereResponse.ResponseID == "" {
		return nil, openai.ErrorWrapperWithMessage(cohereResponse.Message, resp.StatusCode, resp.StatusCode)
	}
	fullTextResponse := Response2OpenAI(meta, &cohereResponse)
	jsonResponse, err := sonic.Marshal(fullTextResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(jsonResponse)
	return &fullTextResponse.Usage, nil
}

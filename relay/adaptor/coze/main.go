package coze

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
	"github.com/labring/aiproxy/relay/adaptor/coze/constant/messagetype"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/model"
)

// https://www.coze.com/open

func stopReasonCoze2OpenAI(reason *string) string {
	if reason == nil {
		return ""
	}
	switch *reason {
	case "end_turn":
		return "stop"
	case "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	default:
		return *reason
	}
}

func StreamResponse2OpenAI(meta *meta.Meta, cozeResponse *StreamResponse) *model.ChatCompletionsStreamResponse {
	var stopReason string
	var choice model.ChatCompletionsStreamResponseChoice

	if cozeResponse.Message != nil {
		if cozeResponse.Message.Type != messagetype.Answer {
			return nil
		}
		choice.Delta.Content = cozeResponse.Message.Content
	}
	choice.Delta.Role = "assistant"
	finishReason := stopReasonCoze2OpenAI(&stopReason)
	if finishReason != "null" {
		choice.FinishReason = &finishReason
	}
	openaiResponse := model.ChatCompletionsStreamResponse{
		ID:      cozeResponse.ConversationID,
		Model:   meta.OriginModel,
		Created: time.Now().Unix(),
		Object:  model.ChatCompletionChunk,
		Choices: []*model.ChatCompletionsStreamResponseChoice{&choice},
	}
	return &openaiResponse
}

func Response2OpenAI(meta *meta.Meta, cozeResponse *Response) *model.TextResponse {
	var responseText string
	for _, message := range cozeResponse.Messages {
		if message.Type == messagetype.Answer {
			responseText = message.Content
			break
		}
	}
	choice := model.TextResponseChoice{
		Index: 0,
		Message: model.Message{
			Role:    "assistant",
			Content: responseText,
			Name:    nil,
		},
		FinishReason: model.StopFinishReason,
	}
	fullTextResponse := model.TextResponse{
		ID:      openai.ChatCompletionID(),
		Model:   meta.OriginModel,
		Object:  model.ChatCompletion,
		Created: time.Now().Unix(),
		Choices: []*model.TextResponseChoice{&choice},
	}
	return &fullTextResponse
}

func StreamHandler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, openai.ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	responseText := strings.Builder{}
	createdTime := time.Now().Unix()

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

		var cozeResponse StreamResponse
		err := sonic.Unmarshal(data, &cozeResponse)
		if err != nil {
			log.Error("error unmarshalling stream response: " + err.Error())
			continue
		}

		response := StreamResponse2OpenAI(meta, &cozeResponse)
		if response == nil {
			continue
		}

		for _, choice := range response.Choices {
			responseText.WriteString(choice.Delta.StringContent())
		}
		response.Model = meta.OriginModel
		response.Created = createdTime

		_ = render.ObjectData(c, response)
	}

	if err := scanner.Err(); err != nil {
		log.Error("error reading stream: " + err.Error())
	}

	render.Done(c)

	return openai.ResponseText2Usage(responseText.String(), meta.ActualModel, meta.InputTokens), nil
}

func Handler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, openai.ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	var cozeResponse Response
	err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&cozeResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if cozeResponse.Code != 0 {
		return nil, openai.ErrorWrapperWithMessage(cozeResponse.Msg, cozeResponse.Code, resp.StatusCode)
	}
	fullTextResponse := Response2OpenAI(meta, &cozeResponse)
	jsonResponse, err := sonic.Marshal(fullTextResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(jsonResponse)
	if err != nil {
		log.Warnf("write response body failed: %v", err)
	}
	var responseText string
	if len(fullTextResponse.Choices) > 0 {
		responseText = fullTextResponse.Choices[0].Message.StringContent()
	}
	return openai.ResponseText2Usage(responseText, meta.ActualModel, meta.InputTokens), nil
}

package anthropic

import (
	"bufio"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common"
	"github.com/labring/aiproxy/common/conv"
	"github.com/labring/aiproxy/common/image"
	"github.com/labring/aiproxy/common/render"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/model"
)

const (
	toolUseType          = "tool_use"
	conetentTypeText     = "text"
	conetentTypeThinking = "thinking"
	conetentTypeImage    = "image"
)

func stopReasonClaude2OpenAI(reason *string) string {
	if reason == nil {
		return ""
	}
	switch *reason {
	case "end_turn", "stop_sequence":
		return model.StopFinishReason
	case "max_tokens":
		return "length"
	case toolUseType:
		return "tool_calls"
	default:
		return *reason
	}
}

type onlyThinkingRequest struct {
	Thinking *Thinking `json:"thinking,omitempty"`
}

func ConvertRequest(meta *meta.Meta, req *http.Request) (*Request, error) {
	var textRequest OpenAIRequest
	err := common.UnmarshalBodyReusable(req, &textRequest)
	if err != nil {
		return nil, err
	}

	var onlyThinking onlyThinkingRequest
	err = common.UnmarshalBodyReusable(req, &onlyThinking)
	if err != nil {
		return nil, err
	}

	textRequest.Model = meta.ActualModel
	meta.Set("stream", textRequest.Stream)
	claudeTools := make([]Tool, 0, len(textRequest.Tools))

	for _, tool := range textRequest.Tools {
		if params, ok := tool.Function.Parameters.(map[string]any); ok {
			t, _ := params["type"].(string)
			claudeTools = append(claudeTools, Tool{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				InputSchema: InputSchema{
					Type:       t,
					Properties: params["properties"],
					Required:   params["required"],
				},
				CacheControl: tool.CacheControl,
			})
		}
	}

	claudeRequest := Request{
		Model:       meta.ActualModel,
		MaxTokens:   textRequest.MaxTokens,
		Temperature: textRequest.Temperature,
		TopP:        textRequest.TopP,
		TopK:        textRequest.TopK,
		Stream:      textRequest.Stream,
		Tools:       claudeTools,
	}

	if claudeRequest.MaxTokens == 0 {
		claudeRequest.MaxTokens = 4096
	}

	if onlyThinking.Thinking != nil {
		claudeRequest.Thinking = onlyThinking.Thinking
	} else if strings.Contains(meta.OriginModel, "think") {
		claudeRequest.Thinking = &Thinking{
			Type: "enabled",
		}
	}

	if claudeRequest.Thinking != nil {
		if claudeRequest.Thinking.BudgetTokens == 0 ||
			claudeRequest.Thinking.BudgetTokens >= claudeRequest.MaxTokens {
			claudeRequest.Thinking.BudgetTokens = claudeRequest.MaxTokens / 2
		}
		if claudeRequest.Thinking.BudgetTokens < 1024 {
			claudeRequest.Thinking.BudgetTokens = 1024
		}
		claudeRequest.Temperature = nil
	}

	if len(claudeTools) > 0 {
		claudeToolChoice := struct {
			Type string `json:"type"`
			Name string `json:"name,omitempty"`
		}{Type: "auto"} // default value https://docs.anthropic.com/en/docs/build-with-claude/tool-use#controlling-claudes-output
		if choice, ok := textRequest.ToolChoice.(map[string]any); ok {
			if function, ok := choice["function"].(map[string]any); ok {
				claudeToolChoice.Type = "tool"
				name, _ := function["name"].(string)
				claudeToolChoice.Name = name
			}
		} else if toolChoiceType, ok := textRequest.ToolChoice.(string); ok {
			if toolChoiceType == "any" {
				claudeToolChoice.Type = toolChoiceType
			}
		}
		claudeRequest.ToolChoice = claudeToolChoice
	}

	for _, message := range textRequest.Messages {
		if message.Role == "system" {
			claudeRequest.System = append(claudeRequest.System, Content{
				Type:         conetentTypeText,
				Text:         message.StringContent(),
				CacheControl: message.CacheControl,
			})
			continue
		}
		claudeMessage := Message{
			Role: message.Role,
		}
		var content Content
		content.CacheControl = message.CacheControl
		if message.IsStringContent() {
			content.Type = conetentTypeText
			content.Text = message.StringContent()
			if message.Role == "tool" {
				claudeMessage.Role = "user"
				content.Type = "tool_result"
				content.Content = content.Text
				content.Text = ""
				content.ToolUseID = message.ToolCallID
			}
			claudeMessage.Content = append(claudeMessage.Content, content)
			for _, toolCall := range message.ToolCalls {
				inputParam := make(map[string]any)
				_ = sonic.Unmarshal(conv.StringToBytes(toolCall.Function.Arguments), &inputParam)
				claudeMessage.Content = append(claudeMessage.Content, Content{
					Type:  toolUseType,
					ID:    toolCall.ID,
					Name:  toolCall.Function.Name,
					Input: inputParam,
				})
			}
			claudeRequest.Messages = append(claudeRequest.Messages, claudeMessage)
			continue
		}
		var contents []Content
		openaiContent := message.ParseContent()
		for _, part := range openaiContent {
			var content Content
			switch part.Type {
			case model.ContentTypeText:
				content.Type = conetentTypeText
				content.Text = part.Text
			case model.ContentTypeImageURL:
				content.Type = conetentTypeImage
				content.Source = &ImageSource{
					Type: "base64",
				}
				mimeType, data, err := image.GetImageFromURL(req.Context(), part.ImageURL.URL)
				if err != nil {
					return nil, err
				}
				content.Source.MediaType = mimeType
				content.Source.Data = data
			}
			contents = append(contents, content)
		}
		claudeMessage.Content = contents
		claudeRequest.Messages = append(claudeRequest.Messages, claudeMessage)
	}

	return &claudeRequest, nil
}

// https://docs.anthropic.com/claude/reference/messages-streaming
func StreamResponse2OpenAI(meta *meta.Meta, claudeResponse *StreamResponse) *model.ChatCompletionsStreamResponse {
	openaiResponse := model.ChatCompletionsStreamResponse{
		ID:      openai.ChatCompletionID(),
		Object:  model.ChatCompletionChunk,
		Created: time.Now().Unix(),
		Model:   meta.OriginModel,
	}
	var content string
	var thinking string
	var stopReason string
	tools := make([]*model.Tool, 0)

	switch claudeResponse.Type {
	case "message_start":
		return nil
	case "content_block_start":
		if claudeResponse.ContentBlock != nil {
			content = claudeResponse.ContentBlock.Text
			if claudeResponse.ContentBlock.Type == toolUseType {
				tools = append(tools, &model.Tool{
					ID:   claudeResponse.ContentBlock.ID,
					Type: "function",
					Function: model.Function{
						Name:      claudeResponse.ContentBlock.Name,
						Arguments: "",
					},
				})
			}
		}
	case "content_block_delta":
		if claudeResponse.Delta != nil {
			switch claudeResponse.Delta.Type {
			case "input_json_delta":
				tools = append(tools, &model.Tool{
					Function: model.Function{
						Arguments: claudeResponse.Delta.PartialJSON,
					},
				})
			case "thinking_delta":
				thinking = claudeResponse.Delta.Thinking
			case "signature_delta":
			default:
				content = claudeResponse.Delta.Text
			}
		}
	case "message_delta":
		if claudeResponse.Usage != nil {
			openaiResponse.Usage = &model.Usage{
				PromptTokens:     claudeResponse.Usage.InputTokens + claudeResponse.Usage.CacheReadInputTokens + claudeResponse.Usage.CacheCreationInputTokens,
				CompletionTokens: claudeResponse.Usage.OutputTokens,
				PromptTokensDetails: &model.PromptTokensDetails{
					CachedTokens:        claudeResponse.Usage.CacheReadInputTokens,
					CacheCreationTokens: claudeResponse.Usage.CacheCreationInputTokens,
				},
			}
			openaiResponse.Usage.TotalTokens = openaiResponse.Usage.PromptTokens + openaiResponse.Usage.CompletionTokens
		}
		if claudeResponse.Delta != nil && claudeResponse.Delta.StopReason != nil {
			stopReason = *claudeResponse.Delta.StopReason
		}
	}

	var choice model.ChatCompletionsStreamResponseChoice
	choice.Delta.Content = content
	choice.Delta.ReasoningContent = thinking

	if len(tools) > 0 {
		choice.Delta.Content = nil // compatible with other OpenAI derivative applications, like LobeOpenAICompatibleFactory ...
		choice.Delta.ToolCalls = tools
	}
	choice.Delta.Role = "assistant"
	finishReason := stopReasonClaude2OpenAI(&stopReason)
	if finishReason != "null" {
		choice.FinishReason = &finishReason
	}
	openaiResponse.Choices = []*model.ChatCompletionsStreamResponseChoice{&choice}

	return &openaiResponse
}

func Response2OpenAI(meta *meta.Meta, claudeResponse *Response) *model.TextResponse {
	var content string
	var thinking string
	for _, v := range claudeResponse.Content {
		switch v.Type {
		case conetentTypeText:
			content = v.Text
		case conetentTypeThinking:
			thinking = v.Thinking
		}
	}
	tools := make([]*model.Tool, 0)
	for _, v := range claudeResponse.Content {
		if v.Type == toolUseType {
			args, _ := sonic.Marshal(v.Input)
			tools = append(tools, &model.Tool{
				ID:   v.ID,
				Type: "function", // compatible with other OpenAI derivative applications
				Function: model.Function{
					Name:      v.Name,
					Arguments: conv.BytesToString(args),
				},
			})
		}
	}
	choice := model.TextResponseChoice{
		Index: 0,
		Message: model.Message{
			Role:             "assistant",
			Content:          content,
			ReasoningContent: thinking,
			Name:             nil,
			ToolCalls:        tools,
		},
		FinishReason: stopReasonClaude2OpenAI(claudeResponse.StopReason),
	}

	fullTextResponse := model.TextResponse{
		ID:      openai.ChatCompletionID(),
		Model:   meta.OriginModel,
		Object:  model.ChatCompletion,
		Created: time.Now().Unix(),
		Choices: []*model.TextResponseChoice{&choice},
		Usage: model.Usage{
			PromptTokens:     claudeResponse.Usage.InputTokens + claudeResponse.Usage.CacheReadInputTokens + claudeResponse.Usage.CacheCreationInputTokens,
			CompletionTokens: claudeResponse.Usage.OutputTokens,
			PromptTokensDetails: &model.PromptTokensDetails{
				CachedTokens:        claudeResponse.Usage.CacheReadInputTokens,
				CacheCreationTokens: claudeResponse.Usage.CacheCreationInputTokens,
			},
		},
	}
	fullTextResponse.Usage.TotalTokens = fullTextResponse.Usage.PromptTokens + fullTextResponse.Usage.CompletionTokens
	return &fullTextResponse
}

func StreamHandler(m *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorHandler(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := slices.Index(data, '\n'); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	common.SetEventStreamHeaders(c)

	responseText := strings.Builder{}

	var usage model.Usage
	var lastToolCallChoice *model.ChatCompletionsStreamResponseChoice
	var usageWrited bool
	for scanner.Scan() {
		data := scanner.Bytes()
		if len(data) < 6 || conv.BytesToString(data[:6]) != "data: " {
			continue
		}
		data = data[6:]
		if conv.BytesToString(data) == "[DONE]" {
			break
		}
		var claudeResponse StreamResponse
		err := sonic.Unmarshal(data, &claudeResponse)
		if err != nil {
			log.Error("error unmarshalling stream response: " + err.Error())
			continue
		}
		response := StreamResponse2OpenAI(m, &claudeResponse)
		if response == nil {
			continue
		}
		if response.Usage != nil {
			if response.Usage.PromptTokens == 0 {
				response.Usage.PromptTokens = m.InputTokens
				response.Usage.TotalTokens += m.InputTokens
			}
			usage = *response.Usage
			usageWrited = true
			responseText.Reset()
		} else if !usageWrited {
			for _, choice := range response.Choices {
				responseText.WriteString(choice.Delta.StringContent())
			}
		}

		if lastToolCallChoice != nil && len(lastToolCallChoice.Delta.ToolCalls) > 0 {
			lastArgs := &lastToolCallChoice.Delta.ToolCalls[len(lastToolCallChoice.Delta.ToolCalls)-1].Function
			if len(lastArgs.Arguments) == 0 { // compatible with OpenAI sending an empty object `{}` when no arguments.
				lastArgs.Arguments = "{}"
				response.Choices[len(response.Choices)-1].Delta.Content = nil
				response.Choices[len(response.Choices)-1].Delta.ToolCalls = lastToolCallChoice.Delta.ToolCalls
			}
		}

		for _, choice := range response.Choices {
			if len(choice.Delta.ToolCalls) > 0 {
				lastToolCallChoice = choice
			}
		}
		_ = render.ObjectData(c, response)
	}
	if err := scanner.Err(); err != nil {
		log.Error("error reading stream: " + err.Error())
	}

	if !usageWrited {
		usage.PromptTokens = m.InputTokens
		usage.CompletionTokens = openai.CountTokenText(responseText.String(), m.OriginModel)
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		_ = render.ObjectData(c, &model.ChatCompletionsStreamResponse{
			ID:      openai.ChatCompletionID(),
			Model:   m.OriginModel,
			Object:  model.ChatCompletionChunk,
			Created: time.Now().Unix(),
			Choices: []*model.ChatCompletionsStreamResponseChoice{},
			Usage:   &usage,
		})
	}
	render.Done(c)
	return &usage, nil
}

func Handler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorHandler(resp)
	}

	defer resp.Body.Close()

	var claudeResponse Response
	err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&claudeResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	fullTextResponse := Response2OpenAI(meta, &claudeResponse)
	jsonResponse, err := sonic.Marshal(fullTextResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(jsonResponse)
	return &fullTextResponse.Usage, nil
}

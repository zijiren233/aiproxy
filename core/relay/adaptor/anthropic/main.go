package anthropic

import (
	"bufio"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/conv"
	"github.com/labring/aiproxy/core/common/render"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func StreamHandler(m *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, OpenAIErrorHandler(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	scanner := bufio.NewScanner(resp.Body)
	buf := openai.GetScannerBuffer()
	defer openai.PutScannerBuffer(buf)
	scanner.Buffer(*buf, cap(*buf))

	common.SetEventStreamHeaders(c)

	responseText := strings.Builder{}

	var usage *relaymodel.Usage

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

		switch {
		case response.Usage != nil:
			if usage == nil {
				usage = &relaymodel.Usage{}
			}
			usage.Add(response.Usage)
			if usage.PromptTokens == 0 {
				usage.PromptTokens = m.InputTokens
				usage.TotalTokens += m.InputTokens
			}
			response.Usage = usage
			responseText.Reset()
		case usage == nil:
			for _, choice := range response.Choices {
				responseText.WriteString(choice.Delta.StringContent())
			}
		default:
			response.Usage = usage
		}

		render.StringData(c, conv.BytesToString(data))
	}

	if err := scanner.Err(); err != nil {
		log.Error("error reading stream: " + err.Error())
	}

	if usage == nil {
		usage = &relaymodel.Usage{
			PromptTokens:     m.InputTokens,
			CompletionTokens: openai.CountTokenText(responseText.String(), m.OriginModel),
			TotalTokens:      m.InputTokens + openai.CountTokenText(responseText.String(), m.OriginModel),
		}
	}

	render.Done(c)

	return usage.ToModelUsage(), nil
}

func Handler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, OpenAIErrorHandler(resp)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "read_response_failed", http.StatusInternalServerError)
	}

	var claudeResponse Response
	err = sonic.Unmarshal(respBody, &claudeResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	fullTextResponse := Response2OpenAI(meta, &claudeResponse)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(respBody)
	return fullTextResponse.Usage.ToModelUsage(), nil
}

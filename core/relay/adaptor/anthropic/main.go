package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/conv"
	"github.com/labring/aiproxy/core/common/image"
	"github.com/labring/aiproxy/core/common/render"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func ConvertRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	// Parse request body into AST node
	node, err := common.UnmarshalBody2Node(req)
	if err != nil {
		return "", nil, nil, err
	}

	// Set the actual model in the request
	node.Set("model", ast.NewString(meta.ActualModel))

	// Process image content if present
	err = ConvertImage2Base64(req.Context(), &node)
	if err != nil {
		return "", nil, nil, err
	}

	// Serialize the modified node
	newBody, err := node.MarshalJSON()
	if err != nil {
		return "", nil, nil, err
	}

	return http.MethodPost, nil, bytes.NewReader(newBody), nil
}

// ConvertImage2Base64 handles converting image URLs to base64 encoded data
func ConvertImage2Base64(ctx context.Context, node *ast.Node) error {
	messagesNode := node.Get("messages")
	if messagesNode == nil || messagesNode.TypeSafe() != ast.V_ARRAY {
		return nil
	}

	return messagesNode.ForEach(func(path ast.Sequence, msgNode *ast.Node) bool {
		contentNode := msgNode.Get("content")
		if contentNode == nil || contentNode.TypeSafe() != ast.V_ARRAY {
			return true
		}

		err := contentNode.ForEach(func(path ast.Sequence, contentItem *ast.Node) bool {
			contentType, err := contentItem.Get("type").String()
			if err == nil && contentType == conetentTypeImage {
				convertImageURLToBase64(ctx, contentItem)
			}
			return true
		})
		return err == nil
	})
}

// convertImageURLToBase64 converts an image URL to base64 encoded data
func convertImageURLToBase64(ctx context.Context, contentItem *ast.Node) {
	sourceNode := contentItem.Get("source")
	if sourceNode == nil {
		return
	}

	imageType, err := sourceNode.Get("type").String()
	if err != nil || imageType != "url" {
		return
	}

	url, err := sourceNode.Get("url").String()
	if err != nil {
		return
	}

	mimeType, data, err := image.GetImageFromURL(ctx, url)
	if err != nil {
		return
	}

	// Update the source node with base64 data
	sourceNode.Set("type", ast.NewString("base64"))
	sourceNode.Set("media_type", ast.NewString(mimeType))
	sourceNode.Set("data", ast.NewString(data))
	sourceNode.Unset("url")
}

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

	responseText := strings.Builder{}

	var usage *relaymodel.Usage
	var writed bool

	for scanner.Scan() {
		data := scanner.Bytes()
		if len(data) < 6 || conv.BytesToString(data[:6]) != "data: " {
			continue
		}
		data = data[6:]

		if conv.BytesToString(data) == "[DONE]" {
			break
		}

		response, err := StreamResponse2OpenAI(m, data)
		if err != nil {
			if writed {
				log.Errorf("response error: %+v", err)
			} else {
				return usage.ToModelUsage(), err
			}
		}
		if response != nil {
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
		}

		render.StringData(c, conv.BytesToString(data))
		writed = true
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

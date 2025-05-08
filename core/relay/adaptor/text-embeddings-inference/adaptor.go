package textembeddingsinference

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

// text-embeddings-inference adaptor supports rerank and embeddings models deployed by https://github.com/huggingface/text-embeddings-inference
type Adaptor struct {
	openai.Adaptor
}

// base url for text-embeddings-inference, fake
const baseURL = "https://api.text-embeddings.net/v1"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) GetModelList() []*model.ModelConfig {
	return ModelList
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	switch meta.Mode {
	case mode.Rerank:
		return a.GetBaseURL() + "/rerank", nil
	case mode.Embeddings:
		return a.GetBaseURL() + "/embeddings", nil
	default:
		return "", fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

// text-embeddings-inference api see https://huggingface.github.io/text-embeddings-inference/#/Text%20Embeddings%20Inference/rerank

func (a *Adaptor) SetupRequestHeader(meta *meta.Meta, _ *gin.Context, req *http.Request) error {
	switch meta.Mode {
	case mode.Rerank:
		req.Header.Set("Content-Type", "application/json")
	case mode.Embeddings:
		req.Header.Set("Content-Type", "application/json")
	default:
		return fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
	return nil
}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	switch meta.Mode {
	case mode.Rerank:
		// Parse request body into AST node
		node, err := common.UnmarshalBody2Node(req)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to parse request body: %w", err)
		}

		// Get the documents array and rename it to texts
		documentsNode := node.Get("documents")
		if !documentsNode.Exists() {
			return "", nil, nil, fmt.Errorf("documents field not found")
		}

		// Get the raw documents value
		documentsValue, err := documentsNode.Raw()
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to get documents value: %w", err)
		}

		// Set the texts field with the documents value
		_, err = node.Set("texts", ast.NewString(documentsValue))
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to set texts field: %w", err)
		}

		// Remove the documents field
		_, err = node.Unset("documents")
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to remove documents field: %w", err)
		}

		// Convert back to JSON
		jsonData, err := node.MarshalJSON()
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		return "", nil, bytes.NewReader(jsonData), nil
	case mode.Embeddings:
		// Handle embeddings request
		return "", nil, req.Body, nil
	default:
		return "", nil, nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) DoRequest(meta *meta.Meta, c *gin.Context, req *http.Request) (*http.Response, error) {
	return utils.DoRequest(req)
}

func (a *Adaptor) DoResponse(_ *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return nil, openai.ErrorWrapperWithMessage(
			fmt.Sprintf("request failed with status %d", resp.StatusCode),
			"upstream_error",
			resp.StatusCode,
		)
	}

	// Write response back to client
	c.Header("Content-Type", "application/json")
	c.Status(resp.StatusCode)
	if _, err := c.Writer.Write(body); err != nil {
		return nil, openai.ErrorWrapper(err, "write_response_failed", http.StatusInternalServerError)
	}

	// For text-embeddings-inference, we don't need to track usage
	// TODO: add usage tracking
	return &model.Usage{}, nil
}

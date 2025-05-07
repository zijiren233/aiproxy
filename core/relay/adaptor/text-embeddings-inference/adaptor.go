package textembeddingsinference

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
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
		// Read the request body
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to read request body: %w", err)
		}

		// Parse the request body
		var requestBody map[string]interface{}
		if err := json.Unmarshal(body, &requestBody); err != nil {
			return "", nil, nil, fmt.Errorf("failed to parse request body: %w", err)
		}

		// Create the rerank request, setting model is not supported
		rerankRequest := map[string]interface{}{
			// "model": meta.OriginModel,
			"query": requestBody["query"],
			"texts": requestBody["documents"],
		}

		// Convert back to JSON
		rerankBody, err := json.Marshal(rerankRequest)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to marshal rerank request: %w", err)
		}

		return "", req.Header, bytes.NewReader(rerankBody), nil

	case mode.Embeddings:
		// Handle embeddings request
		return "", req.Header, req.Body, nil

	default:
		return "", nil, nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) DoRequest(meta *meta.Meta, c *gin.Context, req *http.Request) (*http.Response, error) {
	// Get the request URL
	url, err := a.GetRequestURL(meta)
	if err != nil {
		return nil, err
	}

	// Create a new request with context
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, url, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range req.Header {
		newReq.Header[k] = v
	}

	// Setup headers
	if err := a.SetupRequestHeader(meta, c, newReq); err != nil {
		return nil, err
	}

	// Convert request if needed
	_, headers, body, err := a.ConvertRequest(meta, newReq)
	if err != nil {
		return nil, err
	}

	// Update request with converted data
	for k, v := range headers {
		newReq.Header[k] = v
	}
	if body != nil {
		newReq.Body = io.NopCloser(body)
	}

	// Send request using utils
	return utils.DoRequest(newReq)
}

func (a *Adaptor) DoResponse(_ *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &relaymodel.ErrorWithStatusCode{
			Error: relaymodel.Error{
				Message: fmt.Sprintf("failed to read response body: %v", err),
			},
			StatusCode: resp.StatusCode,
		}
	}

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return nil, &relaymodel.ErrorWithStatusCode{
			Error: relaymodel.Error{
				Message: fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)),
			},
			StatusCode: resp.StatusCode,
		}
	}

	// Write response back to client
	c.Header("Content-Type", "application/json")
	c.Status(resp.StatusCode)
	if _, err := c.Writer.Write(body); err != nil {
		return nil, &relaymodel.ErrorWithStatusCode{
			Error: relaymodel.Error{
				Message: fmt.Sprintf("failed to write response: %v", err),
			},
			StatusCode: http.StatusInternalServerError,
		}
	}

	// For text-embeddings-inference, we don't need to track usage
	return &model.Usage{}, nil
}

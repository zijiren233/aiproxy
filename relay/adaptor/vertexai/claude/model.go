package vertexai

import "github.com/labring/aiproxy/relay/adaptor/anthropic"

type Request struct {
	AnthropicVersion string `json:"anthropic_version"`
	*anthropic.Request
}

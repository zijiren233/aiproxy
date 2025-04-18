package model

type AnthropicMessageRequest struct {
	Model    string     `json:"model,omitempty"`
	Messages []*Message `json:"messages,omitempty"`
}

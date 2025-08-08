package model

import (
	"github.com/labring/aiproxy/core/model"
)

// ResponseStatus represents the status of a response
type ResponseStatus string

const (
	ResponseStatusInProgress ResponseStatus = "in_progress"
	ResponseStatusCompleted  ResponseStatus = "completed"
	ResponseStatusFailed     ResponseStatus = "failed"
	ResponseStatusIncomplete ResponseStatus = "incomplete"
	ResponseStatusCancelled  ResponseStatus = "cancelled"
)

// ResponseError represents an error in a response
type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// IncompleteDetails represents details about why a response is incomplete
type IncompleteDetails struct {
	Reason string `json:"reason"`
}

// ResponseReasoning represents reasoning information
type ResponseReasoning struct {
	Effort  *string `json:"effort"`
	Summary *string `json:"summary"`
}

// ResponseTextFormat represents text format configuration
type ResponseTextFormat struct {
	Type string `json:"type"`
}

// ResponseText represents text configuration
type ResponseText struct {
	Format ResponseTextFormat `json:"format"`
}

// OutputContent represents content in an output item
type OutputContent struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Annotations []any  `json:"annotations,omitempty"`
}

// OutputItem represents an output item in a response
type OutputItem struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Status  ResponseStatus  `json:"status,omitempty"`
	Role    string          `json:"role"`
	Content []OutputContent `json:"content"`
}

// InputContent represents content in an input item
type InputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// InputItem represents an input item
type InputItem struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Role    string         `json:"role"`
	Content []InputContent `json:"content"`
}

// ResponseUsageDetails represents detailed token usage information
type ResponseUsageDetails struct {
	CachedTokens    int64 `json:"cached_tokens,omitempty"`
	ReasoningTokens int64 `json:"reasoning_tokens,omitempty"`
}

// ResponseUsage represents usage information for a response
type ResponseUsage struct {
	InputTokens         int64                 `json:"input_tokens"`
	OutputTokens        int64                 `json:"output_tokens"`
	TotalTokens         int64                 `json:"total_tokens"`
	InputTokensDetails  *ResponseUsageDetails `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *ResponseUsageDetails `json:"output_tokens_details,omitempty"`
}

// Response represents an OpenAI response object
type Response struct {
	ID                 string             `json:"id"`
	Object             string             `json:"object"`
	CreatedAt          int64              `json:"created_at"`
	Status             ResponseStatus     `json:"status"`
	Error              *ResponseError     `json:"error"`
	IncompleteDetails  *IncompleteDetails `json:"incomplete_details"`
	Instructions       *string            `json:"instructions"`
	MaxOutputTokens    *int               `json:"max_output_tokens"`
	Model              string             `json:"model"`
	Output             []OutputItem       `json:"output"`
	ParallelToolCalls  bool               `json:"parallel_tool_calls"`
	PreviousResponseID *string            `json:"previous_response_id"`
	Reasoning          ResponseReasoning  `json:"reasoning"`
	Store              bool               `json:"store"`
	Temperature        float64            `json:"temperature"`
	Text               ResponseText       `json:"text"`
	ToolChoice         any                `json:"tool_choice"`
	Tools              []Tool             `json:"tools"`
	TopP               float64            `json:"top_p"`
	Truncation         string             `json:"truncation"`
	Usage              *ResponseUsage     `json:"usage"`
	User               *string            `json:"user"`
	Metadata           map[string]any     `json:"metadata"`
}

// CreateResponseRequest represents a request to create a response
type CreateResponseRequest struct {
	Model              string         `json:"model"`
	Messages           []Message      `json:"messages"`
	Instructions       *string        `json:"instructions,omitempty"`
	MaxOutputTokens    *int           `json:"max_output_tokens,omitempty"`
	ParallelToolCalls  *bool          `json:"parallel_tool_calls,omitempty"`
	PreviousResponseID *string        `json:"previous_response_id,omitempty"`
	Store              *bool          `json:"store,omitempty"`
	Temperature        *float64       `json:"temperature,omitempty"`
	ToolChoice         any            `json:"tool_choice,omitempty"`
	Tools              []Tool         `json:"tools,omitempty"`
	TopP               *float64       `json:"top_p,omitempty"`
	Truncation         *string        `json:"truncation,omitempty"`
	User               *string        `json:"user,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	Stream             bool           `json:"stream,omitempty"`
}

// InputItemList represents a list of input items
type InputItemList struct {
	Object  string      `json:"object"`
	Data    []InputItem `json:"data"`
	FirstID string      `json:"first_id"`
	LastID  string      `json:"last_id"`
	HasMore bool        `json:"has_more"`
}

// ResponseStreamEvent represents a server-sent event for response streaming
type ResponseStreamEvent struct {
	Type           string      `json:"type"`
	Response       *Response   `json:"response,omitempty"`
	OutputIndex    *int        `json:"output_index,omitempty"`
	Item           *OutputItem `json:"item,omitempty"`
	SequenceNumber int         `json:"sequence_number,omitempty"`
}

func (u *ResponseUsage) ToModelUsage() model.Usage {
	usage := model.Usage{
		InputTokens:  model.ZeroNullInt64(u.InputTokens),
		OutputTokens: model.ZeroNullInt64(u.OutputTokens),
		TotalTokens:  model.ZeroNullInt64(u.TotalTokens),
	}

	if u.InputTokensDetails != nil {
		usage.CachedTokens = model.ZeroNullInt64(u.InputTokensDetails.CachedTokens)
	}

	if u.OutputTokensDetails != nil {
		usage.ReasoningTokens = model.ZeroNullInt64(u.OutputTokensDetails.ReasoningTokens)
	}

	return usage
}

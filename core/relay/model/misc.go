package model

import (
	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/common/conv"
	"github.com/labring/aiproxy/core/model"
)

type Usage struct {
	PromptTokens     int64 `json:"prompt_tokens,omitempty"`
	CompletionTokens int64 `json:"completion_tokens,omitempty"`
	TotalTokens      int64 `json:"total_tokens"`

	WebSearchCount int64 `json:"web_search_count,omitempty"`

	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

func (u *Usage) ToModelUsage() *model.Usage {
	if u == nil {
		return nil
	}
	usage := &model.Usage{
		InputTokens:    u.PromptTokens,
		OutputTokens:   u.CompletionTokens,
		TotalTokens:    u.TotalTokens,
		WebSearchCount: u.WebSearchCount,
	}
	if u.PromptTokensDetails != nil {
		usage.CachedTokens = u.PromptTokensDetails.CachedTokens
		usage.CacheCreationTokens = u.PromptTokensDetails.CacheCreationTokens
	}
	return usage
}

func (u *Usage) Add(other *Usage) {
	if other == nil {
		return
	}
	u.PromptTokens += other.PromptTokens
	u.CompletionTokens += other.CompletionTokens
	u.TotalTokens += other.TotalTokens
	if other.PromptTokensDetails != nil {
		if u.PromptTokensDetails == nil {
			u.PromptTokensDetails = &PromptTokensDetails{}
		}
		u.PromptTokensDetails.Add(other.PromptTokensDetails)
	}
}

type PromptTokensDetails struct {
	CachedTokens        int64 `json:"cached_tokens"`
	AudioTokens         int64 `json:"audio_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens,omitempty"`
}

func (d *PromptTokensDetails) Add(other *PromptTokensDetails) {
	if other == nil {
		return
	}
	d.CachedTokens += other.CachedTokens
	d.AudioTokens += other.AudioTokens
	d.CacheCreationTokens += other.CacheCreationTokens
}

type CompletionTokensDetails struct {
	ReasoningTokens          int64 `json:"reasoning_tokens"`
	AudioTokens              int64 `json:"audio_tokens"`
	AcceptedPredictionTokens int64 `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int64 `json:"rejected_prediction_tokens"`
}

type Error struct {
	Code    any    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
	Param   string `json:"param,omitempty"`
}

func (e *Error) IsEmpty() bool {
	return e == nil || (e.Code == nil && e.Message == "" && e.Type == "" && e.Param == "")
}

func (e *Error) JSONOrEmpty() string {
	if e.IsEmpty() {
		return ""
	}
	jsonBuf, err := sonic.Marshal(e)
	if err != nil {
		return ""
	}
	return conv.BytesToString(jsonBuf)
}

type ErrorWithStatusCode struct {
	Error      Error `json:"error,omitempty"`
	StatusCode int   `json:"-"`
}

func (e *ErrorWithStatusCode) JSONOrEmpty() string {
	if e.StatusCode == 0 && e.Error.IsEmpty() {
		return ""
	}
	jsonBuf, err := sonic.MarshalString(e)
	if err != nil {
		return ""
	}
	return jsonBuf
}

package geminiopenai

import (
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/adaptor/gemini"
	"github.com/labring/aiproxy/relay/adaptor/openai"
)

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) GetModelList() []*model.ModelConfig {
	return gemini.ModelList
}

func (a *Adaptor) GetChannelName() string {
	return "google gemini (openai)"
}

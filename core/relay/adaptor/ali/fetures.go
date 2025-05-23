package ali

import "github.com/labring/aiproxy/core/relay/adaptor"

var _ adaptor.Features = (*Adaptor)(nil)

func (a *Adaptor) Features() []string {
	return []string{
		"OpenAI compatibility",
		"Network search metering support",
		"Rerank support: https://help.aliyun.com/zh/model-studio/text-rerank-api",
		"STT support: https://help.aliyun.com/zh/model-studio/sambert-speech-synthesis/",
	}
}

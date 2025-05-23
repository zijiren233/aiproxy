package gemini

import "github.com/labring/aiproxy/core/relay/adaptor"

var _ adaptor.Features = (*Adaptor)(nil)

func (a *Adaptor) Features() []string {
	return []string{
		"https://ai.google.dev",
		"Chat、Embeddings、Image generation Support",
	}
}

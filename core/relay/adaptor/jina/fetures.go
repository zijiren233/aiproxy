package jina

import "github.com/labring/aiproxy/core/relay/adaptor"

var _ adaptor.Features = (*Adaptor)(nil)

func (a *Adaptor) Features() []string {
	return []string{
		"https://jina.ai",
		"Embeddings、Rerank Support",
	}
}

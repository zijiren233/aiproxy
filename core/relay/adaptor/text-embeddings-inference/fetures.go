package textembeddingsinference

import "github.com/labring/aiproxy/core/relay/adaptor"

var _ adaptor.Features = (*Adaptor)(nil)

func (a *Adaptor) Features() []string {
	return []string{
		"https://github.com/huggingface/text-embeddings-inference",
		"Embeddings、Rerank Support",
	}
}

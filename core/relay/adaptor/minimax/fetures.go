package minimax

import "github.com/labring/aiproxy/core/relay/adaptor"

var _ adaptor.Features = (*Adaptor)(nil)

func (a *Adaptor) Features() []string {
	return []string{
		"Chat、Embeddings、TTS(need group id) Support",
	}
}

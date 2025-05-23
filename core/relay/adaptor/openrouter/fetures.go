package openrouter

import "github.com/labring/aiproxy/core/relay/adaptor"

var _ adaptor.Features = (*Adaptor)(nil)

func (a *Adaptor) Features() []string {
	return []string{
		"The `reasoning` field is converted to `reasoning_content`",
	}
}

package azure2

import "github.com/labring/aiproxy/core/relay/adaptor"

var _ adaptor.Features = (*Adaptor)(nil)

func (a *Adaptor) Features() []string {
	return []string{
		"Model name support contain '.'",
	}
}

package sangforaicp

import (
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
)

var _ adaptor.Adaptor = (*Adaptor)(nil)

type Adaptor struct {
	openai.Adaptor
}

func (a *Adaptor) DefaultBaseURL() string {
	return ""
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{}
}

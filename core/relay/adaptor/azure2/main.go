package azure2

import (
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/azure"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
)

type Adaptor struct {
	azure.Adaptor
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta, _ adaptor.Store) (*adaptor.RequestURL, error) {
	return azure.GetRequestURL(meta, false)
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Features: []string{
			"Model name support contain '.'",
		},
		KeyHelp: "key or key|api-version",
		Models:  openai.ModelList,
	}
}

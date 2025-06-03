package azure2

import (
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/azure"
	"github.com/labring/aiproxy/core/relay/meta"
)

type Adaptor struct {
	azure.Adaptor
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta, _ adaptor.Store) (string, error) {
	return azure.GetRequestURL(meta, false)
}

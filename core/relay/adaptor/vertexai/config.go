package vertexai

import (
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/gemini"
)

func (a *Adaptor) ConfigTemplates() adaptor.ConfigTemplates {
	return gemini.ConfigTemplates
}

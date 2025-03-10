package doc2x

import (
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/relaymode"
)

var ModelList = []*model.ModelConfig{
	{
		Model:      "pdf",
		Type:       relaymode.ParsePdf,
		Owner:      model.ModelOwnerDoc2x,
		InputPrice: 20,
		RPM:        10,
	},
}

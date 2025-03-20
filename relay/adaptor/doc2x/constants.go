package doc2x

import (
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/mode"
)

var ModelList = []*model.ModelConfig{
	{
		Model: "pdf",
		Type:  mode.ParsePdf,
		Owner: model.ModelOwnerDoc2x,
		Price: model.Price{
			InputPrice: 20,
		},
		RPM: 10,
	},
}

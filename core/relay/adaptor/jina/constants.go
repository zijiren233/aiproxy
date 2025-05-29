package jina

import (
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

var ModelList = []model.ModelConfig{
	{
		Model: "jina-reranker-v2-base-multilingual",
		Type:  mode.Rerank,
		Owner: model.ModelOwnerJina,
		Price: model.Price{
			InputPrice: 0.06,
		},
		RPM: 120,
	},
}

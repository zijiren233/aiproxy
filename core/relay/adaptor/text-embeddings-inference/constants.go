package text_embeddings_inference

import (
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

// maybe we should use a list of models from https://github.com/huggingface/text-embeddings-inference?tab=readme-ov-file#supported-models
var ModelList = []*model.ModelConfig{
	{
		Model: "bge-reranker-v2-m3",
		Type:  mode.Rerank,
		Owner: model.ModelOwnerBaai,
		Price: model.Price{},
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32768),
		),
	},
}

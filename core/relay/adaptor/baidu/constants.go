package baidu

import (
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

var ModelList = []model.ModelConfig{
	{
		Model: "BLOOMZ-7B",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.004,
			OutputPrice: 0.004,
		},
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(4800),
		),
	},

	{
		Model: "Embedding-V1",
		Type:  mode.Embeddings,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice: 0.0005,
		},
		RPM: 1200,
	},
	{
		Model: "bge-large-zh",
		Type:  mode.Embeddings,
		Owner: model.ModelOwnerBAAI,
		Price: model.Price{
			InputPrice: 0.0005,
		},
		RPM: 1200,
	},
	{
		Model: "bge-large-en",
		Type:  mode.Embeddings,
		Owner: model.ModelOwnerBAAI,
		Price: model.Price{
			InputPrice: 0.0005,
		},
		RPM: 1200,
	},
	{
		Model: "tao-8k",
		Type:  mode.Embeddings,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice: 0.0005,
		},
		RPM: 1200,
	},

	{
		Model: "bce-reranker-base_v1",
		Type:  mode.Rerank,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice: 0.0005,
		},
		RPM: 1200,
	},

	{
		Model: "Stable-Diffusion-XL",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerStabilityAI,
		ImagePrices: map[string]float64{
			"768x768":   0.06,
			"576x1024":  0.06,
			"1024x576":  0.06,
			"768x1024":  0.08,
			"1024x768":  0.08,
			"1024x1024": 0.08,
			"1536x1536": 0.12,
			"1152x2048": 0.12,
			"2048x1152": 0.12,
			"1536x2048": 0.16,
			"2048x1536": 0.16,
			"2048x2048": 0.16,
		},
	},
}

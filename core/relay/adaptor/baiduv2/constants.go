package baiduv2

import (
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/Fm2vrveyu

var ModelList = []model.ModelConfig{
	{
		Model: "ERNIE-4.0-8K-Latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.03,
			OutputPrice: 0.09,
		},
		RPM: 120,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(5120),
			model.WithModelConfigMaxInputTokens(5120),
			model.WithModelConfigMaxOutputTokens(2048),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-4.0-8K-Preview",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.03,
			OutputPrice: 0.09,
		},
		RPM: 300,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(5120),
			model.WithModelConfigMaxInputTokens(5120),
			model.WithModelConfigMaxOutputTokens(2048),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-4.0-8K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.03,
			OutputPrice: 0.09,
		},
		RPM: 10000,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(5120),
			model.WithModelConfigMaxInputTokens(5120),
			model.WithModelConfigMaxOutputTokens(2048),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-4.0-Turbo-8K-Latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.02,
			OutputPrice: 0.06,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(6144),
			model.WithModelConfigMaxInputTokens(6144),
			model.WithModelConfigMaxOutputTokens(2048),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-4.0-Turbo-8K-Preview",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.02,
			OutputPrice: 0.06,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(6144),
			model.WithModelConfigMaxInputTokens(6144),
			model.WithModelConfigMaxOutputTokens(2048),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-4.0-Turbo-8K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.02,
			OutputPrice: 0.06,
		},
		RPM: 10000,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(6144),
			model.WithModelConfigMaxInputTokens(6144),
			model.WithModelConfigMaxOutputTokens(2048),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-4.0-Turbo-128K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.02,
			OutputPrice: 0.06,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(126976),
			model.WithModelConfigMaxInputTokens(126976),
			model.WithModelConfigMaxOutputTokens(4096),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-3.5-8K-Preview",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0008,
			OutputPrice: 0.002,
		},
		RPM: 300,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(5120),
			model.WithModelConfigMaxInputTokens(5120),
			model.WithModelConfigMaxOutputTokens(2048),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-3.5-8K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0008,
			OutputPrice: 0.002,
		},
		RPM: 10000,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(5120),
			model.WithModelConfigMaxInputTokens(5120),
			model.WithModelConfigMaxOutputTokens(2048),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-3.5-128K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0008,
			OutputPrice: 0.002,
		},
		RPM: 5000,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(126976),
			model.WithModelConfigMaxInputTokens(126976),
			model.WithModelConfigMaxOutputTokens(4096),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-Speed-8K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0001,
			OutputPrice: 0.0001,
		},
		RPM: 500,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(7168),
			model.WithModelConfigMaxInputTokens(7168),
			model.WithModelConfigMaxOutputTokens(2048),
		),
	},
	{
		Model: "ERNIE-Speed-128K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0001,
			OutputPrice: 0.0001,
		},
		RPM: 500,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(126976),
			model.WithModelConfigMaxInputTokens(126976),
			model.WithModelConfigMaxOutputTokens(4096),
		),
	},
	{
		Model: "ERNIE-Speed-Pro-128K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0003,
			OutputPrice: 0.0006,
		},
		RPM: 10000,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(126976),
			model.WithModelConfigMaxInputTokens(126976),
			model.WithModelConfigMaxOutputTokens(4096),
		),
	},
	{
		Model: "ERNIE-Lite-8K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0001,
			OutputPrice: 0.0001,
		},
		RPM: 500,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(6144),
			model.WithModelConfigMaxInputTokens(6144),
			model.WithModelConfigMaxOutputTokens(2048),
		),
	},
	{
		Model: "ERNIE-Lite-Pro-128K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0002,
			OutputPrice: 0.0004,
		},
		RPM: 10000,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(126976),
			model.WithModelConfigMaxInputTokens(126976),
			model.WithModelConfigMaxOutputTokens(4096),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "ERNIE-Tiny-8K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0001,
			OutputPrice: 0.0001,
		},
		RPM: 10000,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(6144),
			model.WithModelConfigMaxInputTokens(6144),
			model.WithModelConfigMaxOutputTokens(2048),
		),
	},
	{
		Model: "ERNIE-Character-8K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0003,
			OutputPrice: 0.0006,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(6144),
			model.WithModelConfigMaxInputTokens(6144),
			model.WithModelConfigMaxOutputTokens(2048),
		),
	},
	{
		Model: "ERNIE-Character-Fiction-8K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.0003,
			OutputPrice: 0.0006,
		},
		RPM: 300,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(5120),
			model.WithModelConfigMaxInputTokens(5120),
			model.WithModelConfigMaxOutputTokens(2048),
		),
	},
	{
		Model: "ERNIE-Novel-8K",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerBaidu,
		Price: model.Price{
			InputPrice:  0.04,
			OutputPrice: 0.12,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(6144),
			model.WithModelConfigMaxInputTokens(6144),
			model.WithModelConfigMaxOutputTokens(2048),
		),
	},

	{
		Model: "DeepSeek-V3",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerDeepSeek,
		Price: model.Price{
			InputPrice:  0.0008,
			OutputPrice: 0.0016,
		},
		RPM: 1000,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(64000),
			model.WithModelConfigMaxOutputTokens(8192),
		),
	},
	{
		Model: "DeepSeek-R1",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerDeepSeek,
		Price: model.Price{
			InputPrice:  0.002,
			OutputPrice: 0.008,
		},
		RPM: 1000,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(64000),
			model.WithModelConfigMaxOutputTokens(8192),
		),
	},
}

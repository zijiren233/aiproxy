package ali

import (
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

// https://help.aliyun.com/zh/model-studio/getting-started/models?spm=a2c4g.11186623.0.i12#ced16cb6cdfsy
// https://help.aliyun.com/zh/model-studio/model-pricing

var ModelList = []model.ModelConfig{
	// 通义千问-Max
	{
		Model: "qwen-max",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.02,
			OutputPrice: 0.06,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32768),
			model.WithModelConfigMaxInputTokens(30720),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-max-latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.02,
			OutputPrice: 0.06,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32768),
			model.WithModelConfigMaxInputTokens(30720),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},

	// 通义千问-Plus
	{
		Model: "qwen-plus",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0008,
			OutputPrice: 0.002,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-plus-latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0008,
			OutputPrice: 0.002,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32000),
			model.WithModelConfigMaxInputTokens(30000),
			model.WithModelConfigMaxOutputTokens(8000),
			model.WithModelConfigToolChoice(true),
		),
	},

	// 通义千问-Turbo
	{
		Model: "qwen-turbo",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0003,
			OutputPrice: 0.0006,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-turbo-latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0003,
			OutputPrice: 0.0006,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(1000000),
			model.WithModelConfigMaxInputTokens(1000000),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},

	// Qwen-Long
	{
		Model: "qwen-long",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0005,
			OutputPrice: 0.002,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(1000000),
			model.WithModelConfigMaxInputTokens(1000000),
			model.WithModelConfigMaxOutputTokens(6000),
			model.WithModelConfigToolChoice(true),
		),
	},

	// 通义千问VL
	{
		Model: "qwen-vl-max",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0016,
			OutputPrice: 0.004,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32000),
			model.WithModelConfigMaxInputTokens(30000),
			model.WithModelConfigMaxOutputTokens(2000),
			model.WithModelConfigVision(true),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-vl-max-latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0016,
			OutputPrice: 0.004,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32000),
			model.WithModelConfigMaxInputTokens(30000),
			model.WithModelConfigMaxOutputTokens(2000),
			model.WithModelConfigVision(true),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-vl-plus",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0008,
			OutputPrice: 0.002,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(8000),
			model.WithModelConfigMaxInputTokens(6000),
			model.WithModelConfigMaxOutputTokens(2000),
			model.WithModelConfigVision(true),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-vl-plus-latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0008,
			OutputPrice: 0.002,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32000),
			model.WithModelConfigMaxInputTokens(30000),
			model.WithModelConfigMaxOutputTokens(2000),
			model.WithModelConfigVision(true),
			model.WithModelConfigToolChoice(true),
		),
	},

	// 通义千问OCR
	{
		Model: "qwen-vl-ocr",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0003,
			OutputPrice: 0.0005,
		},
		RPM:              600,
		ExcludeFromTests: true,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(34096),
			model.WithModelConfigMaxInputTokens(30000),
			model.WithModelConfigMaxOutputTokens(4096),
			model.WithModelConfigVision(true),
		),
	},
	{
		Model: "qwen-vl-ocr-latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0003,
			OutputPrice: 0.0005,
		},
		RPM:              600,
		ExcludeFromTests: true,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(34096),
			model.WithModelConfigMaxInputTokens(30000),
			model.WithModelConfigMaxOutputTokens(4096),
			model.WithModelConfigVision(true),
		),
	},

	// 通义千问Math
	{
		Model: "qwen-math-plus",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.004,
			OutputPrice: 0.012,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(4096),
			model.WithModelConfigMaxInputTokens(3072),
			model.WithModelConfigMaxOutputTokens(3072),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-math-plus-latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.004,
			OutputPrice: 0.012,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(4096),
			model.WithModelConfigMaxInputTokens(3072),
			model.WithModelConfigMaxOutputTokens(3072),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-math-turbo",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.002,
			OutputPrice: 0.006,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(4096),
			model.WithModelConfigMaxInputTokens(3072),
			model.WithModelConfigMaxOutputTokens(3072),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-math-turbo-latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.002,
			OutputPrice: 0.006,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(4096),
			model.WithModelConfigMaxInputTokens(3072),
			model.WithModelConfigMaxOutputTokens(3072),
			model.WithModelConfigToolChoice(true),
		),
	},

	// 通义千问Coder
	{
		Model: "qwen-coder-plus",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0035,
			OutputPrice: 0.007,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-coder-plus-latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0035,
			OutputPrice: 0.007,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-coder-turbo",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.002,
			OutputPrice: 0.006,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-coder-turbo-latest",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.002,
			OutputPrice: 0.006,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},

	// 通义千问2.5
	{
		Model: "qwen2.5-72b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.004,
			OutputPrice: 0.012,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2.5-32b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0035,
			OutputPrice: 0.007,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2.5-14b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.002,
			OutputPrice: 0.006,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2.5-7b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.001,
			OutputPrice: 0.002,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2.5-vl-72b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.016,
			OutputPrice: 0.048,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigVision(true),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2.5-vl-7b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.002,
			OutputPrice: 0.005,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigVision(true),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2.5-vl-3b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0012,
			OutputPrice: 0.0036,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigVision(true),
			model.WithModelConfigToolChoice(true),
		),
	},

	// 通义千问2
	{
		Model: "qwen2-72b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.004,
			OutputPrice: 0.012,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(128000),
			model.WithModelConfigMaxOutputTokens(6144),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2-57b-a14b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0035,
			OutputPrice: 0.007,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(65536),
			model.WithModelConfigMaxInputTokens(63488),
			model.WithModelConfigMaxOutputTokens(6144),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2-7b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.001,
			OutputPrice: 0.002,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(128000),
			model.WithModelConfigMaxOutputTokens(6144),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2-vl-72b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.016,
			OutputPrice: 0.048,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32768),
			model.WithModelConfigMaxInputTokens(30720),
			model.WithModelConfigMaxOutputTokens(2048),
			model.WithModelConfigVision(true),
		),
	},

	// 通义千问1.5
	{
		Model: "qwen1.5-110b-chat",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.007,
			OutputPrice: 0.014,
		},
		RPM: 10,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32000),
			model.WithModelConfigMaxInputTokens(30000),
			model.WithModelConfigMaxOutputTokens(8000),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen1.5-72b-chat",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.005,
			OutputPrice: 0.01,
		},
		RPM: 120,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32000),
			model.WithModelConfigMaxInputTokens(30000),
			model.WithModelConfigMaxOutputTokens(8000),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen1.5-32b-chat",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0035,
			OutputPrice: 0.007,
		},
		RPM: 10,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32000),
			model.WithModelConfigMaxInputTokens(30000),
			model.WithModelConfigMaxOutputTokens(8000),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen1.5-14b-chat",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.002,
			OutputPrice: 0.004,
		},
		RPM: 120,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(8000),
			model.WithModelConfigMaxInputTokens(6000),
			model.WithModelConfigMaxOutputTokens(2000),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen1.5-7b-chat",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.001,
			OutputPrice: 0.002,
		},
		RPM: 120,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(8000),
			model.WithModelConfigMaxInputTokens(6000),
			model.WithModelConfigMaxOutputTokens(2000),
			model.WithModelConfigToolChoice(true),
		),
	},

	// 通义千问
	{
		Model: "qwen-72b-chat",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.02,
			OutputPrice: 0.02,
		},
		RPM: 80,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32000),
			model.WithModelConfigMaxInputTokens(30000),
			model.WithModelConfigMaxOutputTokens(2000),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-14b-chat",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.008,
			OutputPrice: 0.008,
		},
		RPM: 300,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(8000),
			model.WithModelConfigMaxInputTokens(6000),
			model.WithModelConfigMaxOutputTokens(2000),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen-7b-chat",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.006,
			OutputPrice: 0.006,
		},
		RPM: 300,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(7500),
			model.WithModelConfigMaxInputTokens(6000),
			model.WithModelConfigMaxOutputTokens(1500),
			model.WithModelConfigToolChoice(true),
		),
	},

	// 通义千问数学模型
	{
		Model: "qwen2.5-math-72b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.004,
			OutputPrice: 0.012,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(4096),
			model.WithModelConfigMaxInputTokens(3072),
			model.WithModelConfigMaxOutputTokens(3072),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2.5-math-7b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.001,
			OutputPrice: 0.002,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(4096),
			model.WithModelConfigMaxInputTokens(3072),
			model.WithModelConfigMaxOutputTokens(3072),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2-math-72b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.004,
			OutputPrice: 0.012,
		},
		RPM: 10,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(4096),
			model.WithModelConfigMaxInputTokens(3072),
			model.WithModelConfigMaxOutputTokens(3072),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2-math-7b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.001,
			OutputPrice: 0.002,
		},
		RPM: 10,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(4096),
			model.WithModelConfigMaxInputTokens(3072),
			model.WithModelConfigMaxOutputTokens(3072),
			model.WithModelConfigToolChoice(true),
		),
	},

	// 通义千问Coder
	{
		Model: "qwen2.5-coder-32b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0035,
			OutputPrice: 0.007,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2.5-coder-14b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.002,
			OutputPrice: 0.006,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},
	{
		Model: "qwen2.5-coder-7b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.001,
			OutputPrice: 0.002,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(131072),
			model.WithModelConfigMaxInputTokens(129024),
			model.WithModelConfigMaxOutputTokens(8192),
			model.WithModelConfigToolChoice(true),
		),
	},

	{
		Model: "qwq-32b-preview",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.0035,
			OutputPrice: 0.007,
		},
		RPM: 1200,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32768),
			model.WithModelConfigMaxInputTokens(30720),
			model.WithModelConfigMaxOutputTokens(16384),
		),
	},
	{
		Model: "qvq-72b-preview",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.012,
			OutputPrice: 0.036,
		},
		RPM: 60,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(32768),
			model.WithModelConfigMaxInputTokens(16384),
			model.WithModelConfigMaxOutputTokens(16384),
		),
	},

	{
		Model: "qwen-mt-plus",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.015,
			OutputPrice: 0.045,
		},
		RPM:              60,
		ExcludeFromTests: true,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(2048),
			model.WithModelConfigMaxInputTokens(1024),
			model.WithModelConfigMaxOutputTokens(1024),
		),
	},
	{
		Model: "qwen-mt-turbo",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice:  0.001,
			OutputPrice: 0.003,
		},
		RPM:              60,
		ExcludeFromTests: true,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(2048),
			model.WithModelConfigMaxInputTokens(1024),
			model.WithModelConfigMaxOutputTokens(1024),
		),
	},

	// stable-diffusion
	{
		Model: "stable-diffusion-xl",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerStabilityAI,
		RPM:   2,
	},
	{
		Model: "stable-diffusion-v1.5",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerStabilityAI,
		RPM:   2,
	},
	{
		Model: "stable-diffusion-3.5-large",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerStabilityAI,
		RPM:   2,
	},
	{
		Model: "stable-diffusion-3.5-large-turbo",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerStabilityAI,
		RPM:   2,
	},
	{
		Model: "qwen-image",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		// Ali image generation/edit APIs bill by successful output image count.
		// aliImageUsageToOpenAI maps that count into ImageOutputTokens.
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.25),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-plus",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-plus-2026-01-09",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-max",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.5),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-max-2025-12-30",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.5),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-2.0",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-2.0-2026-03-03",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-2.0-pro",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.5),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-2.0-pro-2026-03-03",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.5),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-2.0-pro-2026-04-22",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.5),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-edit",
		Type:  mode.ImagesEdits,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.3),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-edit-plus",
		Type:  mode.ImagesEdits,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-edit-plus-2025-10-30",
		Type:  mode.ImagesEdits,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-edit-plus-2025-12-15",
		Type:  mode.ImagesEdits,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-edit-max",
		Type:  mode.ImagesEdits,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.5),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-image-edit-max-2026-01-16",
		Type:  mode.ImagesEdits,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.5),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "qwen-mt-image",
		Type:  mode.ImagesEdits,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.003),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "z-image-turbo",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wan2.7-image",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wan2.7-image-pro",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.5),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wan2.6-t2i",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wan2.5-t2i-preview",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wan2.2-t2i-flash",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.14),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wan2.2-t2i-plus",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wanx2.1-t2i-turbo",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.14),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wanx2.1-t2i-plus",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.2),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wanx2.0-t2i-turbo",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.04),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wanx-v1",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			ImageOutputPrice:     model.ZeroNullFloat64(0.16),
			ImageOutputPriceUnit: 1,
		},
		RPM: 60,
	},
	{
		Model: "wan2.5-t2v-preview",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		// Ali video APIs bill by successful output video seconds. Async usage
		// normalizes returned dimensions into a resolution condition.
		Price: model.Price{
			OutputPrice:     model.ZeroNullFloat64(1.0),
			OutputPriceUnit: 1,
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"480p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.3),
						OutputPriceUnit: 1,
					},
				},
				{
					Condition: model.PriceCondition{Resolution: []string{"720p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.6),
						OutputPriceUnit: 1,
					},
				},
				{
					Condition: model.PriceCondition{Resolution: []string{"1080p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(1.0),
						OutputPriceUnit: 1,
					},
				},
			},
		},
		RPM: 60,
	},
	{
		Model: "wan2.5-i2v-preview",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			OutputPrice:     model.ZeroNullFloat64(1.0),
			OutputPriceUnit: 1,
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"480p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.3),
						OutputPriceUnit: 1,
					},
				},
				{
					Condition: model.PriceCondition{Resolution: []string{"720p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.6),
						OutputPriceUnit: 1,
					},
				},
				{
					Condition: model.PriceCondition{Resolution: []string{"1080p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(1.0),
						OutputPriceUnit: 1,
					},
				},
			},
		},
		RPM: 60,
	},
	{
		Model: "wan2.2-t2v-plus",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			OutputPrice:     model.ZeroNullFloat64(0.70),
			OutputPriceUnit: 1,
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"480p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.14),
						OutputPriceUnit: 1,
					},
				},
				{
					Condition: model.PriceCondition{Resolution: []string{"1080p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.70),
						OutputPriceUnit: 1,
					},
				},
			},
		},
		RPM: 60,
	},
	{
		Model: "wan2.2-i2v-plus",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			OutputPrice:     model.ZeroNullFloat64(0.70),
			OutputPriceUnit: 1,
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"480p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.14),
						OutputPriceUnit: 1,
					},
				},
				{
					Condition: model.PriceCondition{Resolution: []string{"1080p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.70),
						OutputPriceUnit: 1,
					},
				},
			},
		},
		RPM: 60,
	},
	{
		Model: "wan2.2-videoedit",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		RPM:   60,
	},
	{
		Model: "kling-v1",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		RPM:   60,
	},
	{
		Model: "vidu2.0",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		RPM:   60,
	},
	{
		Model: "pixverse-v4.5",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		RPM:   60,
	},
	{
		Model: "happyhorse-1.0-t2v",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			OutputPrice:     model.ZeroNullFloat64(1.6),
			OutputPriceUnit: 1,
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"720p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.9),
						OutputPriceUnit: 1,
					},
				},
				{
					Condition: model.PriceCondition{Resolution: []string{"1080p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(1.6),
						OutputPriceUnit: 1,
					},
				},
			},
		},
		RPM: 60,
	},
	{
		Model: "happyhorse-1.0-i2v",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			OutputPrice:     model.ZeroNullFloat64(1.6),
			OutputPriceUnit: 1,
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"720p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.9),
						OutputPriceUnit: 1,
					},
				},
				{
					Condition: model.PriceCondition{Resolution: []string{"1080p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(1.6),
						OutputPriceUnit: 1,
					},
				},
			},
		},
		RPM: 60,
	},
	{
		Model: "happyhorse-1.0-r2v",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			OutputPrice:     model.ZeroNullFloat64(1.6),
			OutputPriceUnit: 1,
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"720p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(0.9),
						OutputPriceUnit: 1,
					},
				},
				{
					Condition: model.PriceCondition{Resolution: []string{"1080p"}},
					Price: model.Price{
						OutputPrice:     model.ZeroNullFloat64(1.6),
						OutputPriceUnit: 1,
					},
				},
			},
		},
		RPM: 60,
	},
	{
		Model: "happyhorse-1.0-video-edit",
		Type:  mode.AliVideo,
		Owner: model.ModelOwnerAlibaba,
		// HappyHorse video edit bills both input video seconds and output video
		// seconds. fetchAliVideoJobUsage maps those seconds into VideoInputTokens
		// and OutputTokens.
		Price: model.Price{
			VideoInputPrice:     model.ZeroNullFloat64(1.6),
			VideoInputPriceUnit: 1,
			OutputPrice:         model.ZeroNullFloat64(1.6),
			OutputPriceUnit:     1,
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"720p"}},
					Price: model.Price{
						VideoInputPrice:     model.ZeroNullFloat64(0.9),
						VideoInputPriceUnit: 1,
						OutputPrice:         model.ZeroNullFloat64(0.9),
						OutputPriceUnit:     1,
					},
				},
				{
					Condition: model.PriceCondition{Resolution: []string{"1080p"}},
					Price: model.Price{
						VideoInputPrice:     model.ZeroNullFloat64(1.6),
						VideoInputPriceUnit: 1,
						OutputPrice:         model.ZeroNullFloat64(1.6),
						OutputPriceUnit:     1,
					},
				},
			},
		},
		RPM: 60,
	},

	{
		Model: "sambert-v1",
		Type:  mode.AudioSpeech,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice: 0.1,
		},
		RPM: 20,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxInputTokens(10000),
			model.WithModelConfigSupportFormats([]string{"mp3", "wav", "pcm"}),
			model.WithModelConfigSupportVoices([]string{
				"zhinan",
				"zhiqi",
				"zhichu",
				"zhide",
				"zhijia",
				"zhiru",
				"zhiqian",
				"zhixiang",
				"zhiwei",
				"zhihao",
				"zhijing",
				"zhiming",
				"zhimo",
				"zhina",
				"zhishu",
				"zhistella",
				"zhiting",
				"zhixiao",
				"zhiya",
				"zhiye",
				"zhiying",
				"zhiyuan",
				"zhiyue",
				"zhigui",
				"zhishuo",
				"zhimiao-emo",
				"zhimao",
				"zhilun",
				"zhifei",
				"zhida",
				"indah",
				"clara",
				"hanna",
				"beth",
				"betty",
				"cally",
				"cindy",
				"eva",
				"donna",
				"brian",
				"waan",
			}),
		),
	},

	{
		Model: "paraformer-realtime-v2",
		Type:  mode.AudioTranscription,
		Owner: model.ModelOwnerAlibaba,
		RPM:   20,
		Price: model.Price{
			InputPrice: 0.24,
		},
		Config: model.NewModelConfig(
			model.WithModelConfigMaxInputTokens(10000),
			model.WithModelConfigSupportFormats(
				[]string{"pcm", "wav", "opus", "speex", "aac", "amr"},
			),
		),
	},

	{
		Model: "gte-rerank",
		Type:  mode.Rerank,
		Owner: model.ModelOwnerAlibaba,
		RPM:   300,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxContextTokens(4000),
			model.WithModelConfigMaxInputTokens(4000),
		),
	},

	{
		Model: "text-embedding-v1",
		Type:  mode.Embeddings,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice: 0.0007,
		},
		RPM: 1800,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxInputTokens(2048),
		),
	},
	{
		Model: "text-embedding-v2",
		Type:  mode.Embeddings,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice: 0.0007,
		},
		RPM: 1800,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxInputTokens(2048),
		),
	},
	{
		Model: "text-embedding-v3",
		Type:  mode.Embeddings,
		Owner: model.ModelOwnerAlibaba,
		Price: model.Price{
			InputPrice: 0.0007,
		},
		RPM: 1800,
		Config: model.NewModelConfig(
			model.WithModelConfigMaxInputTokens(8192),
		),
	},
}

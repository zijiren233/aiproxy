package siliconflow

import (
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

// https://docs.siliconflow.cn/docs/getting-started

var ModelList = []model.ModelConfig{
	{
		Model: "BAAI/bge-reranker-v2-m3",
		Type:  mode.Rerank,
		Owner: model.ModelOwnerBAAI,
		RPM:   2000,
	},

	{
		Model: "BAAI/bge-large-zh-v1.5",
		Type:  mode.Embeddings,
		Owner: model.ModelOwnerBAAI,
		RPM:   2000,
	},

	{
		Model: "fishaudio/fish-speech-1.4",
		Type:  mode.AudioSpeech,
		Owner: model.ModelOwnerFishAudio,
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigSupportVoicesKey: []string{
				"fishaudio/fish-speech-1.4:alex",
				"fishaudio/fish-speech-1.4:benjamin",
				"fishaudio/fish-speech-1.4:charles",
				"fishaudio/fish-speech-1.4:david",
				"fishaudio/fish-speech-1.4:anna",
				"fishaudio/fish-speech-1.4:bella",
				"fishaudio/fish-speech-1.4:claire",
				"fishaudio/fish-speech-1.4:diana",
			},
		},
	},

	{
		Model: "FunAudioLLM/SenseVoiceSmall",
		Type:  mode.AudioTranscription,
		Owner: model.ModelOwnerFunAudioLLM,
	},

	{
		Model: "stabilityai/stable-diffusion-3-5-large",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerStabilityAI,
		ImagePrices: map[string]float64{
			"1024x1024": 0,
			"512x1024":  0,
			"768x512":   0,
			"768x1024":  0,
			"1024x576":  0,
			"576x1024":  0,
		},
	},
	{
		Model: "stabilityai/stable-diffusion-3-5-large-turbo",
		Type:  mode.ImagesGenerations,
		Owner: model.ModelOwnerStabilityAI,
		ImagePrices: map[string]float64{
			"1024x1024": 0,
			"512x1024":  0,
			"768x512":   0,
			"768x1024":  0,
			"1024x576":  0,
			"576x1024":  0,
		},
	},
}

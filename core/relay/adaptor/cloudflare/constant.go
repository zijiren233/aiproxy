package cloudflare

import (
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/mode"
)

var ModelList = []model.ModelConfig{
	{
		Model: "@cf/meta/llama-3.1-8b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "@cf/meta/llama-2-7b-chat-fp16",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "@cf/meta/llama-2-7b-chat-int8",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "@cf/mistral/mistral-7b-instruct-v0.1",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "@hf/thebloke/deepseek-coder-6.7b-base-awq",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerDeepSeek,
	},
	{
		Model: "@hf/thebloke/deepseek-coder-6.7b-instruct-awq",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerDeepSeek,
	},
	{
		Model: "@cf/deepseek-ai/deepseek-math-7b-base",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerDeepSeek,
	},
	{
		Model: "@cf/deepseek-ai/deepseek-math-7b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerDeepSeek,
	},
	{
		Model: "@cf/google/gemma-2b-it-lora",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerGoogle,
	},
	{
		Model: "@hf/google/gemma-7b-it",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerGoogle,
	},
	{
		Model: "@cf/google/gemma-7b-it-lora",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerGoogle,
	},
	{
		Model: "@hf/thebloke/llama-2-13b-chat-awq",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "@cf/meta-llama/llama-2-7b-chat-hf-lora",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "@cf/meta/llama-3-8b-instruct",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "@hf/thebloke/llamaguard-7b-awq",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "@hf/thebloke/mistral-7b-instruct-v0.1-awq",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "@hf/mistralai/mistral-7b-instruct-v0.2",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "@cf/mistral/mistral-7b-instruct-v0.2-lora",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "@hf/thebloke/neural-chat-7b-v3-1-awq",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "@cf/openchat/openchat-3.5-0106",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerOpenChat,
	},
	{
		Model: "@hf/thebloke/openhermes-2.5-mistral-7b-awq",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "@cf/microsoft/phi-2",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMicrosoft,
	},
	{
		Model: "@cf/qwen/qwen1.5-0.5b-chat",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
	},
	{
		Model: "@cf/qwen/qwen1.5-1.8b-chat",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
	},
	{
		Model: "@cf/qwen/qwen1.5-14b-chat-awq",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
	},
	{
		Model: "@cf/qwen/qwen1.5-7b-chat-awq",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAlibaba,
	},
	{
		Model: "@cf/defog/sqlcoder-7b-2",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerDefog,
	},
	{
		Model: "@hf/nexusflow/starling-lm-7b-beta",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerNexusFlow,
	},
	{
		Model: "@cf/tinyllama/tinyllama-1.1b-chat-v1.0",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMeta,
	},
	{
		Model: "@hf/thebloke/zephyr-7b-beta-awq",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
}

package cohere

import (
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/mode"
)

var ModelList = []*model.ModelConfig{
	{
		Model: "command",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerCohere,
	},
	{
		Model: "command-nightly",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerCohere,
	},
	{
		Model: "command-light",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerCohere,
	},
	{
		Model: "command-light-nightly",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerCohere,
	},
	{
		Model: "command-r",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerCohere,
	},
	{
		Model: "command-r-plus",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerCohere,
	},
}

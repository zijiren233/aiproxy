package mistral

import (
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/relaymode"
)

var ModelList = []*model.ModelConfig{
	{
		Model: "open-mistral-7b",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "open-mixtral-8x7b",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "mistral-small-latest",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "mistral-medium-latest",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "mistral-large-latest",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerMistral,
	},
	{
		Model: "mistral-embed",
		Type:  relaymode.Embeddings,
		Owner: model.ModelOwnerMistral,
	},
}

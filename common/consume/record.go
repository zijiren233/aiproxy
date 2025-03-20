package consume

import (
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/meta"
	relaymodel "github.com/labring/aiproxy/relay/model"
)

func recordConsume(
	meta *meta.Meta,
	code int,
	usage *relaymodel.Usage,
	modelPrice *model.Price,
	content string,
	ip string,
	requestDetail *model.RequestDetail,
	amount float64,
	retryTimes int,
	downstreamResult bool,
) error {
	promptTokens := 0
	completionTokens := 0
	cachedTokens := 0
	cacheCreationTokens := 0
	if usage != nil {
		promptTokens = usage.PromptTokens
		completionTokens = usage.CompletionTokens
		if usage.PromptTokensDetails != nil {
			cachedTokens = usage.PromptTokensDetails.CachedTokens
			cacheCreationTokens = usage.PromptTokensDetails.CacheCreationTokens
		}
	}

	var channelID int
	if meta.Channel != nil {
		channelID = meta.Channel.ID
	}

	return model.BatchRecordConsume(
		meta.RequestID,
		meta.RequestAt,
		meta.Group.ID,
		code,
		channelID,
		meta.OriginModel,
		meta.Token.ID,
		meta.Token.Name,
		meta.Endpoint,
		content,
		int(meta.Mode),
		ip,
		retryTimes,
		requestDetail,
		downstreamResult,
		model.Usage{
			InputTokens:         promptTokens,
			OutputTokens:        completionTokens,
			TotalTokens:         promptTokens + completionTokens,
			CachedTokens:        cachedTokens,
			CacheCreationTokens: cacheCreationTokens,
		},
		model.Price{
			InputPrice:         modelPrice.InputPrice,
			OutputPrice:        modelPrice.OutputPrice,
			CachedPrice:        modelPrice.CachedPrice,
			CacheCreationPrice: modelPrice.CacheCreationPrice,
		},
		amount,
	)
}

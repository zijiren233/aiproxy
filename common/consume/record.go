package consume

import (
	"time"

	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/meta"
	relaymodel "github.com/labring/aiproxy/relay/model"
)

func recordConsume(
	meta *meta.Meta,
	code int,
	firstByteAt time.Time,
	usage relaymodel.Usage,
	modelPrice model.Price,
	content string,
	ip string,
	requestDetail *model.RequestDetail,
	amount float64,
	retryTimes int,
	downstreamResult bool,
) error {
	us := model.Usage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.PromptTokens + usage.CompletionTokens,
	}
	if usage.PromptTokensDetails != nil {
		us.CachedTokens = usage.PromptTokensDetails.CachedTokens
		us.CacheCreationTokens = usage.PromptTokensDetails.CacheCreationTokens
	}

	return model.BatchRecordConsume(
		meta.RequestID,
		meta.RequestAt,
		meta.RetryAt,
		firstByteAt,
		meta.Group.ID,
		code,
		meta.Channel.ID,
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
		us,
		modelPrice,
		amount,
	)
}

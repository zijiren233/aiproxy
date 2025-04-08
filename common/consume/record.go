package consume

import (
	"time"

	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/meta"
)

func recordConsume(
	meta *meta.Meta,
	code int,
	firstByteAt time.Time,
	usage model.Usage,
	modelPrice model.Price,
	content string,
	ip string,
	requestDetail *model.RequestDetail,
	amount float64,
	retryTimes int,
	downstreamResult bool,
) error {
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
		usage,
		modelPrice,
		amount,
	)
}

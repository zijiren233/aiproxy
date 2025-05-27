package consume

import (
	"time"

	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
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
	user string,
	metadata map[string]string,
	channelModelRate model.RequestRate,
	groupModelTokenRate model.RequestRate,
) error {
	return model.BatchRecordLogs(
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
		user,
		metadata,
		channelModelRate,
		groupModelTokenRate,
	)
}

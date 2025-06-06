package monitor

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common/conv"
	"github.com/labring/aiproxy/core/common/notify"
	"github.com/labring/aiproxy/core/common/reqlimit"
	"github.com/labring/aiproxy/core/common/trylock"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/monitor"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/plugin"
	"github.com/labring/aiproxy/core/relay/plugin/noop"
)

var _ plugin.Plugin = (*Monitor)(nil)

// Monitor implements the monitor functionality
type Monitor struct {
	noop.Noop
}

func NewMonitorPlugin() plugin.Plugin {
	return &Monitor{}
}

var channelNoRetryStatusCodesMap = map[int]struct{}{
	http.StatusBadRequest:                 {},
	http.StatusRequestEntityTooLarge:      {},
	http.StatusUnprocessableEntity:        {},
	http.StatusUnavailableForLegalReasons: {},
}

func ShouldRetry(relayErr adaptor.Error) bool {
	_, ok := channelNoRetryStatusCodesMap[relayErr.StatusCode()]
	return !ok
}

var channelNoPermissionStatusCodesMap = map[int]struct{}{
	http.StatusUnauthorized:    {},
	http.StatusPaymentRequired: {},
	http.StatusForbidden:       {},
	http.StatusNotFound:        {},
}

func ChannelHasPermission(relayErr adaptor.Error) bool {
	_, ok := channelNoPermissionStatusCodesMap[relayErr.StatusCode()]
	return !ok
}

func (m *Monitor) DoRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
	do adaptor.DoRequest,
) (*http.Response, error) {
	count, overLimitCount, secondCount := reqlimit.PushChannelModelRequest(
		context.Background(),
		strconv.Itoa(meta.Channel.ID),
		meta.OriginModel,
	)
	updateChannelModelRequestRate(c, meta, count+overLimitCount, secondCount)
	return do.DoRequest(meta, store, c, req)
}

func (m *Monitor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
	do adaptor.DoResponse,
) (model.Usage, adaptor.Error) {
	log := middleware.GetLogger(c)

	usage, relayErr := do.DoResponse(meta, store, c, resp)

	if usage.TotalTokens > 0 {
		count, overLimitCount, secondCount := reqlimit.PushChannelModelTokensRequest(
			context.Background(),
			strconv.Itoa(meta.Channel.ID),
			meta.OriginModel,
			int64(usage.TotalTokens),
		)
		updateChannelModelTokensRequestRate(c, meta, count+overLimitCount, secondCount)

		count, overLimitCount, secondCount = reqlimit.PushGroupModelTokensRequest(
			context.Background(),
			meta.Group.ID,
			meta.OriginModel,
			meta.ModelConfig.TPM,
			int64(usage.TotalTokens),
		)
		middleware.UpdateGroupModelTokensRequest(c, meta.Group, count+overLimitCount, secondCount)

		count, overLimitCount, secondCount = reqlimit.PushGroupModelTokennameTokensRequest(
			context.Background(),
			meta.Group.ID,
			meta.OriginModel,
			meta.Token.Name,
			int64(usage.TotalTokens),
		)
		middleware.UpdateGroupModelTokennameTokensRequest(c, count+overLimitCount, secondCount)
	}

	if relayErr == nil {
		if _, _, err := monitor.AddRequest(
			context.Background(),
			meta.OriginModel,
			int64(meta.Channel.ID),
			false,
			false,
			meta.ModelConfig.MaxErrorRate,
		); err != nil {
			log.Errorf("add request failed: %+v", err)
		}
		return usage, nil
	}

	if !ShouldRetry(relayErr) {
		return usage, relayErr
	}

	hasPermission := ChannelHasPermission(relayErr)
	beyondThreshold, banExecution, err := monitor.AddRequest(
		context.Background(),
		meta.OriginModel,
		int64(meta.Channel.ID),
		true,
		!hasPermission,
		meta.ModelConfig.MaxErrorRate,
	)
	if err != nil {
		log.Errorf("add request failed: %+v", err)
	}
	switch {
	case banExecution:
		notifyChannelIssue(c, meta, "autoBanned", "Auto Banned", relayErr)
	case beyondThreshold:
		notifyChannelIssue(
			c,
			meta,
			"beyondThreshold",
			"Error Rate Beyond Threshold",
			relayErr,
		)
	case !hasPermission:
		notifyChannelIssue(c, meta, "channelHasPermission", "No Permission", relayErr)
	}

	return usage, relayErr
}

func notifyChannelIssue(
	c *gin.Context,
	meta *meta.Meta,
	issueType, titleSuffix string,
	err adaptor.Error,
) {
	var notifyFunc func(title, message string)

	lockKey := fmt.Sprintf("%s:%d:%s", issueType, meta.Channel.ID, meta.OriginModel)
	switch issueType {
	case "beyondThreshold":
		notifyFunc = func(title, message string) {
			notify.WarnThrottle(lockKey, time.Minute, title, message)
		}
	default:
		notifyFunc = func(title, message string) {
			notify.ErrorThrottle(lockKey, time.Minute, title, message)
		}
	}

	respBody, _ := err.MarshalJSON()

	message := fmt.Sprintf(
		"channel: %s (type: %d, type name: %s, id: %d)\nmodel: %s\nmode: %s\nstatus code: %d\ndetail: %s\nrequest id: %s",
		meta.Channel.Name,
		meta.Channel.Type,
		meta.Channel.Type.String(),
		meta.Channel.ID,
		meta.OriginModel,
		meta.Mode,
		err.StatusCode(),
		conv.BytesToString(respBody),
		meta.RequestID,
	)

	if err.StatusCode() == http.StatusTooManyRequests {
		if !trylock.Lock(lockKey, time.Minute) {
			return
		}
		switch issueType {
		case "beyondThreshold":
			notifyFunc = notify.Warn
		default:
			notifyFunc = notify.Error
		}

		rate := GetChannelModelRequestRate(c, meta)
		message += fmt.Sprintf(
			"\nrpm: %d\nrps: %d\ntpm: %d\ntps: %d",
			rate.RPM,
			rate.RPS,
			rate.TPM,
			rate.TPS,
		)
	}

	notifyFunc(
		fmt.Sprintf("%s `%s` %s", meta.Channel.Name, meta.OriginModel, titleSuffix),
		message,
	)
}

const (
	MetaChannelModelKeyRPM = "channel_model_rpm"
	MetaChannelModelKeyRPS = "channel_model_rps"
	MetaChannelModelKeyTPM = "channel_model_tpm"
	MetaChannelModelKeyTPS = "channel_model_tps"
)

func GetChannelModelRequestRate(c *gin.Context, meta *meta.Meta) model.RequestRate {
	rate := model.RequestRate{}

	if rpm, ok := meta.Get(MetaChannelModelKeyRPM); ok {
		rate.RPM, _ = rpm.(int64)
		rate.RPS = meta.GetInt64(MetaChannelModelKeyRPS)
	} else {
		rpm, rps := reqlimit.GetChannelModelRequest(context.Background(), strconv.Itoa(meta.Channel.ID), meta.OriginModel)
		rate.RPM = rpm
		rate.RPS = rps
		updateChannelModelRequestRate(c, meta, rpm, rps)
	}

	if tpm, ok := meta.Get(MetaChannelModelKeyTPM); ok {
		rate.TPM, _ = tpm.(int64)
		rate.TPS = meta.GetInt64(MetaChannelModelKeyTPS)
	} else {
		tpm, tps := reqlimit.GetChannelModelTokensRequest(context.Background(), strconv.Itoa(meta.Channel.ID), meta.OriginModel)
		rate.TPM = tpm
		rate.TPS = tps
		updateChannelModelTokensRequestRate(c, meta, tpm, tps)
	}

	return rate
}

func updateChannelModelRequestRate(c *gin.Context, meta *meta.Meta, rpm, rps int64) {
	meta.Set(MetaChannelModelKeyRPM, rpm)
	meta.Set(MetaChannelModelKeyRPS, rps)
	log := middleware.GetLogger(c)
	log.Data["ch_rpm"] = rpm
	log.Data["ch_rps"] = rps
}

func updateChannelModelTokensRequestRate(c *gin.Context, meta *meta.Meta, tpm, tps int64) {
	meta.Set(MetaChannelModelKeyTPM, tpm)
	meta.Set(MetaChannelModelKeyTPS, tps)
	log := middleware.GetLogger(c)
	log.Data["ch_tpm"] = tpm
	log.Data["ch_tps"] = tps
}

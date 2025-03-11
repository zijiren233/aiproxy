package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common"
	"github.com/labring/aiproxy/common/config"
	"github.com/labring/aiproxy/common/consume"
	"github.com/labring/aiproxy/common/notify"
	"github.com/labring/aiproxy/middleware"
	dbmodel "github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/monitor"
	"github.com/labring/aiproxy/relay/controller"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/model"
	"github.com/labring/aiproxy/relay/relaymode"
	log "github.com/sirupsen/logrus"
)

// https://platform.openai.com/docs/api-reference/chat

type RelayController func(*meta.Meta, *gin.Context) *controller.HandleResult

func relayController(mode relaymode.Mode) (RelayController, bool) {
	var relayController RelayController
	switch mode {
	case relaymode.ImagesGenerations,
		relaymode.Edits:
		relayController = controller.RelayImageHelper
	case relaymode.AudioSpeech:
		relayController = controller.RelayTTSHelper
	case relaymode.AudioTranslation,
		relaymode.AudioTranscription:
		relayController = controller.RelaySTTHelper
	case relaymode.ParsePdf:
		relayController = controller.RelayParsePdfHelper
	case relaymode.Rerank:
		relayController = controller.RerankHelper
	case relaymode.ChatCompletions,
		relaymode.Embeddings,
		relaymode.Completions,
		relaymode.Moderations:
		relayController = controller.RelayTextHelper
	default:
		return nil, false
	}
	return func(meta *meta.Meta, c *gin.Context) *controller.HandleResult {
		log := middleware.GetLogger(c)
		middleware.SetLogFieldsFromMeta(meta, log.Data)
		return relayController(meta, c)
	}, true
}

func RelayHelper(meta *meta.Meta, c *gin.Context, relayController RelayController) (*controller.HandleResult, bool) {
	result := relayController(meta, c)
	if result.Error == nil {
		if _, _, err := monitor.AddRequest(
			context.Background(),
			meta.OriginModel,
			int64(meta.Channel.ID),
			false,
			false,
		); err != nil {
			log.Errorf("add request failed: %+v", err)
		}
		return result, false
	}
	if result.Error.Error.Code == middleware.GroupBalanceNotEnough {
		return result, false
	}
	if shouldErrorMonitor(result.Error.StatusCode) {
		hasPermission := channelHasPermission(result.Error.StatusCode)
		beyondThreshold, banExecution, err := monitor.AddRequest(
			context.Background(),
			meta.OriginModel,
			int64(meta.Channel.ID),
			true,
			!hasPermission,
		)
		if err != nil {
			log.Errorf("add request failed: %+v", err)
		}
		switch {
		case banExecution:
			notify.ErrorThrottle(
				fmt.Sprintf("autoBanned:%d:%s", meta.Channel.ID, meta.OriginModel),
				time.Minute,
				fmt.Sprintf("channel[%d] %s(%d) model %s is auto banned",
					meta.Channel.Type, meta.Channel.Name, meta.Channel.ID, meta.OriginModel),
				result.Error.JSONOrEmpty(),
			)
		case beyondThreshold:
			notify.WarnThrottle(
				fmt.Sprintf("beyondThreshold:%d:%s", meta.Channel.ID, meta.OriginModel),
				time.Minute,
				fmt.Sprintf("channel[%d] %s(%d) model %s error rate is beyond threshold",
					meta.Channel.Type, meta.Channel.Name, meta.Channel.ID, meta.OriginModel),
				result.Error.JSONOrEmpty(),
			)
		case !hasPermission:
			notify.ErrorThrottle(
				fmt.Sprintf("channelHasPermission:%d:%s", meta.Channel.ID, meta.OriginModel),
				time.Minute,
				fmt.Sprintf("channel[%d] %s(%d) model %s has no permission",
					meta.Channel.Type, meta.Channel.Name, meta.Channel.ID, meta.OriginModel),
				result.Error.JSONOrEmpty(),
			)
		}
	}
	return result, shouldRetry(c, result.Error.StatusCode)
}

func filterChannels(channels []*dbmodel.Channel, ignoreChannel ...int64) []*dbmodel.Channel {
	filtered := make([]*dbmodel.Channel, 0)
	for _, channel := range channels {
		if channel.Status != dbmodel.ChannelStatusEnabled {
			continue
		}
		if slices.Contains(ignoreChannel, int64(channel.ID)) {
			continue
		}
		filtered = append(filtered, channel)
	}
	return filtered
}

var (
	ErrChannelsNotFound  = errors.New("channels not found")
	ErrChannelsExhausted = errors.New("channels exhausted")
)

func GetRandomChannel(c *dbmodel.ModelCaches, model string, errorRates map[int64]float64, ignoreChannel ...int64) (*dbmodel.Channel, error) {
	return getRandomChannel(c.EnabledModel2channels[model], errorRates, ignoreChannel...)
}

func getPriority(channel *dbmodel.Channel, errorRate float64) int32 {
	priority := channel.GetPriority()
	if errorRate > 1 {
		errorRate = 1
	} else if errorRate < 0.1 {
		errorRate = 0.1
	}
	return int32(float64(priority) / errorRate)
}

//nolint:gosec
func getRandomChannel(channels []*dbmodel.Channel, errorRates map[int64]float64, ignoreChannel ...int64) (*dbmodel.Channel, error) {
	if len(channels) == 0 {
		return nil, ErrChannelsNotFound
	}

	channels = filterChannels(channels, ignoreChannel...)
	if len(channels) == 0 {
		return nil, ErrChannelsExhausted
	}

	if len(channels) == 1 {
		return channels[0], nil
	}

	var totalWeight int32
	cachedPrioritys := make([]int32, len(channels))
	for i, ch := range channels {
		priority := getPriority(ch, errorRates[int64(ch.ID)])
		totalWeight += priority
		cachedPrioritys[i] = priority
	}

	if totalWeight == 0 {
		return channels[rand.IntN(len(channels))], nil
	}

	r := rand.Int32N(totalWeight)
	for i, ch := range channels {
		r -= cachedPrioritys[i]
		if r < 0 {
			return ch, nil
		}
	}

	return channels[rand.IntN(len(channels))], nil
}

func getChannelWithFallback(cache *dbmodel.ModelCaches, model string, errorRates map[int64]float64, ignoreChannelIDs ...int64) (*dbmodel.Channel, error) {
	channel, err := GetRandomChannel(cache, model, errorRates, ignoreChannelIDs...)
	if err == nil {
		return channel, nil
	}
	if !errors.Is(err, ErrChannelsExhausted) {
		return nil, err
	}
	return GetRandomChannel(cache, model, errorRates)
}

func NewRelay(mode relaymode.Mode) func(c *gin.Context) {
	relayController, ok := relayController(mode)
	if !ok {
		log.Fatalf("relay mode %d not implemented", mode)
	}
	return func(c *gin.Context) {
		relay(c, mode, relayController)
	}
}

func relay(c *gin.Context, mode relaymode.Mode, relayController RelayController) {
	log := middleware.GetLogger(c)
	requestModel := middleware.GetOriginalModel(c)

	// Get initial channel
	initialChannel, err := getInitialChannel(c, requestModel, log)
	if err != nil || initialChannel == nil || initialChannel.channel == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": &model.Error{
				Message: "the upstream load is saturated, please try again later",
				Code:    "upstream_load_saturated",
				Type:    middleware.ErrorTypeAIPROXY,
			},
		})
		return
	}

	// First attempt
	meta := middleware.NewMetaByContext(c, initialChannel.channel, requestModel, mode)
	result, retry := RelayHelper(meta, c, relayController)

	retryTimes := int(config.GetRetryTimes())
	if handleRelayResult(c, result.Error, retry, retryTimes) {
		recordResult(c, meta, result, 0, true)
		return
	}
	recordResult(c, meta, result, 0, false)

	// Setup retry state
	retryState := initRetryState(
		retryTimes,
		initialChannel.channel,
		result.Error,
		initialChannel.ignoreChannelIDs,
		initialChannel.errorRates,
	)

	// Retry loop
	retryLoop(c, mode, requestModel, retryState, relayController, log)
}

// recordResult records the consumption for the final result
func recordResult(c *gin.Context, meta *meta.Meta, result *controller.HandleResult, retryTimes int, downstreamResult bool) {
	code := http.StatusOK
	content := ""
	if result.Error != nil {
		code = result.Error.StatusCode
		content = result.Error.JSONOrEmpty()
	}

	detail := result.Detail
	if code == http.StatusOK && !config.GetSaveAllLogDetail() {
		detail = nil
	}

	gbc := middleware.GetGroupBalanceConsumerFromContext(c)

	consume.AsyncConsume(
		gbc.Consumer,
		code,
		result.Usage,
		meta,
		result.InputPrice,
		result.OutputPrice,
		content,
		c.ClientIP(),
		retryTimes,
		detail,
		downstreamResult,
	)
}

type retryState struct {
	retryTimes               int
	lastHasPermissionChannel *dbmodel.Channel
	ignoreChannelIDs         []int64
	errorRates               map[int64]float64
	exhausted                bool
	bizErr                   *model.ErrorWithStatusCode
	startTime                time.Time
}

type initialChannel struct {
	channel          *dbmodel.Channel
	ignoreChannelIDs []int64
	errorRates       map[int64]float64
}

func getInitialChannel(c *gin.Context, requestModel string, log *log.Entry) (*initialChannel, error) {
	ids, err := monitor.GetBannedChannelsWithModel(c.Request.Context(), requestModel)
	if err != nil {
		log.Errorf("get %s auto banned channels failed: %+v", requestModel, err)
	}
	log.Debugf("%s model banned channels: %+v", requestModel, ids)

	errorRates, err := monitor.GetModelChannelErrorRate(c.Request.Context(), requestModel)
	if err != nil {
		log.Errorf("get channel model error rates failed: %+v", err)
	}

	mc := middleware.GetModelCaches(c)
	channel, err := getChannelWithFallback(mc, requestModel, errorRates, ids...)
	if err != nil {
		return nil, err
	}

	return &initialChannel{
		channel:          channel,
		ignoreChannelIDs: ids,
		errorRates:       errorRates,
	}, nil
}

func handleRelayResult(c *gin.Context, bizErr *model.ErrorWithStatusCode, retry bool, retryTimes int) bool {
	if bizErr == nil {
		return true
	}
	if !retry || retryTimes == 0 {
		bizErr.Error.Message = middleware.MessageWithRequestID(c, bizErr.Error.Message)
		c.JSON(bizErr.StatusCode, bizErr)
		return true
	}
	return false
}

func initRetryState(retryTimes int, channel *dbmodel.Channel, bizErr *model.ErrorWithStatusCode, ignoreChannelIDs []int64, errorRates map[int64]float64) *retryState {
	state := &retryState{
		retryTimes:       retryTimes,
		ignoreChannelIDs: ignoreChannelIDs,
		errorRates:       errorRates,
		bizErr:           bizErr,
		startTime:        time.Now(),
	}

	if !channelHasPermission(bizErr.StatusCode) {
		state.ignoreChannelIDs = append(state.ignoreChannelIDs, int64(channel.ID))
	} else {
		state.lastHasPermissionChannel = channel
	}

	return state
}

func retryLoop(c *gin.Context, mode relaymode.Mode, requestModel string, state *retryState, relayController RelayController, log *log.Entry) {
	mc := middleware.GetModelCaches(c)

	for i := range state.retryTimes {
		ctxErr := c.Request.Context().Err()
		if ctxErr != nil {
			log.Warnf("retry loop context error: %+v", ctxErr)
			break
		}

		newChannel, err := getRetryChannel(mc, requestModel, state)
		if err != nil {
			break
		}

		log.Data["retry"] = strconv.Itoa(i + 1)

		log.Warnf("using channel %s (type: %d, id: %d) to retry (remain times %d)",
			newChannel.Name,
			newChannel.Type,
			newChannel.ID,
			state.retryTimes-i,
		)

		if !prepareRetry(c, state.bizErr.StatusCode) {
			break
		}

		meta := middleware.NewMetaByContext(c,
			newChannel,
			requestModel,
			mode,
		)
		result, retry := RelayHelper(meta, c, relayController)

		done := handleRetryResult(result.Error, retry, newChannel, state)
		recordResult(c, meta, result, i+1, done || i == state.retryTimes-1)
		if done {
			break
		}
	}

	if state.bizErr != nil {
		state.bizErr.Error.Message = middleware.MessageWithRequestID(c, state.bizErr.Error.Message)
		c.JSON(state.bizErr.StatusCode, state.bizErr)
	}
}

func getRetryChannel(mc *dbmodel.ModelCaches, model string, state *retryState) (*dbmodel.Channel, error) {
	if state.exhausted {
		return state.lastHasPermissionChannel, nil
	}

	newChannel, err := GetRandomChannel(mc, model, state.errorRates, state.ignoreChannelIDs...)
	if err != nil {
		if !errors.Is(err, ErrChannelsExhausted) || state.lastHasPermissionChannel == nil {
			return nil, err
		}
		state.exhausted = true
		return state.lastHasPermissionChannel, nil
	}

	return newChannel, nil
}

func prepareRetry(c *gin.Context, statusCode int) bool {
	requestBody, err := common.GetRequestBody(c.Request)
	if err != nil {
		log.Errorf("get request body failed in prepare retry: %+v", err)
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	if shouldDelay(statusCode) {
		//nolint:gosec
		time.Sleep(time.Duration(rand.Float64()*float64(time.Second)) + time.Second)
	}

	return true
}

func handleRetryResult(bizErr *model.ErrorWithStatusCode, retry bool, newChannel *dbmodel.Channel, state *retryState) (done bool) {
	state.bizErr = bizErr
	if !retry || bizErr == nil {
		return true
	}

	if state.exhausted {
		if !channelHasPermission(bizErr.StatusCode) {
			return true
		}
	} else {
		if !channelHasPermission(bizErr.StatusCode) {
			state.ignoreChannelIDs = append(state.ignoreChannelIDs, int64(newChannel.ID))
			state.retryTimes++
		} else {
			state.lastHasPermissionChannel = newChannel
		}
	}

	return false
}

func shouldRetry(_ *gin.Context, statusCode int) bool {
	return statusCode != http.StatusBadRequest
}

var channelNoPermissionStatusCodesMap = map[int]struct{}{
	http.StatusUnauthorized:    {},
	http.StatusPaymentRequired: {},
	http.StatusForbidden:       {},
}

func channelHasPermission(statusCode int) bool {
	_, ok := channelNoPermissionStatusCodesMap[statusCode]
	return !ok
}

func shouldDelay(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests
}

// 仅当是channel错误时，才需要记录，用户请求参数错误时，不需要记录
func shouldErrorMonitor(statusCode int) bool {
	return statusCode != http.StatusBadRequest
}

func RelayNotImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": &model.Error{
			Message: "API not implemented",
			Type:    middleware.ErrorTypeAIPROXY,
			Code:    "api_not_implemented",
		},
	})
}

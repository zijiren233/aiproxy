package consume

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/labring/aiproxy/core/common/balance"
	"github.com/labring/aiproxy/core/common/config"
	"github.com/labring/aiproxy/core/common/notify"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

var consumeWaitGroup sync.WaitGroup

func Wait() {
	consumeWaitGroup.Wait()
}

func AsyncConsume(
	postGroupConsumer balance.PostGroupConsumer,
	code int,
	firstByteAt time.Time,
	meta *meta.Meta,
	usage model.Usage,
	modelPrice model.Price,
	content string,
	ip string,
	retryTimes int,
	requestDetail *model.RequestDetail,
	downstreamResult bool,
) {
	consumeWaitGroup.Add(1)
	defer func() {
		consumeWaitGroup.Done()
		if r := recover(); r != nil {
			log.Errorf("panic in consume: %v", r)
		}
	}()

	go Consume(
		context.Background(),
		postGroupConsumer,
		firstByteAt,
		code,
		meta,
		usage,
		modelPrice,
		content,
		ip,
		retryTimes,
		requestDetail,
		downstreamResult,
	)
}

func Consume(
	ctx context.Context,
	postGroupConsumer balance.PostGroupConsumer,
	firstByteAt time.Time,
	code int,
	meta *meta.Meta,
	usage model.Usage,
	modelPrice model.Price,
	content string,
	ip string,
	retryTimes int,
	requestDetail *model.RequestDetail,
	downstreamResult bool,
) {
	amount := CalculateAmount(code, usage, modelPrice)
	amount = consumeAmount(ctx, amount, postGroupConsumer, meta)

	if requestDetail != nil && config.GetLogContentStorageHours() < 0 {
		requestDetail = nil
	}

	if requestDetail == nil && config.GetLogStorageHours() < 0 {
		return
	}

	err := recordConsume(
		meta,
		code,
		firstByteAt,
		usage,
		modelPrice,
		content,
		ip,
		requestDetail,
		amount,
		retryTimes,
		downstreamResult,
	)
	if err != nil {
		log.Error("error batch record consume: " + err.Error())
		notify.ErrorThrottle("recordConsume", time.Minute, "record consume failed", err.Error())
	}
}

func consumeAmount(
	ctx context.Context,
	amount float64,
	postGroupConsumer balance.PostGroupConsumer,
	meta *meta.Meta,
) float64 {
	if amount > 0 && postGroupConsumer != nil {
		return processGroupConsume(ctx, amount, postGroupConsumer, meta)
	}
	return amount
}

func CalculateAmount(
	code int,
	usage model.Usage,
	modelPrice model.Price,
) float64 {
	if modelPrice.PerRequestPrice != 0 {
		if code != http.StatusOK {
			return 0
		}
		return modelPrice.PerRequestPrice
	}

	inputTokens := usage.InputTokens
	outputTokens := usage.OutputTokens

	if modelPrice.ImageInputPrice > 0 {
		inputTokens -= usage.ImageInputTokens
	}
	if modelPrice.CachedPrice > 0 {
		inputTokens -= usage.CachedTokens
	}
	if modelPrice.CacheCreationPrice > 0 {
		inputTokens -= usage.CacheCreationTokens
	}

	inputAmount := decimal.NewFromInt(inputTokens).
		Mul(decimal.NewFromFloat(modelPrice.InputPrice)).
		Div(decimal.NewFromInt(modelPrice.GetInputPriceUnit()))

	imageInputAmount := decimal.NewFromInt(usage.ImageInputTokens).
		Mul(decimal.NewFromFloat(modelPrice.ImageInputPrice)).
		Div(decimal.NewFromInt(modelPrice.GetImageInputPriceUnit()))

	cachedAmount := decimal.NewFromInt(usage.CachedTokens).
		Mul(decimal.NewFromFloat(modelPrice.CachedPrice)).
		Div(decimal.NewFromInt(modelPrice.GetCachedPriceUnit()))

	cacheCreationAmount := decimal.NewFromInt(usage.CacheCreationTokens).
		Mul(decimal.NewFromFloat(modelPrice.CacheCreationPrice)).
		Div(decimal.NewFromInt(modelPrice.GetCacheCreationPriceUnit()))

	webSearchAmount := decimal.NewFromInt(usage.WebSearchCount).
		Mul(decimal.NewFromFloat(modelPrice.WebSearchPrice)).
		Div(decimal.NewFromInt(modelPrice.GetWebSearchPriceUnit()))

	outputAmount := decimal.NewFromInt(outputTokens).
		Mul(decimal.NewFromFloat(modelPrice.OutputPrice)).
		Div(decimal.NewFromInt(modelPrice.GetOutputPriceUnit()))

	return inputAmount.
		Add(imageInputAmount).
		Add(cachedAmount).
		Add(cacheCreationAmount).
		Add(webSearchAmount).
		Add(outputAmount).
		InexactFloat64()
}

func processGroupConsume(
	ctx context.Context,
	amount float64,
	postGroupConsumer balance.PostGroupConsumer,
	meta *meta.Meta,
) float64 {
	consumedAmount, err := postGroupConsumer.PostGroupConsume(ctx, meta.Token.Name, amount)
	if err != nil {
		log.Error("error consuming token remain amount: " + err.Error())
		if err := model.CreateConsumeError(
			meta.RequestID,
			meta.RequestAt,
			meta.Group.ID,
			meta.Token.Name,
			meta.OriginModel,
			err.Error(),
			amount,
			meta.Token.ID,
		); err != nil {
			log.Error("failed to create consume error: " + err.Error())
		}
		return amount
	}
	return consumedAmount
}

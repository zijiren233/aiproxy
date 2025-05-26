package reqlimit

import (
	"context"
	"time"

	"github.com/labring/aiproxy/core/common"
	log "github.com/sirupsen/logrus"
)

var (
	memoryGroupModelLimiter = NewInMemoryRecord()
	redisGroupModelLimiter  = NewRedisGroupModelRecord()
)

func memoryPushRequest(group, model string, maxReq int64) (normalCount int64, overCount int64, secondCount int64) {
	return memoryGroupModelLimiter.PushRequest(group, model, maxReq, time.Minute, 1)
}

func memoryGetRequest(group, model string) (int64, int64, error) {
	totalCount, secondCount := memoryGroupModelLimiter.GetRequest(group, model, time.Minute)
	return totalCount, secondCount, nil
}

func PushGroupModelRequest(ctx context.Context, group, model string, maxRequestNum int64) (int64, int64, int64) {
	if common.RedisEnabled {
		count, overLimitCount, secondCount, err := redisGroupModelLimiter.PushRequest(ctx, group, model, maxRequestNum, time.Minute, 1)
		if err == nil {
			return count, overLimitCount, secondCount
		}
		log.Error("redis push request error: " + err.Error())
	}
	return memoryPushRequest(group, model, maxRequestNum)
}

func GetGroupModelRequest(ctx context.Context, group, model string) (int64, int64, error) {
	if common.RedisEnabled {
		totalCount, secondCount, err := redisGroupModelLimiter.GetRequest(ctx, group, model)
		if err == nil {
			return totalCount, secondCount, nil
		}
		log.Error("redis get request error: " + err.Error())
	}
	return memoryGetRequest(group, model)
}

var (
	memoryChannelModelRecord = NewInMemoryRecord()
	redisChannelModelRecord  = NewRedisChannelModelRecord()
)

func memoryRecordChannelModelRequest(channel, model string) (int64, int64) {
	count, overLimitCount, secondCount := memoryChannelModelRecord.PushRequest(channel, model, 0, time.Minute, 1)
	return count + overLimitCount, secondCount
}

func memoryGetChannelModelRequest(channel, model string) (int64, int64) {
	totalCount, secondCount := memoryChannelModelRecord.GetRequest(channel, model, time.Minute)
	return totalCount, secondCount
}

func PushChannelModelRequest(ctx context.Context, channel, model string) (int64, int64) {
	if common.RedisEnabled {
		count, overLimitCount, secondCount, err := redisChannelModelRecord.PushRequest(ctx, channel, model, 0, time.Minute, 1)
		if err == nil {
			return count + overLimitCount, secondCount
		}
		log.Error("redis push request error: " + err.Error())
	}
	return memoryRecordChannelModelRequest(channel, model)
}

func GetChannelModelRequest(ctx context.Context, channel, model string) (int64, int64) {
	if common.RedisEnabled {
		totalCount, secondCount, err := redisChannelModelRecord.GetRequest(ctx, channel, model)
		if err == nil {
			return totalCount, secondCount
		}
		log.Error("redis get request error: " + err.Error())
	}
	return memoryGetChannelModelRequest(channel, model)
}

var (
	memoryGroupModelTokensLimiter = NewInMemoryRecord()
	redisGroupModelTokensLimiter  = NewRedisGroupModelTokensRecord()
)

func memoryRecordGroupModelTokensRequest(group, model string, maxTokens int64, tokens int64) (int64, int64) {
	count, overLimitCount, secondCount := memoryGroupModelTokensLimiter.PushRequest(group, model, maxTokens, time.Minute, tokens)
	return count + overLimitCount, secondCount
}

func memoryGetGroupModelTokensRequest(group, model string) (int64, int64) {
	totalCount, secondCount := memoryGroupModelTokensLimiter.GetRequest(group, model, time.Minute)
	return totalCount, secondCount
}

func PushGroupModelTokensRequest(ctx context.Context, group, model string, maxTokens int64, tokens int64) (int64, int64) {
	if common.RedisEnabled {
		count, overLimitCount, secondCount, err := redisGroupModelTokensLimiter.PushRequest(ctx, group, model, maxTokens, time.Minute, tokens)
		if err == nil {
			return count + overLimitCount, secondCount
		}
		log.Error("redis push request error: " + err.Error())
	}
	return memoryRecordGroupModelTokensRequest(group, model, maxTokens, tokens)
}

func GetGroupModelTokensRequest(ctx context.Context, group, model string) (int64, int64) {
	if common.RedisEnabled {
		totalCount, secondCount, err := redisGroupModelTokensLimiter.GetRequest(ctx, group, model)
		if err == nil {
			return totalCount, secondCount
		}
		log.Error("redis get request error: " + err.Error())
	}
	return memoryGetGroupModelTokensRequest(group, model)
}

var (
	memoryChannelModelTokensRecord = NewInMemoryRecord()
	redisChannelModelTokensRecord  = NewRedisChannelModelTokensRecord()
)

func memoryRecordChannelModelTokensRequest(channel, model string, tokens int64) (int64, int64) {
	count, overLimitCount, secondCount := memoryChannelModelTokensRecord.PushRequest(channel, model, 0, time.Minute, tokens)
	return count + overLimitCount, secondCount
}

func memoryGetChannelModelTokensRequest(channel, model string) (int64, int64) {
	totalCount, secondCount := memoryChannelModelTokensRecord.GetRequest(channel, model, time.Minute)
	return totalCount, secondCount
}

func PushChannelModelTokensRequest(ctx context.Context, channel, model string, tokens int64) (int64, int64) {
	if common.RedisEnabled {
		count, overLimitCount, secondCount, err := redisChannelModelTokensRecord.PushRequest(ctx, channel, model, 0, time.Minute, tokens)
		if err == nil {
			return count + overLimitCount, secondCount
		}
		log.Error("redis push request error: " + err.Error())
	}
	return memoryRecordChannelModelTokensRequest(channel, model, tokens)
}

func GetChannelModelTokensRequest(ctx context.Context, channel, model string) (int64, int64) {
	if common.RedisEnabled {
		totalCount, secondCount, err := redisChannelModelTokensRecord.GetRequest(ctx, channel, model)
		if err == nil {
			return totalCount, secondCount
		}
		log.Error("redis get request error: " + err.Error())
	}
	return memoryGetChannelModelTokensRequest(channel, model)
}

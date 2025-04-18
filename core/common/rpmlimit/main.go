package rpmlimit

import (
	"context"
	"time"

	"github.com/labring/aiproxy/core/common"
	log "github.com/sirupsen/logrus"
)

func PushRequest(ctx context.Context, group, model string, maxRequestNum int64, duration time.Duration) (int64, int64, error) {
	if common.RedisEnabled {
		return redisPushRequest(ctx, group, model, maxRequestNum, duration)
	}
	count, overLimitCount := MemoryPushRequest(group, model, maxRequestNum, duration)
	return count, overLimitCount, nil
}

func PushRequestAnyWay(ctx context.Context, group, model string, maxRequestNum int64, duration time.Duration) (int64, int64) {
	if common.RedisEnabled {
		count, overLimitCount, err := redisPushRequest(ctx, group, model, maxRequestNum, duration)
		if err == nil {
			return count, overLimitCount
		}
		log.Error("redis push request error: " + err.Error())
	}
	return MemoryPushRequest(group, model, maxRequestNum, duration)
}

func RateLimit(ctx context.Context, group, model string, maxRequestNum int64, duration time.Duration) (bool, error) {
	if maxRequestNum == 0 {
		return true, nil
	}
	if common.RedisEnabled {
		return redisRateLimitRequest(ctx, group, model, maxRequestNum, duration)
	}
	return MemoryRateLimit(group, model, maxRequestNum, duration), nil
}

func GetRPM(ctx context.Context, group, model string) (int64, error) {
	if common.RedisEnabled {
		return redisGetRPM(ctx, group, model)
	}
	return GetMemoryRPM(group, model)
}

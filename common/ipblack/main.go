package ipblack

import (
	"context"
	"time"

	"github.com/labring/aiproxy/common"
)

func SetIPBlack(ip string, duration time.Duration) {
	if common.RedisEnabled {
		redisSetIPBlack(context.Background(), ip, duration)
	} else {
		memSetIPBlack(ip, duration)
	}
}

func GetIPIsBlock(ctx context.Context, ip string) (bool, error) {
	if common.RedisEnabled {
		return redisGetIPIsBlock(ctx, ip)
	}
	return memGetIPIsBlock(ip)
}

package ipblack

import (
	"context"
	"fmt"
	"time"

	"github.com/labring/aiproxy/core/common"
)

const (
	ipBlackKey = "ip_black:%s"
)

func redisSetIPBlack(ctx context.Context, ip string, duration time.Duration) (bool, error) {
	key := fmt.Sprintf(ipBlackKey, ip)
	success, err := common.RDB.SetNX(ctx, key, duration.Seconds(), duration).Result()
	if err != nil {
		return false, err
	}
	return !success, nil
}

func redisGetIPIsBlock(ctx context.Context, ip string) (bool, error) {
	key := fmt.Sprintf(ipBlackKey, ip)
	exists, err := common.RDB.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

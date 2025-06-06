package common

import (
	"context"
	"os"
	"time"

	"github.com/labring/aiproxy/core/common/env"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

var (
	RDB          *redis.Client
	RedisEnabled = false
)

// InitRedisClient This function is called after init()
func InitRedisClient() (err error) {
	redisConn := env.String("REDIS", os.Getenv("REDIS_CONN_STRING"))
	if redisConn == "" {
		log.Info("REDIS not set, redis is not enabled")
		return nil
	}
	RedisEnabled = true
	log.Info("redis is enabled")
	opt, err := redis.ParseURL(redisConn)
	if err != nil {
		log.Fatal("failed to parse redis connection string: " + err.Error())
	}
	RDB = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = RDB.Ping(ctx).Result()
	if err != nil {
		log.Errorf("failed to ping redis: %s", err.Error())
	}

	return nil
}

func RedisSet(key, value string, expiration time.Duration) error {
	ctx := context.Background()
	return RDB.Set(ctx, key, value, expiration).Err()
}

func RedisGet(key string) (string, error) {
	ctx := context.Background()
	return RDB.Get(ctx, key).Result()
}

func RedisDel(key string) error {
	ctx := context.Background()
	return RDB.Del(ctx, key).Err()
}

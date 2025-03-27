package rpmlimit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/labring/aiproxy/common"
	"github.com/redis/go-redis/v9"
)

const (
	groupModelRPMHashKey = "group_model_rpm_hash:%s:%s"
)

const pushRequestLuaScript = `
local key = KEYS[1]
local window_seconds = tonumber(ARGV[1])
local current_time = tonumber(ARGV[2])
local max_requests = tonumber(ARGV[3])
local cutoff_slice = current_time - window_seconds

local function parse_count(value)
    if not value then return 0, 0 end
    local r, e = value:match("^(%d+):(%d+)$")
    return tonumber(r) or 0, tonumber(e) or 0
end

local count = 0
local over_count = 0

local all_fields = redis.call('HGETALL', key)
for i = 1, #all_fields, 2 do
    local field_slice = tonumber(all_fields[i])
    if field_slice < cutoff_slice then
        redis.call('HDEL', key, all_fields[i])
	else
		local c, oc = parse_count(all_fields[i+1])
		count = count + c
		over_count = over_count + oc
	end
end

if max_requests == 0 or count <= max_requests then
    local current_value = redis.call('HGET', key, tostring(current_time))
    local c, oc = parse_count(current_value)
    redis.call('HSET', key, current_time, (c+1) .. ":" .. oc)
    count = count + 1
else
    local current_value = redis.call('HGET', key, tostring(current_time))
    local c, oc = parse_count(current_value)
    redis.call('HSET', key, current_time, c .. ":" .. (oc+1))
    over_count = over_count + 1
end

redis.call('EXPIRE', key, window_seconds)
return string.format("%d:%d", count, over_count)
`

const getRequestCountLuaScript = `
local pattern = KEYS[1]
local window_seconds = tonumber(ARGV[1])
local current_time = tonumber(ARGV[2])
local cutoff_slice = current_time - window_seconds

local function parse_count(value)
    if not value then return 0, 0 end
    local r, e = value:match("^(%d+):(%d+)$")
    return tonumber(r) or 0, tonumber(e) or 0
end

local total = 0

local keys = redis.call('KEYS', pattern)
for _, key in ipairs(keys) do
    local count = 0
    local over = 0

    local all_fields = redis.call('HGETALL', key)
    for i=1, #all_fields, 2 do
        local field_slice = tonumber(all_fields[i])
        if field_slice < cutoff_slice then
			redis.call('HDEL', key, all_fields[i])
		else
			local c, oc = parse_count(all_fields[i+1])
			count = count + c
			over = over + oc
		end
    end

    total = total + count + over
end

return total
`

var (
	pushRequestScript     = redis.NewScript(pushRequestLuaScript)
	getRequestCountScript = redis.NewScript(getRequestCountLuaScript)
)

func redisGetRPM(ctx context.Context, group, model string) (int64, error) {
	if !common.RedisEnabled {
		return 0, nil
	}

	var pattern string
	switch {
	case model == "":
		model = "*"
		fallthrough
	default:
		pattern = fmt.Sprintf("group_model_rpm_hash:%s:%s", group, model)
	}

	result, err := getRequestCountScript.Run(
		ctx,
		common.RDB,
		[]string{pattern},
		time.Minute.Seconds(),
		time.Now().Unix(),
	).Int64()
	if err != nil {
		return 0, err
	}
	return result, nil
}

func redisPushRequest(ctx context.Context, group, model string, maxRequestNum int64, duration time.Duration) (int64, int64, error) {
	result, err := pushRequestScript.Run(
		ctx,
		common.RDB,
		[]string{fmt.Sprintf(groupModelRPMHashKey, group, model)},
		duration.Seconds(),
		time.Now().Unix(),
		maxRequestNum,
	).Text()
	if err != nil {
		return 0, 0, err
	}
	count, overLimitCount, ok := strings.Cut(result, ":")
	if !ok {
		return 0, 0, errors.New("invalid result")
	}
	countInt, err := strconv.ParseInt(count, 10, 64)
	if err != nil {
		return 0, 0, err
	}
	overLimitCountInt, err := strconv.ParseInt(overLimitCount, 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return countInt, overLimitCountInt, nil
}

func redisRateLimitRequest(ctx context.Context, group, model string, maxRequestNum int64, duration time.Duration) (bool, error) {
	count, _, err := PushRequest(ctx, group, model, maxRequestNum, duration)
	if err != nil {
		return false, err
	}
	return count <= maxRequestNum, nil
}

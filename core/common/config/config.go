package config

import (
	"math"
	"os"
	"slices"
	"strconv"
	"sync/atomic"

	"github.com/labring/aiproxy/core/common/env"
)

var (
	DebugEnabled    = env.Bool("DEBUG", false)
	DebugSQLEnabled = env.Bool("DEBUG_SQL", false)
)

var (
	DisableAutoMigrateDB = env.Bool("DISABLE_AUTO_MIGRATE_DB", false)
	AdminKey             = os.Getenv("ADMIN_KEY")
	WebPath              = os.Getenv("WEB_PATH")
	DisableWeb           = env.Bool("DISABLE_WEB", false)
	FfmpegEnabled        = env.Bool("FFMPEG_ENABLED", false)
)

var (
	disableServe                 atomic.Bool
	logStorageHours              int64 // default 0 means no limit
	retryLogStorageHours         int64 // default 0 means no limit
	saveAllLogDetail             atomic.Bool
	logDetailRequestBodyMaxSize  int64 = 128 * 1024 // 128KB
	logDetailResponseBodyMaxSize int64 = 128 * 1024 // 128KB
	logDetailStorageHours        int64 = 3 * 24     // 3 days
	cleanLogBatchSize            int64 = 2000
	internalToken                atomic.Value
	notifyNote                   atomic.Value
	ipGroupsThreshold            int64
	ipGroupsBanThreshold         int64
)

var (
	retryTimes              atomic.Int64
	enableModelErrorAutoBan atomic.Bool
	modelErrorAutoBanRate   = math.Float64bits(0.3)
	timeoutWithModelType    atomic.Value
	disableModelConfig      = env.Bool("DISABLE_MODEL_CONFIG", false)
)

var (
	defaultChannelModels       atomic.Value
	defaultChannelModelMapping atomic.Value
	groupMaxTokenNum           atomic.Int64
	groupConsumeLevelRatio     atomic.Value
)

var geminiSafetySetting atomic.Value

var billingEnabled atomic.Bool

func init() {
	timeoutWithModelType.Store(make(map[int]int64))
	defaultChannelModels.Store(make(map[int][]string))
	defaultChannelModelMapping.Store(make(map[int]map[string]string))
	groupConsumeLevelRatio.Store(make(map[float64]float64))
	geminiSafetySetting.Store("BLOCK_NONE")
	billingEnabled.Store(true)
	internalToken.Store(os.Getenv("INTERNAL_TOKEN"))
	notifyNote.Store(os.Getenv("NOTIFY_NOTE"))
}

func GetDisableModelConfig() bool {
	return disableModelConfig
}

func GetRetryTimes() int64 {
	return retryTimes.Load()
}

func SetRetryTimes(times int64) {
	times = env.Int64("RETRY_TIMES", times)
	retryTimes.Store(times)
}

func GetEnableModelErrorAutoBan() bool {
	return enableModelErrorAutoBan.Load()
}

func SetEnableModelErrorAutoBan(enabled bool) {
	enabled = env.Bool("ENABLE_MODEL_ERROR_AUTO_BAN", enabled)
	enableModelErrorAutoBan.Store(enabled)
}

func GetModelErrorAutoBanRate() float64 {
	return math.Float64frombits(atomic.LoadUint64(&modelErrorAutoBanRate))
}

func SetModelErrorAutoBanRate(rate float64) {
	rate = env.Float64("MODEL_ERROR_AUTO_BAN_RATE", rate)
	atomic.StoreUint64(&modelErrorAutoBanRate, math.Float64bits(rate))
}

func GetTimeoutWithModelType() map[int]int64 {
	t, _ := timeoutWithModelType.Load().(map[int]int64)
	return t
}

func SetTimeoutWithModelType(timeout map[int]int64) {
	timeout = env.JSON("TIMEOUT_WITH_MODEL_TYPE", timeout)
	timeoutWithModelType.Store(timeout)
}

func GetLogStorageHours() int64 {
	return atomic.LoadInt64(&logStorageHours)
}

func SetLogStorageHours(hours int64) {
	hours = env.Int64("LOG_STORAGE_HOURS", hours)
	atomic.StoreInt64(&logStorageHours, hours)
}

func GetRetryLogStorageHours() int64 {
	return atomic.LoadInt64(&retryLogStorageHours)
}

func SetRetryLogStorageHours(hours int64) {
	hours = env.Int64("RETRY_LOG_STORAGE_HOURS", hours)
	atomic.StoreInt64(&retryLogStorageHours, hours)
}

func GetLogDetailStorageHours() int64 {
	return atomic.LoadInt64(&logDetailStorageHours)
}

func SetLogDetailStorageHours(hours int64) {
	hours = env.Int64("LOG_DETAIL_STORAGE_HOURS", hours)
	atomic.StoreInt64(&logDetailStorageHours, hours)
}

func GetCleanLogBatchSize() int64 {
	return atomic.LoadInt64(&cleanLogBatchSize)
}

func SetCleanLogBatchSize(size int64) {
	size = env.Int64("CLEAN_LOG_BATCH_SIZE", size)
	atomic.StoreInt64(&cleanLogBatchSize, size)
}

func GetIPGroupsThreshold() int64 {
	return atomic.LoadInt64(&ipGroupsThreshold)
}

func SetIPGroupsThreshold(threshold int64) {
	threshold = env.Int64("IP_GROUPS_THRESHOLD", threshold)
	atomic.StoreInt64(&ipGroupsThreshold, threshold)
}

func GetIPGroupsBanThreshold() int64 {
	return atomic.LoadInt64(&ipGroupsBanThreshold)
}

func SetIPGroupsBanThreshold(threshold int64) {
	threshold = env.Int64("IP_GROUPS_BAN_THRESHOLD", threshold)
	atomic.StoreInt64(&ipGroupsBanThreshold, threshold)
}

func GetSaveAllLogDetail() bool {
	return saveAllLogDetail.Load()
}

func SetSaveAllLogDetail(enabled bool) {
	enabled = env.Bool("SAVE_ALL_LOG_DETAIL", enabled)
	saveAllLogDetail.Store(enabled)
}

func GetLogDetailRequestBodyMaxSize() int64 {
	return atomic.LoadInt64(&logDetailRequestBodyMaxSize)
}

func SetLogDetailRequestBodyMaxSize(size int64) {
	size = env.Int64("LOG_DETAIL_REQUEST_BODY_MAX_SIZE", size)
	atomic.StoreInt64(&logDetailRequestBodyMaxSize, size)
}

func GetLogDetailResponseBodyMaxSize() int64 {
	return atomic.LoadInt64(&logDetailResponseBodyMaxSize)
}

func SetLogDetailResponseBodyMaxSize(size int64) {
	size = env.Int64("LOG_DETAIL_RESPONSE_BODY_MAX_SIZE", size)
	atomic.StoreInt64(&logDetailResponseBodyMaxSize, size)
}

func GetDisableServe() bool {
	return disableServe.Load()
}

func SetDisableServe(disabled bool) {
	disabled = env.Bool("DISABLE_SERVE", disabled)
	disableServe.Store(disabled)
}

func GetDefaultChannelModels() map[int][]string {
	d, _ := defaultChannelModels.Load().(map[int][]string)
	return d
}

func SetDefaultChannelModels(models map[int][]string) {
	models = env.JSON("DEFAULT_CHANNEL_MODELS", models)
	for key, ms := range models {
		slices.Sort(ms)
		models[key] = slices.Compact(ms)
	}
	defaultChannelModels.Store(models)
}

func GetDefaultChannelModelMapping() map[int]map[string]string {
	d, _ := defaultChannelModelMapping.Load().(map[int]map[string]string)
	return d
}

func SetDefaultChannelModelMapping(mapping map[int]map[string]string) {
	mapping = env.JSON("DEFAULT_CHANNEL_MODEL_MAPPING", mapping)
	defaultChannelModelMapping.Store(mapping)
}

func GetGroupConsumeLevelRatio() map[float64]float64 {
	r, _ := groupConsumeLevelRatio.Load().(map[float64]float64)
	return r
}

func GetGroupConsumeLevelRatioStringKeyMap() map[string]float64 {
	ratio := GetGroupConsumeLevelRatio()
	stringMap := make(map[string]float64)
	for k, v := range ratio {
		stringMap[strconv.FormatFloat(k, 'f', -1, 64)] = v
	}
	return stringMap
}

func SetGroupConsumeLevelRatio(ratio map[float64]float64) {
	ratio = env.JSON("GROUP_CONSUME_LEVEL_RATIO", ratio)
	groupConsumeLevelRatio.Store(ratio)
}

// GetGroupMaxTokenNum returns max number of tokens per group, 0 means unlimited
func GetGroupMaxTokenNum() int64 {
	return groupMaxTokenNum.Load()
}

func SetGroupMaxTokenNum(num int64) {
	num = env.Int64("GROUP_MAX_TOKEN_NUM", num)
	groupMaxTokenNum.Store(num)
}

func GetGeminiSafetySetting() string {
	s, _ := geminiSafetySetting.Load().(string)
	return s
}

func SetGeminiSafetySetting(setting string) {
	setting = env.String("GEMINI_SAFETY_SETTING", setting)
	geminiSafetySetting.Store(setting)
}

func GetBillingEnabled() bool {
	return billingEnabled.Load()
}

func SetBillingEnabled(enabled bool) {
	enabled = env.Bool("BILLING_ENABLED", enabled)
	billingEnabled.Store(enabled)
}

func GetInternalToken() string {
	t, _ := internalToken.Load().(string)
	return t
}

func SetInternalToken(token string) {
	token = env.String("INTERNAL_TOKEN", token)
	internalToken.Store(token)
}

func GetNotifyNote() string {
	n, _ := notifyNote.Load().(string)
	return n
}

func SetNotifyNote(note string) {
	note = env.String("NOTIFY_NOTE", note)
	notifyNote.Store(note)
}

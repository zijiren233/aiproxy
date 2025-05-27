package model

import (
	"context"
	"encoding"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/config"
	"github.com/labring/aiproxy/core/common/conv"
	"github.com/labring/aiproxy/core/common/notify"
	"github.com/maruel/natural"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

const (
	SyncFrequency    = time.Minute * 3
	TokenCacheKey    = "token:%s"
	GroupCacheKey    = "group:%s"
	GroupModelTPMKey = "group:%s:model_tpm"
)

var (
	_ encoding.BinaryMarshaler = (*redisStringSlice)(nil)
	_ redis.Scanner            = (*redisStringSlice)(nil)
)

type redisStringSlice []string

func (r *redisStringSlice) ScanRedis(value string) error {
	return sonic.Unmarshal(conv.StringToBytes(value), r)
}

func (r redisStringSlice) MarshalBinary() ([]byte, error) {
	return sonic.Marshal(r)
}

type redisTime time.Time

var (
	_ redis.Scanner            = (*redisTime)(nil)
	_ encoding.BinaryMarshaler = (*redisTime)(nil)
)

func (t *redisTime) ScanRedis(value string) error {
	return (*time.Time)(t).UnmarshalBinary(conv.StringToBytes(value))
}

func (t redisTime) MarshalBinary() ([]byte, error) {
	return time.Time(t).MarshalBinary()
}

type TokenCache struct {
	ExpiredAt     redisTime        `json:"expired_at"  redis:"e"`
	Group         string           `json:"group"       redis:"g"`
	Key           string           `json:"-"           redis:"-"`
	Name          string           `json:"name"        redis:"n"`
	Subnets       redisStringSlice `json:"subnets"     redis:"s"`
	Models        redisStringSlice `json:"models"      redis:"m"`
	ID            int              `json:"id"          redis:"i"`
	Status        int              `json:"status"      redis:"st"`
	Quota         float64          `json:"quota"       redis:"q"`
	UsedAmount    float64          `json:"used_amount" redis:"u"`
	availableSets []string
	modelsBySet   map[string][]string
}

func (t *TokenCache) SetAvailableSets(availableSets []string) {
	t.availableSets = availableSets
}

func (t *TokenCache) SetModelsBySet(modelsBySet map[string][]string) {
	t.modelsBySet = modelsBySet
}

func (t *TokenCache) ContainsModel(model string) bool {
	if len(t.Models) != 0 {
		if !slices.Contains(t.Models, model) {
			return false
		}
	}
	return containsModel(model, t.availableSets, t.modelsBySet)
}

func containsModel(model string, sets []string, modelsBySet map[string][]string) bool {
	for _, set := range sets {
		if slices.Contains(modelsBySet[set], model) {
			return true
		}
	}
	return false
}

func (t *TokenCache) Range(fn func(model string) bool) {
	ranged := make(map[string]struct{})
	if len(t.Models) != 0 {
		for _, model := range t.Models {
			if _, ok := ranged[model]; !ok && containsModel(model, t.availableSets, t.modelsBySet) {
				if !fn(model) {
					return
				}
			}
			ranged[model] = struct{}{}
		}
		return
	}

	for _, set := range t.availableSets {
		for _, model := range t.modelsBySet[set] {
			if _, ok := ranged[model]; !ok {
				if !fn(model) {
					return
				}
			}
			ranged[model] = struct{}{}
		}
	}
}

func (t *Token) ToTokenCache() *TokenCache {
	return &TokenCache{
		ID:         t.ID,
		Group:      t.GroupID,
		Key:        t.Key,
		Name:       t.Name.String(),
		Models:     t.Models,
		Subnets:    t.Subnets,
		Status:     t.Status,
		ExpiredAt:  redisTime(t.ExpiredAt),
		Quota:      t.Quota,
		UsedAmount: t.UsedAmount,
	}
}

func CacheDeleteToken(key string) error {
	if !common.RedisEnabled {
		return nil
	}
	return common.RedisDel(fmt.Sprintf(TokenCacheKey, key))
}

//nolint:gosec
func CacheSetToken(token *TokenCache) error {
	if !common.RedisEnabled {
		return nil
	}
	key := fmt.Sprintf(TokenCacheKey, token.Key)
	pipe := common.RDB.Pipeline()
	pipe.HSet(context.Background(), key, token)
	expireTime := SyncFrequency + time.Duration(rand.Int64N(60)-30)*time.Second
	pipe.Expire(context.Background(), key, expireTime)
	_, err := pipe.Exec(context.Background())
	return err
}

func CacheGetTokenByKey(key string) (*TokenCache, error) {
	if !common.RedisEnabled {
		token, err := GetTokenByKey(key)
		if err != nil {
			return nil, err
		}
		return token.ToTokenCache(), nil
	}

	cacheKey := fmt.Sprintf(TokenCacheKey, key)
	tokenCache := &TokenCache{}
	err := common.RDB.HGetAll(context.Background(), cacheKey).Scan(tokenCache)
	if err == nil && tokenCache.ID != 0 {
		tokenCache.Key = key
		return tokenCache, nil
	} else if err != nil && !errors.Is(err, redis.Nil) {
		log.Errorf("get token (%s) from redis error: %s", key, err.Error())
	}

	token, err := GetTokenByKey(key)
	if err != nil {
		return nil, err
	}

	tc := token.ToTokenCache()

	if err := CacheSetToken(tc); err != nil {
		log.Error("redis set token error: " + err.Error())
	}

	return tc, nil
}

var updateTokenUsedAmountOnlyIncreaseScript = redis.NewScript(`
	local used_amount = redis.call("HGet", KEYS[1], "ua")
	if used_amount == false then
		return redis.status_reply("ok")
	end
	if ARGV[1] < used_amount then
		return redis.status_reply("ok")
	end
	redis.call("HSet", KEYS[1], "ua", ARGV[1])
	return redis.status_reply("ok")
`)

func CacheUpdateTokenUsedAmountOnlyIncrease(key string, amount float64) error {
	if !common.RedisEnabled {
		return nil
	}
	return updateTokenUsedAmountOnlyIncreaseScript.Run(context.Background(), common.RDB, []string{fmt.Sprintf(TokenCacheKey, key)}, amount).Err()
}

var updateTokenNameScript = redis.NewScript(`
	if redis.call("HExists", KEYS[1], "n") then
		redis.call("HSet", KEYS[1], "n", ARGV[1])
	end
	return redis.status_reply("ok")
`)

func CacheUpdateTokenName(key string, name string) error {
	if !common.RedisEnabled {
		return nil
	}
	return updateTokenNameScript.Run(context.Background(), common.RDB, []string{fmt.Sprintf(TokenCacheKey, key)}, name).Err()
}

var updateTokenStatusScript = redis.NewScript(`
	if redis.call("HExists", KEYS[1], "st") then
		redis.call("HSet", KEYS[1], "st", ARGV[1])
	end
	return redis.status_reply("ok")
`)

func CacheUpdateTokenStatus(key string, status int) error {
	if !common.RedisEnabled {
		return nil
	}
	return updateTokenStatusScript.Run(context.Background(), common.RDB, []string{fmt.Sprintf(TokenCacheKey, key)}, status).Err()
}

type redisGroupModelConfigMap map[string]GroupModelConfig

var (
	_ redis.Scanner            = (*redisGroupModelConfigMap)(nil)
	_ encoding.BinaryMarshaler = (*redisGroupModelConfigMap)(nil)
)

func (r *redisGroupModelConfigMap) ScanRedis(value string) error {
	return sonic.UnmarshalString(value, r)
}

func (r redisGroupModelConfigMap) MarshalBinary() ([]byte, error) {
	return sonic.Marshal(r)
}

type GroupCache struct {
	ID            string                   `json:"-"              redis:"-"`
	Status        int                      `json:"status"         redis:"st"`
	UsedAmount    float64                  `json:"used_amount"    redis:"ua"`
	RPMRatio      float64                  `json:"rpm_ratio"      redis:"rpm_r"`
	TPMRatio      float64                  `json:"tpm_ratio"      redis:"tpm_r"`
	AvailableSets redisStringSlice         `json:"available_sets" redis:"ass"`
	ModelConfigs  redisGroupModelConfigMap `json:"model_configs"  redis:"mc"`

	BalanceAlertEnabled   bool    `json:"balance_alert_enabled"   redis:"bae"`
	BalanceAlertThreshold float64 `json:"balance_alert_threshold" redis:"bat"`
}

func (g *GroupCache) GetAvailableSets() []string {
	if len(g.AvailableSets) == 0 {
		return []string{ChannelDefaultSet}
	}
	return g.AvailableSets
}

func (g *Group) ToGroupCache() *GroupCache {
	modelConfigs := make(redisGroupModelConfigMap, len(g.GroupModelConfigs))
	for _, modelConfig := range g.GroupModelConfigs {
		modelConfigs[modelConfig.Model] = modelConfig
	}
	return &GroupCache{
		ID:            g.ID,
		Status:        g.Status,
		UsedAmount:    g.UsedAmount,
		RPMRatio:      g.RPMRatio,
		TPMRatio:      g.TPMRatio,
		AvailableSets: g.AvailableSets,
		ModelConfigs:  modelConfigs,

		BalanceAlertEnabled:   g.BalanceAlertEnabled,
		BalanceAlertThreshold: g.BalanceAlertThreshold,
	}
}

func CacheDeleteGroup(id string) error {
	if !common.RedisEnabled {
		return nil
	}
	return common.RedisDel(fmt.Sprintf(GroupCacheKey, id))
}

var updateGroupRPMRatioScript = redis.NewScript(`
	if redis.call("HExists", KEYS[1], "rpm_r") then
		redis.call("HSet", KEYS[1], "rpm_r", ARGV[1])
	end
	return redis.status_reply("ok")
`)

func CacheUpdateGroupRPMRatio(id string, rpmRatio float64) error {
	if !common.RedisEnabled {
		return nil
	}
	return updateGroupRPMRatioScript.Run(context.Background(), common.RDB, []string{fmt.Sprintf(GroupCacheKey, id)}, rpmRatio).Err()
}

var updateGroupTPMRatioScript = redis.NewScript(`
	if redis.call("HExists", KEYS[1], "tpm_r") then
		redis.call("HSet", KEYS[1], "tpm_r", ARGV[1])
	end
	return redis.status_reply("ok")
`)

func CacheUpdateGroupTPMRatio(id string, tpmRatio float64) error {
	if !common.RedisEnabled {
		return nil
	}
	return updateGroupTPMRatioScript.Run(context.Background(), common.RDB, []string{fmt.Sprintf(GroupCacheKey, id)}, tpmRatio).Err()
}

var updateGroupStatusScript = redis.NewScript(`
	if redis.call("HExists", KEYS[1], "st") then
		redis.call("HSet", KEYS[1], "st", ARGV[1])
	end
	return redis.status_reply("ok")
`)

func CacheUpdateGroupStatus(id string, status int) error {
	if !common.RedisEnabled {
		return nil
	}
	return updateGroupStatusScript.Run(context.Background(), common.RDB, []string{fmt.Sprintf(GroupCacheKey, id)}, status).Err()
}

//nolint:gosec
func CacheSetGroup(group *GroupCache) error {
	if !common.RedisEnabled {
		return nil
	}
	key := fmt.Sprintf(GroupCacheKey, group.ID)
	pipe := common.RDB.Pipeline()
	pipe.HSet(context.Background(), key, group)
	expireTime := SyncFrequency + time.Duration(rand.Int64N(60)-30)*time.Second
	pipe.Expire(context.Background(), key, expireTime)
	_, err := pipe.Exec(context.Background())
	return err
}

func CacheGetGroup(id string) (*GroupCache, error) {
	if !common.RedisEnabled {
		group, err := GetGroupByID(id, true)
		if err != nil {
			return nil, err
		}
		return group.ToGroupCache(), nil
	}

	cacheKey := fmt.Sprintf(GroupCacheKey, id)
	groupCache := &GroupCache{}
	err := common.RDB.HGetAll(context.Background(), cacheKey).Scan(groupCache)
	if err == nil && groupCache.Status != 0 {
		groupCache.ID = id
		return groupCache, nil
	} else if err != nil && !errors.Is(err, redis.Nil) {
		log.Errorf("get group (%s) from redis error: %s", id, err.Error())
	}

	group, err := GetGroupByID(id, true)
	if err != nil {
		return nil, err
	}

	gc := group.ToGroupCache()

	if err := CacheSetGroup(gc); err != nil {
		log.Error("redis set group error: " + err.Error())
	}

	return gc, nil
}

var updateGroupUsedAmountOnlyIncreaseScript = redis.NewScript(`
	local used_amount = redis.call("HGet", KEYS[1], "ua")
	if used_amount == false then
		return redis.status_reply("ok")
	end
	if ARGV[1] < used_amount then
		return redis.status_reply("ok")
	end
	redis.call("HSet", KEYS[1], "ua", ARGV[1])
	return redis.status_reply("ok")
`)

func CacheUpdateGroupUsedAmountOnlyIncrease(id string, amount float64) error {
	if !common.RedisEnabled {
		return nil
	}
	return updateGroupUsedAmountOnlyIncreaseScript.Run(context.Background(), common.RDB, []string{fmt.Sprintf(GroupCacheKey, id)}, amount).Err()
}

//nolint:revive
type ModelConfigCache interface {
	GetModelConfig(model string) (*ModelConfig, bool)
}

// read-only cache
//
//nolint:revive
type ModelCaches struct {
	ModelConfig ModelConfigCache

	// map[set][]model
	EnabledModelsBySet map[string][]string
	// map[set][]modelconfig
	EnabledModelConfigsBySet map[string][]*ModelConfig
	// map[model]modelconfig
	EnabledModelConfigsMap map[string]*ModelConfig

	// map[set]map[model][]channel
	EnabledModel2ChannelsBySet map[string]map[string][]*Channel
	// map[set]map[model][]channel
	DisabledModel2ChannelsBySet map[string]map[string][]*Channel
}

var modelCaches atomic.Pointer[ModelCaches]

func init() {
	modelCaches.Store(new(ModelCaches))
}

func LoadModelCaches() *ModelCaches {
	return modelCaches.Load()
}

// InitModelConfigAndChannelCache initializes the channel cache from database
func InitModelConfigAndChannelCache() error {
	modelConfig, err := initializeModelConfigCache()
	if err != nil {
		return err
	}

	// Load enabled channels from database
	enabledChannels, err := LoadEnabledChannels()
	if err != nil {
		return err
	}

	// Build model to channels map by set
	enabledModel2ChannelsBySet := buildModelToChannelsBySetMap(enabledChannels)

	// Sort channels by priority within each set
	sortChannelsByPriorityBySet(enabledModel2ChannelsBySet)

	// Build enabled models and configs by set
	enabledModelsBySet, enabledModelConfigsBySet, enabledModelConfigsMap := buildEnabledModelsBySet(enabledModel2ChannelsBySet, modelConfig)

	// Load disabled channels
	disabledChannels, err := LoadDisabledChannels()
	if err != nil {
		return err
	}

	// Build disabled model to channels map by set
	disabledModel2ChannelsBySet := buildModelToChannelsBySetMap(disabledChannels)

	// Update global cache atomically
	modelCaches.Store(&ModelCaches{
		ModelConfig: modelConfig,

		EnabledModelsBySet:       enabledModelsBySet,
		EnabledModelConfigsBySet: enabledModelConfigsBySet,
		EnabledModelConfigsMap:   enabledModelConfigsMap,

		EnabledModel2ChannelsBySet:  enabledModel2ChannelsBySet,
		DisabledModel2ChannelsBySet: disabledModel2ChannelsBySet,
	})

	return nil
}

func LoadEnabledChannels() ([]*Channel, error) {
	var channels []*Channel
	err := DB.Where("status = ?", ChannelStatusEnabled).Find(&channels).Error
	if err != nil {
		return nil, err
	}

	for _, channel := range channels {
		initializeChannelModels(channel)
		initializeChannelModelMapping(channel)
	}

	return channels, nil
}

func LoadDisabledChannels() ([]*Channel, error) {
	var channels []*Channel
	err := DB.Where("status = ?", ChannelStatusDisabled).Find(&channels).Error
	if err != nil {
		return nil, err
	}

	for _, channel := range channels {
		initializeChannelModels(channel)
		initializeChannelModelMapping(channel)
	}

	return channels, nil
}

func LoadChannels() ([]*Channel, error) {
	var channels []*Channel
	err := DB.Find(&channels).Error
	if err != nil {
		return nil, err
	}

	for _, channel := range channels {
		initializeChannelModels(channel)
		initializeChannelModelMapping(channel)
	}

	return channels, nil
}

func LoadChannelByID(id int) (*Channel, error) {
	var channel Channel
	err := DB.First(&channel, id).Error
	if err != nil {
		return nil, err
	}

	initializeChannelModels(&channel)
	initializeChannelModelMapping(&channel)

	return &channel, nil
}

var _ ModelConfigCache = (*modelConfigMapCache)(nil)

type modelConfigMapCache struct {
	modelConfigMap map[string]*ModelConfig
}

func (m *modelConfigMapCache) GetModelConfig(model string) (*ModelConfig, bool) {
	config, ok := m.modelConfigMap[model]
	return config, ok
}

var _ ModelConfigCache = (*disabledModelConfigCache)(nil)

type disabledModelConfigCache struct {
	modelConfigs ModelConfigCache
}

func (d *disabledModelConfigCache) GetModelConfig(model string) (*ModelConfig, bool) {
	if config, ok := d.modelConfigs.GetModelConfig(model); ok {
		return config, true
	}
	return NewDefaultModelConfig(model), true
}

func initializeModelConfigCache() (ModelConfigCache, error) {
	modelConfigs, err := GetAllModelConfigs()
	if err != nil {
		return nil, err
	}
	newModelConfigMap := make(map[string]*ModelConfig)
	for _, modelConfig := range modelConfigs {
		newModelConfigMap[modelConfig.Model] = modelConfig
	}

	configs := &modelConfigMapCache{modelConfigMap: newModelConfigMap}
	if config.GetDisableModelConfig() {
		return &disabledModelConfigCache{modelConfigs: configs}, nil
	}
	return configs, nil
}

func initializeChannelModels(channel *Channel) {
	if len(channel.Models) == 0 {
		channel.Models = config.GetDefaultChannelModels()[int(channel.Type)]
		return
	}

	findedModels, missingModels, err := GetModelConfigWithModels(channel.Models)
	if err != nil {
		return
	}

	if len(missingModels) > 0 {
		slices.Sort(missingModels)
		log.Errorf("model config not found: %v", missingModels)
	}
	slices.Sort(findedModels)
	channel.Models = findedModels
}

func initializeChannelModelMapping(channel *Channel) {
	if len(channel.ModelMapping) == 0 {
		channel.ModelMapping = config.GetDefaultChannelModelMapping()[int(channel.Type)]
	}
}

func buildModelToChannelsBySetMap(channels []*Channel) map[string]map[string][]*Channel {
	modelMapBySet := make(map[string]map[string][]*Channel)

	for _, channel := range channels {
		sets := channel.GetSets()
		for _, set := range sets {
			if _, ok := modelMapBySet[set]; !ok {
				modelMapBySet[set] = make(map[string][]*Channel)
			}

			for _, model := range channel.Models {
				modelMapBySet[set][model] = append(modelMapBySet[set][model], channel)
			}
		}
	}

	return modelMapBySet
}

func sortChannelsByPriorityBySet(modelMapBySet map[string]map[string][]*Channel) {
	for _, modelMap := range modelMapBySet {
		for _, channels := range modelMap {
			sort.Slice(channels, func(i, j int) bool {
				return channels[i].GetPriority() > channels[j].GetPriority()
			})
		}
	}
}

func buildEnabledModelsBySet(modelMapBySet map[string]map[string][]*Channel, modelConfigCache ModelConfigCache) (
	map[string][]string,
	map[string][]*ModelConfig,
	map[string]*ModelConfig,
) {
	modelsBySet := make(map[string][]string)
	modelConfigsBySet := make(map[string][]*ModelConfig)
	modelConfigsMap := make(map[string]*ModelConfig)

	for set, modelMap := range modelMapBySet {
		models := make([]string, 0)
		configs := make([]*ModelConfig, 0)
		appended := make(map[string]struct{})

		for model := range modelMap {
			if _, ok := appended[model]; ok {
				continue
			}

			if config, ok := modelConfigCache.GetModelConfig(model); ok {
				models = append(models, model)
				configs = append(configs, config)
				appended[model] = struct{}{}
				modelConfigsMap[model] = config
			}
		}

		slices.Sort(models)
		slices.SortStableFunc(configs, SortModelConfigsFunc)

		modelsBySet[set] = models
		modelConfigsBySet[set] = configs
	}

	return modelsBySet, modelConfigsBySet, modelConfigsMap
}

func SortModelConfigsFunc(i, j *ModelConfig) int {
	if i.Owner != j.Owner {
		if natural.Less(string(i.Owner), string(j.Owner)) {
			return -1
		}
		return 1
	}
	if i.Type != j.Type {
		if i.Type < j.Type {
			return -1
		}
		return 1
	}
	if i.Model == j.Model {
		return 0
	}
	if natural.Less(i.Model, j.Model) {
		return -1
	}
	return 1
}

func SyncModelConfigAndChannelCache(ctx context.Context, wg *sync.WaitGroup, frequency time.Duration) {
	defer wg.Done()

	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := InitModelConfigAndChannelCache()
			if err != nil {
				notify.ErrorThrottle("syncModelChannel", time.Minute, "failed to sync channels", err.Error())
			}
		}
	}
}

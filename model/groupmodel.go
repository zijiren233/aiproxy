package model

import log "github.com/sirupsen/logrus"

const (
	GroupModelConfigCacheKey = "group_model_config"
)

type GroupModelConfig struct {
	GroupID string `gorm:"primaryKey"         json:"group_id"`
	Group   *Group `gorm:"foreignKey:GroupID" json:"-"`
	Model   string `gorm:"primaryKey"         json:"model"`
	RPM     int64  `json:"rpm,omitempty"`
	TPM     int64  `json:"tpm,omitempty"`

	OverwritePrice bool               `json:"overwrite_price,omitempty"`
	ImagePrices    map[string]float64 `gorm:"serializer:fastjson;type:text" json:"image_prices,omitempty"`
	Price          Price              `gorm:"embedded"                      json:"price,omitempty"`
}

func SaveGroupModelConfig(groupModelConfig *GroupModelConfig) (err error) {
	defer func() {
		if err == nil {
			if err := CacheDeleteGroup(groupModelConfig.GroupID); err != nil {
				log.Error("cache delete group failed: " + err.Error())
			}
		}
	}()
	return HandleNotFound(DB.Save(groupModelConfig).Error, GroupModelConfigCacheKey)
}

func DeleteGroupModelConfig(groupID, model string) error {
	err := DB.
		Where("group_id = ? AND model = ?", groupID, model).
		Delete(&GroupModelConfig{}).
		Error
	return HandleNotFound(err, GroupModelConfigCacheKey)
}

func GetGroupModelConfigs(groupID string) ([]GroupModelConfig, error) {
	var groupModelConfigs []GroupModelConfig
	err := DB.Where("group_id = ?", groupID).Find(&groupModelConfigs).Error
	return groupModelConfigs, HandleNotFound(err, GroupModelConfigCacheKey)
}

func GetGroupModelConfig(groupID, model string) (*GroupModelConfig, error) {
	var groupModelConfig GroupModelConfig
	err := DB.Where("group_id = ? AND model = ?", groupID, model).First(&groupModelConfig).Error
	return &groupModelConfig, HandleNotFound(err, GroupModelConfigCacheKey)
}

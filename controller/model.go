package controller

import (
	"net/http"
	"slices"
	"sort"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common/config"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/channeltype"
	log "github.com/sirupsen/logrus"
)

// https://platform.openai.com/docs/api-reference/models/list

type OpenAIModelPermission struct {
	Group              *string `json:"group"`
	ID                 string  `json:"id"`
	Object             string  `json:"object"`
	Organization       string  `json:"organization"`
	Created            int     `json:"created"`
	AllowCreateEngine  bool    `json:"allow_create_engine"`
	AllowSampling      bool    `json:"allow_sampling"`
	AllowLogprobs      bool    `json:"allow_logprobs"`
	AllowSearchIndices bool    `json:"allow_search_indices"`
	AllowView          bool    `json:"allow_view"`
	AllowFineTuning    bool    `json:"allow_fine_tuning"`
	IsBlocking         bool    `json:"is_blocking"`
}

type OpenAIModels struct {
	Parent     *string                 `json:"parent"`
	ID         string                  `json:"id"`
	Object     string                  `json:"object"`
	OwnedBy    string                  `json:"owned_by"`
	Root       string                  `json:"root"`
	Permission []OpenAIModelPermission `json:"permission"`
	Created    int                     `json:"created"`
}

type BuiltinModelConfig model.ModelConfig

func (c *BuiltinModelConfig) MarshalJSON() ([]byte, error) {
	type Alias BuiltinModelConfig
	return sonic.Marshal(&struct {
		*Alias
		CreatedAt int64 `json:"created_at,omitempty"`
		UpdatedAt int64 `json:"updated_at,omitempty"`
	}{
		Alias: (*Alias)(c),
	})
}

func SortBuiltinModelConfigsFunc(i, j *BuiltinModelConfig) int {
	return model.SortModelConfigsFunc((*model.ModelConfig)(i), (*model.ModelConfig)(j))
}

var (
	builtinModels           []*BuiltinModelConfig
	builtinModelsMap        map[string]*OpenAIModels
	builtinChannelID2Models map[int][]*BuiltinModelConfig
)

var permission = []OpenAIModelPermission{
	{
		ID:                 "modelperm-LwHkVFn8AcMItP432fKKDIKJ",
		Object:             "model_permission",
		Created:            1626777600,
		AllowCreateEngine:  true,
		AllowSampling:      true,
		AllowLogprobs:      true,
		AllowSearchIndices: false,
		AllowView:          true,
		AllowFineTuning:    false,
		Organization:       "*",
		Group:              nil,
		IsBlocking:         false,
	},
}

func init() {
	builtinChannelID2Models = make(map[int][]*BuiltinModelConfig)
	builtinModelsMap = make(map[string]*OpenAIModels)
	// https://platform.openai.com/docs/models/model-endpoint-compatibility
	for i, adaptor := range channeltype.ChannelAdaptor {
		modelNames := adaptor.GetModelList()
		builtinChannelID2Models[i] = make([]*BuiltinModelConfig, len(modelNames))
		for idx, _model := range modelNames {
			if _model.Owner == "" {
				_model.Owner = model.ModelOwner(adaptor.GetChannelName())
			}
			if v, ok := builtinModelsMap[_model.Model]; !ok {
				builtinModelsMap[_model.Model] = &OpenAIModels{
					ID:         _model.Model,
					Object:     "model",
					Created:    1626777600,
					OwnedBy:    string(_model.Owner),
					Permission: permission,
					Root:       _model.Model,
					Parent:     nil,
				}
				builtinModels = append(builtinModels, (*BuiltinModelConfig)(_model))
			} else if v.OwnedBy != string(_model.Owner) {
				log.Fatalf("model %s owner mismatch, expect %s, actual %s", _model.Model, string(_model.Owner), v.OwnedBy)
			}
			builtinChannelID2Models[i][idx] = (*BuiltinModelConfig)(_model)
		}
	}
	for _, models := range builtinChannelID2Models {
		sort.Slice(models, func(i, j int) bool {
			return models[i].Model < models[j].Model
		})
		slices.SortStableFunc(models, SortBuiltinModelConfigsFunc)
	}
	slices.SortStableFunc(builtinModels, SortBuiltinModelConfigsFunc)
}

// BuiltinModels godoc
//
//	@Summary		Get builtin models
//	@Description	Returns a list of builtin models
//	@Tags			model
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	middleware.APIResponse{data=[]BuiltinModelConfig}
//	@Router			/api/models/builtin [get]
func BuiltinModels(c *gin.Context) {
	middleware.SuccessResponse(c, builtinModels)
}

// ChannelBuiltinModels godoc
//
//	@Summary		Get channel builtin models
//	@Description	Returns a list of channel builtin models
//	@Tags			model
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	middleware.APIResponse{data=map[int][]BuiltinModelConfig}
//	@Router			/api/models/builtin/channel [get]
func ChannelBuiltinModels(c *gin.Context) {
	middleware.SuccessResponse(c, builtinChannelID2Models)
}

// ChannelBuiltinModelsByType godoc
//
//	@Summary		Get channel builtin models by type
//	@Description	Returns a list of channel builtin models by type
//	@Tags			model
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			type	path		string	true	"Channel type"
//	@Success		200		{object}	middleware.APIResponse{data=[]BuiltinModelConfig}
//	@Router			/api/models/builtin/channel/{type} [get]
func ChannelBuiltinModelsByType(c *gin.Context) {
	channelType := c.Param("type")
	if channelType == "" {
		middleware.ErrorResponse(c, http.StatusOK, "type is required")
		return
	}
	channelTypeInt, err := strconv.Atoi(channelType)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "invalid type")
		return
	}
	middleware.SuccessResponse(c, builtinChannelID2Models[channelTypeInt])
}

// ChannelDefaultModelsAndMapping godoc
//
//	@Summary		Get channel default models and mapping
//	@Description	Returns a list of channel default models and mapping
//	@Tags			model
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	middleware.APIResponse{data=map[string]any{models=[]string,mapping=map[string]string}}
//	@Router			/api/models/default [get]
func ChannelDefaultModelsAndMapping(c *gin.Context) {
	middleware.SuccessResponse(c, gin.H{
		"models":  config.GetDefaultChannelModels(),
		"mapping": config.GetDefaultChannelModelMapping(),
	})
}

// ChannelDefaultModelsAndMappingByType godoc
//
//	@Summary		Get channel default models and mapping by type
//	@Description	Returns a list of channel default models and mapping by type
//	@Tags			model
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			type	path		string	true	"Channel type"
//	@Success		200		{object}	middleware.APIResponse{data=map[string]any{models=[]string,mapping=map[string]string}}
//	@Router			/api/models/default/{type} [get]
func ChannelDefaultModelsAndMappingByType(c *gin.Context) {
	channelType := c.Param("type")
	if channelType == "" {
		middleware.ErrorResponse(c, http.StatusOK, "type is required")
		return
	}
	channelTypeInt, err := strconv.Atoi(channelType)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "invalid type")
		return
	}
	middleware.SuccessResponse(c, gin.H{
		"models":  config.GetDefaultChannelModels()[channelTypeInt],
		"mapping": config.GetDefaultChannelModelMapping()[channelTypeInt],
	})
}

// EnabledModels godoc
//
//	@Summary		Get enabled models
//	@Description	Returns a list of enabled models
//	@Tags			model
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	middleware.APIResponse{data=[]model.ModelConfig}
//	@Router			/api/models/enabled [get]
func EnabledModels(c *gin.Context) {
	middleware.SuccessResponse(c, model.LoadModelCaches().EnabledModelConfigs)
}

// ChannelEnabledModels godoc
//
//	@Summary		Get channel enabled models
//	@Description	Returns a list of channel enabled models
//	@Tags			model
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	middleware.APIResponse{data=map[int][]model.ModelConfig}
//	@Router			/api/models/enabled/channel [get]
func ChannelEnabledModels(c *gin.Context) {
	middleware.SuccessResponse(c, model.LoadModelCaches().EnabledChannelType2ModelConfigs)
}

// ChannelEnabledModelsByType godoc
//
//	@Summary		Get channel enabled models by type
//	@Description	Returns a list of channel enabled models by type
//	@Tags			model
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			type	path		string	true	"Channel type"
//	@Success		200		{object}	middleware.APIResponse{data=[]model.ModelConfig}
//	@Router			/api/models/enabled/channel/{type} [get]
func ChannelEnabledModelsByType(c *gin.Context) {
	channelTypeStr := c.Param("type")
	if channelTypeStr == "" {
		middleware.ErrorResponse(c, http.StatusOK, "type is required")
		return
	}
	channelTypeInt, err := strconv.Atoi(channelTypeStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "invalid type")
		return
	}
	middleware.SuccessResponse(c, model.LoadModelCaches().EnabledChannelType2ModelConfigs[channelTypeInt])
}

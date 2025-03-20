package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common"
	"github.com/labring/aiproxy/common/notify"
	"github.com/labring/aiproxy/common/render"
	"github.com/labring/aiproxy/common/trylock"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/monitor"
	"github.com/labring/aiproxy/relay/channeltype"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/relaymode"
	"github.com/labring/aiproxy/relay/utils"
	log "github.com/sirupsen/logrus"
)

const channelTestRequestID = "channel-test"

var (
	modelConfigCache     map[string]*model.ModelConfig = make(map[string]*model.ModelConfig)
	modelConfigCacheOnce sync.Once
)

func guessModelConfig(model string) *model.ModelConfig {
	modelConfigCacheOnce.Do(func() {
		for _, c := range channeltype.ChannelAdaptor {
			for _, m := range c.GetModelList() {
				if _, ok := modelConfigCache[m.Model]; !ok {
					modelConfigCache[m.Model] = m
				}
			}
		}
	})

	if cachedConfig, ok := modelConfigCache[model]; ok {
		return cachedConfig
	}
	return nil
}

// testSingleModel tests a single model in the channel
func testSingleModel(mc *model.ModelCaches, channel *model.Channel, modelName string) (*model.ChannelTest, error) {
	modelConfig, ok := mc.ModelConfig.GetModelConfig(modelName)
	if !ok {
		return nil, errors.New(modelName + " model config not found")
	}
	if modelConfig.Type == relaymode.Unknown {
		newModelConfig := guessModelConfig(modelName)
		if newModelConfig != nil {
			modelConfig = newModelConfig
		}
	}

	if modelConfig.ExcludeFromTests {
		return &model.ChannelTest{
			TestAt:      time.Now(),
			Model:       modelName,
			ActualModel: modelName,
			Success:     true,
			Code:        http.StatusOK,
			Mode:        modelConfig.Type,
			ChannelName: channel.Name,
			ChannelType: channel.Type,
			ChannelID:   channel.ID,
		}, nil
	}

	body, mode, err := utils.BuildRequest(modelConfig)
	if err != nil {
		return nil, err
	}

	w := httptest.NewRecorder()
	newc, _ := gin.CreateTestContext(w)
	newc.Request = &http.Request{
		URL:    &url.URL{},
		Body:   io.NopCloser(body),
		Header: make(http.Header),
	}
	middleware.SetRequestID(newc, channelTestRequestID)

	meta := meta.NewMeta(
		channel,
		mode,
		modelName,
		modelConfig,
		meta.WithRequestID(channelTestRequestID),
	)
	relayController, ok := relayController(mode)
	if !ok {
		return nil, fmt.Errorf("relay mode %d not implemented", mode)
	}
	result := relayController(meta, newc)
	success := result.Error == nil
	var respStr string
	var code int
	if success {
		switch meta.Mode {
		case relaymode.AudioSpeech,
			relaymode.ImagesGenerations:
			respStr = ""
		default:
			respStr = w.Body.String()
		}
		code = w.Code
	} else {
		respStr = result.Error.JSONOrEmpty()
		code = result.Error.StatusCode
	}

	return channel.UpdateModelTest(
		meta.RequestAt,
		meta.OriginModel,
		meta.ActualModel,
		meta.Mode,
		time.Since(meta.RequestAt).Seconds(),
		success,
		respStr,
		code,
	)
}

// @Summary      Test channel model
// @Description  Tests a single model in the channel
// @Tags         channel
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id  path      int  true  "Channel ID"
// @Param        model  path      string  true  "Model name"
// @Success      200  {object}  middleware.APIResponse{data=model.ChannelTest}
// @Router       /api/channel/{id}/{model} [get]
//
//nolint:goconst
func TestChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, middleware.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	modelName := c.Param("model")
	if modelName == "" {
		c.JSON(http.StatusOK, middleware.APIResponse{
			Success: false,
			Message: "model is required",
		})
		return
	}

	channel, err := model.LoadChannelByID(id)
	if err != nil {
		c.JSON(http.StatusOK, middleware.APIResponse{
			Success: false,
			Message: "channel not found",
		})
		return
	}

	if !slices.Contains(channel.Models, modelName) {
		c.JSON(http.StatusOK, middleware.APIResponse{
			Success: false,
			Message: "model not supported by channel",
		})
		return
	}

	ct, err := testSingleModel(model.LoadModelCaches(), channel, modelName)
	if err != nil {
		log.Errorf("failed to test channel %s(%d) model %s: %s", channel.Name, channel.ID, modelName, err.Error())
		c.JSON(http.StatusOK, middleware.APIResponse{
			Success: false,
			Message: fmt.Sprintf("failed to test channel %s(%d) model %s: %s", channel.Name, channel.ID, modelName, err.Error()),
		})
		return
	}

	if c.Query("success_body") != "true" && ct.Success {
		ct.Response = ""
	}

	c.JSON(http.StatusOK, middleware.APIResponse{
		Success: true,
		Data:    ct,
	})
}

type TestResult struct {
	Data    *model.ChannelTest `json:"data,omitempty"`
	Message string             `json:"message,omitempty"`
	Success bool               `json:"success"`
}

func processTestResult(mc *model.ModelCaches, channel *model.Channel, modelName string, returnSuccess bool, successResponseBody bool) *TestResult {
	ct, err := testSingleModel(mc, channel, modelName)

	e := &utils.UnsupportedModelTypeError{}
	if errors.As(err, &e) {
		log.Errorf("model %s not supported test: %s", modelName, err.Error())
		return nil
	}

	result := &TestResult{
		Success: err == nil,
	}
	if err != nil {
		result.Message = fmt.Sprintf("failed to test channel %s(%d) model %s: %s", channel.Name, channel.ID, modelName, err.Error())
		return result
	}

	if !ct.Success {
		result.Data = ct
		return result
	}

	if !returnSuccess {
		return nil
	}

	if !successResponseBody {
		ct.Response = ""
	}
	result.Data = ct
	return result
}

// @Summary      Test channel models
// @Description  Tests all models in the channel
// @Tags         channel
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id  path      int  true  "Channel ID"
// @Success      200  {object}  middleware.APIResponse{data=[]TestResult}
// @Router       /api/channel/{id}/models [get]
func TestChannelModels(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, middleware.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	channel, err := model.LoadChannelByID(id)
	if err != nil {
		c.JSON(http.StatusOK, middleware.APIResponse{
			Success: false,
			Message: "channel not found",
		})
		return
	}

	returnSuccess := c.Query("return_success") == "true"
	successResponseBody := c.Query("success_body") == "true"
	isStream := c.Query("stream") == "true"

	if isStream {
		common.SetEventStreamHeaders(c)
	}

	results := make([]*TestResult, 0)
	resultsMutex := sync.Mutex{}
	hasError := atomic.Bool{}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	models := slices.Clone(channel.Models)
	rand.Shuffle(len(models), func(i, j int) {
		models[i], models[j] = models[j], models[i]
	})

	mc := model.LoadModelCaches()

	for _, modelName := range models {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(model string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			result := processTestResult(mc, channel, model, returnSuccess, successResponseBody)
			if result == nil {
				return
			}
			if !result.Success || (result.Data != nil && !result.Data.Success) {
				hasError.Store(true)
			}
			resultsMutex.Lock()
			if isStream {
				err := render.ObjectData(c, result)
				if err != nil {
					log.Errorf("failed to render result: %s", err.Error())
				}
			} else {
				results = append(results, result)
			}
			resultsMutex.Unlock()
		}(modelName)
	}

	wg.Wait()

	if !hasError.Load() {
		err := model.ClearLastTestErrorAt(channel.ID)
		if err != nil {
			log.Errorf("failed to clear last test error at for channel %s(%d): %s", channel.Name, channel.ID, err.Error())
		}
	}

	if !isStream {
		c.JSON(http.StatusOK, middleware.APIResponse{
			Success: true,
			Data:    results,
		})
	}
}

// @Summary      Test all channels
// @Description  Tests all channels
// @Tags         channel
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  middleware.APIResponse{data=[]TestResult}
// @Router       /api/channels/test [get]
func TestAllChannels(c *gin.Context) {
	testDisabled := c.Query("test_disabled") == "true"
	var channels []*model.Channel
	var err error
	if testDisabled {
		channels, err = model.LoadChannels()
	} else {
		channels, err = model.LoadEnabledChannels()
	}
	if err != nil {
		c.JSON(http.StatusOK, middleware.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	returnSuccess := c.Query("return_success") == "true"
	successResponseBody := c.Query("success_body") == "true"
	isStream := c.Query("stream") == "true"

	if isStream {
		common.SetEventStreamHeaders(c)
	}

	results := make([]*TestResult, 0)
	resultsMutex := sync.Mutex{}
	hasErrorMap := make(map[int]*atomic.Bool)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	newChannels := slices.Clone(channels)
	rand.Shuffle(len(newChannels), func(i, j int) {
		newChannels[i], newChannels[j] = newChannels[j], newChannels[i]
	})

	mc := model.LoadModelCaches()

	for _, channel := range newChannels {
		channelHasError := &atomic.Bool{}
		hasErrorMap[channel.ID] = channelHasError

		models := slices.Clone(channel.Models)
		rand.Shuffle(len(models), func(i, j int) {
			models[i], models[j] = models[j], models[i]
		})

		for _, modelName := range models {
			wg.Add(1)
			semaphore <- struct{}{}

			go func(model string, ch *model.Channel, hasError *atomic.Bool) {
				defer wg.Done()
				defer func() { <-semaphore }()

				result := processTestResult(mc, ch, model, returnSuccess, successResponseBody)
				if result == nil {
					return
				}
				if !result.Success || (result.Data != nil && !result.Data.Success) {
					hasError.Store(true)
				}
				resultsMutex.Lock()
				if isStream {
					err := render.ObjectData(c, result)
					if err != nil {
						log.Errorf("failed to render result: %s", err.Error())
					}
				} else {
					results = append(results, result)
				}
				resultsMutex.Unlock()
			}(modelName, channel, channelHasError)
		}
	}

	wg.Wait()

	for id, hasError := range hasErrorMap {
		if !hasError.Load() {
			err := model.ClearLastTestErrorAt(id)
			if err != nil {
				log.Errorf("failed to clear last test error at for channel %d: %s", id, err.Error())
			}
		}
	}

	if !isStream {
		c.JSON(http.StatusOK, middleware.APIResponse{
			Success: true,
			Data:    results,
		})
	}
}

func tryTestChannel(channelID int, modelName string) bool {
	return trylock.Lock(fmt.Sprintf("channel_test_lock:%d:%s", channelID, modelName), 30*time.Second)
}

func AutoTestBannedModels() {
	log := log.WithFields(log.Fields{
		"auto_test_banned_models": "true",
	})
	channels, err := monitor.GetAllBannedModelChannels(context.Background())
	if err != nil {
		log.Errorf("failed to get banned channels: %s", err.Error())
		return
	}
	if len(channels) == 0 {
		return
	}

	mc := model.LoadModelCaches()

	for modelName, ids := range channels {
		for _, id := range ids {
			if !tryTestChannel(int(id), modelName) {
				continue
			}
			channel, err := model.LoadChannelByID(int(id))
			if err != nil {
				log.Errorf("failed to get channel by model %s: %s", modelName, err.Error())
				continue
			}
			result, err := testSingleModel(mc, channel, modelName)
			if err != nil {
				notify.Error(fmt.Sprintf("channel[%d] %s(%d) model %s test failed", channel.Type, channel.Name, channel.ID, modelName), err.Error())
				continue
			}
			if result.Success {
				notify.Info(fmt.Sprintf("channel[%d] %s(%d) model %s test success", channel.Type, channel.Name, channel.ID, modelName), "unban it")
				err = monitor.ClearChannelModelErrors(context.Background(), modelName, channel.ID)
				if err != nil {
					log.Errorf("clear channel errors failed: %+v", err)
				}
			} else {
				notify.Error(fmt.Sprintf("channel[%d] %s(%d) model %s test failed", channel.Type, channel.Name, channel.ID, modelName),
					fmt.Sprintf("code: %d, response: %s", result.Code, result.Response))
			}
		}
	}
}

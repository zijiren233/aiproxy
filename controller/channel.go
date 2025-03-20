package controller

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/monitor"
	"github.com/labring/aiproxy/relay/adaptor"
	"github.com/labring/aiproxy/relay/channeltype"
	log "github.com/sirupsen/logrus"
)

// ChannelTypeNames godoc
// @Summary      Get all channel type names
// @Description  Returns a list of all available channel type names
// @Tags         channels
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  middleware.APIResponse{data=map[int]string}
// @Router       /api/channels/type_names [get]
func ChannelTypeNames(c *gin.Context) {
	middleware.SuccessResponse(c, channeltype.ChannelNames)
}

// ChannelTypeMetas godoc
// @Summary      Get channel type metadata
// @Description  Returns metadata for all channel types
// @Tags         channels
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  middleware.APIResponse{data=map[int]channeltype.AdaptorMeta}
// @Router       /api/channels/type_metas [get]
func ChannelTypeMetas(c *gin.Context) {
	middleware.SuccessResponse(c, channeltype.ChannelMetas)
}

// GetChannels godoc
// @Summary      Get channels with pagination
// @Description  Returns a paginated list of channels with optional filters
// @Tags         channels
// @Produce      json
// @Security     ApiKeyAuth
// @Param        page         query    int     false  "Page number"
// @Param        per_page     query    int     false  "Items per page"
// @Param        id           query    int     false  "Filter by id"
// @Param        name         query    string  false  "Filter by name"
// @Param        key          query    string  false  "Filter by key"
// @Param        channel_type query    int     false  "Filter by channel type"
// @Param        base_url     query    string  false  "Filter by base URL"
// @Param        order        query    string  false  "Order by field"
// @Success      200  {object}  middleware.APIResponse{data=map[string]any{channels=[]model.Channel,total=int}}
// @Router       /api/channels [get]
func GetChannels(c *gin.Context) {
	page, perPage := parsePageParams(c)
	id, _ := strconv.Atoi(c.Query("id"))
	name := c.Query("name")
	key := c.Query("key")
	channelType, _ := strconv.Atoi(c.Query("channel_type"))
	baseURL := c.Query("base_url")
	order := c.Query("order")
	channels, total, err := model.GetChannels(page, perPage, id, name, key, channelType, baseURL, order)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, gin.H{
		"channels": channels,
		"total":    total,
	})
}

// GetAllChannels godoc
// @Summary      Get all channels
// @Description  Returns a list of all channels without pagination
// @Tags         channels
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  middleware.APIResponse{data=[]model.Channel}
// @Router       /api/channels/all [get]
func GetAllChannels(c *gin.Context) {
	channels, err := model.GetAllChannels()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, channels)
}

// AddChannels godoc
// @Summary      Add multiple channels
// @Description  Adds multiple channels in a batch operation
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        channels  body      []AddChannelRequest  true  "Channel information"
// @Success      200       {object}  middleware.APIResponse
// @Router       /api/channels [post]
func AddChannels(c *gin.Context) {
	channels := make([]*AddChannelRequest, 0)
	err := c.ShouldBindJSON(&channels)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	_channels := make([]*model.Channel, 0, len(channels))
	for _, channel := range channels {
		channels, err := channel.ToChannels()
		if err != nil {
			middleware.ErrorResponse(c, http.StatusOK, err.Error())
			return
		}
		_channels = append(_channels, channels...)
	}
	err = model.BatchInsertChannels(_channels)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

// SearchChannels godoc
// @Summary      Search channels
// @Description  Search channels with keyword and optional filters
// @Tags         channels
// @Produce      json
// @Security     ApiKeyAuth
// @Param        keyword      query    string  true   "Search keyword"
// @Param        page         query    int     false  "Page number"
// @Param        per_page     query    int     false  "Items per page"
// @Param        id           query    int     false  "Filter by id"
// @Param        name         query    string  false  "Filter by name"
// @Param        key          query    string  false  "Filter by key"
// @Param        channel_type query    int     false  "Filter by channel type"
// @Param        base_url     query    string  false  "Filter by base URL"
// @Param        order        query    string  false  "Order by field"
// @Success      200  {object}  middleware.APIResponse{data=map[string]any{channels=[]model.Channel,total=int}}
// @Router       /api/channels/search [get]
func SearchChannels(c *gin.Context) {
	keyword := c.Query("keyword")
	page, perPage := parsePageParams(c)
	id, _ := strconv.Atoi(c.Query("id"))
	name := c.Query("name")
	key := c.Query("key")
	channelType, _ := strconv.Atoi(c.Query("channel_type"))
	baseURL := c.Query("base_url")
	order := c.Query("order")
	channels, total, err := model.SearchChannels(keyword, page, perPage, id, name, key, channelType, baseURL, order)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, gin.H{
		"channels": channels,
		"total":    total,
	})
}

// GetChannel godoc
// @Summary      Get a channel by ID
// @Description  Returns detailed information about a specific channel
// @Tags         channel
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      int  true  "Channel ID"
// @Success      200  {object}  middleware.APIResponse{data=model.Channel}
// @Router       /api/channel/{id} [get]
func GetChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	channel, err := model.GetChannelByID(id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, channel)
}

// AddChannelRequest represents the request body for adding a channel
type AddChannelRequest struct {
	ModelMapping map[string]string    `json:"model_mapping"`
	Config       *model.ChannelConfig `json:"config"`
	Name         string               `json:"name"`
	Key          string               `json:"key"`
	BaseURL      string               `json:"base_url"`
	Models       []string             `json:"models"`
	Type         int                  `json:"type"`
	Priority     int32                `json:"priority"`
	Status       int                  `json:"status"`
}

func (r *AddChannelRequest) ToChannel() (*model.Channel, error) {
	channelType, ok := channeltype.GetAdaptor(r.Type)
	if !ok {
		return nil, fmt.Errorf("invalid channel type: %d", r.Type)
	}
	if validator, ok := channelType.(adaptor.KeyValidator); ok {
		err := validator.ValidateKey(r.Key)
		if err != nil {
			keyHelp := validator.KeyHelp()
			if keyHelp == "" {
				return nil, fmt.Errorf("%s [%s(%d)] invalid key: %w", r.Name, channeltype.ChannelNames[r.Type], r.Type, err)
			}
			return nil, fmt.Errorf("%s [%s(%d)] invalid key: %w, %s", r.Name, channeltype.ChannelNames[r.Type], r.Type, err, keyHelp)
		}
	}
	return &model.Channel{
		Type:         r.Type,
		Name:         r.Name,
		Key:          r.Key,
		BaseURL:      r.BaseURL,
		Models:       slices.Clone(r.Models),
		ModelMapping: maps.Clone(r.ModelMapping),
		Priority:     r.Priority,
		Status:       r.Status,
		Config:       r.Config,
	}, nil
}

func (r *AddChannelRequest) ToChannels() ([]*model.Channel, error) {
	keys := strings.Split(r.Key, "\n")
	channels := make([]*model.Channel, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		c, err := r.ToChannel()
		if err != nil {
			return nil, err
		}
		c.Key = key
		channels = append(channels, c)
	}
	if len(channels) == 0 {
		ch, err := r.ToChannel()
		if err != nil {
			return nil, err
		}
		return []*model.Channel{ch}, nil
	}
	return channels, nil
}

// AddChannel godoc
// @Summary      Add a single channel
// @Description  Adds a new channel to the system
// @Tags         channel
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        channel  body      AddChannelRequest  true  "Channel information"
// @Success      200      {object}  middleware.APIResponse
// @Router       /api/channel [post]
func AddChannel(c *gin.Context) {
	channel := AddChannelRequest{}
	err := c.ShouldBindJSON(&channel)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	channels, err := channel.ToChannels()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	err = model.BatchInsertChannels(channels)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

// DeleteChannel godoc
// @Summary      Delete a channel
// @Description  Deletes a channel by its ID
// @Tags         channel
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      int  true  "Channel ID"
// @Success      200  {object}  middleware.APIResponse
// @Router       /api/channel/{id} [delete]
func DeleteChannel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteChannelByID(id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

// DeleteChannels godoc
// @Summary      Delete multiple channels
// @Description  Deletes multiple channels by their IDs
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        ids  body      []int  true  "Channel IDs"
// @Success      200  {object}  middleware.APIResponse
// @Router       /api/channels/batch_delete [post]
func DeleteChannels(c *gin.Context) {
	ids := []int{}
	err := c.ShouldBindJSON(&ids)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	err = model.DeleteChannelsByIDs(ids)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

// UpdateChannel godoc
// @Summary      Update a channel
// @Description  Updates an existing channel by its ID
// @Tags         channel
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id       path      int               true  "Channel ID"
// @Param        channel  body      AddChannelRequest true  "Updated channel information"
// @Success      200      {object}  middleware.APIResponse{data=model.Channel}
// @Router       /api/channel/{id} [put]
func UpdateChannel(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		middleware.ErrorResponse(c, http.StatusOK, "id is required")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	channel := AddChannelRequest{}
	err = c.ShouldBindJSON(&channel)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	ch, err := channel.ToChannel()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	ch.ID = id
	err = model.UpdateChannel(ch)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	err = monitor.ClearChannelAllModelErrors(c.Request.Context(), id)
	if err != nil {
		log.Errorf("failed to clear channel all model errors: %+v", err)
	}
	middleware.SuccessResponse(c, ch)
}

// UpdateChannelStatusRequest represents the request body for updating a channel's status
type UpdateChannelStatusRequest struct {
	Status int `json:"status"`
}

// UpdateChannelStatus godoc
// @Summary      Update channel status
// @Description  Updates the status of a channel by its ID
// @Tags         channel
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id      path      int                       true  "Channel ID"
// @Param        status  body      UpdateChannelStatusRequest true  "Status information"
// @Success      200     {object}  middleware.APIResponse
// @Router       /api/channel/{id}/status [post]
func UpdateChannelStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	status := UpdateChannelStatusRequest{}
	err := c.ShouldBindJSON(&status)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	err = model.UpdateChannelStatusByID(id, status.Status)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	err = monitor.ClearChannelAllModelErrors(c.Request.Context(), id)
	if err != nil {
		log.Errorf("failed to clear channel all model errors: %+v", err)
	}
	middleware.SuccessResponse(c, nil)
}

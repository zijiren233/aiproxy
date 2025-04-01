package controller

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common/rpmlimit"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/model"
	"gorm.io/gorm"
)

func getDashboardTime(t string, startTimestamp int64, endTimestamp int64) (time.Time, time.Time, model.TimeSpanType) {
	end := time.Now()
	if endTimestamp != 0 {
		end = time.Unix(endTimestamp, 0)
	}
	var start time.Time
	var timeSpan model.TimeSpanType
	switch t {
	case "month":
		start = end.AddDate(0, 0, -30)
		timeSpan = model.TimeSpanDay
	case "two_week":
		start = end.AddDate(0, 0, -15)
		timeSpan = model.TimeSpanDay
	case "week":
		start = end.AddDate(0, 0, -7)
		timeSpan = model.TimeSpanDay
	case "day":
		fallthrough
	default:
		start = end.AddDate(0, 0, -1)
		timeSpan = model.TimeSpanHour
	}
	if startTimestamp != 0 {
		start = time.Unix(startTimestamp, 0)
	}
	return start, end, timeSpan
}

func fillGaps(data []*model.ChartData, start, end time.Time, t model.TimeSpanType) []*model.ChartData {
	if len(data) == 0 {
		return data
	}

	var timeSpan time.Duration
	switch t {
	case model.TimeSpanDay:
		timeSpan = time.Hour * 24
	default:
		timeSpan = time.Hour
	}

	// Handle first point
	firstPoint := time.Unix(data[0].Timestamp, 0)
	firstAlignedTime := firstPoint
	for !firstAlignedTime.Add(-timeSpan).Before(start) {
		firstAlignedTime = firstAlignedTime.Add(-timeSpan)
	}
	var firstIsZero bool
	if !firstAlignedTime.Equal(firstPoint) {
		data = append([]*model.ChartData{
			{
				Timestamp: firstAlignedTime.Unix(),
			},
		}, data...)
		firstIsZero = true
	}

	// Handle last point
	lastPoint := time.Unix(data[len(data)-1].Timestamp, 0)
	lastAlignedTime := lastPoint
	for !lastAlignedTime.Add(timeSpan).After(end) {
		lastAlignedTime = lastAlignedTime.Add(timeSpan)
	}
	var lastIsZero bool
	if !lastAlignedTime.Equal(lastPoint) {
		data = append(data, &model.ChartData{
			Timestamp: lastAlignedTime.Unix(),
		})
		lastIsZero = true
	}

	result := make([]*model.ChartData, 0, len(data))
	result = append(result, data[0])

	for i := 1; i < len(data); i++ {
		curr := data[i]
		prev := data[i-1]
		hourDiff := (curr.Timestamp - prev.Timestamp) / int64(timeSpan.Seconds())

		// If gap is 1 hour or less, continue
		if hourDiff <= 1 {
			result = append(result, curr)
			continue
		}

		// If gap is more than 3 hours, only add boundary points
		if hourDiff > 3 {
			// Add point for hour after prev
			if i != 1 || (i == 1 && !firstIsZero) {
				result = append(result, &model.ChartData{
					Timestamp: prev.Timestamp + int64(timeSpan.Seconds()),
				})
			}
			// Add point for hour before curr
			if i != len(data)-1 || (i == len(data)-1 && !lastIsZero) {
				result = append(result, &model.ChartData{
					Timestamp: curr.Timestamp - int64(timeSpan.Seconds()),
				})
			}
			result = append(result, curr)
			continue
		}

		// Fill gaps of 2-3 hours with zero points
		for j := prev.Timestamp + int64(timeSpan.Seconds()); j < curr.Timestamp; j += int64(timeSpan.Seconds()) {
			result = append(result, &model.ChartData{
				Timestamp: j,
			})
		}
		result = append(result, curr)
	}

	return result
}

// GetDashboard godoc
//
//	@Summary		Get dashboard data
//	@Description	Returns the general dashboard data including usage statistics and metrics
//	@Tags			dashboard
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			group			query		string	false	"Group or *"
//	@Param			channel			query		int		false	"Channel ID"
//	@Param			type			query		string	false	"Type of time span (day, week, month, two_week)"
//	@Param			model			query		string	false	"Model name"
//	@Param			result_only		query		bool	false	"Only return result"
//	@Param			token_usage		query		bool	false	"Token usage"
//	@Param			start_timestamp	query		int64	false	"Start second timestamp"
//	@Param			end_timestamp	query		int64	false	"End second timestamp"
//	@Param			from_log		query		bool	false	"From log"
//	@Param			timezone		query		string	false	"Timezone"
//	@Success		200				{object}	middleware.APIResponse{data=model.DashboardResponse}
//	@Router			/api/dashboard [get]
func GetDashboard(c *gin.Context) {
	log := middleware.GetLogger(c)

	group := c.Query("group")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	start, end, timeSpan := getDashboardTime(c.Query("type"), startTimestamp, endTimestamp)
	modelName := c.Query("model")
	resultOnly, _ := strconv.ParseBool(c.Query("result_only"))
	tokenUsage, _ := strconv.ParseBool(c.Query("token_usage"))
	channelID, _ := strconv.Atoi(c.Query("channel"))
	fromLog, _ := strconv.ParseBool(c.Query("from_log"))
	timezoneLocation, _ := time.LoadLocation(c.Query("timezone"))
	if timezoneLocation == nil {
		timezoneLocation = time.UTC
	}

	needRPM := channelID != 0

	dashboards, err := model.GetDashboardData(group, start, end, modelName, channelID, timeSpan, resultOnly, needRPM, tokenUsage, fromLog, timezoneLocation)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}

	dashboards.ChartData = fillGaps(dashboards.ChartData, start, end, timeSpan)

	if !needRPM {
		rpm, err := rpmlimit.GetRPM(c.Request.Context(), group, modelName)
		if err != nil {
			log.Errorf("failed to get rpm: %v", err)
		} else {
			dashboards.RPM = rpm
		}
	}

	middleware.SuccessResponse(c, dashboards)
}

// GetGroupDashboard godoc
//
//	@Summary		Get dashboard data for a specific group
//	@Description	Returns dashboard data and metrics specific to the given group
//	@Tags			dashboard
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			group			path		string	true	"Group"
//	@Param			type			query		string	false	"Type of time span (day, week, month, two_week)"
//	@Param			token_name		query		string	false	"Token name"
//	@Param			model			query		string	false	"Model or *"
//	@Param			result_only		query		bool	false	"Only return result"
//	@Param			token_usage		query		bool	false	"Token usage"
//	@Param			start_timestamp	query		int64	false	"Start second timestamp"
//	@Param			end_timestamp	query		int64	false	"End second timestamp"
//	@Param			from_log		query		bool	false	"From log"
//	@Param			timezone		query		string	false	"Timezone"
//	@Success		200				{object}	middleware.APIResponse{data=model.GroupDashboardResponse}
//	@Router			/api/dashboard/{group} [get]
func GetGroupDashboard(c *gin.Context) {
	log := middleware.GetLogger(c)

	group := c.Param("group")
	if group == "" || group == "*" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid group parameter")
		return
	}

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	start, end, timeSpan := getDashboardTime(c.Query("type"), startTimestamp, endTimestamp)
	tokenName := c.Query("token_name")
	modelName := c.Query("model")
	resultOnly, _ := strconv.ParseBool(c.Query("result_only"))
	tokenUsage, _ := strconv.ParseBool(c.Query("token_usage"))
	fromLog, _ := strconv.ParseBool(c.Query("from_log"))
	timezoneLocation, _ := time.LoadLocation(c.Query("timezone"))
	if timezoneLocation == nil {
		timezoneLocation = time.UTC
	}

	needRPM := tokenName != ""

	dashboards, err := model.GetGroupDashboardData(group, start, end, tokenName, modelName, timeSpan, resultOnly, needRPM, tokenUsage, fromLog, timezoneLocation)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "failed to get statistics")
		return
	}

	dashboards.ChartData = fillGaps(dashboards.ChartData, start, end, timeSpan)

	if !needRPM {
		rpm, err := rpmlimit.GetRPM(c.Request.Context(), group, modelName)
		if err != nil {
			log.Errorf("failed to get rpm: %v", err)
		} else {
			dashboards.RPM = rpm
		}
	}

	middleware.SuccessResponse(c, dashboards)
}

// GetGroupDashboardModels godoc
//
//	@Summary		Get model usage data for a specific group
//	@Description	Returns model-specific metrics and usage data for the given group
//	@Tags			dashboard
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			group	path		string	true	"Group"
//	@Success		200		{object}	middleware.APIResponse{data=[]model.ModelConfig}
//	@Router			/api/dashboard/{group}/models [get]
func GetGroupDashboardModels(c *gin.Context) {
	group := c.Param("group")
	if group == "" || group == "*" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid group parameter")
		return
	}
	groupCache, err := model.CacheGetGroup(group)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			middleware.SuccessResponse(c, model.LoadModelCaches().EnabledModelConfigsBySet[model.ChannelDefaultSet])
		} else {
			middleware.ErrorResponse(c, http.StatusOK, fmt.Sprintf("failed to get group: %v", err))
		}
		return
	}

	availableSet := groupCache.GetAvailableSets()
	enabledModelConfigs := model.LoadModelCaches().EnabledModelConfigsBySet
	newEnabledModelConfigs := make([]model.ModelConfig, 0)
	for _, set := range availableSet {
		for _, mc := range enabledModelConfigs[set] {
			if slices.ContainsFunc(newEnabledModelConfigs, func(m model.ModelConfig) bool {
				return m.Model == mc.Model
			}) {
				continue
			}
			newEnabledModelConfigs = append(newEnabledModelConfigs, middleware.GetGroupAdjustedModelConfig(groupCache, *mc))
		}
	}
	middleware.SuccessResponse(c, newEnabledModelConfigs)
}

// GetModelCostRank godoc
//
//	@Summary		Get model cost ranking data
//	@Description	Returns ranking data for models based on cost
//	@Tags			dashboard
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			group			query		string	false	"Group or *"
//	@Param			channel			query		int		false	"Channel ID"
//	@Param			start_timestamp	query		int64	false	"Start timestamp"
//	@Param			end_timestamp	query		int64	false	"End timestamp"
//	@Param			token_usage		query		bool	false	"Token usage"
//	@Success		200				{object}	middleware.APIResponse{data=[]model.ModelCostRank}
//	@Router			/api/model_cost_rank [get]
func GetModelCostRank(c *gin.Context) {
	group := c.Query("group")
	channelID, _ := strconv.Atoi(c.Query("channel"))
	startTime, endTime := parseTimeRange(c)
	tokenUsage, _ := strconv.ParseBool(c.Query("token_usage"))
	models, err := model.GetModelCostRank(group, channelID, startTime, endTime, tokenUsage)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, models)
}

// GetGroupModelCostRank godoc
//
//	@Summary		Get model cost ranking data for a specific group
//	@Description	Returns model cost ranking data specific to the given group
//	@Tags			dashboard
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			group			path		string	true	"Group"
//	@Param			start_timestamp	query		int64	false	"Start timestamp"
//	@Param			end_timestamp	query		int64	false	"End timestamp"
//	@Param			token_usage		query		bool	false	"Token usage"
//	@Success		200				{object}	middleware.APIResponse{data=[]model.ModelCostRank}
//	@Router			/api/model_cost_rank/{group} [get]
func GetGroupModelCostRank(c *gin.Context) {
	group := c.Param("group")
	if group == "" || group == "*" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid group parameter")
		return
	}
	startTime, endTime := parseTimeRange(c)
	tokenUsage, _ := strconv.ParseBool(c.Query("token_usage"))
	models, err := model.GetModelCostRank(group, 0, startTime, endTime, tokenUsage)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, models)
}

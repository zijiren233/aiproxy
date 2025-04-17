package middleware

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common/config"
	"github.com/labring/aiproxy/core/common/network"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/sirupsen/logrus"
)

type APIResponse struct {
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
	Success bool   `json:"success"`
}

func SuccessResponse(c *gin.Context, data any) {
	c.JSON(http.StatusOK, &APIResponse{
		Success: true,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(code, &APIResponse{
		Success: false,
		Message: message,
	})
}

func AdminAuth(c *gin.Context) {
	accessToken := c.Request.Header.Get("Authorization")
	if config.AdminKey != "" && (accessToken == "" || strings.TrimPrefix(accessToken, "Bearer ") != config.AdminKey) {
		ErrorResponse(c, http.StatusUnauthorized, "unauthorized, no access token provided")
		c.Abort()
		return
	}

	group := c.Param("group")
	if group != "" {
		log := GetLogger(c)
		log.Data["gid"] = group
	}

	c.Next()
}

func TokenAuth(c *gin.Context) {
	log := GetLogger(c)
	key := c.Request.Header.Get("Authorization")
	key = strings.TrimPrefix(
		strings.TrimPrefix(key, "Bearer "),
		"sk-",
	)

	var token *model.TokenCache
	var useInternalToken bool
	if config.AdminKey != "" && config.AdminKey == key ||
		config.GetInternalToken() != "" && config.GetInternalToken() == key {
		token = &model.TokenCache{
			Key: key,
		}
		useInternalToken = true
	} else {
		var err error
		token, err = model.ValidateAndGetToken(key)
		if err != nil {
			AbortLogWithMessage(c, http.StatusUnauthorized, err.Error(), &ErrorField{
				Code: "invalid_token",
			})
			return
		}
	}

	SetLogTokenFields(log.Data, token, useInternalToken)

	if len(token.Subnets) > 0 {
		if ok, err := network.IsIPInSubnets(c.ClientIP(), token.Subnets); err != nil {
			AbortLogWithMessage(c, http.StatusInternalServerError, err.Error())
			return
		} else if !ok {
			AbortLogWithMessage(c, http.StatusForbidden,
				fmt.Sprintf("token (%s[%d]) can only be used in the specified subnets: %v, current ip: %s",
					token.Name,
					token.ID,
					token.Subnets,
					c.ClientIP(),
				),
			)
			return
		}
	}

	modelCaches := model.LoadModelCaches()

	var group *model.GroupCache
	if useInternalToken {
		group = &model.GroupCache{
			Status:        model.GroupStatusInternal,
			AvailableSets: slices.AppendSeq(make([]string, 0, len(modelCaches.EnabledModelsBySet)), maps.Keys(modelCaches.EnabledModelsBySet)),
		}
	} else {
		var err error
		group, err = model.CacheGetGroup(token.Group)
		if err != nil {
			AbortLogWithMessage(c, http.StatusInternalServerError, fmt.Sprintf("failed to get group: %v", err))
			return
		}
	}
	SetLogGroupFields(log.Data, group)
	if group.Status != model.GroupStatusEnabled && group.Status != model.GroupStatusInternal {
		AbortLogWithMessage(c, http.StatusForbidden, "group is disabled")
		return
	}

	token.SetAvailableSets(group.GetAvailableSets())
	token.SetModelsBySet(modelCaches.EnabledModelsBySet)

	c.Set(Group, group)
	c.Set(Token, token)
	c.Set(ModelCaches, modelCaches)

	c.Next()
}

func GetGroup(c *gin.Context) *model.GroupCache {
	return c.MustGet(Group).(*model.GroupCache)
}

func GetToken(c *gin.Context) *model.TokenCache {
	return c.MustGet(Token).(*model.TokenCache)
}

func GetModelCaches(c *gin.Context) *model.ModelCaches {
	return c.MustGet(ModelCaches).(*model.ModelCaches)
}

func GetChannel(c *gin.Context) *model.Channel {
	ch, exists := c.Get(Channel)
	if !exists {
		return nil
	}
	return ch.(*model.Channel)
}

func SetLogFieldsFromMeta(m *meta.Meta, fields logrus.Fields) {
	SetLogRequestIDField(fields, m.RequestID)

	SetLogModeField(fields, m.Mode)
	SetLogModelFields(fields, m.OriginModel)
	SetLogActualModelFields(fields, m.ActualModel)

	SetLogGroupFields(fields, m.Group)
	SetLogTokenFields(fields, m.Token, false)
	SetLogChannelFields(fields, m.Channel)
}

func SetLogModeField(fields logrus.Fields, mode mode.Mode) {
	fields["mode"] = mode.String()
}

func SetLogActualModelFields(fields logrus.Fields, actualModel string) {
	fields["actmodel"] = actualModel
}

func SetLogModelFields(fields logrus.Fields, model string) {
	fields["model"] = model
}

func SetLogChannelFields(fields logrus.Fields, channel meta.ChannelMeta) {
	if channel.ID > 0 {
		fields["chid"] = channel.ID
	}
	if channel.Name != "" {
		fields["chname"] = channel.Name
	}
	if channel.Type > 0 {
		fields["chtype"] = channel.Type
	}
	if channel.TypeName != "" {
		fields["chtype_name"] = channel.TypeName
	}
}

func SetLogRequestIDField(fields logrus.Fields, requestID string) {
	fields["reqid"] = requestID
}

func SetLogGroupFields(fields logrus.Fields, group *model.GroupCache) {
	if group == nil {
		return
	}
	if group.ID != "" {
		fields["gid"] = group.ID
	}
}

func SetLogTokenFields(fields logrus.Fields, token *model.TokenCache, internal bool) {
	if token == nil {
		return
	}
	if token.ID > 0 {
		fields["kid"] = token.ID
	}
	if token.Name != "" {
		fields["kname"] = token.Name
	}
	if token.Key != "" {
		fields["key"] = maskTokenKey(token.Key)
	}
	if internal {
		fields["internal"] = "true"
	}
}

func maskTokenKey(key string) string {
	if len(key) <= 8 {
		return "*****"
	}
	return key[:4] + "*****" + key[len(key)-4:]
}

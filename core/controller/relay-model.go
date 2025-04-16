package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/middleware"
	model "github.com/labring/aiproxy/core/relay/model"
)

// ListModels godoc
//
//	@Summary		List models
//	@Description	List all models
//	@Tags			relay
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	object{object=string,data=[]OpenAIModels}
//	@Router			/v1/models [get]
func ListModels(c *gin.Context) {
	enabledModelConfigsMap := middleware.GetModelCaches(c).EnabledModelConfigsMap
	token := middleware.GetToken(c)

	availableOpenAIModels := make([]*OpenAIModels, 0)

	token.Range(func(model string) bool {
		if mc, ok := enabledModelConfigsMap[model]; ok {
			availableOpenAIModels = append(availableOpenAIModels, &OpenAIModels{
				ID:         model,
				Object:     "model",
				Created:    1626777600,
				OwnedBy:    string(mc.Owner),
				Root:       model,
				Permission: permission,
				Parent:     nil,
			})
		}
		return true
	})

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   availableOpenAIModels,
	})
}

// RetrieveModel godoc
//
//	@Summary		Retrieve model
//	@Description	Retrieve a model
//	@Tags			relay
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	OpenAIModels
//	@Router			/v1/models/{model} [get]
func RetrieveModel(c *gin.Context) {
	modelName := c.Param("model")
	enabledModelConfigsMap := middleware.GetModelCaches(c).EnabledModelConfigsMap

	mc, ok := enabledModelConfigsMap[modelName]
	if ok {
		token := middleware.GetToken(c)
		ok = token.ContainsModel(modelName)
	}

	if !ok {
		c.JSON(200, gin.H{
			"error": &model.Error{
				Message: fmt.Sprintf("the model '%s' does not exist", modelName),
				Type:    "invalid_request_error",
				Param:   "model",
				Code:    "model_not_found",
			},
		})
		return
	}

	c.JSON(200, &OpenAIModels{
		ID:         modelName,
		Object:     "model",
		Created:    1626777600,
		OwnedBy:    string(mc.Owner),
		Root:       modelName,
		Permission: permission,
		Parent:     nil,
	})
}

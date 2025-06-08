package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/controller/utils"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

type GroupPublicMCPResponse struct {
	model.PublicMCP
	Reusing map[string]model.ReusingParam `json:"reusing"`
	Params  map[string]string             `json:"params"`
}

func NewGroupPublicMCPResponse(
	mcp model.PublicMCP,
	groupID string,
) (GroupPublicMCPResponse, error) {
	r := GroupPublicMCPResponse{
		PublicMCP: mcp,
	}
	r.PublicMCP.ProxyConfig = nil
	r.PublicMCP.EmbedConfig = nil
	r.PublicMCP.OpenAPIConfig = nil

	switch mcp.Type {
	case model.PublicMCPTypeProxySSE, model.PublicMCPTypeProxyStreamable:
		for _, v := range mcp.ProxyConfig.Reusing {
			r.Reusing[v.Name] = v.ReusingParam
		}
	case model.PublicMCPTypeEmbed:
		r.Reusing = mcp.EmbedConfig.Reusing
	}

	reusingParams, err := model.GetPublicMCPReusingParam(mcp.ID, groupID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return r, err
		}
	}
	r.Params = reusingParams.Params

	return r, nil
}

func NewGroupPublicMCPResponses(
	host string,
	mcps []model.PublicMCP,
	groupID string,
) ([]GroupPublicMCPResponse, error) {
	responses := make([]GroupPublicMCPResponse, len(mcps))
	for i, mcp := range mcps {
		response, err := NewGroupPublicMCPResponse(mcp, groupID)
		if err != nil {
			return nil, err
		}
		responses[i] = response
	}
	return responses, nil
}

// GetGroupPublicMCPs godoc
//
//	@Summary		Get MCPs by group
//	@Description	Get MCPs by group
//	@Tags			mcp
//	@Tags			group
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			group		path		string	true	"Group ID"
//	@Param			page		query		int		false	"Page"
//	@Param			per_page	query		int		false	"Per Page"
//	@Param			type		query		string	false	"Type"
//	@Param			keyword		query		string	false	"Keyword"
//	@Success		200			{object}	middleware.APIResponse{data=[]GroupPublicMCPResponse}
//	@Router			/api/group/{group}/mcp [get]
func GetGroupPublicMCPs(c *gin.Context) {
	groupID := c.Param("group")

	page, perPage := utils.ParsePageParams(c)
	mcpType := model.PublicMCPType(c.Query("type"))
	keyword := c.Query("keyword")

	mcps, total, err := model.GetPublicMCPs(
		page,
		perPage,
		mcpType,
		keyword,
		model.PublicMCPStatusEnabled,
	)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	responses, err := NewGroupPublicMCPResponses(c.Request.Host, mcps, groupID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{
		"mcps":  responses,
		"total": total,
	})
}

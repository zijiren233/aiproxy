package controller

import (
	"maps"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/embedmcp"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
)

type EmbedMCPConfigTemplate struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Example     string `json:"example,omitempty"`
	Description string `json:"description,omitempty"`
}

func newEmbedMCPConfigTemplate(template embedmcp.ConfigTemplate) EmbedMCPConfigTemplate {
	return EmbedMCPConfigTemplate{
		Name:        template.Name,
		Required:    template.Required == embedmcp.ConfigRequiredTypeInitOnly,
		Example:     template.Example,
		Description: template.Description,
	}
}

type EmbedMCPConfigTemplates = map[string]EmbedMCPConfigTemplate

func newEmbedMCPConfigTemplates(templates embedmcp.ConfigTemplates) EmbedMCPConfigTemplates {
	emcpTemplates := make(EmbedMCPConfigTemplates, len(templates))
	for key, template := range templates {
		emcpTemplates[key] = newEmbedMCPConfigTemplate(template)
	}
	return emcpTemplates
}

type EmbedMCP struct {
	ID              string                  `json:"id"`
	Enabled         bool                    `json:"enabled"`
	Name            string                  `json:"name"`
	Readme          string                  `json:"readme"`
	Tags            []string                `json:"tags"`
	ConfigTemplates EmbedMCPConfigTemplates `json:"config_templates"`
}

func newEmbedMCP(mcp *embedmcp.EmbedMcp, enabled bool) *EmbedMCP {
	emcp := &EmbedMCP{
		ID:              mcp.ID,
		Enabled:         enabled,
		Name:            mcp.Name,
		Readme:          mcp.Readme,
		Tags:            mcp.Tags,
		ConfigTemplates: newEmbedMCPConfigTemplates(mcp.ConfigTemplates),
	}
	return emcp
}

// GetEmbedMCPs godoc
//
//	@Summary		Get embed mcp
//	@Description	Get embed mcp
//	@Tags			embedmcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{array}	EmbedMCP
//	@Router			/api/embedmcp/ [get]
func GetEmbedMCPs(c *gin.Context) {
	embeds := embedmcp.Servers()
	enabledMCPs, err := model.GetPublicMCPsEnabled(slices.Collect(maps.Keys(embeds)))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	emcps := make([]*EmbedMCP, 0, len(embeds))
	for _, mcp := range embeds {
		emcps = append(emcps, newEmbedMCP(&mcp, slices.Contains(enabledMCPs, mcp.ID)))
	}

	middleware.SuccessResponse(c, emcps)
}

type SaveEmbedMCPRequest struct {
	ID         string            `json:"id"`
	Enabled    bool              `json:"enabled"`
	InitConfig map[string]string `json:"init_config"`
}

// SaveEmbedMCP godoc
//
//	@Summary		Save embed mcp
//	@Description	Save embed mcp
//	@Tags			embedmcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			body	body		SaveEmbedMCPRequest	true	"Save embed mcp request"
//	@Success		200		{object}	nil
//	@Router			/api/embedmcp/ [post]
func SaveEmbedMCP(c *gin.Context) {
	var req SaveEmbedMCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	emcp, ok := embedmcp.GetEmbedMCP(req.ID)
	if !ok {
		middleware.ErrorResponse(c, http.StatusNotFound, "embed mcp not found")
		return
	}

	pmcp, err := emcp.ToPublicMCP(req.InitConfig, req.Enabled)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := model.SavePublicMCP(pmcp); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, nil)
}

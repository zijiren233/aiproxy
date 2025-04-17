package controller

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
)

// GetMCPs godoc
//
//	@Summary		Get MCPs
//	@Description	Get a list of MCPs with pagination and filtering
//	@Tags			mcp
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			page		query		int		false	"Page number"
//	@Param			per_page	query		int		false	"Items per page"
//	@Param			type		query		string	false	"MCP type"
//	@Param			keyword		query		string	false	"Search keyword"
//	@Success		200			{object}	middleware.APIResponse{data=[]model.PublicMCP}
//	@Router			/api/mcp/ [get]
func GetMCPs(c *gin.Context) {
	page, perPage := parsePageParams(c)
	mcpType := model.MCPType(c.Query("type"))
	keyword := c.Query("keyword")

	mcps, total, err := model.GetMCPs(page, perPage, mcpType, keyword)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{
		"mcps":  mcps,
		"total": total,
	})
}

// GetMCPByID godoc
//
//	@Summary		Get MCP by ID
//	@Description	Get a specific MCP by its ID
//	@Tags			mcp
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		string	true	"MCP ID"
//	@Success		200	{object}	middleware.APIResponse{data=model.PublicMCP}
//	@Router			/api/mcp/{id} [get]
func GetMCPByIDHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID is required")
		return
	}

	mcp, err := model.GetMCPByID(id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	middleware.SuccessResponse(c, mcp)
}

// CreateMCP godoc
//
//	@Summary		Create MCP
//	@Description	Create a new MCP
//	@Tags			mcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			mcp	body		model.PublicMCP	true	"MCP object"
//	@Success		200	{object}	middleware.APIResponse
//	@Router			/api/mcp/ [post]
func CreateMCP(c *gin.Context) {
	var mcp model.PublicMCP
	if err := c.ShouldBindJSON(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	mcp.CreatedAt = time.Now()
	mcp.UpdateAt = time.Now()

	if err := model.CreateMCP(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, mcp)
}

// UpdateMCP godoc
//
//	@Summary		Update MCP
//	@Description	Update an existing MCP
//	@Tags			mcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		string			true	"MCP ID"
//	@Param			mcp	body		model.PublicMCP	true	"MCP object"
//	@Success		200	{object}	middleware.APIResponse
//	@Router			/api/mcp/{id} [put]
func UpdateMCP(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID is required")
		return
	}

	var mcp model.PublicMCP
	if err := c.ShouldBindJSON(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	mcp.ID = id
	mcp.UpdateAt = time.Now()

	if err := model.UpdateMCP(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, mcp)
}

// DeleteMCP godoc
//
//	@Summary		Delete MCP
//	@Description	Delete an MCP by ID
//	@Tags			mcp
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		string	true	"MCP ID"
//	@Success		200	{object}	middleware.APIResponse
//	@Router			/api/mcp/{id} [delete]
func DeleteMCP(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID is required")
		return
	}

	if err := model.DeleteMCP(id); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{"id": id})
}

// GetGroupMCPReusingParam godoc
//
//	@Summary		Get group MCP reusing parameters
//	@Description	Get reusing parameters for a specific group and MCP
//	@Tags			mcp
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		string	true	"MCP ID"
//	@Param			group	path		string	true	"Group ID"
//	@Success		200		{object}	middleware.APIResponse{data=model.GroupMCPReusingParam}
//	@Router			/api/mcp/{id}/group/{group}/params [get]
func GetGroupMCPReusingParam(c *gin.Context) {
	mcpID := c.Param("id")
	groupID := c.Param("group")

	if mcpID == "" || groupID == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID and Group ID are required")
		return
	}

	param, err := model.GetGroupMCPReusingParam(mcpID, groupID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	middleware.SuccessResponse(c, param)
}

// SaveGroupMCPReusingParam godoc
//
//	@Summary		Create or update group MCP reusing parameters
//	@Description	Create or update reusing parameters for a specific group and MCP
//	@Tags			mcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		string						true	"MCP ID"
//	@Param			group	path		string						true	"Group ID"
//	@Param			params	body		model.GroupMCPReusingParam	true	"Reusing parameters"
//	@Success		200		{object}	middleware.APIResponse
//	@Router			/api/mcp/{id}/group/{group}/params [post]
func SaveGroupMCPReusingParam(c *gin.Context) {
	mcpID := c.Param("id")
	groupID := c.Param("group")

	if mcpID == "" || groupID == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID and Group ID are required")
		return
	}

	var param model.GroupMCPReusingParam
	if err := c.ShouldBindJSON(&param); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	param.MCPID = mcpID
	param.GroupID = groupID

	if err := model.SaveGroupMCPReusingParam(&param); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, param)
}

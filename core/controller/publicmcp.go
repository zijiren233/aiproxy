package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
)

// GetPublicMCPs godoc
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
//	@Param			status		query		int		false	"MCP status"
//	@Success		200			{object}	middleware.APIResponse{data=[]model.PublicMCP}
//	@Router			/api/mcp/public/ [get]
func GetPublicMCPs(c *gin.Context) {
	page, perPage := parsePageParams(c)
	mcpType := model.PublicMCPType(c.Query("type"))
	keyword := c.Query("keyword")
	status, _ := strconv.Atoi(c.Query("status"))

	mcps, total, err := model.GetPublicMCPs(
		page,
		perPage,
		mcpType,
		keyword,
		model.PublicMCPStatus(status),
	)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{
		"mcps":  mcps,
		"total": total,
	})
}

// GetPublicMCPByIDHandler godoc
//
//	@Summary		Get MCP by ID
//	@Description	Get a specific MCP by its ID
//	@Tags			mcp
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		string	true	"MCP ID"
//	@Success		200	{object}	middleware.APIResponse{data=model.PublicMCP}
//	@Router			/api/mcp/public/{id} [get]
func GetPublicMCPByIDHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID is required")
		return
	}

	mcp, err := model.GetPublicMCPByID(id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	middleware.SuccessResponse(c, mcp)
}

// CreatePublicMCP godoc
//
//	@Summary		Create MCP
//	@Description	Create a new MCP
//	@Tags			mcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			mcp	body		model.PublicMCP	true	"MCP object"
//	@Success		200	{object}	middleware.APIResponse
//	@Router			/api/mcp/public/ [post]
func CreatePublicMCP(c *gin.Context) {
	var mcp model.PublicMCP
	if err := c.ShouldBindJSON(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	mcp.CreatedAt = time.Now()
	mcp.UpdateAt = time.Now()

	if err := model.CreatePublicMCP(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, mcp)
}

type UpdatePublicMCPStatusRequest struct {
	Status model.PublicMCPStatus `json:"status"`
}

// UpdatePublicMCPStatus godoc
//
//	@Summary		Update MCP status
//	@Description	Update the status of an MCP
//	@Tags			mcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		string							true	"MCP ID"
//	@Param			status	body		UpdatePublicMCPStatusRequest	true	"MCP status"
//	@Success		200		{object}	middleware.APIResponse
//	@Router			/api/mcp/public/{id}/status [post]
func UpdatePublicMCPStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID is required")
		return
	}

	var status UpdatePublicMCPStatusRequest
	if err := c.ShouldBindJSON(&status); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := model.UpdatePublicMCPStatus(id, status.Status); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, nil)
}

// UpdatePublicMCP godoc
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
//	@Router			/api/mcp/public/{id} [put]
func UpdatePublicMCP(c *gin.Context) {
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

	if err := model.UpdatePublicMCP(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, mcp)
}

// DeletePublicMCP godoc
//
//	@Summary		Delete MCP
//	@Description	Delete an MCP by ID
//	@Tags			mcp
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		string	true	"MCP ID"
//	@Success		200	{object}	middleware.APIResponse
//	@Router			/api/mcp/public/{id} [delete]
func DeletePublicMCP(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID is required")
		return
	}

	if err := model.DeletePublicMCP(id); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{"id": id})
}

// GetGroupPublicMCPReusingParam godoc
//
//	@Summary		Get group MCP reusing parameters
//	@Description	Get reusing parameters for a specific group and MCP
//	@Tags			mcp
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		string	true	"MCP ID"
//	@Param			group	path		string	true	"Group ID"
//	@Success		200		{object}	middleware.APIResponse{data=model.PublicMCPReusingParam}
//	@Router			/api/mcp/public/{id}/group/{group}/params [get]
func GetGroupPublicMCPReusingParam(c *gin.Context) {
	mcpID := c.Param("id")
	groupID := c.Param("group")

	if mcpID == "" || groupID == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID and Group ID are required")
		return
	}

	param, err := model.GetPublicMCPReusingParam(mcpID, groupID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	middleware.SuccessResponse(c, param)
}

// SaveGroupPublicMCPReusingParam godoc
//
//	@Summary		Create or update group MCP reusing parameters
//	@Description	Create or update reusing parameters for a specific group and MCP
//	@Tags			mcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		string						true	"MCP ID"
//	@Param			group	path		string						true	"Group ID"
//	@Param			params	body		model.PublicMCPReusingParam	true	"Reusing parameters"
//	@Success		200		{object}	middleware.APIResponse
//	@Router			/api/mcp/public/{id}/group/{group}/params [post]
func SaveGroupPublicMCPReusingParam(c *gin.Context) {
	mcpID := c.Param("id")
	groupID := c.Param("group")

	if mcpID == "" || groupID == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID and Group ID are required")
		return
	}

	var param model.PublicMCPReusingParam
	if err := c.ShouldBindJSON(&param); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	param.MCPID = mcpID
	param.GroupID = groupID

	if err := model.SavePublicMCPReusingParam(&param); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, param)
}

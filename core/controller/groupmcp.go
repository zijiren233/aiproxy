package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
)

// GetGroupMCPs godoc
//
//	@Summary		Get Group MCPs
//	@Description	Get a list of Group MCPs with pagination and filtering
//	@Tags			mcp
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			group		path		string	true	"Group ID"
//	@Param			page		query		int		false	"Page number"
//	@Param			per_page	query		int		false	"Items per page"
//	@Param			type		query		string	false	"MCP type"
//	@Param			keyword		query		string	false	"Search keyword"
//	@Success		200			{object}	middleware.APIResponse{data=[]model.GroupMCP}
//	@Router			/api/mcp/group/{group} [get]
func GetGroupMCPs(c *gin.Context) {
	groupID := c.Param("group")
	if groupID == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Group ID is required")
		return
	}

	page, perPage := parsePageParams(c)
	mcpType := model.PublicMCPType(c.Query("type"))
	keyword := c.Query("keyword")

	mcps, total, err := model.GetGroupMCPs(groupID, page, perPage, mcpType, keyword)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{
		"mcps":  mcps,
		"total": total,
	})
}

// GetGroupMCPByID godoc
//
//	@Summary		Get Group MCP by ID
//	@Description	Get a specific Group MCP by its ID and Group ID
//	@Tags			mcp
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		string	true	"MCP ID"
//	@Param			group	path		string	true	"Group ID"
//	@Success		200		{object}	middleware.APIResponse{data=model.GroupMCP}
//	@Router			/api/mcp/group/{group}/{id} [get]
func GetGroupMCPByID(c *gin.Context) {
	id := c.Param("id")
	groupID := c.Param("group")

	if id == "" || groupID == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID and Group ID are required")
		return
	}

	mcp, err := model.GetGroupMCPByID(id, groupID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	middleware.SuccessResponse(c, mcp)
}

// CreateGroupMCP godoc
//
//	@Summary		Create Group MCP
//	@Description	Create a new Group MCP
//	@Tags			mcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			group	path		string			true	"Group ID"
//	@Param			mcp		body		model.GroupMCP	true	"Group MCP object"
//	@Success		200		{object}	middleware.APIResponse{data=model.GroupMCP}
//	@Router			/api/mcp/group/{group} [post]
func CreateGroupMCP(c *gin.Context) {
	groupID := c.Param("group")
	if groupID == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Group ID is required")
		return
	}

	var mcp model.GroupMCP
	if err := c.ShouldBindJSON(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	mcp.GroupID = groupID

	if err := model.CreateGroupMCP(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, mcp)
}

// UpdateGroupMCP godoc
//
//	@Summary		Update Group MCP
//	@Description	Update an existing Group MCP
//	@Tags			mcp
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		string			true	"MCP ID"
//	@Param			group	path		string			true	"Group ID"
//	@Param			mcp		body		model.GroupMCP	true	"Group MCP object"
//	@Success		200		{object}	middleware.APIResponse{data=model.GroupMCP}
//	@Router			/api/mcp/group/{group}/{id} [put]
func UpdateGroupMCP(c *gin.Context) {
	id := c.Param("id")
	groupID := c.Param("group")

	if id == "" || groupID == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID and Group ID are required")
		return
	}

	var mcp model.GroupMCP
	if err := c.ShouldBindJSON(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	mcp.ID = id
	mcp.GroupID = groupID

	if err := model.UpdateGroupMCP(&mcp); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, mcp)
}

// DeleteGroupMCP godoc
//
//	@Summary		Delete Group MCP
//	@Description	Delete a Group MCP by ID and Group ID
//	@Tags			mcp
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		string	true	"MCP ID"
//	@Param			group	path		string	true	"Group ID"
//	@Success		200		{object}	middleware.APIResponse
//	@Router			/api/mcp/group/{group}/{id} [delete]
func DeleteGroupMCP(c *gin.Context) {
	id := c.Param("id")
	groupID := c.Param("group")

	if id == "" || groupID == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID and Group ID are required")
		return
	}

	if err := model.DeleteGroupMCP(id, groupID); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, nil)
}

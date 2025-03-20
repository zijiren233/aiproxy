package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/model"
)

type GroupResponse struct {
	*model.Group
	AccessedAt time.Time `json:"accessed_at,omitempty"`
}

func (g *GroupResponse) MarshalJSON() ([]byte, error) {
	type Alias model.Group
	return sonic.Marshal(&struct {
		*Alias
		CreatedAt  int64 `json:"created_at,omitempty"`
		AccessedAt int64 `json:"accessed_at,omitempty"`
	}{
		Alias:      (*Alias)(g.Group),
		CreatedAt:  g.CreatedAt.UnixMilli(),
		AccessedAt: g.AccessedAt.UnixMilli(),
	})
}

// GetGroups godoc
// @Summary      Get all groups
// @Description  Returns a list of all groups with pagination
// @Tags         groups
// @Produce      json
// @Security     ApiKeyAuth
// @Param        page      query    int     false  "Page number"
// @Param        per_page  query    int     false  "Items per page"
// @Success      200       {object}  middleware.APIResponse{data=map[string]any{groups=[]GroupResponse,total=int}}
// @Router       /api/groups [get]
func GetGroups(c *gin.Context) {
	page, perPage := parsePageParams(c)
	order := c.DefaultQuery("order", "")
	groups, total, err := model.GetGroups(page, perPage, order, false)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	groupResponses := make([]*GroupResponse, len(groups))
	for i, group := range groups {
		lastRequestAt, _ := model.GetGroupLastRequestTime(group.ID)
		groupResponses[i] = &GroupResponse{
			Group:      group,
			AccessedAt: lastRequestAt,
		}
	}
	middleware.SuccessResponse(c, gin.H{
		"groups": groupResponses,
		"total":  total,
	})
}

// SearchGroups godoc
// @Summary      Search groups
// @Description  Search groups with keyword and pagination
// @Tags         groups
// @Produce      json
// @Security     ApiKeyAuth
// @Param        keyword   query    string  true   "Search keyword"
// @Param        page      query    int     false  "Page number"
// @Param        per_page  query    int     false  "Items per page"
// @Success      200       {object}  middleware.APIResponse{data=map[string]any{groups=[]GroupResponse,total=int}}
// @Router       /api/groups/search [get]
func SearchGroups(c *gin.Context) {
	keyword := c.Query("keyword")
	page, perPage := parsePageParams(c)
	order := c.DefaultQuery("order", "")
	status, _ := strconv.Atoi(c.Query("status"))
	groups, total, err := model.SearchGroup(keyword, page, perPage, order, status)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	groupResponses := make([]*GroupResponse, len(groups))
	for i, group := range groups {
		lastRequestAt, _ := model.GetGroupLastRequestTime(group.ID)
		groupResponses[i] = &GroupResponse{
			Group:      group,
			AccessedAt: lastRequestAt,
		}
	}
	middleware.SuccessResponse(c, gin.H{
		"groups": groupResponses,
		"total":  total,
	})
}

// GetGroup godoc
// @Summary      Get a group
// @Description  Returns detailed information about a specific group
// @Tags         group
// @Produce      json
// @Security     ApiKeyAuth
// @Param        group  path      string  true  "Group name"
// @Success      200    {object}  middleware.APIResponse{data=GroupResponse}
// @Router       /api/group/{group} [get]
func GetGroup(c *gin.Context) {
	group := c.Param("group")
	if group == "" {
		middleware.ErrorResponse(c, http.StatusOK, "group id is empty")
		return
	}
	_group, err := model.GetGroupByID(group)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	lastRequestAt, _ := model.GetGroupLastRequestTime(group)
	groupResponse := &GroupResponse{
		Group:      _group,
		AccessedAt: lastRequestAt,
	}
	middleware.SuccessResponse(c, groupResponse)
}

type UpdateGroupRPMRatioRequest struct {
	RPMRatio float64 `json:"rpm_ratio"`
}

// UpdateGroupRPMRatio godoc
// @Summary      Update group RPM ratio
// @Description  Updates the RPM (Requests Per Minute) ratio for a group
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        group  path      string  true  "Group name"
// @Param        data   body      object  true  "RPM ratio information"
// @Success      200    {object}  middleware.APIResponse
// @Router       /api/group/{group}/rpm_ratio [post]
func UpdateGroupRPMRatio(c *gin.Context) {
	group := c.Param("group")
	if group == "" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	req := UpdateGroupRPMRatioRequest{}
	err := sonic.ConfigDefault.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	err = model.UpdateGroupRPMRatio(group, req.RPMRatio)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

type UpdateGroupRPMRequest struct {
	RPM map[string]int64 `json:"rpm"`
}

// UpdateGroupRPM godoc
// @Summary      Update group RPM
// @Description  Updates the RPM (Requests Per Minute) for a group
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        group  path      string  true  "Group name"
// @Param        data   body      object  true  "RPM information"
// @Success      200    {object}  middleware.APIResponse
// @Router       /api/group/{group}/rpm [post]
func UpdateGroupRPM(c *gin.Context) {
	group := c.Param("group")
	if group == "" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	req := UpdateGroupRPMRequest{}
	err := sonic.ConfigDefault.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	err = model.UpdateGroupRPM(group, req.RPM)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

type UpdateGroupTPMRequest struct {
	TPM map[string]int64 `json:"tpm"`
}

// UpdateGroupTPM godoc
// @Summary      Update group TPM
// @Description  Updates the TPM (Tokens Per Minute) for a group
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        group  path      string  true  "Group name"
// @Param        data   body      object  true  "TPM information"
// @Success      200    {object}  middleware.APIResponse
// @Router       /api/group/{group}/tpm [post]
func UpdateGroupTPM(c *gin.Context) {
	group := c.Param("group")
	if group == "" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	req := UpdateGroupTPMRequest{}
	err := sonic.ConfigDefault.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	err = model.UpdateGroupTPM(group, req.TPM)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

type UpdateGroupTPMRatioRequest struct {
	TPMRatio float64 `json:"tpm_ratio"`
}

// UpdateGroupTPMRatio godoc
// @Summary      Update group TPM ratio
// @Description  Updates the TPM (Tokens Per Minute) ratio for a group
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        group  path      string  true  "Group name"
// @Param        data   body      object  true  "TPM ratio information"
// @Success      200    {object}  middleware.APIResponse
// @Router       /api/group/{group}/tpm_ratio [post]
func UpdateGroupTPMRatio(c *gin.Context) {
	group := c.Param("group")
	if group == "" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	req := UpdateGroupTPMRatioRequest{}
	err := sonic.ConfigDefault.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	err = model.UpdateGroupTPMRatio(group, req.TPMRatio)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

type UpdateGroupStatusRequest struct {
	Status int `json:"status"`
}

// UpdateGroupStatus godoc
// @Summary      Update group status
// @Description  Updates the status of a group
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        group   path      string  true  "Group name"
// @Param        status  body      object  true  "Status information"
// @Success      200     {object}  middleware.APIResponse
// @Router       /api/group/{group}/status [post]
func UpdateGroupStatus(c *gin.Context) {
	group := c.Param("group")
	if group == "" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	req := UpdateGroupStatusRequest{}
	err := sonic.ConfigDefault.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	err = model.UpdateGroupStatus(group, req.Status)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

// DeleteGroup godoc
// @Summary      Delete a group
// @Description  Deletes a group by its name
// @Tags         group
// @Produce      json
// @Security     ApiKeyAuth
// @Param        group  path      string  true  "Group name"
// @Success      200    {object}  middleware.APIResponse
// @Router       /api/group/{group} [delete]
func DeleteGroup(c *gin.Context) {
	group := c.Param("group")
	if group == "" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	err := model.DeleteGroupByID(group)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

// DeleteGroups godoc
// @Summary      Delete multiple groups
// @Description  Deletes multiple groups by their IDs
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        ids  body      []string  true  "Group IDs"
// @Success      200  {object}  middleware.APIResponse
// @Router       /api/groups/batch_delete [post]
func DeleteGroups(c *gin.Context) {
	ids := []string{}
	err := c.ShouldBindJSON(&ids)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	err = model.DeleteGroupsByIDs(ids)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, nil)
}

type CreateGroupRequest struct {
	RPM      map[string]int64 `json:"rpm"`
	RPMRatio float64          `json:"rpm_ratio"`
	TPM      map[string]int64 `json:"tpm"`
	TPMRatio float64          `json:"tpm_ratio"`
}

// CreateGroup godoc
// @Summary      Create a new group
// @Description  Creates a new group with the given information
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        group  path      string  true  "Group name"
// @Param        data   body      object  true  "Group information"
// @Success      200    {object}  middleware.APIResponse{data=model.Group}
// @Router       /api/group/{group} [post]
func CreateGroup(c *gin.Context) {
	group := c.Param("group")
	if group == "" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	req := CreateGroupRequest{}
	err := sonic.ConfigDefault.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	g := &model.Group{
		ID:       group,
		RPMRatio: req.RPMRatio,
		RPM:      req.RPM,
		TPMRatio: req.TPMRatio,
		TPM:      req.TPM,
	}
	if err := model.CreateGroup(g); err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, g)
}

// UpdateGroup godoc
// @Summary      Update a group
// @Description  Updates an existing group with the given information
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        group  path      string  true  "Group name"
// @Param        data   body      object  true  "Updated group information"
// @Success      200    {object}  middleware.APIResponse{data=model.Group}
// @Router       /api/group/{group} [put]
func UpdateGroup(c *gin.Context) {
	group := c.Param("group")
	if group == "" {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	req := CreateGroupRequest{}
	err := sonic.ConfigDefault.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, "invalid parameter")
		return
	}
	g := &model.Group{
		RPMRatio: req.RPMRatio,
		RPM:      req.RPM,
		TPMRatio: req.TPMRatio,
		TPM:      req.TPM,
	}
	err = model.UpdateGroup(group, g)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusOK, err.Error())
		return
	}
	middleware.SuccessResponse(c, g)
}

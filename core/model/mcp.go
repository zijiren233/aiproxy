package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	ErrPublicMCPNotFound            = "public mcp"
	ErrGroupMCPReusingParamNotFound = "group mcp reusing param"
)

type MCPType string

const (
	MCPTypeProxySSE MCPType = "mcp_proxy_sse"
	MCPTypeGitRepo  MCPType = "mcp_git_repo" // read only
	MCPTypeOpenAPI  MCPType = "mcp_openapi"
)

type ParamType string

const (
	ParamTypeHeader ParamType = "header"
	ParamTypeQuery  ParamType = "query"
)

type ReusingParam struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        ParamType `json:"type"`
	Required    bool      `json:"required"`
}

type MCPProxySSEPrice struct {
	DefaultToolsCallPrice float64
	ToolsCallPrices       map[string]float64
}

type MCPProxySSEConfig struct {
	URL           string                  `json:"url"`
	Querys        map[string]string       `json:"querys"`
	Headers       map[string]string       `json:"headers"`
	ReusingParams map[string]ReusingParam `json:"reusing_params"`
	Price         MCPProxySSEPrice        `json:"price"`
}

type GroupMCPReusingParam struct {
	MCPID         string            `gorm:"primaryKey"                    json:"mcp_id"`
	GroupID       string            `gorm:"primaryKey"                    json:"group_id"`
	Group         *Group            `gorm:"foreignKey:GroupID"            json:"-"`
	ReusingParams map[string]string `gorm:"serializer:fastjson;type:text" json:"reusing_params"`
}

func (l *GroupMCPReusingParam) BeforeCreate(_ *gorm.DB) (err error) {
	if l.MCPID == "" {
		return errors.New("mcp id is empty")
	}
	if l.GroupID == "" {
		return errors.New("group is empty")
	}
	return
}

type MCPOpenAPIConfig struct {
	OpenAPISpec    string `json:"openapi_spec"`
	OpenAPIContent string `json:"openapi_content,omitempty"`
	Server         string `json:"server,omitempty"`
	Authorization  string `json:"authorization,omitempty"`
}

type PublicMCP struct {
	ID                    string                 `gorm:"primaryKey"                    json:"id"`
	CreatedAt             time.Time              `gorm:"index"                         json:"created_at"`
	UpdateAt              time.Time              `gorm:"index"                         json:"update_at"`
	GroupMCPReusingParams []GroupMCPReusingParam `gorm:"foreignKey:MCPID"              json:"-"`
	Name                  string                 `json:"name"`
	Type                  MCPType                `gorm:"index"                         json:"type"`
	RepoURL               string                 `json:"repo_url"`
	ReadmeURL             string                 `json:"readme_url"`
	Readme                string                 `gorm:"type:text"                     json:"readme"`
	Tags                  []string               `gorm:"serializer:fastjson;type:text" json:"tags,omitempty"`
	Author                string                 `json:"author"`
	LogoURL               string                 `json:"logo_url"`
	ProxySSEConfig        *MCPProxySSEConfig     `gorm:"serializer:fastjson;type:text" json:"proxy_sse_config,omitempty"`
	OpenAPIConfig         *MCPOpenAPIConfig      `gorm:"serializer:fastjson;type:text" json:"openapi_config,omitempty"`
}

func (l *PublicMCP) BeforeCreate(_ *gorm.DB) (err error) {
	if l.ID == "" {
		return errors.New("mcp id is empty")
	}
	return
}

func (c *PublicMCP) BeforeDelete(tx *gorm.DB) (err error) {
	return tx.Model(&GroupMCPReusingParam{}).Where("mcp_id = ?", c.ID).Delete(&GroupMCPReusingParam{}).Error
}

// CreateMCP creates a new MCP
func CreateMCP(mcp *PublicMCP) error {
	err := DB.Create(mcp).Error
	if err != nil && errors.Is(err, gorm.ErrDuplicatedKey) {
		return errors.New("mcp server already exist")
	}
	return err
}

// UpdateMCP updates an existing MCP
func UpdateMCP(mcp *PublicMCP) error {
	selects := []string{
		"repo_url",
		"readme",
		"tags",
		"author",
		"logo_url",
		"proxy_sse_config",
		"openapi_config",
	}
	if mcp.Name != "" {
		selects = append(selects, "name")
	}
	if mcp.Type != "" {
		selects = append(selects, "type")
	}
	result := DB.
		Select(selects).
		Where("id = ?", mcp.ID).
		Updates(mcp)
	return HandleUpdateResult(result, ErrPublicMCPNotFound)
}

// DeleteMCP deletes an MCP by ID
func DeleteMCP(id string) error {
	if id == "" {
		return errors.New("MCP id is empty")
	}
	result := DB.Delete(&PublicMCP{ID: id})
	return HandleUpdateResult(result, ErrPublicMCPNotFound)
}

// GetMCPByID retrieves an MCP by ID
func GetMCPByID(id string) (*PublicMCP, error) {
	if id == "" {
		return nil, errors.New("MCP id is empty")
	}
	var mcp PublicMCP
	err := DB.Where("id = ?", id).First(&mcp).Error
	return &mcp, HandleNotFound(err, ErrPublicMCPNotFound)
}

// GetMCPs retrieves MCPs with pagination and filtering
func GetMCPs(page int, perPage int, mcpType MCPType, keyword string) (mcps []*PublicMCP, total int64, err error) {
	tx := DB.Model(&PublicMCP{})

	if mcpType != "" {
		tx = tx.Where("type = ?", mcpType)
	}

	if keyword != "" {
		keyword = "%" + keyword + "%"
		tx = tx.Where("name LIKE ? OR author LIKE ? OR tags LIKE ?", keyword, keyword, keyword)
	}

	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total <= 0 {
		return nil, 0, nil
	}

	limit, offset := toLimitOffset(page, perPage)
	err = tx.
		Limit(limit).
		Offset(offset).
		Find(&mcps).
		Error

	return mcps, total, err
}

func SaveGroupMCPReusingParam(param *GroupMCPReusingParam) (err error) {
	return DB.Save(param).Error
}

// UpdateGroupMCPReusingParam updates an existing GroupMCPReusingParam
func UpdateGroupMCPReusingParam(param *GroupMCPReusingParam) error {
	result := DB.
		Select([]string{
			"reusing_params",
		}).
		Where("mcp_id = ? AND group_id = ?", param.MCPID, param.GroupID).
		Updates(param)
	return HandleUpdateResult(result, ErrGroupMCPReusingParamNotFound)
}

// DeleteGroupMCPReusingParam deletes a GroupMCPReusingParam
func DeleteGroupMCPReusingParam(mcpID string, groupID string) error {
	if mcpID == "" || groupID == "" {
		return errors.New("MCP ID or Group ID is empty")
	}
	result := DB.
		Where("mcp_id = ? AND group_id = ?", mcpID, groupID).
		Delete(&GroupMCPReusingParam{})
	return HandleUpdateResult(result, ErrGroupMCPReusingParamNotFound)
}

// GetGroupMCPReusingParam retrieves a GroupMCPReusingParam by MCP ID and Group ID
func GetGroupMCPReusingParam(mcpID string, groupID string) (*GroupMCPReusingParam, error) {
	if mcpID == "" || groupID == "" {
		return nil, errors.New("MCP ID or Group ID is empty")
	}
	var param GroupMCPReusingParam
	err := DB.Where("mcp_id = ? AND group_id = ?", mcpID, groupID).First(&param).Error
	return &param, HandleNotFound(err, ErrGroupMCPReusingParamNotFound)
}

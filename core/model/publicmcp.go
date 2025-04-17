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

type MCPPrice struct {
	DefaultToolsCallPrice float64
	ToolsCallPrices       map[string]float64
}

type PublicMCPProxySSEConfig struct {
	URL           string                  `json:"url"`
	Querys        map[string]string       `json:"querys"`
	Headers       map[string]string       `json:"headers"`
	ReusingParams map[string]ReusingParam `json:"reusing_params"`
	Price         MCPPrice                `json:"price"`
}

type GroupPublicMCPReusingParam struct {
	MCPID         string            `gorm:"primaryKey"                    json:"mcp_id"`
	GroupID       string            `gorm:"primaryKey"                    json:"group_id"`
	Group         *Group            `gorm:"foreignKey:GroupID"            json:"-"`
	ReusingParams map[string]string `gorm:"serializer:fastjson;type:text" json:"reusing_params"`
}

func (l *GroupPublicMCPReusingParam) BeforeCreate(_ *gorm.DB) (err error) {
	if l.MCPID == "" {
		return errors.New("mcp id is empty")
	}
	if l.GroupID == "" {
		return errors.New("group is empty")
	}
	return
}

type MCPOpenAPIConfig struct {
	OpenAPISpec    string   `json:"openapi_spec"`
	OpenAPIContent string   `json:"openapi_content,omitempty"`
	V2             bool     `json:"v2"`
	Server         string   `json:"server,omitempty"`
	Authorization  string   `json:"authorization,omitempty"`
	Price          MCPPrice `json:"price"`
}

type PublicMCP struct {
	ID                          string                       `gorm:"primaryKey"                    json:"id"`
	CreatedAt                   time.Time                    `gorm:"index"                         json:"created_at"`
	UpdateAt                    time.Time                    `gorm:"index"                         json:"update_at"`
	GroupPublicMCPReusingParams []GroupPublicMCPReusingParam `gorm:"foreignKey:MCPID"              json:"-"`
	Name                        string                       `json:"name"`
	Type                        MCPType                      `gorm:"index"                         json:"type"`
	RepoURL                     string                       `json:"repo_url"`
	ReadmeURL                   string                       `json:"readme_url"`
	Readme                      string                       `gorm:"type:text"                     json:"readme"`
	Tags                        []string                     `gorm:"serializer:fastjson;type:text" json:"tags,omitempty"`
	Author                      string                       `json:"author"`
	LogoURL                     string                       `json:"logo_url"`
	ProxySSEConfig              *PublicMCPProxySSEConfig     `gorm:"serializer:fastjson;type:text" json:"proxy_sse_config,omitempty"`
	OpenAPIConfig               *MCPOpenAPIConfig            `gorm:"serializer:fastjson;type:text" json:"openapi_config,omitempty"`
}

func (l *PublicMCP) BeforeCreate(_ *gorm.DB) (err error) {
	if l.ID == "" {
		return errors.New("mcp id is empty")
	}
	return
}

func (c *PublicMCP) BeforeDelete(tx *gorm.DB) (err error) {
	return tx.Model(&GroupPublicMCPReusingParam{}).Where("mcp_id = ?", c.ID).Delete(&GroupPublicMCPReusingParam{}).Error
}

// CreatePublicMCP creates a new MCP
func CreatePublicMCP(mcp *PublicMCP) error {
	err := DB.Create(mcp).Error
	if err != nil && errors.Is(err, gorm.ErrDuplicatedKey) {
		return errors.New("mcp server already exist")
	}
	return err
}

// UpdatePublicMCP updates an existing MCP
func UpdatePublicMCP(mcp *PublicMCP) error {
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

// DeletePublicMCP deletes an MCP by ID
func DeletePublicMCP(id string) error {
	if id == "" {
		return errors.New("MCP id is empty")
	}
	result := DB.Delete(&PublicMCP{ID: id})
	return HandleUpdateResult(result, ErrPublicMCPNotFound)
}

// GetPublicMCPByID retrieves an MCP by ID
func GetPublicMCPByID(id string) (*PublicMCP, error) {
	if id == "" {
		return nil, errors.New("MCP id is empty")
	}
	var mcp PublicMCP
	err := DB.Where("id = ?", id).First(&mcp).Error
	return &mcp, HandleNotFound(err, ErrPublicMCPNotFound)
}

// GetPublicMCPs retrieves MCPs with pagination and filtering
func GetPublicMCPs(page int, perPage int, mcpType MCPType, keyword string) (mcps []*PublicMCP, total int64, err error) {
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

func SaveGroupPublicMCPReusingParam(param *GroupPublicMCPReusingParam) (err error) {
	return DB.Save(param).Error
}

// UpdateGroupPublicMCPReusingParam updates an existing GroupMCPReusingParam
func UpdateGroupPublicMCPReusingParam(param *GroupPublicMCPReusingParam) error {
	result := DB.
		Select([]string{
			"reusing_params",
		}).
		Where("mcp_id = ? AND group_id = ?", param.MCPID, param.GroupID).
		Updates(param)
	return HandleUpdateResult(result, ErrGroupMCPReusingParamNotFound)
}

// DeleteGroupPublicMCPReusingParam deletes a GroupMCPReusingParam
func DeleteGroupPublicMCPReusingParam(mcpID string, groupID string) error {
	if mcpID == "" || groupID == "" {
		return errors.New("MCP ID or Group ID is empty")
	}
	result := DB.
		Where("mcp_id = ? AND group_id = ?", mcpID, groupID).
		Delete(&GroupPublicMCPReusingParam{})
	return HandleUpdateResult(result, ErrGroupMCPReusingParamNotFound)
}

// GetGroupPublicMCPReusingParam retrieves a GroupMCPReusingParam by MCP ID and Group ID
func GetGroupPublicMCPReusingParam(mcpID string, groupID string) (*GroupPublicMCPReusingParam, error) {
	if mcpID == "" || groupID == "" {
		return nil, errors.New("MCP ID or Group ID is empty")
	}
	var param GroupPublicMCPReusingParam
	err := DB.Where("mcp_id = ? AND group_id = ?", mcpID, groupID).First(&param).Error
	return &param, HandleNotFound(err, ErrGroupMCPReusingParamNotFound)
}

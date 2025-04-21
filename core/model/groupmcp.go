package model

import (
	"errors"
	"time"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/common"
	"gorm.io/gorm"
)

const (
	ErrGroupMCPNotFound = "group mcp"
)

type GroupMCPType string

const (
	GroupMCPTypeProxySSE GroupMCPType = "mcp_proxy_sse"
	GroupMCPTypeOpenAPI  GroupMCPType = "mcp_openapi"
)

type GroupMCPProxySSEConfig struct {
	URL     string            `json:"url"`
	Querys  map[string]string `json:"querys"`
	Headers map[string]string `json:"headers"`
}

type GroupMCP struct {
	ID             string                  `gorm:"primaryKey"                    json:"id"`
	GroupID        string                  `gorm:"primaryKey"                    json:"group_id"`
	Group          *Group                  `gorm:"foreignKey:GroupID"            json:"-"`
	CreatedAt      time.Time               `gorm:"index"                         json:"created_at"`
	UpdateAt       time.Time               `gorm:"index"                         json:"update_at"`
	Name           string                  `json:"name"`
	Type           GroupMCPType            `gorm:"index"                         json:"type"`
	ProxySSEConfig *GroupMCPProxySSEConfig `gorm:"serializer:fastjson;type:text" json:"proxy_sse_config,omitempty"`
	OpenAPIConfig  *MCPOpenAPIConfig       `gorm:"serializer:fastjson;type:text" json:"openapi_config,omitempty"`
}

func (g *GroupMCP) BeforeCreate(_ *gorm.DB) (err error) {
	if g.GroupID == "" {
		return errors.New("group id is empty")
	}
	if g.ID == "" {
		g.ID = common.ShortUUID()
	}

	if g.OpenAPIConfig != nil {
		config := g.OpenAPIConfig
		if config.OpenAPISpec != "" {
			return validateHTTPURL(config.OpenAPISpec)
		}
		if config.OpenAPIContent != "" {
			return nil
		}
		return errors.New("openapi spec and content is empty")
	}

	if g.ProxySSEConfig != nil {
		config := g.ProxySSEConfig
		return validateHTTPURL(config.URL)
	}

	return
}

func (g *GroupMCP) MarshalJSON() ([]byte, error) {
	type Alias GroupMCP
	a := &struct {
		*Alias
		CreatedAt int64 `json:"created_at"`
		UpdateAt  int64 `json:"update_at"`
	}{
		Alias:     (*Alias)(g),
		CreatedAt: g.CreatedAt.UnixMilli(),
		UpdateAt:  g.UpdateAt.UnixMilli(),
	}
	return sonic.Marshal(a)
}

// CreateGroupMCP creates a new GroupMCP
func CreateGroupMCP(mcp *GroupMCP) error {
	err := DB.Create(mcp).Error
	if err != nil && errors.Is(err, gorm.ErrDuplicatedKey) {
		return errors.New("group mcp already exists")
	}
	return err
}

// UpdateGroupMCP updates an existing GroupMCP
func UpdateGroupMCP(mcp *GroupMCP) error {
	selects := []string{
		"name",
		"proxy_sse_config",
		"openapi_config",
	}
	if mcp.Type != "" {
		selects = append(selects, "type")
	}
	result := DB.
		Select(selects).
		Where("id = ? AND group_id = ?", mcp.ID, mcp.GroupID).
		Updates(mcp)
	return HandleUpdateResult(result, ErrGroupMCPNotFound)
}

// DeleteGroupMCP deletes a GroupMCP by ID and GroupID
func DeleteGroupMCP(id string, groupID string) error {
	if id == "" || groupID == "" {
		return errors.New("group mcp id or group id is empty")
	}
	result := DB.Where("id = ? AND group_id = ?", id, groupID).Delete(&GroupMCP{})
	return HandleUpdateResult(result, ErrGroupMCPNotFound)
}

// GetGroupMCPByID retrieves a GroupMCP by ID and GroupID
func GetGroupMCPByID(id string, groupID string) (*GroupMCP, error) {
	if id == "" || groupID == "" {
		return nil, errors.New("group mcp id or group id is empty")
	}
	var mcp GroupMCP
	err := DB.Where("id = ? AND group_id = ?", id, groupID).First(&mcp).Error
	return &mcp, HandleNotFound(err, ErrGroupMCPNotFound)
}

// GetGroupMCPs retrieves GroupMCPs with pagination and filtering
func GetGroupMCPs(groupID string, page int, perPage int, mcpType PublicMCPType, keyword string) (mcps []*GroupMCP, total int64, err error) {
	if groupID == "" {
		return nil, 0, errors.New("group id is empty")
	}

	tx := DB.Model(&GroupMCP{}).Where("group_id = ?", groupID)

	if mcpType != "" {
		tx = tx.Where("type = ?", mcpType)
	}

	if keyword != "" {
		keyword = "%" + keyword + "%"
		if common.UsingPostgreSQL {
			tx = tx.Where("name ILIKE ? OR id ILIKE ?", keyword, keyword)
		} else {
			tx = tx.Where("name LIKE ? OR id LIKE ?", keyword, keyword)
		}
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

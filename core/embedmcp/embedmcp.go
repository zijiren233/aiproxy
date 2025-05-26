package embedmcp

import (
	"fmt"

	"github.com/labring/aiproxy/core/model"
	"github.com/mark3labs/mcp-go/server"
)

type ConfigValueValidator func(value string) error

type ConfigRequiredType int

const (
	ConfigRequiredTypeInitOptional ConfigRequiredType = iota
	ConfigRequiredTypeReusingOptional
	ConfigRequiredTypeInitOnly
	ConfigRequiredTypeReusingOnly
	ConfigRequiredTypeInitOrReusingOnly
)

func (c ConfigRequiredType) Validate(config any, reusingConfig any) error {
	switch c {
	case ConfigRequiredTypeInitOnly:
		if config == nil {
			return fmt.Errorf("config is required")
		}
	case ConfigRequiredTypeReusingOnly:
		if reusingConfig == nil {
			return fmt.Errorf("reusing config is required")
		}
	case ConfigRequiredTypeInitOrReusingOnly:
		if config == nil && reusingConfig == nil {
			return fmt.Errorf("config or reusing config is required")
		}
		if config != nil && reusingConfig != nil {
			return fmt.Errorf("config and reusing config are both provided, but only one is allowed")
		}
	}
	return nil
}

type ConfigTemplate struct {
	Name        string               `json:"name"`
	Required    ConfigRequiredType   `json:"required"`
	Example     string               `json:"example,omitempty"`
	Description string               `json:"description,omitempty"`
	Validator   ConfigValueValidator `json:"-"`
}

type ConfigTemplates = map[string]ConfigTemplate

func ValidateConfigTemplatesConfig(ct ConfigTemplates, config map[string]string, reusingConfig map[string]string) error {
	if len(ct) == 0 {
		return nil
	}

	for key, template := range ct {
		c := config[key]
		rc := reusingConfig[key]
		if err := template.Required.Validate(c, rc); err != nil {
			return fmt.Errorf("config required %s is invalid: %w", key, err)
		}
		if template.Validator != nil {
			if c != "" {
				if err := template.Validator(c); err != nil {
					return fmt.Errorf("config %s is invalid: %w", key, err)
				}
			} else if rc != "" {
				if err := template.Validator(rc); err != nil {
					return fmt.Errorf("reusing config %s is invalid: %w", key, err)
				}
			}
		}
	}

	return nil
}

func CheckConfigTemplatesValidate(ct ConfigTemplates) error {
	for key, value := range ct {
		if value.Name == "" {
			return fmt.Errorf("config %s name is required", key)
		}
		if value.Description == "" {
			return fmt.Errorf("config %s description is required", key)
		}
		if value.Example == "" || value.Validator == nil {
			continue
		}
		if err := value.Validator(value.Example); err != nil {
			return fmt.Errorf("config %s example is invalid: %w", key, err)
		}
	}
	return nil
}

func GetEmbedConfig(ct ConfigTemplates, initConfig map[string]string) (*model.MCPEmbeddingConfig, error) {
	reusingConfig := make(map[string]model.MCPEmbeddingReusingConfig)
	embedConfig := &model.MCPEmbeddingConfig{
		Init: initConfig,
	}
	for key, value := range ct {
		switch value.Required {
		case ConfigRequiredTypeInitOnly:
			if _, ok := initConfig[key]; !ok {
				return nil, fmt.Errorf("config %s is required", key)
			}
		case ConfigRequiredTypeReusingOnly:
			if _, ok := initConfig[key]; ok {
				return nil, fmt.Errorf("config %s is provided, but it is not allowed", key)
			}
			reusingConfig[key] = model.MCPEmbeddingReusingConfig{
				Name:        value.Name,
				Description: value.Description,
				Required:    true,
			}
		case ConfigRequiredTypeInitOrReusingOnly:
			if _, ok := initConfig[key]; ok {
				continue
			}
			reusingConfig[key] = model.MCPEmbeddingReusingConfig{
				Name:        value.Name,
				Description: value.Description,
				Required:    true,
			}
		}
	}
	embedConfig.Reusing = reusingConfig
	return embedConfig, nil
}

type NewServerFunc func(config map[string]string, reusingConfig map[string]string) (*server.MCPServer, error)

type EmbedMcp struct {
	ID              string
	Name            string
	Readme          string
	Tags            []string
	ConfigTemplates ConfigTemplates
	NewServer       NewServerFunc
}

func (e EmbedMcp) ToPublicMCP(initConfig map[string]string, enabled bool) (*model.PublicMCP, error) {
	embedConfig, err := GetEmbedConfig(e.ConfigTemplates, initConfig)
	if err != nil {
		return nil, err
	}
	pmcp := &model.PublicMCP{
		ID:          e.ID,
		Name:        e.Name,
		Readme:      e.Readme,
		Tags:        e.Tags,
		EmbedConfig: embedConfig,
	}
	if enabled {
		pmcp.Status = model.PublicMCPStatusEnabled
	} else {
		pmcp.Status = model.PublicMCPStatusDisabled
	}
	return pmcp, nil
}

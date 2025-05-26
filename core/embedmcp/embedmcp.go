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
	Required  ConfigRequiredType   `json:"required"`
	Example   string               `json:"example,omitempty"`
	Help      string               `json:"help,omitempty"`
	Validator ConfigValueValidator `json:"-"`
}

type ConfigTemplates = map[string]ConfigTemplate

func ValidateConfigTemplates(ct ConfigTemplates, config map[string]string, reusingConfig map[string]string) error {
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

func CheckConfigTemplatesExample(ct ConfigTemplates) error {
	for key, value := range ct {
		if value.Example == "" || value.Validator == nil {
			continue
		}
		if err := value.Validator(value.Example); err != nil {
			return fmt.Errorf("config %s example is invalid: %w", key, err)
		}
	}
	return nil
}

type NewServerFunc func(config map[string]string, reusingConfig map[string]string) (*server.MCPServer, error)

type EmbeddingMcp struct {
	ID              string
	Name            string
	Readme          string
	Tags            []string
	ConfigTemplates ConfigTemplates
	NewServer       NewServerFunc
}

func (e *EmbeddingMcp) ToPublicMCP() *model.PublicMCP {
	return &model.PublicMCP{
		ID:     e.ID,
		Name:   e.Name,
		Readme: e.Readme,
		Tags:   e.Tags,
	}
}

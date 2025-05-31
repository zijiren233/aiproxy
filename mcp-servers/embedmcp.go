package mcpservers

import (
	"errors"
	"fmt"

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

func (c ConfigRequiredType) Validate(config, reusingConfig string) error {
	switch c {
	case ConfigRequiredTypeInitOnly:
		if config == "" {
			return errors.New("config is required")
		}
	case ConfigRequiredTypeReusingOnly:
		if reusingConfig == "" {
			return errors.New("reusing config is required")
		}
	case ConfigRequiredTypeInitOrReusingOnly:
		if config == "" && reusingConfig == "" {
			return errors.New("config or reusing config is required")
		}
		if config != "" && reusingConfig != "" {
			return errors.New(
				"config and reusing config are both provided, but only one is allowed",
			)
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

func ValidateConfigTemplatesConfig(
	ct ConfigTemplates,
	config, reusingConfig map[string]string,
) error {
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

type NewServerFunc func(config, reusingConfig map[string]string) (*server.MCPServer, error)

type EmbedMcp struct {
	ID              string
	Name            string
	Readme          string
	Tags            []string
	ConfigTemplates ConfigTemplates
	newServer       NewServerFunc
}

type EmbedMcpConfig func(*EmbedMcp)

func WithReadme(readme string) EmbedMcpConfig {
	return func(e *EmbedMcp) {
		e.Readme = readme
	}
}

func WithTags(tags []string) EmbedMcpConfig {
	return func(e *EmbedMcp) {
		e.Tags = tags
	}
}

func WithConfigTemplates(configTemplates ConfigTemplates) EmbedMcpConfig {
	return func(e *EmbedMcp) {
		e.ConfigTemplates = configTemplates
	}
}

func NewEmbedMcp(id, name string, newServer NewServerFunc, opts ...EmbedMcpConfig) EmbedMcp {
	e := EmbedMcp{
		ID:        id,
		Name:      name,
		newServer: newServer,
	}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

func (e *EmbedMcp) NewServer(config, reusingConfig map[string]string) (*server.MCPServer, error) {
	if err := ValidateConfigTemplatesConfig(e.ConfigTemplates, config, reusingConfig); err != nil {
		return nil, fmt.Errorf("mcp %s config is invalid: %w", e.ID, err)
	}
	return e.newServer(config, reusingConfig)
}

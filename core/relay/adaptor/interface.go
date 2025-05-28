package adaptor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
)

type Adaptor interface {
	GetBaseURL() string
	GetRequestURL(meta *meta.Meta) (string, error)
	SetupRequestHeader(meta *meta.Meta, c *gin.Context, req *http.Request) error
	ConvertRequest(meta *meta.Meta, req *http.Request) (*ConvertRequestResult, error)
	DoRequest(meta *meta.Meta, c *gin.Context, req *http.Request) (*http.Response, error)
	DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, Error)
	GetModelList() []*model.ModelConfig
}

type ConvertRequestResult struct {
	Method string
	Header http.Header
	Body   io.Reader
}

type Error interface {
	json.Marshaler
	StatusCode() int
}

type ErrorImpl[T any] struct {
	error      T
	statusCode int
}

func (e ErrorImpl[T]) MarshalJSON() ([]byte, error) {
	return sonic.Marshal(e.error)
}

func (e ErrorImpl[T]) StatusCode() int {
	return e.statusCode
}

func NewError[T any](statusCode int, err T) Error {
	return ErrorImpl[T]{
		error:      err,
		statusCode: statusCode,
	}
}

var ErrGetBalanceNotImplemented = errors.New("get balance not implemented")

type Balancer interface {
	GetBalance(channel *model.Channel) (float64, error)
}

type KeyValidator interface {
	ValidateKey(key string) error
	KeyHelp() string
}

type Features interface {
	Features() []string
}

type ConfigType string

const (
	ConfigTypeString ConfigType = "string"
	ConfigTypeNumber ConfigType = "number"
	ConfigTypeBool   ConfigType = "bool"
	ConfigTypeObject ConfigType = "object"
)

type ConfigTemplate struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Example     any             `json:"example,omitempty"`
	Validator   func(any) error `json:"-"`
	Required    bool            `json:"required"`
	Type        ConfigType      `json:"type"`
}

func ValidateConfigTemplate(template ConfigTemplate) error {
	if template.Name == "" {
		return errors.New("config template is invalid: name is empty")
	}
	if template.Type == "" {
		return fmt.Errorf("config template %s is invalid: type is empty", template.Name)
	}
	if template.Example != nil {
		if err := ValidateConfigTemplateValue(template, template.Example); err != nil {
			return fmt.Errorf("config template %s is invalid: %w", template.Name, err)
		}
	}
	return nil
}

func ValidateConfigTemplateValue(template ConfigTemplate, value any) error {
	if template.Validator == nil {
		return nil
	}
	switch template.Type {
	case ConfigTypeString:
		_, ok := value.(string)
		if !ok {
			return fmt.Errorf("config template %s is invalid: value is not a string", template.Name)
		}
	case ConfigTypeNumber:
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return nil
		default:
			return fmt.Errorf("config template %s is invalid: value is not a number", template.Name)
		}
	case ConfigTypeBool:
		_, ok := value.(bool)
		if !ok {
			return fmt.Errorf("config template %s is invalid: value is not a bool", template.Name)
		}
	case ConfigTypeObject:
		if reflect.TypeOf(value).Kind() != reflect.Map &&
			reflect.TypeOf(value).Kind() != reflect.Struct {
			return fmt.Errorf("config template %s is invalid: value is not a object", template.Name)
		}
	}
	if err := template.Validator(value); err != nil {
		return fmt.Errorf("config template %s(%s) is invalid: %w", template.Name, template.Name, err)
	}
	return nil
}

type ConfigTemplates = map[string]ConfigTemplate

type Config interface {
	ConfigTemplates() ConfigTemplates
}

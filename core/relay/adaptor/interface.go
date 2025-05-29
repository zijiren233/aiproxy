package adaptor

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
)

type GetRequestURL interface {
	GetRequestURL(meta *meta.Meta) (string, error)
}

type SetupRequestHeader interface {
	SetupRequestHeader(meta *meta.Meta, c *gin.Context, req *http.Request) error
}

type ConvertRequest interface {
	ConvertRequest(meta *meta.Meta, req *http.Request) (*ConvertRequestResult, error)
}

type DoRequest interface {
	DoRequest(meta *meta.Meta, c *gin.Context, req *http.Request) (*http.Response, error)
}

type DoResponse interface {
	DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, Error)
}

type Adaptor interface {
	GetBaseURL() string
	GetModelList() []model.ModelConfig
	GetRequestURL
	SetupRequestHeader
	ConvertRequest
	DoRequest
	DoResponse
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

type ConfigTemplates = map[string]ConfigTemplate

type Config interface {
	ConfigTemplates() ConfigTemplates
}

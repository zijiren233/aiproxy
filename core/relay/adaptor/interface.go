package adaptor

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
)

type StoreCache struct {
	ID        string
	GroupID   string
	TokenID   int
	ChannelID int
	Model     string
	ExpiresAt time.Time
}

type Store interface {
	GetStore(group string, tokenID int, id string) (StoreCache, error)
	SaveStore(store StoreCache) error
}

type Metadata struct {
	ConfigTemplates ConfigTemplates
	KeyHelp         string
	Readme          string
	Models          []model.ModelConfig
}

type SetupRequestHeader interface {
	SetupRequestHeader(meta *meta.Meta, store Store, c *gin.Context, req *http.Request) error
}

type ConvertRequest interface {
	ConvertRequest(
		meta *meta.Meta,
		store Store,
		c *gin.Context,
		req *http.Request,
	) (ConvertResult, error)
}

type DoRequest interface {
	DoRequest(
		meta *meta.Meta,
		store Store,
		c *gin.Context,
		req *http.Request,
	) (*http.Response, error)
}

type DoResponse interface {
	DoResponse(
		meta *meta.Meta,
		store Store,
		c *gin.Context,
		resp *http.Response,
	) (UsageResult, Error)
}

type Adaptor interface {
	Metadata() Metadata
	SupportMode(mode mode.Mode) bool
	DefaultBaseURL() string
	SetupRequestHeader
	ConvertRequest
	DoRequest
	DoResponse
}

// ConvertResult represents the result of request conversion
type ConvertResult struct {
	Method string      // HTTP Method (defaults to POST if empty)
	URL    string      // Full request URL
	Header http.Header // Request headers
	Body   io.Reader   // Request body
}

// UsageResult represents the result of usage calculation
type UsageResult interface {
	// Usage returns the current usage (may be zero for async tasks)
	Usage() model.Usage
	// IsAsync returns true if usage needs to be fetched asynchronously
	IsAsync() bool
	// AsyncInfo returns async metadata if IsAsync is true, nil otherwise
	AsyncInfo() *model.AsyncUsageInfo
}

// SyncUsage represents synchronous usage (most common case)
type SyncUsage struct {
	usage model.Usage
}

// NewSyncUsage creates a new SyncUsage
func NewSyncUsage(usage model.Usage) SyncUsage {
	return SyncUsage{usage: usage}
}

func (s SyncUsage) Usage() model.Usage               { return s.usage }
func (s SyncUsage) IsAsync() bool                    { return false }
func (s SyncUsage) AsyncInfo() *model.AsyncUsageInfo { return nil }

// AsyncUsage represents asynchronous usage (e.g., Video API)
type AsyncUsage struct {
	info *model.AsyncUsageInfo
}

// NewAsyncUsage creates a new AsyncUsage
func NewAsyncUsage(info *model.AsyncUsageInfo) AsyncUsage {
	return AsyncUsage{info: info}
}

func (a AsyncUsage) Usage() model.Usage               { return model.Usage{} }
func (a AsyncUsage) IsAsync() bool                    { return true }
func (a AsyncUsage) AsyncInfo() *model.AsyncUsageInfo { return a.info }

type Error interface {
	json.Marshaler
	error
	StatusCode() int
}

var ErrGetBalanceNotImplemented = errors.New("get balance not implemented")

type Balancer interface {
	GetBalance(channel *model.Channel) (float64, error)
}

type KeyValidator interface {
	ValidateKey(key string) error
}

type ConfigTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
	Required    bool   `json:"required"`
}

type ConfigTemplates struct {
	Configs   map[string]ConfigTemplate
	Validator func(model.ChannelConfigs) error `json:"-"`
}

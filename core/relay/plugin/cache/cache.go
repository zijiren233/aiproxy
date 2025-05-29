package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/plugin"
	"github.com/labring/aiproxy/core/relay/plugin/noop"
	gcache "github.com/patrickmn/go-cache"
)

// Constants for cache metadata keys
const (
	cacheKey   = "cache_key"
	cacheHit   = "cache_hit"
	cacheValue = "cache_value"
)

// Constants for plugin configuration
const (
	pluginConfigCacheKey = "cache-config"
	cacheHeader          = "X-Aiproxy-Cache"
)

// Buffer size constants
const (
	defaultBufferSize = 512 * 1024
	maxBufferSize     = 4 * defaultBufferSize
)

// Item represents a cached response
type Item struct {
	Body   []byte
	Header http.Header
	Usage  *model.Usage
}

// Cache implements caching functionality for AI requests
type Cache struct {
	noop.Noop
}

var (
	_ plugin.Plugin = (*Cache)(nil)
	// Global cache instance with 5 minute default TTL and 10 minute cleanup interval
	cache = gcache.New(5*time.Minute, 10*time.Minute)
	// Buffer pool for response writers
	bufferPool = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 0, defaultBufferSize))
		},
	}
)

// NewCachePlugin creates a new cache plugin
func NewCachePlugin() plugin.Plugin {
	return &Cache{}
}

// Cache metadata helpers
func getCacheKey(meta *meta.Meta) string {
	return meta.GetString(cacheKey)
}

func setCacheKey(meta *meta.Meta, key string) {
	meta.Set(cacheKey, key)
}

func isCacheHit(meta *meta.Meta) bool {
	return meta.GetBool(cacheHit)
}

func getCacheItem(meta *meta.Meta) *Item {
	v, ok := meta.Get(cacheValue)
	if !ok {
		return nil
	}
	item, ok := v.(*Item)
	if !ok {
		panic(fmt.Sprintf("cache item type not match: %T", v))
	}
	return item
}

func setCacheHit(meta *meta.Meta, item *Item) {
	meta.Set(cacheHit, true)
	meta.Set(cacheValue, item)
}

// Buffer pool helpers
func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	if buf.Cap() > maxBufferSize {
		return
	}
	bufferPool.Put(buf)
}

// getPluginConfig retrieves the plugin configuration from metadata
func getPluginConfig(meta *meta.Meta) (config *Config, err error) {
	v, ok := meta.Get(pluginConfigCacheKey)
	if ok {
		config, ok := v.(*Config)
		if !ok {
			panic(fmt.Sprintf("cache config type not match: %T", v))
		}
		return config, nil
	}

	pluginConfig := Config{}
	if err := meta.ModelConfig.LoadPluginConfig("cache", &pluginConfig); err != nil {
		return nil, err
	}
	meta.Set(pluginConfigCacheKey, &pluginConfig)
	return &pluginConfig, nil
}

// ConvertRequest handles the request conversion phase
func (c *Cache) ConvertRequest(meta *meta.Meta, req *http.Request, do adaptor.ConvertRequest) (*adaptor.ConvertRequestResult, error) {
	pluginConfig, err := getPluginConfig(meta)
	if err != nil {
		return do.ConvertRequest(meta, req)
	}
	if !pluginConfig.EnablePlugin {
		return do.ConvertRequest(meta, req)
	}

	body, err := common.GetRequestBody(req)
	if err != nil {
		return nil, err
	}

	// Generate hash as cache key
	hash := sha256.Sum256(body)
	cacheKey := fmt.Sprintf("%d:%s", meta.Mode, hex.EncodeToString(hash[:]))
	setCacheKey(meta, cacheKey)

	item, ok := cache.Get(cacheKey)
	if ok {
		cacheItem, ok := item.(Item)
		if !ok {
			panic(fmt.Sprintf("cache item type not match: %T", item))
		}
		setCacheHit(meta, &cacheItem)
		return &adaptor.ConvertRequestResult{}, nil
	}

	return do.ConvertRequest(meta, req)
}

// DoRequest handles the request execution phase
func (c *Cache) DoRequest(meta *meta.Meta, ctx *gin.Context, req *http.Request, do adaptor.DoRequest) (*http.Response, error) {
	if isCacheHit(meta) {
		return &http.Response{}, nil
	}

	return do.DoRequest(meta, ctx, req)
}

// Custom response writer to capture response for caching
type responseWriter struct {
	gin.ResponseWriter
	cacheBody *bytes.Buffer
	maxSize   int
	overflow  bool
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.overflow {
		return rw.ResponseWriter.Write(b)
	}
	if rw.maxSize > 0 && rw.cacheBody.Len()+len(b) > rw.maxSize {
		rw.overflow = true
		rw.cacheBody.Reset()
		return rw.ResponseWriter.Write(b)
	}
	rw.cacheBody.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteString(s string) (int, error) {
	if rw.overflow {
		return rw.ResponseWriter.WriteString(s)
	}
	if rw.maxSize > 0 && rw.cacheBody.Len()+len(s) > rw.maxSize {
		rw.overflow = true
		rw.cacheBody.Reset()
		return rw.ResponseWriter.WriteString(s)
	}
	rw.cacheBody.WriteString(s)
	return rw.ResponseWriter.WriteString(s)
}

// DoResponse handles the response processing phase
func (c *Cache) DoResponse(meta *meta.Meta, ctx *gin.Context, resp *http.Response, do adaptor.DoResponse) (usage *model.Usage, adapterErr adaptor.Error) {
	pluginConfig, err := getPluginConfig(meta)
	if err != nil {
		return do.DoResponse(meta, ctx, resp)
	}

	// Handle cache hit
	if isCacheHit(meta) {
		item := getCacheItem(meta)
		if item == nil {
			return do.DoResponse(meta, ctx, resp)
		}

		ctx.Header("Content-Type", item.Header.Get("Content-Type"))
		ctx.Header("Content-Length", strconv.Itoa(len(item.Body)))
		if pluginConfig.AddCacheHitHeader {
			header := pluginConfig.CacheHitHeader
			if header == "" {
				header = cacheHeader
			}
			ctx.Header(header, "hit")
		}
		ctx.Status(http.StatusOK)
		_, _ = ctx.Writer.Write(item.Body)
		return item.Usage, nil
	}

	if !pluginConfig.EnablePlugin {
		return do.DoResponse(meta, ctx, resp)
	}

	// Set up response capture for caching
	buf := getBuffer()
	defer putBuffer(buf)

	rw := &responseWriter{
		ResponseWriter: ctx.Writer,
		maxSize:        pluginConfig.MaxSize,
		cacheBody:      buf,
	}
	ctx.Writer = rw
	defer func() {
		ctx.Writer = rw.ResponseWriter
		if adapterErr != nil || rw.overflow {
			return
		}
		respBody := rw.cacheBody.Bytes()
		respHeader := rw.Header()
		cache.Set(getCacheKey(meta), Item{
			Body:   bytes.Clone(respBody),
			Header: respHeader,
			Usage:  usage,
		}, time.Duration(pluginConfig.TTL)*time.Second)
	}()

	return do.DoResponse(meta, ctx, resp)
}

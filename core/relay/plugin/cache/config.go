package cache

type Config struct {
	EnablePlugin      bool   `json:"enable_plugin"`
	TTL               int    `json:"ttl"`
	MaxSize           int    `json:"max_size"`
	MaxItems          int    `json:"max_items"`
	AddCacheHitHeader bool   `json:"add_cache_hit_header"`
	CacheHitHeader    string `json:"cache_hit_header"`
}

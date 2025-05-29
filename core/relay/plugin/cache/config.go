package cache

type Config struct {
	EnablePlugin      bool   `json:"enable_plugin"`
	TTL               int    `json:"ttl"`
	ItemMaxSize       int    `json:"item_max_size"`
	AddCacheHitHeader bool   `json:"add_cache_hit_header"`
	CacheHitHeader    string `json:"cache_hit_header"`
}

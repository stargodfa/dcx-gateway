package cache

import (
	"context"
	"encoding/json"
	"time"

	gateway "dcx-gateway/internal/model/protocol/gateway"

	"github.com/redis/go-redis/v9"
)

type ModuleCache interface {
	GetModuleData(ctx context.Context, dataKey string) (*gateway.ModuleData, bool, error)
	SetModuleData(ctx context.Context, dataKey string, data gateway.ModuleData, ttl time.Duration) error
}

type moduleCache struct {
	rdb       *redis.Client
	keyPrefix string
}

func NewModuleCache(rdb *redis.Client, keyPrefix ...string) ModuleCache {
	return &moduleCache{rdb: rdb, keyPrefix: normalizedKeyPrefix(keyPrefix...)}
}

func (c *moduleCache) GetModuleData(ctx context.Context, dataKey string) (*gateway.ModuleData, bool, error) {
	val, err := c.rdb.Get(ctx, c.keyPrefix+KeyPrefixModuleData+dataKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var data gateway.ModuleData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, false, err
	}
	return &data, true, nil
}

func (c *moduleCache) SetModuleData(ctx context.Context, dataKey string, data gateway.ModuleData, ttl time.Duration) error {
	if dataKey == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = DefaultModuleTTL
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.keyPrefix+KeyPrefixModuleData+dataKey, b, ttl).Err()
}

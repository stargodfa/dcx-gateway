package cache

import (
	"context"
	"encoding/json"
	"time"

	gateway "dcx-gateway/internal/model/protocol/gateway"

	"github.com/redis/go-redis/v9"
)

type ScriptCache interface {
	GetPreloadBundle(ctx context.Context, scriptID string) (*gateway.DataBundle, bool, error)
	SetPreloadBundle(ctx context.Context, bundle *gateway.DataBundle, ttl time.Duration) error
	SetCurrentScript(ctx context.Context, scriptID string, ttl time.Duration) error
	GetCurrentScript(ctx context.Context) (string, bool, error)
}

type scriptCache struct {
	rdb       *redis.Client
	keyPrefix string
}

func NewScriptCache(rdb *redis.Client, keyPrefix ...string) ScriptCache {
	return &scriptCache{rdb: rdb, keyPrefix: normalizedKeyPrefix(keyPrefix...)}
}

func (c *scriptCache) GetPreloadBundle(ctx context.Context, scriptID string) (*gateway.DataBundle, bool, error) {
	val, err := c.rdb.Get(ctx, c.keyPrefix+KeyPrefixScriptBundle+scriptID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var bundle gateway.DataBundle
	if err := json.Unmarshal([]byte(val), &bundle); err != nil {
		return nil, false, err
	}
	return &bundle, true, nil
}

func (c *scriptCache) SetPreloadBundle(ctx context.Context, bundle *gateway.DataBundle, ttl time.Duration) error {
	if bundle == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = DefaultScriptTTL
	}
	b, err := json.Marshal(bundle)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.keyPrefix+KeyPrefixScriptBundle+bundle.ScriptID, b, ttl).Err()
}

func (c *scriptCache) SetCurrentScript(ctx context.Context, scriptID string, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.keyPrefix+KeyCurrentScript, scriptID, ttl).Err()
}

func (c *scriptCache) GetCurrentScript(ctx context.Context) (string, bool, error) {
	val, err := c.rdb.Get(ctx, c.keyPrefix+KeyCurrentScript).Result()
	if err != nil {
		if err == redis.Nil {
			return "", false, nil
		}
		return "", false, err
	}
	return val, true, nil
}

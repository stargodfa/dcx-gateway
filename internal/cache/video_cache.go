package cache

import (
	"context"
	"encoding/json"
	"time"

	"dcx-gateway/internal/model/domain"

	"github.com/redis/go-redis/v9"
)

type VideoCache interface {
	GetVideoStatus(ctx context.Context, videoID string) (*domain.VideoStatus, bool, error)
	SetVideoStatus(ctx context.Context, status domain.VideoStatus, ttl time.Duration) error
}

type videoCache struct {
	rdb       *redis.Client
	keyPrefix string
}

func NewVideoCache(rdb *redis.Client, keyPrefix ...string) VideoCache {
	return &videoCache{rdb: rdb, keyPrefix: normalizedKeyPrefix(keyPrefix...)}
}

func (c *videoCache) GetVideoStatus(ctx context.Context, videoID string) (*domain.VideoStatus, bool, error) {
	val, err := c.rdb.Get(ctx, c.keyPrefix+KeyPrefixVideoStatus+videoID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var status domain.VideoStatus
	if err := json.Unmarshal([]byte(val), &status); err != nil {
		return nil, false, err
	}
	return &status, true, nil
}

func (c *videoCache) SetVideoStatus(ctx context.Context, status domain.VideoStatus, ttl time.Duration) error {
	if status.VideoID == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = DefaultVideoTTL
	}
	b, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.keyPrefix+KeyPrefixVideoStatus+status.VideoID, b, ttl).Err()
}

package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type AuthSession struct {
	UserID    uint64    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuthCache interface {
	SetSession(ctx context.Context, tokenHash string, session AuthSession, ttl time.Duration) error
	GetSession(ctx context.Context, tokenHash string) (*AuthSession, bool, error)
	DeleteSession(ctx context.Context, tokenHash string) error
}

type authCache struct {
	rdb       *redis.Client
	keyPrefix string
}

func NewAuthCache(rdb *redis.Client, keyPrefix ...string) AuthCache {
	return &authCache{rdb: rdb, keyPrefix: normalizedKeyPrefix(keyPrefix...)}
}

func (c *authCache) SetSession(ctx context.Context, tokenHash string, session AuthSession, ttl time.Duration) error {
	b, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.sessionKey(tokenHash), b, ttl).Err()
}

func (c *authCache) GetSession(ctx context.Context, tokenHash string) (*AuthSession, bool, error) {
	val, err := c.rdb.Get(ctx, c.sessionKey(tokenHash)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}
	var session AuthSession
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, false, err
	}
	return &session, true, nil
}

func (c *authCache) DeleteSession(ctx context.Context, tokenHash string) error {
	return c.rdb.Del(ctx, c.sessionKey(tokenHash)).Err()
}

func (c *authCache) sessionKey(tokenHash string) string {
	return c.keyPrefix + "auth:session:" + tokenHash
}

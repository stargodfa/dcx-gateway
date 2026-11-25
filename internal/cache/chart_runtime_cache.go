package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type ChartRuntimeData struct {
	DataKey    string     `json:"data_key"`
	BindingID  string     `json:"binding_id"`
	ChartID    string     `json:"chart_id"`
	CellValues [][]string `json:"cell_values"`
	UpdatedAt  string     `json:"updated_at"`
}

type ChartRuntimeCache interface {
	SetChartRuntime(ctx context.Context, data ChartRuntimeData, ttl time.Duration) error
	GetByBindingID(ctx context.Context, bindingID string) (*ChartRuntimeData, error)
	GetByChartID(ctx context.Context, chartID string) (*ChartRuntimeData, error)
}

type chartRuntimeCache struct {
	rdb       *redis.Client
	keyPrefix string
}

func NewChartRuntimeCache(rdb *redis.Client, keyPrefix ...string) ChartRuntimeCache {
	return &chartRuntimeCache{rdb: rdb, keyPrefix: normalizedKeyPrefix(keyPrefix...)}
}

func (c *chartRuntimeCache) SetChartRuntime(ctx context.Context, data ChartRuntimeData, ttl time.Duration) error {
	if data.BindingID == "" || data.ChartID == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = DefaultChartRuntimeTTL
	}
	b, err := encodeChartRuntimeData(data)
	if err != nil {
		return err
	}
	pipe := c.rdb.TxPipeline()
	pipe.Set(ctx, c.keyPrefix+KeyPrefixChartRuntimeBinding+data.BindingID, b, ttl)
	pipe.Set(ctx, c.keyPrefix+KeyPrefixChartRuntimeChart+data.ChartID, b, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *chartRuntimeCache) GetByBindingID(ctx context.Context, bindingID string) (*ChartRuntimeData, error) {
	if bindingID == "" {
		return nil, nil
	}
	return c.get(ctx, KeyPrefixChartRuntimeBinding+bindingID)
}

func (c *chartRuntimeCache) GetByChartID(ctx context.Context, chartID string) (*ChartRuntimeData, error) {
	if chartID == "" {
		return nil, nil
	}
	return c.get(ctx, KeyPrefixChartRuntimeChart+chartID)
}

func (c *chartRuntimeCache) get(ctx context.Context, key string) (*ChartRuntimeData, error) {
	val, err := c.rdb.Get(ctx, c.keyPrefix+key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return decodeChartRuntimeData([]byte(val))
}

func encodeChartRuntimeData(data ChartRuntimeData) ([]byte, error) {
	return json.Marshal(data)
}

func decodeChartRuntimeData(data []byte) (*ChartRuntimeData, error) {
	var out ChartRuntimeData
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

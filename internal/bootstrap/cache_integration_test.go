package bootstrap

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"dcx-gateway/internal/cache"
	"dcx-gateway/internal/model/domain"
	gateway "dcx-gateway/internal/model/protocol/gateway"

	"github.com/redis/go-redis/v9"
)

func TestCachesRedisNamespaceIsolation(t *testing.T) {
	addr := os.Getenv("DCX_GATEWAY_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set DCX_GATEWAY_TEST_REDIS_ADDR to run Redis namespace integration")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stem := fmt.Sprintf("dcx-gateway-cache-test-%d-", time.Now().UnixNano())
	a := NewCaches(client, stem+"a") // Both prefix forms must produce one separator.
	b := NewCaches(client, stem+"b:")
	keys := []string{}
	for _, prefix := range []string{stem + "a:", stem + "b:"} {
		for _, suffix := range []string{"script:bundle:shared", "script:current", "module:data:shared", "video:status:shared", "chart:runtime:binding:shared", "chart:runtime:chart:shared", "auth:session:shared"} {
			keys = append(keys, prefix+suffix)
		}
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		if err := client.Del(cleanup, keys...).Err(); err != nil {
			t.Error(err)
		}
	}()
	write := func(c Caches, label string) {
		t.Helper()
		operations := []func() error{
			func() error {
				return c.Script.SetPreloadBundle(ctx, &gateway.DataBundle{ScriptID: "shared", BundleID: label}, time.Minute)
			},
			func() error { return c.Script.SetCurrentScript(ctx, label, time.Minute) },
			func() error {
				return c.Module.SetModuleData(ctx, "shared", gateway.ModuleData{Title: label}, time.Minute)
			},
			func() error {
				return c.Video.SetVideoStatus(ctx, domain.VideoStatus{VideoID: "shared", Status: label}, time.Minute)
			},
			func() error {
				return c.ChartRuntime.SetChartRuntime(ctx, cache.ChartRuntimeData{BindingID: "shared", ChartID: "shared", DataKey: label}, time.Minute)
			},
			func() error { return c.Auth.SetSession(ctx, "shared", cache.AuthSession{Username: label}, time.Minute) },
		}
		for _, operation := range operations {
			if err := operation(); err != nil {
				t.Fatal(err)
			}
		}
	}
	check := func(c Caches, label string, present bool) {
		t.Helper()
		bundle, ok, err := c.Script.GetPreloadBundle(ctx, "shared")
		if err != nil || ok != present || (present && (bundle == nil || bundle.BundleID != label)) {
			t.Fatalf("bundle: value=%+v found=%v err=%v", bundle, ok, err)
		}
		current, ok, err := c.Script.GetCurrentScript(ctx)
		if err != nil || ok != present || (present && current != label) {
			t.Fatalf("current script: value=%q found=%v err=%v", current, ok, err)
		}
		module, ok, err := c.Module.GetModuleData(ctx, "shared")
		if err != nil || ok != present || (present && (module == nil || module.Title != label)) {
			t.Fatalf("module: value=%+v found=%v err=%v", module, ok, err)
		}
		video, ok, err := c.Video.GetVideoStatus(ctx, "shared")
		if err != nil || ok != present || (present && (video == nil || video.Status != label)) {
			t.Fatalf("video: value=%+v found=%v err=%v", video, ok, err)
		}
		for _, read := range []func(context.Context, string) (*cache.ChartRuntimeData, error){c.ChartRuntime.GetByBindingID, c.ChartRuntime.GetByChartID} {
			chart, err := read(ctx, "shared")
			if err != nil || (chart != nil) != present || (present && chart.DataKey != label) {
				t.Fatalf("chart: value=%+v err=%v", chart, err)
			}
		}
		session, ok, err := c.Auth.GetSession(ctx, "shared")
		if err != nil || ok != present || (present && (session == nil || session.Username != label)) {
			t.Fatalf("session: value=%+v found=%v err=%v", session, ok, err)
		}
	}
	write(a, "a")
	check(a, "a", true)
	check(b, "", false)
	write(b, "b")
	check(a, "a", true)
	check(b, "b", true)
	for _, key := range keys {
		ttl, err := client.TTL(ctx, key).Result()
		if err != nil || ttl <= 0 || ttl > time.Minute {
			t.Fatalf("namespaced key TTL: key=%s ttl=%v err=%v", key, ttl, err)
		}
	}
	if err := b.Auth.DeleteSession(ctx, "shared"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := a.Auth.GetSession(ctx, "shared"); err != nil || !ok {
		t.Fatalf("deleting B's session affected A: found=%v err=%v", ok, err)
	}
}

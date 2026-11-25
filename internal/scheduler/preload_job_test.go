package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	gateway "dcx-gateway/internal/model/protocol/gateway"
	"dcx-gateway/internal/service"

	"go.uber.org/zap"
)

type fakePreloader struct {
	mu          sync.Mutex
	calls       []string
	forceValues []bool
	pushValues  []bool
	fail        map[string]error
}

func (p *fakePreloader) Preload(ctx context.Context, scriptID string, force bool) (*gateway.DataBundle, error) {
	return p.PreloadWithOptions(ctx, scriptID, service.PreloadOptions{Force: force, PushWS: true})
}

func (p *fakePreloader) PreloadWithOptions(ctx context.Context, scriptID string, opts service.PreloadOptions) (*gateway.DataBundle, error) {
	p.mu.Lock()
	p.calls = append(p.calls, scriptID)
	p.forceValues = append(p.forceValues, opts.Force)
	p.pushValues = append(p.pushValues, opts.PushWS)
	p.mu.Unlock()

	if err := p.fail[scriptID]; err != nil {
		return nil, err
	}
	return &gateway.DataBundle{
		ScriptID: scriptID,
		Modules:  []gateway.ModuleData{{ModuleID: "m1"}},
		Charts:   []gateway.ChartData{{ChartID: "c1"}},
		Videos:   []gateway.VideoSource{{VideoID: "v1"}},
	}, nil
}

func (p *fakePreloader) snapshotCalls() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.calls))
	copy(out, p.calls)
	return out
}

func (p *fakePreloader) snapshotForceValues() []bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]bool, len(p.forceValues))
	copy(out, p.forceValues)
	return out
}

func (p *fakePreloader) snapshotPushValues() []bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]bool, len(p.pushValues))
	copy(out, p.pushValues)
	return out
}

func TestPreloadJobRunsImmediately(t *testing.T) {
	preloader := &fakePreloader{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := NewPreloadJob(preloader, []string{"security_main"}, time.Hour, zap.NewNop())
	job.Start(ctx)

	calls := waitForCallCount(t, preloader, 1, 200*time.Millisecond)
	if calls[0] != "security_main" {
		t.Fatalf("calls = %#v", calls)
	}
	assertAllForceTrue(t, preloader)
	assertAllPushValues(t, preloader, false)
}

func TestPreloadJobRunsConfiguredScriptsAndIsolatesFailure(t *testing.T) {
	preloader := &fakePreloader{fail: map[string]error{"energy_main": errors.New("boom")}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := NewPreloadJob(preloader, []string{"security_main", "energy_main", "education_main"}, time.Hour, zap.NewNop())
	job.Start(ctx)

	calls := waitForCallCount(t, preloader, 3, 200*time.Millisecond)
	want := []string{"security_main", "energy_main", "education_main"}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls = %#v, want prefix %#v", calls, want)
		}
	}
	assertAllForceTrue(t, preloader)
	assertAllPushValues(t, preloader, false)
}

func TestPreloadJobIntervalTriggersSecondRound(t *testing.T) {
	preloader := &fakePreloader{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := NewPreloadJob(preloader, []string{"security_main"}, 10*time.Millisecond, zap.NewNop())
	job.Start(ctx)

	calls := waitForCallCount(t, preloader, 2, 300*time.Millisecond)
	if len(calls) < 2 {
		t.Fatalf("calls = %#v, want at least 2", calls)
	}
	assertAllForceTrue(t, preloader)
	assertAllPushValues(t, preloader, false)
}

func TestPreloadJobCanPushWS(t *testing.T) {
	preloader := &fakePreloader{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := NewPreloadJob(preloader, []string{"security_main"}, time.Hour, zap.NewNop(), WithPushWS(true), WithSource("mock_push"))
	job.Start(ctx)

	waitForCallCount(t, preloader, 1, 200*time.Millisecond)
	assertAllForceTrue(t, preloader)
	assertAllPushValues(t, preloader, true)
}

func TestPreloadJobStopsWhenContextCanceled(t *testing.T) {
	preloader := &fakePreloader{}
	ctx, cancel := context.WithCancel(context.Background())

	job := NewPreloadJob(preloader, []string{"security_main"}, 10*time.Millisecond, zap.NewNop())
	job.Start(ctx)
	waitForCallCount(t, preloader, 1, 200*time.Millisecond)
	cancel()

	countAfterCancel := len(preloader.snapshotCalls())
	time.Sleep(40 * time.Millisecond)
	if got := len(preloader.snapshotCalls()); got != countAfterCancel {
		t.Fatalf("calls after cancel = %d, want %d", got, countAfterCancel)
	}
}

func assertAllPushValues(t *testing.T, preloader *fakePreloader, want bool) {
	t.Helper()
	values := preloader.snapshotPushValues()
	if len(values) == 0 {
		t.Fatal("expected preload push values to be recorded")
	}
	for i, value := range values {
		if value != want {
			t.Fatalf("pushValues[%d] = %v, want %v; all values = %#v", i, value, want, values)
		}
	}
}

func TestPreloadJobEmptyScriptsDoesNotPanic(t *testing.T) {
	preloader := &fakePreloader{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := NewPreloadJob(preloader, nil, 10*time.Millisecond, zap.NewNop())
	job.Start(ctx)

	time.Sleep(30 * time.Millisecond)
	if calls := preloader.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("calls = %#v, want empty", calls)
	}
}

func waitForCallCount(t *testing.T, preloader *fakePreloader, want int, timeout time.Duration) []string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		calls := preloader.snapshotCalls()
		if len(calls) >= want {
			return calls
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d calls, got %#v", want, preloader.snapshotCalls())
	return nil
}

func assertAllForceTrue(t *testing.T, preloader *fakePreloader) {
	t.Helper()
	forces := preloader.snapshotForceValues()
	if len(forces) == 0 {
		t.Fatal("expected preload force values to be recorded")
	}
	for i, force := range forces {
		if !force {
			t.Fatalf("forceValues[%d] = false, want true; all values = %#v", i, forces)
		}
	}
}

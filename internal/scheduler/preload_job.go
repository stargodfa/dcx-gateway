package scheduler

import (
	"context"
	"strings"
	"sync"
	"time"

	gateway "dcx-gateway/internal/model/protocol/gateway"
	"dcx-gateway/internal/service"

	"go.uber.org/zap"
)

type ScriptPreloader interface {
	PreloadWithOptions(ctx context.Context, scriptID string, opts service.PreloadOptions) (*gateway.DataBundle, error)
}

type PreloadJob struct {
	preloader ScriptPreloader
	scriptIDs []string
	interval  time.Duration
	pushWS    bool
	source    string
	log       *zap.Logger
	stop      chan struct{}
	stopOnce  sync.Once
}

type PreloadJobOption func(*PreloadJob)

func WithPushWS(pushWS bool) PreloadJobOption {
	return func(j *PreloadJob) {
		j.pushWS = pushWS
	}
}

func WithSource(source string) PreloadJobOption {
	return func(j *PreloadJob) {
		source = strings.TrimSpace(source)
		if source != "" {
			j.source = source
		}
	}
}

func NewPreloadJob(preloader ScriptPreloader, scriptIDs []string, interval time.Duration, log *zap.Logger, opts ...PreloadJobOption) *PreloadJob {
	if log == nil {
		log = zap.NewNop()
	}
	job := &PreloadJob{
		preloader: preloader,
		scriptIDs: normalizeScriptIDs(scriptIDs),
		interval:  interval,
		source:    "scheduler",
		log:       log,
		stop:      make(chan struct{}),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(job)
		}
	}
	return job
}

func (j *PreloadJob) Start(ctx context.Context) {
	if j == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if j.preloader == nil {
		j.log.Warn("preload job skipped: script preloader is nil")
		return
	}
	if j.interval <= 0 {
		j.log.Warn("preload job skipped: interval must be positive", zap.Duration("interval", j.interval))
		return
	}
	if len(j.scriptIDs) == 0 {
		j.log.Warn("preload job skipped: scripts is empty")
		return
	}

	go func() {
		j.runOnce(ctx)

		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				j.log.Info("preload job stopped")
				return
			case <-j.stop:
				j.log.Info("preload job stopped")
				return
			case <-ticker.C:
				j.runOnce(ctx)
			}
		}
	}()
}

func (j *PreloadJob) Stop() {
	if j == nil {
		return
	}
	j.stopOnce.Do(func() {
		close(j.stop)
	})
}

func (j *PreloadJob) runOnce(ctx context.Context) {
	for _, scriptID := range j.scriptIDs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		start := time.Now()
		bundle, err := j.preloader.PreloadWithOptions(ctx, scriptID, service.PreloadOptions{
			Force:    true,
			PushWS:   j.pushWS,
			Source:   j.source,
			AuthType: "system",
		})
		duration := time.Since(start)
		if err != nil {
			j.log.Warn("preload job failed",
				zap.String("source", j.source),
				zap.String("auth_type", "system"),
				zap.String("script_id", scriptID),
				zap.Bool("push_ws", j.pushWS),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
			continue
		}

		modules, charts, videos := 0, 0, 0
		if bundle != nil {
			modules = len(bundle.Modules)
			charts = len(bundle.Charts)
			videos = len(bundle.Videos)
		}
		j.log.Info("preload job success",
			zap.String("source", j.source),
			zap.String("auth_type", "system"),
			zap.String("script_id", scriptID),
			zap.Bool("push_ws", j.pushWS),
			zap.Int("modules", modules),
			zap.Int("charts", charts),
			zap.Int("videos", videos),
			zap.Duration("duration", duration),
		)
	}
}

func normalizeScriptIDs(scriptIDs []string) []string {
	out := make([]string, 0, len(scriptIDs))
	seen := make(map[string]struct{}, len(scriptIDs))
	for _, scriptID := range scriptIDs {
		scriptID = strings.TrimSpace(scriptID)
		if scriptID == "" {
			continue
		}
		if _, ok := seen[scriptID]; ok {
			continue
		}
		seen[scriptID] = struct{}{}
		out = append(out, scriptID)
	}
	return out
}

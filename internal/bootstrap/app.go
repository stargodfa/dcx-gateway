package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"dcx-gateway/internal/adapter/education"
	"dcx-gateway/internal/adapter/energy"
	"dcx-gateway/internal/adapter/security"
	"dcx-gateway/internal/api"
	"dcx-gateway/internal/api/handler"
	apimiddleware "dcx-gateway/internal/api/middleware"
	"dcx-gateway/internal/config"
	"dcx-gateway/internal/pkg/logger"
	"dcx-gateway/internal/scheduler"
	"dcx-gateway/internal/service"
	"dcx-gateway/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	Config        *config.Config
	Logger        *zap.Logger
	Router        *gin.Engine
	DB            *gorm.DB
	Redis         *redis.Client
	FastAPIClient websocket.FastAPIClient
	PreloadJob    *scheduler.PreloadJob
	MockPushJob   *scheduler.PreloadJob
}

func NewApp(configPath string) (*App, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config failed: %w", err)
	}

	log, err := logger.New(cfg.App.Env)
	if err != nil {
		return nil, fmt.Errorf("init logger failed: %w", err)
	}

	db, err := NewMySQL(cfg.MySQL.DSN)
	if err != nil {
		return nil, fmt.Errorf("init mysql failed: %w", err)
	}

	rdb := NewRedis(cfg.Redis)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		closeRedis(rdb, log)
		closeMySQL(db, log)
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	repos := NewRepositories(db, log)
	caches := NewCaches(rdb, cfg.Redis.KeyPrefix)

	securityAdapter := newSecurityAdapter(cfg.Adapters.Security, log)
	energyAdapter := newEnergyAdapter(cfg.Adapters.Energy, log)
	educationAdapter := newEducationAdapter(cfg.Adapters.Education, log)

	var fastAPIClient websocket.FastAPIClient
	if cfg.FastAPI.WSDataURL != "" {
		fastAPIClient = websocket.NewFastAPIClient(
			cfg.FastAPI.WSDataURL,
			log,
			fastAPIWSAuthOption(cfg.FastAPI.WSAuth),
			websocket.WithTimings(
				time.Duration(cfg.FastAPI.WSConnectTimeoutSeconds)*time.Second,
				time.Duration(cfg.FastAPI.WSPingIntervalSeconds)*time.Second,
				time.Duration(cfg.FastAPI.WSReconnectMinSeconds)*time.Second,
				time.Duration(cfg.FastAPI.WSReconnectMaxSeconds)*time.Second,
			),
		)
	}

	scriptService := service.NewScriptService(
		repos.Script,
		repos.ScriptModule,
		repos.DataBinding,
		repos.VideoSource,
		securityAdapter,
		energyAdapter,
		educationAdapter,
		caches.Script,
		caches.Module,
		caches.Video,
		caches.ChartRuntime,
		fastAPIClient,
		cfg.FastAPI.PushVideoStreams,
		log,
	)
	var preloadJob *scheduler.PreloadJob
	if cfg.Scheduler.Preload.Enabled {
		preloadJob = scheduler.NewPreloadJob(
			scriptService,
			cfg.Scheduler.Preload.Scripts,
			time.Duration(cfg.Scheduler.Preload.IntervalSeconds)*time.Second,
			log,
			scheduler.WithPushWS(cfg.Scheduler.Preload.PushWS),
			scheduler.WithSource("scheduler"),
		)
	}
	var mockPushJob *scheduler.PreloadJob
	if cfg.MockPush.Enabled {
		mockPushJob = scheduler.NewPreloadJob(
			scriptService,
			cfg.MockPush.Scripts,
			time.Duration(cfg.MockPush.IntervalSeconds)*time.Second,
			log,
			scheduler.WithPushWS(true),
			scheduler.WithSource("mock_push"),
		)
	}
	ueConfigService := service.NewUEConfigService(
		repos.Script,
		repos.ScriptModule,
		repos.DataBinding,
		repos.VideoSource,
		repos.UEConfigDocument,
		repos.UEChartConfig,
		repos.UESceneDesired,
		caches.ChartRuntime,
		service.UEModelConfigRepositories{
			ModelType:     repos.UEModelType,
			ModelInstance: repos.UEModelInstance,
			Placement:     repos.UEModelPlacement,
		},
	)
	runtimeService := service.NewRuntimeService(
		repos.Script,
		repos.ScriptModule,
		repos.DataBinding,
		repos.VideoSource,
		repos.UEModelInstance,
		repos.UEConfigDocument,
		scriptService,
		log,
	)

	scriptHandler := handler.NewScriptHandler(scriptService, log)
	ueConfigHandler := handler.NewUEConfigHandler(ueConfigService, log, handler.WithMaxRawConfigBodyBytes(cfg.Server.MaxBodyBytes))
	authService := service.NewAuthService(repos.User, caches.Auth, cfg.Security)
	authHandler := handler.NewAuthHandler(authService, log)
	authMiddleware := apimiddleware.Auth(cfg.Security, authService)
	healthHandler := handler.NewHealthHandler(db, rdb, cfg.FastAPI.WSDataURL, log)
	runtimeHandler := handler.NewRuntimeHandler(fastAPIClient, runtimeService)
	ueAuxConfigService := service.NewUEAuxConfigService(repos.UEAuxConfig)
	ueAuxConfigHandler := handler.NewUEAuxConfigHandler(ueAuxConfigService, log, cfg.Server.MaxBodyBytes)

	router := gin.New()

	api.RegisterRoutes(router, api.Handlers{
		Script:      scriptHandler,
		UEConfig:    ueConfigHandler,
		Auth:        authHandler,
		Health:      healthHandler,
		Runtime:     runtimeHandler,
		UEAuxConfig: ueAuxConfigHandler,
	}, apimiddleware.RequestID(), apimiddleware.RequestLogger(log), gin.Recovery(), authMiddleware)

	return &App{
		Config:        cfg,
		Logger:        log,
		Router:        router,
		DB:            db,
		Redis:         rdb,
		FastAPIClient: fastAPIClient,
		PreloadJob:    preloadJob,
		MockPushJob:   mockPushJob,
	}, nil
}

func fastAPIWSAuthOption(auth config.OutboundAuthConfig) websocket.FastAPIClientOption {
	header := http.Header{}
	if !config.ApplyOutboundAuthHeader(header, auth) {
		return nil
	}
	for name, values := range header {
		if len(values) > 0 {
			return websocket.WithAuthHeader(name, values[0])
		}
	}
	return nil
}

func newSecurityAdapter(cfg config.SecurityAdapterConfig, log *zap.Logger) security.Adapter {
	switch strings.ToLower(strings.TrimSpace(cfg.Mode)) {
	case "", "mock":
		return security.NewMockAdapter(log)
	case "http":
		if strings.TrimSpace(cfg.BaseURL) == "" {
			log.Warn("security adapter http mode missing base_url, fallback to mock")
			return security.NewMockAdapter(log)
		}
		return security.NewHTTPAdapter(
			cfg.BaseURL,
			time.Duration(cfg.TimeoutSeconds)*time.Second,
			cfg.AccessSummaryPath,
			cfg.FireAlertsPath,
			log,
			cfg.Auth,
		)
	default:
		log.Warn("unknown security adapter mode, fallback to mock", zap.String("mode", cfg.Mode))
		return security.NewMockAdapter(log)
	}
}

func newEnergyAdapter(cfg config.EnergyAdapterConfig, log *zap.Logger) energy.Adapter {
	switch strings.ToLower(strings.TrimSpace(cfg.Mode)) {
	case "", "mock":
		return energy.NewMockAdapter(log)
	case "http":
		if strings.TrimSpace(cfg.BaseURL) == "" {
			log.Warn("energy adapter http mode missing base_url, fallback to mock")
			return energy.NewMockAdapter(log)
		}
		return energy.NewHTTPAdapter(
			cfg.BaseURL,
			time.Duration(cfg.TimeoutSeconds)*time.Second,
			cfg.SnapshotPath,
			log,
			cfg.Auth,
		)
	default:
		log.Warn("unknown energy adapter mode, fallback to mock", zap.String("mode", cfg.Mode))
		return energy.NewMockAdapter(log)
	}
}

func newEducationAdapter(cfg config.EducationAdapterConfig, log *zap.Logger) education.Adapter {
	switch strings.ToLower(strings.TrimSpace(cfg.Mode)) {
	case "", "mock":
		return education.NewMockAdapter(log)
	case "http":
		if strings.TrimSpace(cfg.BaseURL) == "" {
			log.Warn("education adapter http mode missing base_url, fallback to mock")
			return education.NewMockAdapter(log)
		}
		return education.NewHTTPAdapter(
			cfg.BaseURL,
			time.Duration(cfg.TimeoutSeconds)*time.Second,
			cfg.ClassroomUsagePath,
			log,
			cfg.Auth,
		)
	default:
		log.Warn("unknown education adapter mode, fallback to mock", zap.String("mode", cfg.Mode))
		return education.NewMockAdapter(log)
	}
}

func (a *App) Close() {
	if a == nil {
		return
	}
	if a.FastAPIClient != nil {
		if err := a.FastAPIClient.Close(); err != nil && a.Logger != nil {
			a.Logger.Warn("close fastapi ws client failed", zap.Error(err))
		}
	}
	if a.PreloadJob != nil {
		a.PreloadJob.Stop()
	}
	if a.MockPushJob != nil {
		a.MockPushJob.Stop()
	}

	closeRedis(a.Redis, a.Logger)
	closeMySQL(a.DB, a.Logger)

	if a.Logger != nil {
		_ = a.Logger.Sync()
	}
}

func closeRedis(rdb *redis.Client, log *zap.Logger) {
	if rdb == nil {
		return
	}
	if err := rdb.Close(); err != nil && log != nil {
		log.Warn("close redis client failed", zap.Error(err))
	}
}

func closeMySQL(db *gorm.DB, log *zap.Logger) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		if log != nil {
			log.Warn("get mysql sql db failed", zap.Error(err))
		}
		return
	}
	if err := sqlDB.Close(); err != nil && log != nil {
		log.Warn("close mysql db failed", zap.Error(err))
	}
}

func (a *App) Run() error {
	if a == nil {
		return errors.New("app is nil")
	}
	if a.Config == nil {
		return errors.New("app config is nil")
	}
	if a.Logger == nil {
		return errors.New("app logger is nil")
	}
	if a.Router == nil {
		return errors.New("app router is nil")
	}

	port := strings.TrimSpace(a.Config.App.Port)
	if port == "" {
		return errors.New("app port is empty")
	}
	addr := ":" + strings.TrimPrefix(port, ":")

	// Use http.Server directly so the app owns its lifecycle.
	srv := &http.Server{
		Addr:              addr,
		Handler:           a.Router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		a.Logger.Info("starting dcx-gateway", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("server listen failed: %w", err)
		}
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if a.PreloadJob != nil {
		a.PreloadJob.Start(signalCtx)
	}
	if a.MockPushJob != nil {
		a.MockPushJob.Start(signalCtx)
	}

	select {
	case err := <-serverErr:
		return err
	case <-signalCtx.Done():
	}

	a.Logger.Info("shutdown signal received, draining connections...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	a.Logger.Info("server exited cleanly")
	return nil
}

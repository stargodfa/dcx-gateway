package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PingFunc func(ctx context.Context) error

type HealthHandler struct {
	mysqlPing    PingFunc
	redisPing    PingFunc
	fastAPIWSURL string
	log          *zap.Logger
}

func NewHealthHandler(db *gorm.DB, rdb *redis.Client, fastAPIWSURL string, loggers ...*zap.Logger) *HealthHandler {
	return &HealthHandler{
		mysqlPing: func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
		},
		redisPing: func(ctx context.Context) error {
			return rdb.Ping(ctx).Err()
		},
		fastAPIWSURL: fastAPIWSURL,
		log:          firstLogger(loggers),
	}
}

func NewHealthHandlerWithPings(mysqlPing PingFunc, redisPing PingFunc, fastAPIWSURL string, loggers ...*zap.Logger) *HealthHandler {
	return &HealthHandler{
		mysqlPing:    mysqlPing,
		redisPing:    redisPing,
		fastAPIWSURL: fastAPIWSURL,
		log:          firstLogger(loggers),
	}
}

func (h *HealthHandler) Get(c *gin.Context) {
	if strings.ToLower(strings.TrimSpace(c.Query("deep"))) != "true" {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	checks := gin.H{}
	degraded := false
	if err := h.mysqlPing(c.Request.Context()); err != nil {
		degraded = true
		h.warnDependencyError("mysql", err)
		checks["mysql"] = gin.H{"status": "error"}
	} else {
		checks["mysql"] = gin.H{"status": "ok"}
	}
	if err := h.redisPing(c.Request.Context()); err != nil {
		degraded = true
		h.warnDependencyError("redis", err)
		checks["redis"] = gin.H{"status": "error"}
	} else {
		checks["redis"] = gin.H{"status": "ok"}
	}

	wsConfigured := strings.TrimSpace(h.fastAPIWSURL) != ""
	checks["fastapi_ws"] = gin.H{
		"status":     fastAPIWSStatus(wsConfigured),
		"configured": wsConfigured,
	}

	status := "ok"
	httpStatus := http.StatusOK
	if degraded {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}
	c.JSON(httpStatus, gin.H{
		"status": status,
		"checks": checks,
	})
}

func fastAPIWSStatus(configured bool) string {
	if configured {
		return "configured"
	}
	return "not_configured"
}

func (h *HealthHandler) warnDependencyError(name string, err error) {
	if h.log == nil || err == nil {
		return
	}
	h.log.Warn("deep health dependency check failed",
		zap.String("dependency", name),
		zap.Error(err),
	)
}

func firstLogger(loggers []*zap.Logger) *zap.Logger {
	if len(loggers) == 0 {
		return nil
	}
	return loggers[0]
}

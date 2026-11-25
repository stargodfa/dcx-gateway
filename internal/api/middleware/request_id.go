package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	ContextRequestID = "request_id"
	HeaderRequestID  = "X-Request-Id"
	HeaderTraceID    = "X-Trace-Id"

	maxRequestIDLength = 128
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := normalizeRequestID(c.GetHeader(HeaderRequestID))
		if requestID == "" {
			requestID = normalizeRequestID(c.GetHeader(HeaderTraceID))
		}
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set(ContextRequestID, requestID)
		c.Header(HeaderRequestID, requestID)
		c.Next()
	}
}

func RequestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if log == nil {
			return
		}
		log.Info("http request",
			zap.String("request_id", RequestIDFromContext(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}

func RequestIDFromContext(c *gin.Context) string {
	v, _ := c.Get(ContextRequestID)
	s, _ := v.(string)
	return s
}

func normalizeRequestID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > maxRequestIDLength {
		return value[:maxRequestIDLength]
	}
	return value
}

func generateRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b[:])
}

func IsRequestTooLarge(err error) bool {
	_, ok := err.(*http.MaxBytesError)
	return ok
}

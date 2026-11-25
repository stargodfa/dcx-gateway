package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"dcx-gateway/internal/model/protocol/ue"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type FastAPIClient interface {
	PushMessage(ctx context.Context, msg ue.WSMessage) error
	PushMessages(ctx context.Context, messages []ue.WSMessage) error
	Stats() FastAPIClientStats
	Close() error
}

type FastAPIClientStats struct {
	Enabled            bool             `json:"enabled"`
	Connected          bool             `json:"connected"`
	WSDataURL          string           `json:"ws_data_url,omitempty"`
	LastConnectTime    string           `json:"last_connect_time,omitempty"`
	LastDisconnectTime string           `json:"last_disconnect_time,omitempty"`
	LastPongTime       string           `json:"last_pong_time,omitempty"`
	LastPushTime       string           `json:"last_push_time,omitempty"`
	LastSuccessTime    string           `json:"last_success_time,omitempty"`
	LastErrorTime      string           `json:"last_error_time,omitempty"`
	LastError          string           `json:"last_error,omitempty"`
	SuccessCount       int64            `json:"success_count"`
	FailedCount        int64            `json:"failed_count"`
	ReconnectCount     int64            `json:"reconnect_count"`
	MessageCountByType map[string]int64 `json:"message_count_by_type"`
}

type FastAPIClientOption func(*fastAPIClientOptions)

type fastAPIClientOptions struct {
	headers        http.Header
	connectTimeout time.Duration
	pingInterval   time.Duration
	reconnectMin   time.Duration
	reconnectMax   time.Duration
}

func WithAuthHeader(name string, value string) FastAPIClientOption {
	return func(opts *fastAPIClientOptions) {
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" || value == "" {
			return
		}
		opts.headers.Set(name, value)
	}
}

func WithTimings(connectTimeout, pingInterval, reconnectMin, reconnectMax time.Duration) FastAPIClientOption {
	return func(opts *fastAPIClientOptions) {
		if connectTimeout > 0 {
			opts.connectTimeout = connectTimeout
		}
		if pingInterval > 0 {
			opts.pingInterval = pingInterval
		}
		if reconnectMin > 0 {
			opts.reconnectMin = reconnectMin
		}
		if reconnectMax > 0 {
			opts.reconnectMax = reconnectMax
		}
		if opts.reconnectMax < opts.reconnectMin {
			opts.reconnectMax = opts.reconnectMin
		}
	}
}

type fastAPIClient struct {
	url          string
	safeURL      string
	log          *zap.Logger
	opts         fastAPIClientOptions
	mu           sync.Mutex
	dialMu       sync.Mutex
	writeMu      sync.Mutex
	conn         *websocket.Conn
	closed       bool
	backoff      time.Duration
	nextDialTime time.Time
	stats        FastAPIClientStats
}

func NewFastAPIClient(rawURL string, log *zap.Logger, options ...FastAPIClientOption) FastAPIClient {
	if log == nil {
		log = zap.NewNop()
	}
	opts := fastAPIClientOptions{
		headers:        http.Header{},
		connectTimeout: 5 * time.Second,
		pingInterval:   20 * time.Second,
		reconnectMin:   time.Second,
		reconnectMax:   30 * time.Second,
	}
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}
	safeURL := sanitizeWSURL(rawURL)
	return &fastAPIClient{
		url:     strings.TrimSpace(rawURL),
		safeURL: safeURL,
		log:     log,
		opts:    opts,
		backoff: opts.reconnectMin,
		stats: FastAPIClientStats{
			Enabled:            strings.TrimSpace(rawURL) != "",
			Connected:          false,
			WSDataURL:          safeURL,
			MessageCountByType: map[string]int64{},
		},
	}
}

func (c *fastAPIClient) PushMessages(ctx context.Context, messages []ue.WSMessage) error {
	var lastErr error
	for _, msg := range messages {
		if err := c.PushMessage(ctx, msg); err != nil {
			lastErr = err
			if c.log != nil {
				c.log.Warn("push ws message failed",
					zap.String("type", msg.Type),
					zap.Int64("seq", msg.Seq),
					zap.String("ws_data_url", c.safeURL),
					zap.Error(err),
				)
			}
		}
	}
	return lastErr
}

func (c *fastAPIClient) PushMessage(ctx context.Context, msg ue.WSMessage) error {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		c.recordPushFailure(msg.Type, err)
		return err
	}

	b, err := json.Marshal(msg)
	if err != nil {
		c.recordPushFailure(msg.Type, err)
		return err
	}

	start := time.Now()
	c.writeMu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, b)
	c.writeMu.Unlock()
	if err != nil {
		c.markDisconnected(conn, "ws.write.failed", err)
		c.recordPushFailure(msg.Type, err)
		return err
	}

	c.recordPushSuccess(msg.Type)
	if c.log != nil {
		c.log.Debug("ws push success",
			zap.String("event", "ws.push.success"),
			zap.String("message_type", msg.Type),
			zap.Int64("seq", msg.Seq),
			zap.String("ws_data_url", c.safeURL),
			zap.Duration("duration", time.Since(start)),
		)
	}
	return nil
}

func (c *fastAPIClient) ensureConn(ctx context.Context) (*websocket.Conn, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("fastapi websocket client is closed")
	}
	if c.conn != nil {
		conn := c.conn
		c.mu.Unlock()
		return conn, nil
	}
	if c.url == "" {
		c.mu.Unlock()
		return nil, fmt.Errorf("fastapi websocket url is empty")
	}
	if wait := time.Until(c.nextDialTime); wait > 0 {
		c.mu.Unlock()
		return nil, fmt.Errorf("fastapi websocket reconnect backoff active for %s", wait.Round(time.Millisecond))
	}
	c.mu.Unlock()

	c.dialMu.Lock()
	defer c.dialMu.Unlock()

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("fastapi websocket client is closed")
	}
	if c.conn != nil {
		conn := c.conn
		c.mu.Unlock()
		return conn, nil
	}
	if wait := time.Until(c.nextDialTime); wait > 0 {
		c.mu.Unlock()
		return nil, fmt.Errorf("fastapi websocket reconnect backoff active for %s", wait.Round(time.Millisecond))
	}
	c.mu.Unlock()

	dialer := websocket.Dialer{HandshakeTimeout: c.opts.connectTimeout}
	conn, _, err := dialer.DialContext(ctx, c.url, cloneHeader(c.opts.headers))
	if err != nil {
		c.recordConnectFailure(err)
		return nil, err
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		_ = conn.Close()
		return nil, fmt.Errorf("fastapi websocket client is closed")
	}
	c.conn = conn
	now := time.Now()
	c.stats.Connected = true
	c.stats.LastConnectTime = formatStatTime(now)
	c.stats.LastError = ""
	c.stats.ReconnectCount++
	c.backoff = c.opts.reconnectMin
	c.nextDialTime = time.Time{}
	c.mu.Unlock()

	conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.stats.LastPongTime = formatStatTime(time.Now())
		c.mu.Unlock()
		return nil
	})
	go c.readLoop(conn)
	go c.pingLoop(conn)

	c.log.Info("connected to FastAPI ws_data",
		zap.String("event", "ws.connect.success"),
		zap.String("ws_data_url", c.safeURL),
	)
	return conn, nil
}

func (c *fastAPIClient) readLoop(conn *websocket.Conn) {
	for {
		if c.opts.pingInterval > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(c.opts.pingInterval * 3))
		}
		if _, _, err := conn.ReadMessage(); err != nil {
			c.markDisconnected(conn, "ws.read.closed", err)
			return
		}
	}
}

func (c *fastAPIClient) pingLoop(conn *websocket.Conn) {
	if c.opts.pingInterval <= 0 {
		return
	}
	ticker := time.NewTicker(c.opts.pingInterval)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		active := !c.closed && c.conn == conn
		c.mu.Unlock()
		if !active {
			return
		}
		deadline := time.Now().Add(5 * time.Second)
		c.writeMu.Lock()
		err := conn.WriteControl(websocket.PingMessage, []byte("ping"), deadline)
		c.writeMu.Unlock()
		if err != nil {
			c.markDisconnected(conn, "ws.ping.failed", err)
			return
		}
	}
}

func (c *fastAPIClient) markDisconnected(conn *websocket.Conn, event string, err error) {
	c.mu.Lock()
	if c.conn != conn {
		c.mu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
		return
	}
	c.conn = nil
	now := time.Now()
	c.stats.Connected = false
	c.stats.LastDisconnectTime = formatStatTime(now)
	c.stats.LastErrorTime = formatStatTime(now)
	c.stats.LastError = sanitizeError(err)
	c.nextDialTime = now.Add(c.backoff)
	if c.backoff <= 0 {
		c.backoff = c.opts.reconnectMin
	}
	c.backoff *= 2
	if c.backoff > c.opts.reconnectMax {
		c.backoff = c.opts.reconnectMax
	}
	c.mu.Unlock()

	_ = conn.Close()
	if c.log != nil {
		c.log.Warn("fastapi ws disconnected",
			zap.String("event", event),
			zap.String("ws_data_url", c.safeURL),
			zap.Error(err),
		)
	}
}

func (c *fastAPIClient) recordConnectFailure(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.stats.Connected = false
	c.stats.LastErrorTime = formatStatTime(now)
	c.stats.LastError = sanitizeError(err)
	c.nextDialTime = now.Add(c.backoff)
	if c.backoff <= 0 {
		c.backoff = c.opts.reconnectMin
	}
	c.backoff *= 2
	if c.backoff > c.opts.reconnectMax {
		c.backoff = c.opts.reconnectMax
	}
	if c.log != nil {
		c.log.Warn("connect to FastAPI ws_data failed",
			zap.String("event", "ws.connect.failed"),
			zap.String("ws_data_url", c.safeURL),
			zap.Error(err),
		)
	}
}

func (c *fastAPIClient) recordPushSuccess(messageType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.stats.LastPushTime = formatStatTime(now)
	c.stats.LastSuccessTime = formatStatTime(now)
	c.stats.SuccessCount++
	if c.stats.MessageCountByType == nil {
		c.stats.MessageCountByType = map[string]int64{}
	}
	c.stats.MessageCountByType[messageType]++
}

func (c *fastAPIClient) recordPushFailure(messageType string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.stats.LastPushTime = formatStatTime(now)
	c.stats.LastErrorTime = formatStatTime(now)
	c.stats.LastError = sanitizeError(err)
	c.stats.FailedCount++
}

func (c *fastAPIClient) Stats() FastAPIClientStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	stats := c.stats
	stats.MessageCountByType = make(map[string]int64, len(c.stats.MessageCountByType))
	for k, v := range c.stats.MessageCountByType {
		stats.MessageCountByType[k] = v
	}
	return stats
}

func (c *fastAPIClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.conn == nil {
		c.stats.Connected = false
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.stats.Connected = false
	c.stats.LastDisconnectTime = formatStatTime(time.Now())
	return err
}

func cloneHeader(src http.Header) http.Header {
	if len(src) == 0 {
		return nil
	}
	dst := make(http.Header, len(src))
	for k, values := range src {
		dst[k] = append([]string(nil), values...)
	}
	return dst
}

func sanitizeWSURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) > 240 {
		msg = msg[:240]
	}
	return msg
}

func formatStatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

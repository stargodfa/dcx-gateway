package config

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

const (
	DefaultSessionTTLSeconds     = 7200
	DefaultPreloadIntervalSecond = 60
	DefaultWSTimeoutSeconds      = 5
	DefaultWSPingSeconds         = 20
	DefaultWSReconnectMinSeconds = 1
	DefaultWSReconnectMaxSeconds = 30
	DefaultMockPushInterval      = 5
)

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	MySQL     MySQLConfig     `mapstructure:"mysql"`
	Redis     RedisConfig     `mapstructure:"redis"`
	FastAPI   FastAPIConfig   `mapstructure:"fastapi"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Security  SecurityConfig  `mapstructure:"security"`
	Server    ServerConfig    `mapstructure:"server"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	MockPush  MockPushConfig  `mapstructure:"mock_push"`
	Adapters  AdaptersConfig  `mapstructure:"adapters"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Port string `mapstructure:"port"`
}

type MySQLConfig struct {
	DSN string `mapstructure:"dsn"`
}

type RedisConfig struct {
	Addr      string `mapstructure:"addr"`
	Password  string `mapstructure:"password"`
	DB        int    `mapstructure:"db"`
	KeyPrefix string `mapstructure:"key_prefix"`
}

type FastAPIConfig struct {
	BaseURL                 string             `mapstructure:"base_url"`
	WSDataURL               string             `mapstructure:"ws_data_url"`
	PushVideoStreams        bool               `mapstructure:"push_video_streams"`
	WSAuth                  OutboundAuthConfig `mapstructure:"ws_auth"`
	WSConnectTimeoutSeconds int                `mapstructure:"ws_connect_timeout_seconds"`
	WSPingIntervalSeconds   int                `mapstructure:"ws_ping_interval_seconds"`
	WSReconnectMinSeconds   int                `mapstructure:"ws_reconnect_min_seconds"`
	WSReconnectMaxSeconds   int                `mapstructure:"ws_reconnect_max_seconds"`
}

// AuthConfig is kept for compatibility with older config files. New deployments
// should use security.session_ttl_seconds.
type AuthConfig struct {
	TokenTTLMinutes int `mapstructure:"token_ttl_minutes"`
}

type SecurityConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	ProtectRead        bool   `mapstructure:"protect_read"`
	StaticTokenEnabled bool   `mapstructure:"static_token_enabled"`
	StaticToken        string `mapstructure:"static_token"`
	SessionTTLSeconds  int    `mapstructure:"session_ttl_seconds"`
}

type ServerConfig struct {
	MaxBodyBytes int64 `mapstructure:"max_body_bytes"`
}

type SchedulerConfig struct {
	Preload SchedulerPreloadConfig `mapstructure:"preload"`
}

type SchedulerPreloadConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	IntervalSeconds int      `mapstructure:"interval_seconds"`
	PushWS          bool     `mapstructure:"push_ws"`
	Scripts         []string `mapstructure:"scripts"`
}

type MockPushConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	IntervalSeconds int      `mapstructure:"interval_seconds"`
	Scripts         []string `mapstructure:"scripts"`
}

type AdaptersConfig struct {
	Security  SecurityAdapterConfig  `mapstructure:"security"`
	Energy    EnergyAdapterConfig    `mapstructure:"energy"`
	Education EducationAdapterConfig `mapstructure:"education"`
}

type SecurityAdapterConfig struct {
	Mode              string             `mapstructure:"mode"`
	BaseURL           string             `mapstructure:"base_url"`
	TimeoutSeconds    int                `mapstructure:"timeout_seconds"`
	AccessSummaryPath string             `mapstructure:"access_summary_path"`
	FireAlertsPath    string             `mapstructure:"fire_alerts_path"`
	Auth              OutboundAuthConfig `mapstructure:"auth"`
}

type EnergyAdapterConfig struct {
	Mode           string             `mapstructure:"mode"`
	BaseURL        string             `mapstructure:"base_url"`
	TimeoutSeconds int                `mapstructure:"timeout_seconds"`
	SnapshotPath   string             `mapstructure:"snapshot_path"`
	Auth           OutboundAuthConfig `mapstructure:"auth"`
}

type EducationAdapterConfig struct {
	Mode               string             `mapstructure:"mode"`
	BaseURL            string             `mapstructure:"base_url"`
	TimeoutSeconds     int                `mapstructure:"timeout_seconds"`
	ClassroomUsagePath string             `mapstructure:"classroom_usage_path"`
	Auth               OutboundAuthConfig `mapstructure:"auth"`
}

type OutboundAuthConfig struct {
	Type       string `mapstructure:"type"`
	HeaderName string `mapstructure:"header_name"`
	Token      string `mapstructure:"token"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	normalizeConfig(&cfg)
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func normalizeConfig(cfg *Config) {
	if cfg.Security.SessionTTLSeconds <= 0 {
		if cfg.Auth.TokenTTLMinutes > 0 {
			cfg.Security.SessionTTLSeconds = cfg.Auth.TokenTTLMinutes * 60
		} else {
			cfg.Security.SessionTTLSeconds = DefaultSessionTTLSeconds
		}
	}
	if cfg.Server.MaxBodyBytes <= 0 {
		cfg.Server.MaxBodyBytes = 8 * 1024 * 1024
	}
	if cfg.Scheduler.Preload.IntervalSeconds <= 0 {
		cfg.Scheduler.Preload.IntervalSeconds = DefaultPreloadIntervalSecond
	}
	if cfg.MockPush.IntervalSeconds <= 0 {
		cfg.MockPush.IntervalSeconds = DefaultMockPushInterval
	}
	normalizeFastAPIConfig(&cfg.FastAPI)
	normalizeAdapterConfig(&cfg.Adapters)
}

func normalizeFastAPIConfig(cfg *FastAPIConfig) {
	if cfg.WSConnectTimeoutSeconds <= 0 {
		cfg.WSConnectTimeoutSeconds = DefaultWSTimeoutSeconds
	}
	if cfg.WSPingIntervalSeconds <= 0 {
		cfg.WSPingIntervalSeconds = DefaultWSPingSeconds
	}
	if cfg.WSReconnectMinSeconds <= 0 {
		cfg.WSReconnectMinSeconds = DefaultWSReconnectMinSeconds
	}
	if cfg.WSReconnectMaxSeconds <= 0 {
		cfg.WSReconnectMaxSeconds = DefaultWSReconnectMaxSeconds
	}
	if cfg.WSReconnectMaxSeconds < cfg.WSReconnectMinSeconds {
		cfg.WSReconnectMaxSeconds = cfg.WSReconnectMinSeconds
	}
	cfg.WSAuth = NormalizeOutboundAuth(cfg.WSAuth)
}

func normalizeAdapterConfig(cfg *AdaptersConfig) {
	cfg.Security.Mode = normalizeAdapterMode(cfg.Security.Mode)
	cfg.Security.Auth = NormalizeOutboundAuth(cfg.Security.Auth)
	if cfg.Security.TimeoutSeconds <= 0 {
		cfg.Security.TimeoutSeconds = 3
	}
	if strings.TrimSpace(cfg.Security.AccessSummaryPath) == "" {
		cfg.Security.AccessSummaryPath = "/api/security/access-summary"
	}
	if strings.TrimSpace(cfg.Security.FireAlertsPath) == "" {
		cfg.Security.FireAlertsPath = "/api/security/fire-alerts"
	}

	cfg.Energy.Mode = normalizeAdapterMode(cfg.Energy.Mode)
	cfg.Energy.Auth = NormalizeOutboundAuth(cfg.Energy.Auth)
	if cfg.Energy.TimeoutSeconds <= 0 {
		cfg.Energy.TimeoutSeconds = 3
	}
	if strings.TrimSpace(cfg.Energy.SnapshotPath) == "" {
		cfg.Energy.SnapshotPath = "/api/energy/snapshot"
	}

	cfg.Education.Mode = normalizeAdapterMode(cfg.Education.Mode)
	cfg.Education.Auth = NormalizeOutboundAuth(cfg.Education.Auth)
	if cfg.Education.TimeoutSeconds <= 0 {
		cfg.Education.TimeoutSeconds = 3
	}
	if strings.TrimSpace(cfg.Education.ClassroomUsagePath) == "" {
		cfg.Education.ClassroomUsagePath = "/api/education/classroom-usage"
	}
}

func NormalizeOutboundAuth(auth OutboundAuthConfig) OutboundAuthConfig {
	auth.Type = strings.ToLower(strings.TrimSpace(auth.Type))
	auth.HeaderName = strings.TrimSpace(auth.HeaderName)
	auth.Token = strings.TrimSpace(auth.Token)
	if auth.Type == "" {
		auth.Type = "none"
	}
	if auth.Type == "api_key" && auth.HeaderName == "" {
		auth.HeaderName = "X-API-Key"
	}
	return auth
}

func ApplyOutboundAuthHeader(header http.Header, auth OutboundAuthConfig) bool {
	auth = NormalizeOutboundAuth(auth)
	if header == nil || auth.Token == "" {
		return false
	}
	switch auth.Type {
	case "bearer":
		header.Set("Authorization", "Bearer "+auth.Token)
		return true
	case "api_key", "header":
		if auth.HeaderName == "" {
			return false
		}
		header.Set(auth.HeaderName, auth.Token)
		return true
	default:
		return false
	}
}

func normalizeAdapterMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return "mock"
	}
	return mode
}

func validateConfig(cfg *Config) error {
	if cfg.Security.Enabled && cfg.Security.StaticTokenEnabled && strings.TrimSpace(cfg.Security.StaticToken) == "" {
		return fmt.Errorf("security.static_token is required when security.enabled and security.static_token_enabled are true")
	}
	return nil
}

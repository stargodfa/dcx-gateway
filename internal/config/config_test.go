package config

import "testing"

func TestNormalizeConfigSchedulerPreloadDefaults(t *testing.T) {
	cfg := Config{}
	normalizeConfig(&cfg)

	if cfg.Scheduler.Preload.IntervalSeconds != DefaultPreloadIntervalSecond {
		t.Fatalf("interval = %d, want %d", cfg.Scheduler.Preload.IntervalSeconds, DefaultPreloadIntervalSecond)
	}
	if cfg.Scheduler.Preload.Enabled {
		t.Fatal("missing scheduler config should not enable preload")
	}
	if cfg.Scheduler.Preload.PushWS {
		t.Fatal("scheduler preload push_ws should default false")
	}
	if cfg.MockPush.Enabled {
		t.Fatal("mock_push should default disabled")
	}
	if cfg.MockPush.IntervalSeconds != DefaultMockPushInterval {
		t.Fatalf("mock_push interval = %d, want %d", cfg.MockPush.IntervalSeconds, DefaultMockPushInterval)
	}
}

func TestNormalizeConfigSchedulerPreloadKeepsConfiguredValues(t *testing.T) {
	cfg := Config{
		Scheduler: SchedulerConfig{
			Preload: SchedulerPreloadConfig{
				Enabled:         true,
				IntervalSeconds: 15,
				Scripts:         []string{"security_main", "energy_main"},
			},
		},
	}
	normalizeConfig(&cfg)

	if !cfg.Scheduler.Preload.Enabled {
		t.Fatal("enabled should be preserved")
	}
	if cfg.Scheduler.Preload.IntervalSeconds != 15 {
		t.Fatalf("interval = %d, want 15", cfg.Scheduler.Preload.IntervalSeconds)
	}
	if len(cfg.Scheduler.Preload.Scripts) != 2 {
		t.Fatalf("scripts = %#v", cfg.Scheduler.Preload.Scripts)
	}
}

func TestNormalizeConfigAdapterDefaults(t *testing.T) {
	cfg := Config{}
	normalizeConfig(&cfg)

	if cfg.Adapters.Security.Mode != "mock" || cfg.Adapters.Energy.Mode != "mock" || cfg.Adapters.Education.Mode != "mock" {
		t.Fatalf("adapter modes = %#v", cfg.Adapters)
	}
	if cfg.Adapters.Security.TimeoutSeconds != 3 || cfg.Adapters.Energy.TimeoutSeconds != 3 || cfg.Adapters.Education.TimeoutSeconds != 3 {
		t.Fatalf("adapter timeouts = %#v", cfg.Adapters)
	}
	if cfg.Adapters.Security.AccessSummaryPath != "/api/security/access-summary" {
		t.Fatalf("security access path = %s", cfg.Adapters.Security.AccessSummaryPath)
	}
	if cfg.Adapters.Energy.SnapshotPath != "/api/energy/snapshot" {
		t.Fatalf("energy snapshot path = %s", cfg.Adapters.Energy.SnapshotPath)
	}
	if cfg.Adapters.Education.ClassroomUsagePath != "/api/education/classroom-usage" {
		t.Fatalf("education usage path = %s", cfg.Adapters.Education.ClassroomUsagePath)
	}
	if cfg.Adapters.Security.Auth.Type != "none" || cfg.Adapters.Energy.Auth.Type != "none" || cfg.Adapters.Education.Auth.Type != "none" {
		t.Fatalf("adapter auth defaults = %#v", cfg.Adapters)
	}
	if cfg.FastAPI.WSAuth.Type != "none" {
		t.Fatalf("fastapi ws auth = %#v", cfg.FastAPI.WSAuth)
	}
	if cfg.FastAPI.WSConnectTimeoutSeconds != DefaultWSTimeoutSeconds ||
		cfg.FastAPI.WSPingIntervalSeconds != DefaultWSPingSeconds ||
		cfg.FastAPI.WSReconnectMinSeconds != DefaultWSReconnectMinSeconds ||
		cfg.FastAPI.WSReconnectMaxSeconds != DefaultWSReconnectMaxSeconds {
		t.Fatalf("fastapi ws timing defaults = %#v", cfg.FastAPI)
	}
}

func TestNormalizeConfigAdapterConfiguredValues(t *testing.T) {
	cfg := Config{
		Adapters: AdaptersConfig{
			Security: SecurityAdapterConfig{Mode: "HTTP", TimeoutSeconds: 5, AccessSummaryPath: "/s/access", FireAlertsPath: "/s/fire"},
			Energy:   EnergyAdapterConfig{Mode: "http", TimeoutSeconds: 6, SnapshotPath: "/e/snapshot"},
			Education: EducationAdapterConfig{
				Mode:               "mock",
				TimeoutSeconds:     7,
				ClassroomUsagePath: "/ed/usage",
			},
		},
	}
	normalizeConfig(&cfg)

	if cfg.Adapters.Security.Mode != "http" || cfg.Adapters.Energy.Mode != "http" || cfg.Adapters.Education.Mode != "mock" {
		t.Fatalf("adapter modes = %#v", cfg.Adapters)
	}
	if cfg.Adapters.Security.TimeoutSeconds != 5 || cfg.Adapters.Energy.TimeoutSeconds != 6 || cfg.Adapters.Education.TimeoutSeconds != 7 {
		t.Fatalf("adapter timeouts = %#v", cfg.Adapters)
	}
}

func TestNormalizeOutboundAuth(t *testing.T) {
	auth := NormalizeOutboundAuth(OutboundAuthConfig{Type: "API_KEY", Token: " secret "})
	if auth.Type != "api_key" || auth.HeaderName != "X-API-Key" || auth.Token != "secret" {
		t.Fatalf("auth = %#v", auth)
	}
}

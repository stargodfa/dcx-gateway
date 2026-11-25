package bootstrap

import (
	"testing"

	"dcx-gateway/internal/adapter/education"
	"dcx-gateway/internal/adapter/energy"
	"dcx-gateway/internal/adapter/security"
	"dcx-gateway/internal/config"

	"go.uber.org/zap"
)

func TestNewSecurityAdapterSelection(t *testing.T) {
	log := zap.NewNop()

	if _, ok := newSecurityAdapter(config.SecurityAdapterConfig{Mode: "mock"}, log).(*security.MockAdapter); !ok {
		t.Fatal("mock mode should create security mock adapter")
	}
	if _, ok := newSecurityAdapter(config.SecurityAdapterConfig{Mode: "http", BaseURL: "http://example.test"}, log).(*security.HTTPAdapter); !ok {
		t.Fatal("http mode with base_url should create security http adapter")
	}
	if _, ok := newSecurityAdapter(config.SecurityAdapterConfig{Mode: "http"}, log).(*security.MockAdapter); !ok {
		t.Fatal("http mode without base_url should fallback to security mock adapter")
	}
	if _, ok := newSecurityAdapter(config.SecurityAdapterConfig{Mode: "bad"}, log).(*security.MockAdapter); !ok {
		t.Fatal("unknown mode should fallback to security mock adapter")
	}
}

func TestNewEnergyAdapterSelection(t *testing.T) {
	log := zap.NewNop()

	if _, ok := newEnergyAdapter(config.EnergyAdapterConfig{Mode: "mock"}, log).(*energy.MockAdapter); !ok {
		t.Fatal("mock mode should create energy mock adapter")
	}
	if _, ok := newEnergyAdapter(config.EnergyAdapterConfig{Mode: "http", BaseURL: "http://example.test"}, log).(*energy.HTTPAdapter); !ok {
		t.Fatal("http mode with base_url should create energy http adapter")
	}
	if _, ok := newEnergyAdapter(config.EnergyAdapterConfig{Mode: "http"}, log).(*energy.MockAdapter); !ok {
		t.Fatal("http mode without base_url should fallback to energy mock adapter")
	}
	if _, ok := newEnergyAdapter(config.EnergyAdapterConfig{Mode: "bad"}, log).(*energy.MockAdapter); !ok {
		t.Fatal("unknown mode should fallback to energy mock adapter")
	}
}

func TestNewEducationAdapterSelection(t *testing.T) {
	log := zap.NewNop()

	if _, ok := newEducationAdapter(config.EducationAdapterConfig{Mode: "mock"}, log).(*education.MockAdapter); !ok {
		t.Fatal("mock mode should create education mock adapter")
	}
	if _, ok := newEducationAdapter(config.EducationAdapterConfig{Mode: "http", BaseURL: "http://example.test"}, log).(*education.HTTPAdapter); !ok {
		t.Fatal("http mode with base_url should create education http adapter")
	}
	if _, ok := newEducationAdapter(config.EducationAdapterConfig{Mode: "http"}, log).(*education.MockAdapter); !ok {
		t.Fatal("http mode without base_url should fallback to education mock adapter")
	}
	if _, ok := newEducationAdapter(config.EducationAdapterConfig{Mode: "bad"}, log).(*education.MockAdapter); !ok {
		t.Fatal("unknown mode should fallback to education mock adapter")
	}
}

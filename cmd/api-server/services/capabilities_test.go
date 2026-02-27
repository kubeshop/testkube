package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/internal/config"
)

func TestAgentCapabilities_Persona(t *testing.T) {
	tests := []struct {
		name    string
		agentID string
		want    string
	}{
		{"runner persona", "tkcrun_abc123", "persona:runner"},
		{"sync persona", "tkcsync_abc123", "persona:sync"},
		{"default persona empty", "", "persona:default"},
		{"default persona regular", "tkcagnt_abc123", "persona:default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.TestkubeProAgentID = tt.agentID
			caps := AgentCapabilities(cfg)
			assert.Contains(t, caps, tt.want)
		})
	}
}

func TestAgentCapabilities_Mode(t *testing.T) {
	t.Run("connected mode", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.TestkubeProAPIKey = "tkcagnt_key"
		caps := AgentCapabilities(cfg)
		assert.Contains(t, caps, "mode:connected")
		assert.NotContains(t, caps, "mode:standalone")
	})

	t.Run("standalone mode", func(t *testing.T) {
		cfg := &config.Config{}
		caps := AgentCapabilities(cfg)
		assert.Contains(t, caps, "mode:standalone")
		assert.NotContains(t, caps, "mode:connected")
	})
}

func TestAgentCapabilities_Features(t *testing.T) {
	t.Run("all features enabled", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.EnableK8sControllers = true
		cfg.EnableK8sEvents = true
		cfg.EnableCronJobs = "true"
		cfg.EnableCloudWebhooks = true
		cfg.GitOpsSyncKubernetesToCloudEnabled = true
		cfg.FeatureCloudStorage = true
		cfg.TracingEnabled = true
		cfg.FloatingRunner = true

		caps := AgentCapabilities(cfg)
		assert.Contains(t, caps, "runner")
		assert.Contains(t, caps, "floating-runner")
		assert.Contains(t, caps, "k8s-controllers")
		assert.Contains(t, caps, "k8s-events")
		assert.Contains(t, caps, "cron-jobs")
		assert.Contains(t, caps, "webhooks")
		assert.Contains(t, caps, "cloud-webhooks")
		assert.Contains(t, caps, "gitops-sync")
		assert.Contains(t, caps, "cloud-storage")
		assert.Contains(t, caps, "tracing")
	})

	t.Run("features disabled", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.DisableRunner = true
		cfg.DisableWebhooks = true
		cfg.EnableCronJobs = "false"

		caps := AgentCapabilities(cfg)
		assert.NotContains(t, caps, "runner")
		assert.NotContains(t, caps, "webhooks")
		assert.NotContains(t, caps, "cron-jobs")
		assert.NotContains(t, caps, "floating-runner")
		assert.NotContains(t, caps, "k8s-controllers")
		assert.NotContains(t, caps, "cloud-webhooks")
		assert.NotContains(t, caps, "gitops-sync")
		assert.NotContains(t, caps, "cloud-storage")
		assert.NotContains(t, caps, "tracing")
	})

	t.Run("cron jobs empty string means not explicitly enabled", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.EnableCronJobs = ""
		caps := AgentCapabilities(cfg)
		assert.NotContains(t, caps, "cron-jobs")
	})
}

func TestAgentCapabilities_RunnerPersonaOverrides(t *testing.T) {
	cfg := &config.Config{}
	cfg.TestkubeProAgentID = "tkcrun_abc"
	cfg.TestkubeProAPIKey = "some-key"
	cfg.DisableWebhooks = true
	cfg.EnableCronJobs = "false"
	cfg.EnableK8sControllers = false
	cfg.DisableDefaultAgent = true

	caps := AgentCapabilities(cfg)
	assert.Contains(t, caps, "persona:runner")
	assert.Contains(t, caps, "mode:connected")
	assert.NotContains(t, caps, "webhooks")
	assert.NotContains(t, caps, "cron-jobs")
	assert.NotContains(t, caps, "k8s-controllers")
}

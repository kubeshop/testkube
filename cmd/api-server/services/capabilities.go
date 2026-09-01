package services

import (
	"strings"

	"github.com/kubeshop/testkube/internal/config"
)

// hostedRunnerNamePrefix identifies runners that Testkube provisions itself for
// trial users rather than ones deployed into a user's own infrastructure. The
// control plane assigns the name at provisioning time; it must stay in sync with
// naming.HostedRunnerAgentName in testkube-cloud-api.
const hostedRunnerNamePrefix = "tkcagent_hr_"

// AgentCapabilities extracts a list of capability tags from the agent
// configuration. These are included in heartbeat and start telemetry events
// to provide visibility into how the agent is deployed.
func AgentCapabilities(cfg *config.Config) []string {
	var caps []string

	switch {
	case strings.HasPrefix(cfg.TestkubeProAgentID, "tkcrun_"):
		caps = append(caps, "persona:runner")
	case strings.HasPrefix(cfg.TestkubeProAgentID, "tkcsync_"):
		caps = append(caps, "persona:sync")
	default:
		caps = append(caps, "persona:default")
	}

	if cfg.TestkubeProAPIKey != "" {
		caps = append(caps, "mode:connected")
	} else {
		caps = append(caps, "mode:standalone")
	}

	if !cfg.DisableRunner {
		caps = append(caps, "runner")
	}
	if cfg.FloatingRunner {
		caps = append(caps, "floating-runner")
	}
	if strings.HasPrefix(cfg.RunnerName, hostedRunnerNamePrefix) {
		caps = append(caps, "hosted-runner")
	}
	if cfg.EnableK8sControllers {
		caps = append(caps, "k8s-controllers")
	}
	if cfg.EnableK8sEvents {
		caps = append(caps, "k8s-events")
	}
	if !strings.EqualFold(cfg.EnableCronJobs, "false") {
		caps = append(caps, "cron-jobs")
	}
	if !cfg.DisableWebhooks {
		caps = append(caps, "webhooks")
	}
	if cfg.EnableCloudWebhooks {
		caps = append(caps, "cloud-webhooks")
	}
	if cfg.GitOpsSyncKubernetesToCloudEnabled {
		caps = append(caps, "gitops-sync")
	}
	if cfg.FeatureCloudStorage {
		caps = append(caps, "cloud-storage")
	}
	if cfg.TracingEnabled {
		caps = append(caps, "tracing")
	}

	return caps
}

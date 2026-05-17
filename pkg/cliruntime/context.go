// Package cliruntime exposes lightweight runtime-context detection helpers
// (Docker, CI, interactive local). It is intentionally dependency-free so it
// can be imported by both the telemetry package and the CLI commands packages
// without creating an import cycle.
package cliruntime

import (
	"os"
	"runtime"
	"strings"
)

// CliRunContextLocal is the value returned by CliRunContext when the CLI is
// running interactively on a developer machine (i.e. not inside a known CI
// system, Docker container, or Kubernetes pod).
const CliRunContextLocal = "others|local"

// IsRunningInDocker detects whether the current process is running inside a
// Docker (or compatible OCI) container.
//
// The detection deliberately ignores Docker *client* configuration variables
// (DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH, DOCKER_MACHINE_NAME,
// DOCKER_BUILDKIT) because those are commonly exported on developer
// workstations talking to a remote or alternative daemon (docker-machine,
// Colima, lima, etc.) and do not imply the current process itself is
// containerized. We rely instead on filesystem/cgroup markers that only exist
// inside a container, plus env vars that are conventionally set *by* a
// container image rather than by the local shell.
func IsRunningInDocker() bool {
	// /.dockerenv is the canonical marker for Docker; other runtimes (podman,
	// containerd-via-Docker-API) typically replicate it.
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// DOCKER_CONTAINER is conventionally set by container images themselves
	// (e.g. via ENV in a Dockerfile) to signal in-container execution to
	// child processes. It is not a Docker client variable.
	if _, exists := os.LookupEnv("DOCKER_CONTAINER"); exists {
		return true
	}

	if runtime.GOOS == "linux" {
		if cgroupData, err := os.ReadFile("/proc/1/cgroup"); err == nil {
			content := string(cgroupData)
			if strings.Contains(content, "docker") || strings.Contains(content, "containerd") {
				return true
			}
		}
	}

	return false
}

// DockerContext returns a non-empty descriptor when the CLI is running inside
// a known containerized environment (Docker Compose, Kubernetes pod, etc.).
// Returns an empty string when not inside a container.
func DockerContext() string {
	if !IsRunningInDocker() {
		return ""
	}

	if _, ok := os.LookupEnv("COMPOSE_PROJECT_NAME"); ok {
		if projectName := os.Getenv("COMPOSE_PROJECT_NAME"); projectName != "" {
			return "docker-compose:" + projectName
		}
		return "docker-compose"
	}
	if _, ok := os.LookupEnv("DOCKER_SWARM"); ok {
		return "docker-swarm"
	}
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		if namespace := os.Getenv("POD_NAMESPACE"); namespace != "" {
			return "kubernetes:" + namespace
		}
		return "kubernetes"
	}
	if _, ok := os.LookupEnv("DOCKER_DESKTOP"); ok {
		return "docker-desktop"
	}

	if runtime.GOOS == "linux" {
		if cgroupData, err := os.ReadFile("/proc/1/cgroup"); err == nil {
			content := string(cgroupData)
			if strings.Contains(content, "containerd") {
				return "containerd"
			}
			if strings.Contains(content, "crio") {
				return "cri-o"
			}
		}
	}

	if version, ok := os.LookupEnv("TESTKUBE_DOCKER_IMAGE_VERSION"); ok {
		return "docker:testkube:" + version
	}

	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		return "docker:github-actions"
	}
	if _, ok := os.LookupEnv("CIRCLECI"); ok {
		return "docker:circleci"
	}
	if _, ok := os.LookupEnv("GITLAB_CI"); ok {
		return "docker:gitlab-ci"
	}
	if _, ok := os.LookupEnv("BUILDKITE"); ok {
		return "docker:buildkite"
	}

	if _, ok := os.LookupEnv("AWS_EXECUTION_ENV"); ok {
		return "docker:aws"
	}
	if _, ok := os.LookupEnv("GOOGLE_CLOUD_PROJECT"); ok {
		return "docker:gcp"
	}
	if _, ok := os.LookupEnv("AZURE_CONTAINER_REGISTRY"); ok {
		return "docker:azure"
	}

	return "docker"
}

// CliRunContext returns a stable identifier describing where the CLI is
// running: a Docker-derived string when inside a container, a CI system name
// when run from a known CI, or CliRunContextLocal otherwise. Callers can use
// the CliRunContextLocal sentinel to decide whether to emit interactive-only
// output (e.g. update-check hints).
func CliRunContext() string {
	if dockerContext := DockerContext(); dockerContext != "" {
		return dockerContext
	}

	if value, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		if value == "true" {
			return "github-actions"
		}
	}
	if _, ok := os.LookupEnv("TF_BUILD"); ok {
		return "azure-pipelines"
	}
	if _, ok := os.LookupEnv("JENKINS_URL"); ok {
		return "jenkins"
	}
	if _, ok := os.LookupEnv("JENKINS_HOME"); ok {
		return "jenkins"
	}
	if _, ok := os.LookupEnv("CIRCLECI"); ok {
		return "circleci"
	}
	if _, ok := os.LookupEnv("GITLAB_CI"); ok {
		return "gitlab-ci"
	}
	if _, ok := os.LookupEnv("BUILDKITE"); ok {
		return "buildkite"
	}
	if _, ok := os.LookupEnv("TRAVIS"); ok {
		return "travis-ci"
	}
	if _, ok := os.LookupEnv("AIRFLOW_HOME"); ok {
		return "airflow"
	}
	if _, ok := os.LookupEnv("TEAMCITY_VERSION"); ok {
		return "teamcity"
	}
	if _, ok := os.LookupEnv("GO_PIPELINE_NAME"); ok {
		return "gocd"
	}
	if _, ok := os.LookupEnv("SEMAPHORE"); ok {
		return "semaphore-ci"
	}
	if _, ok := os.LookupEnv("BITBUCKET_BUILD_NUMBER"); ok {
		return "bitbucket-pipelines"
	}
	if _, ok := os.LookupEnv("DRONE"); ok {
		return "drone"
	}

	return CliRunContextLocal
}

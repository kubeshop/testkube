variable "GOCACHE"       { default = "/go/pkg" }
variable "GOMODCACHE"    { default = "/root/.cache/go-build" }
variable "BUSYBOX_IMAGE" { default = "busybox:1.36.1-musl"}
variable "ALPINE_IMAGE"  { default = "alpine:3.20.8" }
variable "VERSION"       { default = "0.0.0-unknown"}

variable "GIT_SHA"                 { default = ""}
variable "SLACK_BOT_CLIENT_ID"     { default = ""}
variable "SLACK_BOT_CLIENT_SECRET" { default = ""}
variable "ANALYTICS_TRACKING_ID"   { default = ""}
variable "ANALYTICS_API_KEY"       { default = ""}
variable "SEGMENTIO_KEY"           { default = ""}
variable "CLOUD_SEGMENTIO_KEY"     { default = ""}
variable "KEYGEN_PUBLIC_KEY"       { default = ""}

group "default" {
  targets = ["api", "cli", "tw-init", "tw-toolkit", "mcp-server"]
}

target "api-meta" {}
target "api" {
  inherits = ["api-meta"]
  context="."
  dockerfile = "build/new/api-server.Dockerfile"
  platforms = ["linux/arm64", "linux/amd64"]
  args = {
    VERSION = "${VERSION}"
    GIT_SHA = "${GIT_SHA}"
    SLACK_BOT_CLIENT_ID = "${SLACK_BOT_CLIENT_ID}"
    SLACK_BOT_CLIENT_SECRET = "${SLACK_BOT_CLIENT_SECRET}"
    ANALYTICS_TRACKING_ID = "${ANALYTICS_TRACKING_ID}"
    ANALYTICS_API_KEY = "${ANALYTICS_API_KEY}"
    SEGMENTIO_KEY = "${SEGMENTIO_KEY}"
    CLOUD_SEGMENTIO_KEY = "${CLOUD_SEGMENTIO_KEY}"
    BUSYBOX_IMAGE = "${BUSYBOX_IMAGE}"
    ALPINE_IMAGE = "${ALPINE_IMAGE}"
  }
}

target "cli-meta" {}
target "cli" {
  inherits = ["cli-meta"]
  context="."
  dockerfile = "build/new/cli.Dockerfile"
  platforms = ["linux/arm64", "linux/amd64"]
  args = {
    VERSION = "${VERSION}"
    GIT_SHA = "${GIT_SHA}"
    ANALYTICS_TRACKING_ID = "${ANALYTICS_TRACKING_ID}"
    ANALYTICS_API_KEY = "${ANALYTICS_API_KEY}"
    ALPINE_IMAGE = "${ALPINE_IMAGE}"
    KEYGEN_PUBLIC_KEY="${KEYGEN_PUBLIC_KEY}"
  }
}

target "tw-init-meta" {}
target "tw-init" {
  inherits = ["tw-init-meta"]
  context="."
  dockerfile = "build/new/tw-init.Dockerfile"
  platforms = ["linux/arm64", "linux/amd64"]
  args = {
    BUSYBOX_IMAGE = "${BUSYBOX_IMAGE}"
    ALPINE_IMAGE = "${ALPINE_IMAGE}"
  }
}

target "tw-toolkit-meta" {}
target "tw-toolkit" {
  inherits = ["tw-toolkit-meta"]
  context="."
  dockerfile = "build/new/tw-toolkit.Dockerfile"
  platforms = ["linux/arm64", "linux/amd64"]
  args = {
    BUSYBOX_IMAGE = "${BUSYBOX_IMAGE}"
    ALPINE_IMAGE = "${ALPINE_IMAGE}"
  }
}

target "agent-server-meta" {}
target "agent-server" {
  inherits = ["agent-server-meta"]
  context="."
  dockerfile = "build/_local/agent-server.Dockerfile"
  platforms = ["linux/arm64"]
  args = {
    GOCACHE = "${GOCACHE}"
    GOMODCACHE = "${GOMODCACHE}"
    VERSION = "${VERSION}"
    GIT_SHA = "${GIT_SHA}"
    SLACK_BOT_CLIENT_ID = "${SLACK_BOT_CLIENT_ID}"
    SLACK_BOT_CLIENT_SECRET = "${SLACK_BOT_CLIENT_SECRET}"
    ANALYTICS_TRACKING_ID = "${ANALYTICS_TRACKING_ID}"
    ANALYTICS_API_KEY = "${ANALYTICS_API_KEY}"
    SEGMENTIO_KEY = "${SEGMENTIO_KEY}"
    CLOUD_SEGMENTIO_KEY = "${CLOUD_SEGMENTIO_KEY}"
  }
}

target "testworkflow-init-meta" {}
target "testworkflow-init" {
  inherits = ["testworkflow-init-meta"]
  context="."
  dockerfile = "build/_local/testworkflow-init.Dockerfile"
  platforms = ["linux/arm64"]
  args = {
    GOCACHE = "${GOCACHE}"
    GOMODCACHE = "${GOMODCACHE}"
  }
}

target "testworkflow-toolkit-meta" {}
target "testworkflow-toolkit" {
  inherits = ["testworkflow-toolkit-meta"]
  context="."
  dockerfile = "build/_local/testworkflow-toolkit.Dockerfile"
  platforms = ["linux/arm64"]
  args = {
    GOCACHE = "${GOCACHE}"
    GOMODCACHE = "${GOMODCACHE}"
  }
}

target "mcp-server-meta" {}
target "mcp-server" {
  inherits = ["mcp-server-meta"]
  context="."
  dockerfile = "build/mcp-server/Dockerfile"
  platforms = ["linux/arm64", "linux/amd64"]
  args = {
    VERSION = "${VERSION}"
    GIT_SHA = "${GIT_SHA}"
    ALPINE_IMAGE = "${ALPINE_IMAGE}"
  }
}

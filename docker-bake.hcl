variable "GOCACHE" {
  default = "/go/pkg"
}
variable "GOMODCACHE" {
  default = "/root/.cache/go-build"
}
variable "BUSYBOX_IMAGE" {
  default = "busybox:1.36.1-musl"
}
variable "ALPINE_IMAGE" {
  default = "alpine:3.20.3"
}

group "default" {
  targets = ["agent-server", "testworkflow-init", "testworkflow-toolkit"]
}

target "api-meta" {}
target "api" {
  inherits = ["api-meta"]
  context="."
  dockerfile = "build/new/api-server.Dockerfile"
  platforms = ["linux/arm64", "linux/amd64"]
  args = {
    BUSYBOX_IMAGE = "${BUSYBOX_IMAGE}"
    ALPINE_IMAGE = "${ALPINE_IMAGE}"
  }
}

target "tw-init-meta" {}
target "tw-init" {
  inherits = ["testworkflow-init-meta"]
  context="."
  dockerfile = "build/new/tw-init.Dockerfile"
  platforms = ["linux/arm64"]
  args = {
    BUSYBOX_IMAGE = "${BUSYBOX_IMAGE}"
    ALPINE_IMAGE = "${ALPINE_IMAGE}"
  }
}

target "tw-toolkit-meta" {}
target "tw-toolkit" {
  inherits = ["testworkflow-toolkit-meta"]
  context="."
  dockerfile = "build/new/tw-toolkit.Dockerfile"
  platforms = ["linux/arm64"]
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

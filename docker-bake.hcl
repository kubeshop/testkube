variable "GOCACHE" {
  default = "/go/pkg"
}
variable "GOMODCACHE" {
  default = "/root/.cache/go-build"
}

group "default" {
  targets = ["agent-server", "testworkflow-init", "testworkflow-toolkit"]
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

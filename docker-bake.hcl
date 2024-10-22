variable "GOCACHE" {
  default = "/go/pkg"
}
variable "GOMODCACHE" {
  default = "/root/.cache/go-build"
}

group "default" {
  targets = ["agent-server"]
}

target "agent-server-meta" {
  tags = ["kubeshop/tk-agent-server:dev"]
}
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

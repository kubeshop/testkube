// docker-bake.hcl
target "docker-metadata-action" {}

group "default" {
    targets = ["npm", "pnpm", "yarn"]
}

target "npm" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.npm"
  platforms = [
    "linux/amd64",
    "linux/arm64"
  ]
}

target "pnpm" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.pnpm"
  platforms = [
    "linux/amd64",
    "linux/arm64"
  ]
}

target "yarn" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.yarn"
  platforms = [
    "linux/amd64",
    "linux/arm64"
  ]
}
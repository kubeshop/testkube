// docker-bake.hcl
target "docker-metadata-action" {}

group "default" {
    targets = ["yarn", "cypress8", "cypress9", "cypress10", "cypress11"]
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

target "yarn" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.yarn"
  platforms = [
    "linux/amd64",
    "linux/arm64"
  ]
}

target "cypress8" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.cypress8"
  platforms = [
    "linux/amd64",
    "linux/arm64"
  ]
}

target "cypress9" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.cypress9"
  platforms = [
    "linux/amd64",
    "linux/arm64"
  ]
}

target "cypress10" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.cypress10"
  platforms = [
    "linux/amd64",
    "linux/arm64"
  ]
}

target "cypress11" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.cypress11"
  platforms = [
    "linux/amd64",
    "linux/arm64"
  ]
}

// docker-bake.hcl is run from root directory so part
target "docker-metadata-action" {}

group "default" {
    targets = ["jdk11","jdk17","jdk18"]
}


target "jdk11" {
  inherits = ["docker-metadata-action"]
  context = "./executor/gradle/"
  dockerfile = "build/agent/Dockerfile.jdk11"
  platforms = [
    "linux/amd64",
  ]
}

target "jdk17" {
  inherits = ["docker-metadata-action"]
  context = "./executor/gradle/"
  dockerfile = "build/agent/Dockerfile.jdk17"
  platforms = [
    "linux/amd64",
  ]
}


target "jdk18" {
  inherits = ["docker-metadata-action"]
  context = "./executor/gradle/"
  dockerfile = "build/agent/Dockerfile.jdk17"
  platforms = [
    "linux/amd64",
  ]
}
target "docker-metadata-action" {}

group "default" {
    targets = ["jdk11","jdk8","jdk18","jdk17"]
}


target "jdk11" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "contrib/executor/maven/build/agent/Dockerfile.jdk11"
  platforms = [
    "linux/amd64",
    "linux/arm64",
  ]
}

target "jdk8" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "contrib/executor/maven/build/agent/Dockerfile.jdk8"
  platforms = [
    "linux/amd64",
    "linux/arm64",
  ]
}


target "jdk18" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "contrib/executor/maven/build/agent/Dockerfile.jdk18"
  platforms = [
    "linux/amd64",
    "linux/arm64",
  ]
}

target "jdk17" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "contrib/executor/maven/build/agent/Dockerfile.jdk17"
  platforms = [
    "linux/amd64",
    "linux/arm64",
  ]
}

// docker-bake.hcl
target "docker-metadata-action" {}

group "default" {
    targets = ["build gradle:7.4.2-jdk11"]
}


target "build gradle:7.4.2-jdk11" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.jdk11"
  platforms = [
    "linux/amd64",
  ]
}

target "build gradle:7.4.2-jdk17" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.jdk17"
  platforms = [
    "linux/amd64",
  ]
}


target "build gradle:7.4.2-jdk18" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/agent/Dockerfile.jdk17"
  platforms = [
    "linux/amd64",
  ]
}
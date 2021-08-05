// docker-bake.hcl
target "docker-metadata-action" {}

target "build" {
  inherits = ["docker-metadata-action"]
  context = "./"
  dockerfile = "build/postman-executor/Dockerfile"
  platforms = [
    "linux/amd64",
  ]
}
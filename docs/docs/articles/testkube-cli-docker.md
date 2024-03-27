# Testkube Docker CLI

Starting with Testkube version 1.13, the easiest way to start managing your team's tests on a remote Testkube server is to run the Testkube CLI using the official Docker image. The Testkube CLI Docker image is a self-contained environment that allows you to run Testkube commands in a consistent and isolated manner.

## Prerequisites

Before using the Testkube CLI Docker image, ensure that you have Docker installed and running on your system. You can download and install Docker from the official Docker website (<https://www.docker.com/>).

To pull the image, run:

```bash
docker pull kubeshop/testkube-cli:latest
```

## Obtaining the Testkube CLI Docker Image

To obtain the Testkube CLI Docker image, you have two options:

### 1. Pulling from Docker Hub

The Testkube CLI Docker image is available on Docker Hub. You can pull it using the following command:

```bash
docker pull testkube/cli:latest
```

### 2. Building from Source

If you prefer to build the Docker image from source, you can clone the Testkube CLI repository from GitHub and build it locally using GoReleaser, the provided Dockerfile and the Makefile. Follow these steps:

1. Clone the Testkube CLI repository:

```bash
git clone https://github.com/kubeshop/testkube.git
```

2. Build the Docker image:

```bash
make docker-build-cli DOCKER_BUILDX_CACHE_FROM=type=registry,ref=docker.io/kubeshop/testkube-cli:latest ALPINE_IMAGE=alpine:3.18.0 DOCKER_IMAGE_TAG=local ANALYTICS_TRACKING_ID="" ANALYTICS_API_KEY=""
```

## Running the Testkube CLI Docker Image

Once you have the image, run the following command pointing to the Testkube demo:

```bash
docker run kubeshop/testkube-cli:latest version --namespace testkube --api-uri https://demo.testkube.io/results --client direct
```

This command starts a new Docker container with the Testkube CLI image and executes the command `testkube version`, pointing to the api-server running on the Testkube demo environment.

There are multiple *client types* you can set for the Testkube CLI:

* direct - for connecting to a remotely deployed environment
* proxy - for connecting to local environments, not relevant in the case of a Docker container
* cloud - for connecting to Testkube Cloud
* cluster - for connecting to current Kubernetes cluster environment (when running inside the cluster pod, make sure pod service account can access `services/proxy` resource)

You can also use your existing `kubectl` configuration file as a volume:

```bash
docker run -v ~/.testkube:/.testkube kubeshop/testkube-cli:1.11.13-SNAPSHOT-5f34248fd-arm64v8 --api-uri https://demo.testkube.io/results --client direct version 
```

## Conclusion

This user documentation has provided an overview of the Testkube CLI Docker image and guided you through the process of obtaining, running, and using the Testkube CLI within the Docker container. With the Testkube CLI, you can conveniently manage your Testkube deployments and perform various operations with ease.

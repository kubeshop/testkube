# syntax=docker/dockerfile:1
# Step 1: Use a base image with Docker installed
FROM docker:20.10.24-dind

ENV TINI_SUBREAPER=true

# Step 2: Install necessary dependencies (curl, bash, tini, jq)
RUN apk add --no-cache bash curl tini jq

# Step 3: Install Kind (Kubernetes in Docker)
RUN curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64 && \
    chmod +x ./kind && \
    mv ./kind /usr/local/bin/kind

# Step 4: Install kubectl (for interacting with the Kubernetes cluster)
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

# Step 5: Install Helm (package manager for Kubernetes)
RUN curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Step 6: Script to automatically create Kind cluster and install Testkube
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Step 7: Example K6 Test Workflow CRD and preload Kind images
COPY ./images /images

ARG segmentio_key
ENV SEGMENTIO_KEY=$segmentio_key
ARG ga_id
ENV GA_ID=$ga_id
ARG ga_secret
ENV GA_SECRET=$ga_secret
ARG docker_image_version
ENV DOCKER_IMAGE_VERSION=$docker_image_version
ARG cloud_url
ENV CLOUD_URL=$cloud_url

# Step 8: Set Docker entry point for DIND (Docker-in-Docker)
ENTRYPOINT ["tini", "--", "/usr/local/bin/entrypoint.sh"]

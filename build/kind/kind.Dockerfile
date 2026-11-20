# syntax=docker/dockerfile:1
# Step 1: Use a base image with Docker installed
FROM docker:29.9.3-dind

ARG TARGETARCH

ENV TINI_SUBREAPER=true

# Step 2: Install necessary dependencies (curl, bash, tini, jq)
RUN apk upgrade --no-cache && apk add --no-cache bash curl tini jq 'ca-certificates>=20261020-r0' 'ca-certificates-bundle>=20261020-r0'

# Steps 3-5 pin kind, kubectl and helm and verify their SHA256, for this image's own
# architecture: unpinned or amd64-only downloads shipped stale binaries with known CVEs.
# kind's node image (kindest/node in entrypoint.sh and release.yaml) must be one listed
# for this kind release; kubectl stays within one minor version of that node image.

# Step 3: Install Kind (Kubernetes in Docker)
ARG KIND_VERSION=v0.33.0
RUN case "$TARGETARCH" in \
      amd64) sha=aee6151561422756b764a4ae28e7f44cda5af5a9eead3cc9985112b1de8d8e0d ;; \
      arm64) sha=20022bee6cfcd5086cb7234d218e3454e6090022f2a8f55d1fa7fcf42c3867a2 ;; \
      *) echo "unsupported arch $TARGETARCH" && exit 1 ;; \
    esac && \
    curl -fsSLo /usr/local/bin/kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-${TARGETARCH}" && \
    echo "$sha  /usr/local/bin/kind" | sha256sum -c - && \
    chmod +x /usr/local/bin/kind

# Step 4: Install kubectl (for interacting with the Kubernetes cluster)
ARG KUBECTL_VERSION=v1.34.13
RUN case "$TARGETARCH" in \
      amd64) sha=b4bb494e9466a7a94bb851414548bc0d3e2f8da90757e6eccc3371e648648591 ;; \
      arm64) sha=66e8283d52d9bc1f9c0e9daade7c971cd20db5a379306862897dfb25395f3e03 ;; \
      *) echo "unsupported arch $TARGETARCH" && exit 1 ;; \
    esac && \
    curl -fsSLo /usr/local/bin/kubectl "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${TARGETARCH}/kubectl" && \
    echo "$sha  /usr/local/bin/kubectl" | sha256sum -c - && \
    chmod +x /usr/local/bin/kubectl

# Step 5: Install Helm (package manager for Kubernetes)
ARG HELM_VERSION=v3.22.1
RUN case "$TARGETARCH" in \
      amd64) sha=896cf63684a42a723d61c980369d6e588330927deb13676ca0d85006dd6da0ea ;; \
      arm64) sha=4c29e286b279d170fa048ddfc3e4e83dcd069ca9c52102766ba672d7149e694d ;; \
      *) echo "unsupported arch $TARGETARCH" && exit 1 ;; \
    esac && \
    curl -fsSLo /tmp/helm.tar.gz "https://get.helm.sh/helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz" && \
    echo "$sha  /tmp/helm.tar.gz" | sha256sum -c - && \
    tar -xzf /tmp/helm.tar.gz -C /tmp "linux-${TARGETARCH}/helm" && \
    mv "/tmp/linux-${TARGETARCH}/helm" /usr/local/bin/helm && \
    rm -rf /tmp/helm.tar.gz "/tmp/linux-${TARGETARCH}"

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

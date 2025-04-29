ARG BUSYBOX_IMAGE="busybox:1.36.1-musl"

###################################
## Build testworkflow init
###################################
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG GOMODCACHE="/root/.cache/go-build"
ARG GOCACHE="/go/pkg"
ARG SKAFFOLD_GO_GCFLAGS

WORKDIR /app
COPY . .
RUN --mount=type=cache,target="$GOMODCACHE" \
    --mount=type=cache,target="$GOCACHE" \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    CGO_ENABLED=0 \
    go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o build/_local/workflow-init cmd/testworkflow-init/main.go

###################################
## Debug
###################################
FROM golang:1.23-alpine AS debug

ENV GOTRACEBACK=all
RUN go install github.com/go-delve/delve/cmd/dlv@v1.23.1

COPY --from=builder /app/build/_local/workflow-init /testkube/

ENTRYPOINT ["/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56401", "--api-version=2", "/testkube/workflow-init"]

###################################
## Distribution
###################################
FROM ${BUSYBOX_IMAGE} AS dist
RUN cp -rf /bin /.tktw-bin
COPY --from=builder /app/build/_local/workflow-init /testkube/init
USER 1001
ENTRYPOINT ["/init"]


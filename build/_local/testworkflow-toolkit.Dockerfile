ARG BUSYBOX_IMAGE="busybox:1.36.1-musl"
ARG ALPINE_IMAGE="alpine:3.20.0"
FROM ${BUSYBOX_IMAGE} AS busybox

###################################
## Build testworkflow-init
###################################
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder-init

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
## Build testworkflow-toolkit
###################################
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder-toolkit

ARG TARGETOS
ARG TARGETARCH
ARG GOMODCACHE="/root/.cache/go-build"
ARG GOCACHE="/go/pkg"
ARG SKAFFOLD_GO_GCFLAGS

RUN go install github.com/go-delve/delve/cmd/dlv@v1.23.1

WORKDIR /app
COPY . .
RUN --mount=type=cache,target="$GOMODCACHE" \
    --mount=type=cache,target="$GOCACHE" \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    CGO_ENABLED=0 \
    go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o build/_local/workflow-toolkit cmd/testworkflow-toolkit/main.go

###################################
## Debug
###################################
FROM ${ALPINE_IMAGE} AS debug
RUN apk --no-cache add ca-certificates libssl3 git openssh-client
ENV GOTRACEBACK=all
COPY --from=builder-toolkit /go/bin/dlv /
COPY --from=busybox /bin /.tktw-bin
COPY --from=builder-toolkit /app/build/_local/workflow-toolkit /toolkit
COPY --from=builder-init /app/build/_local/workflow-init /init
RUN adduser --disabled-password --home / --no-create-home --uid 1001 default
USER 1001
ENTRYPOINT ["/dlv", "exec", "--headless", "--accept-multiclient", "--listen=:56300", "--api-version=2", "/toolkit"]

###################################
## Distribution
###################################
FROM ${ALPINE_IMAGE} AS dist
RUN apk --no-cache add ca-certificates libssl3 git openssh-client
COPY --from=busybox /bin /.tktw-bin
COPY --from=builder-toolkit /app/build/_local/workflow-toolkit /toolkit
COPY --from=builder-init /app/build/_local/workflow-init /init
RUN adduser --disabled-password --home / --no-create-home --uid 1001 default
USER 1001
ENTRYPOINT ["/toolkit"]

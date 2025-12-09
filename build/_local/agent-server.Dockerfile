###################################
## Build
###################################
FROM --platform=$BUILDPLATFORM golang:1.25 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG GOMODCACHE="/root/.cache/go-build"
ARG GOCACHE="/go/pkg"
ARG SKAFFOLD_GO_GCFLAGS

ARG VERSION
ARG GIT_SHA
ARG SLACK_BOT_CLIENT_ID
ARG SLACK_BOT_CLIENT_SECRET
ARG BUSYBOX_IMAGE
ARG ANALYTICS_TRACKING_ID
ARG ANALYTICS_API_KEY
ARG SEGMENTIO_KEY
ARG CLOUD_SEGMENTIO_KEY

WORKDIR /app
COPY . .
RUN --mount=type=cache,target="$GOMODCACHE" \
    --mount=type=cache,target="$GOCACHE" \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    CGO_ENABLED=0 \
    go build \
      -gcflags="${SKAFFOLD_GO_GCFLAGS}" \
      -ldflags="-X github.com/kubeshop/testkube/pkg/version.Version=${VERSION} -X github.com/kubeshop/testkube/pkg/version.Commit=${GIT_SHA} -X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientID=${SLACK_BOT_CLIENT_ID} -X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientSecret=${SLACK_BOT_CLIENT_SECRET} -X github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants.DefaultImage=${BUSYBOX_IMAGE} -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=${ANALYTICS_TRACKING_ID} -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementSecret=${ANALYTICS_API_KEY} -X github.com/kubeshop/testkube/pkg/telemetry.SegmentioKey=${SEGMENTIO_KEY} -X github.com/kubeshop/testkube/pkg/telemetry.CloudSegmentioKey=${CLOUD_SEGMENTIO_KEY}" \
      -o build/_local/agent-server ./cmd/api-server/...

###################################
## Debug
###################################
FROM golang:1.25 AS debug

ENV GOTRACEBACK=all
RUN go install github.com/go-delve/delve/cmd/dlv@v1.25.2

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/build/_local/agent-server /testkube/

EXPOSE 8080 8088 8089 56268
ENTRYPOINT ["/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "/testkube/agent-server"]

###################################
## Distribution
###################################
FROM gcr.io/distroless/static AS dist

COPY LICENSE /testkube/
COPY --from=builder /app/build/_local/agent-server /testkube/

EXPOSE 8080 8088 8089
ENTRYPOINT ["/testkube/agent-server"]

# this arg has to be defined before the first FROM otherwise the value will be empty
ARG ALPINE_IMAGE

FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build
ARG TARGETOS
ARG TARGETARCH

ARG VERSION
ARG GIT_SHA
ARG SLACK_BOT_CLIENT_ID
ARG SLACK_BOT_CLIENT_SECRET
ARG BUSYBOX_IMAGE
ARG ANALYTICS_TRACKING_ID
ARG ANALYTICS_API_KEY
ARG SEGMENTIO_KEY
ARG CLOUD_SEGMENTIO_KEY

WORKDIR /build
COPY . .
RUN cd cmd/api-server; \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags \
     "-X github.com/kubeshop/testkube/pkg/version.Version=${VERSION} \
      -X github.com/kubeshop/testkube/pkg/version.Commit=${GIT_SHA} \
      -X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientID=${SLACK_BOT_CLIENT_ID} \
      -X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientSecret=${SLACK_BOT_CLIENT_SECRET} \
      -X github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants.DefaultImage=${BUSYBOX_IMAGE} \
      -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=${ANALYTICS_TRACKING_ID} \
      -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementSecret=${ANALYTICS_API_KEY} \
      -X github.com/kubeshop/testkube/pkg/telemetry.SegmentioKey=${SEGMENTIO_KEY} \
      -X github.com/kubeshop/testkube/pkg/telemetry.CloudSegmentioKey=${CLOUD_SEGMENTIO_KEY}" \
    -o /app -mod mod -a .

FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates libssl3 git
WORKDIR /root/
COPY --from=build /app /bin/app
USER 1001
EXPOSE 8088
ENTRYPOINT ["/bin/app"]
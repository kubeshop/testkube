# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.18 AS build
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

ARG VERSION
ARG COMMIT
ARG ANALYTICS_API_KEY
ARG ANALYTICS_TRACKING_ID
ARG SEGMENTIO_KEY
ARG SLACK_BOT_CLIENT_ID
ARG SLACK_BOT_CLIENT_SECRET
ARG TARGETOS TARGETARCH

RUN cd cmd/api-server; \
    GOOS=$TARGETOS GOARCH=$TARGETARCH  go build \
        -ldflags "-X github.com/kubeshop/testkube/pkg/version.Version=$VERSION \
            -X github.com/kubeshop/testkube/pkg/version.Commit=$COMMIT \
	    -X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientID=$SLACK_BOT_CLIENT_ID \
	    -X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientSecret=$SLACK_BOT_CLIENT_SECRET \
            -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=$ANALYTICS_TRACKING_ID \
            -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementSecret=$ANALYTICS_API_KEY \
            -X github.com/kubeshop/testkube/pkg/telemetry.SegmentioKey=$SEGMENTIO_KEY" \
        -o /app -mod mod -a .

FROM alpine:3.16  
RUN apk --no-cache add ca-certificates libssl1.1 git
WORKDIR /root/
COPY  --from=build /app /bin/app

USER 1001

EXPOSE 8088
CMD ["/bin/app"]

# syntax=docker/dockerfile:1
FROM golang:1.18
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

ARG VERSION 
ARG COMMIT 
ARG ANALYTICS_API_KEY 
ARG ANALYTICS_TRACKING_ID
ARG SLACK_BOT_CLIENT_ID 
ARG SLACK_BOT_CLIENT_SECRET

RUN cd cmd/api-server; \
    go build \
        -ldflags "-X github.com/kubeshop/testkube/internal/pkg/api.Version=$VERSION \
            -X github.com/kubeshop/testkube/internal/pkg/api.Commit=$COMMIT \
            -X github.com/kubeshop/testkube/pkg/analytics.TestkubeMeasurementID=$ANALYTICS_TRACKING_ID \
            -X github.com/kubeshop/testkube/pkg/analytics.TestkubeMeasurementSecret=$ANALYTICS_API_KEY \
			-X github.com/kubeshop/testkube/pkg/slacknotifier.SlackBotClientID=$SLACK_BOT_CLIENT_ID \
			-X github.com/kubeshop/testkube/pkg/slacknotifier.SlackBotClientSecret=$SLACK_BOT_CLIENT_SECRET" \
        -o /app -mod mod -a .

FROM alpine:3.15  
RUN apk --no-cache add ca-certificates libssl1.1
WORKDIR /root/
COPY --from=0 /app /bin/app
EXPOSE 8088
CMD ["/bin/app"]

###################################
## Build
###################################
FROM --platform=$BUILDPLATFORM golang:1.25 AS builder

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
    go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o build/_local/agent-server cmd/api-server/main.go

###################################
## Debug
###################################
FROM golang:1.25.0-alpine AS debug

ENV GOTRACEBACK=all
RUN go install github.com/go-delve/delve/cmd/dlv@v1.25.0

RUN apk --no-cache --update add ca-certificates && (rm -rf /var/cache/apk/* || 0)

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/build/_local/agent-server /testkube/

ENTRYPOINT ["/go/bin/dlv", "exec", "--headless", "--continue", "--accept-multiclient", "--listen=:56268", "--api-version=2", "/testkube/agent-server"]

###################################
## Distribution
###################################
FROM gcr.io/distroless/static AS dist

COPY LICENSE /testkube/
COPY --from=builder /app/build/_local/agent-server /testkube/

EXPOSE 8080
ENTRYPOINT ["/testkube/agent-server"]

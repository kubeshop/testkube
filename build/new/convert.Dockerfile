# this arg has to be defined before the first FROM otherwise the value will be empty
ARG ALPINE_IMAGE

FROM --platform=$BUILDPLATFORM golang:1.27.0-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG GOCACHE="/root/.cache/go-build"
ARG GOMODCACHE="/go/pkg/mod"

ARG VERSION
ARG GIT_SHA

WORKDIR /build
COPY . .
RUN --mount=type=cache,target="$GOMODCACHE" \
    --mount=type=cache,target="$GOCACHE" \
    cd cmd/convert; \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags \
     "-X github.com/kubeshop/testkube/pkg/version.Version=${VERSION} \
      -X github.com/kubeshop/testkube/pkg/version.Commit=${GIT_SHA} \
      -X main.commit=${GIT_SHA}" \
    -o /app -mod mod .

# alpine rather than scratch: the tool needs a writable /tmp for its scratch
# space, and CA certificates to reach a TLS-enabled MongoDB or DocumentDB.
FROM ${ALPINE_IMAGE}
RUN apk --no-cache upgrade && apk --no-cache add ca-certificates libssl3
WORKDIR /tmp
COPY --from=build /app /bin/app
USER 1001
ENTRYPOINT ["/bin/app"]

ARG BUSYBOX_IMAGE
ARG ALPINE_IMAGE

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG GOCACHE="/root/.cache/go-build"
ARG GOMODCACHE="/go/pkg/mod"
WORKDIR /app
COPY . .
RUN --mount=type=cache,target="$GOMODCACHE" \
    --mount=type=cache,target="$GOCACHE" \
    cd cmd/testworkflow-toolkit; \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags \
    "-X github.com/kubeshop/testkube/pkg/version.Version=${VERSION} \
     -X github.com/kubeshop/testkube/pkg/version.Commit=${GIT_SHA} \
     -s -w" \
     -o /app/testworkflow-toolkit -mod mod .

RUN --mount=type=cache,target="$GOMODCACHE" \
    --mount=type=cache,target="$GOCACHE" \
    cd cmd/testworkflow-init; \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags \
    "-X github.com/kubeshop/testkube/pkg/version.Version=${VERSION} \
     -X github.com/kubeshop/testkube/pkg/version.Commit=${GIT_SHA} \
     -s -w" \
     -o /app/testworkflow-init -mod mod .

FROM ${BUSYBOX_IMAGE} AS busybox
FROM ${ALPINE_IMAGE}
RUN apk --no-cache upgrade && apk --no-cache add ca-certificates libssl3 git openssh-client
COPY --from=busybox /bin /.tktw-bin
COPY --from=build /app/testworkflow-toolkit /toolkit
COPY --from=build /app/testworkflow-init /init
RUN adduser --disabled-password --home / --no-create-home --uid 1001 default
USER 1001
ENTRYPOINT ["/toolkit"]

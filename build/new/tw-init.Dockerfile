ARG BUSYBOX_IMAGE

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY . .
RUN cd cmd/testworkflow-init; \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags \
       "-X github.com/kubeshop/testkube/pkg/version.Version=${VERSION} \
        -X github.com/kubeshop/testkube/pkg/version.Commit=${GIT_SHA} \
        -s -w" \
    -o /app/testworkflow-init -mod mod -a .


FROM ${BUSYBOX_IMAGE}
RUN cp -rf /bin /.tktw-bin
COPY --from=build /app/testworkflow-init /init
USER 1001
ENTRYPOINT ["/init"]



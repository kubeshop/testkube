ARG ALPINE_IMAGE

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG GOMODCACHE="/root/.cache/go-build"
ARG GOCACHE="/go/pkg"

ARG VERSION
ARG DATE
ARG GIT_SHA
ARG ANALYTICS_TRACKING_ID
ARG ANALYTICS_API_KEY
ARG KEYGEN_PUBLIC_KEY

WORKDIR /app
COPY . .
RUN --mount=type=cache,target="$GOMODCACHE" \
    --mount=type=cache,target="$GOCACHE" \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    CGO_ENABLED=0 \
    go build -ldflags "-s -X main.version=${VERSION} -X main.commit=${GIT_SHA} -X main.date=${DATE} -X main.builtBy=docker -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=${ANALYTICS_TRACKING_ID} -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementSecret=${ANALYTICS_API_KEY} -X github.com/kubeshop/testkube/pkg/diagnostics/validators/license.KeygenOfflinePublicKey=${KEYGEN_PUBLIC_KEY}" \
    -o /app/kubectl-testkube cmd/kubectl-testkube/main.go

FROM ${ALPINE_IMAGE}
COPY --from=build /app/kubectl-testkube /bin/kubectl-testkube

# Create symbolic links for 'testkube' and 'tk' as aliases for 'kubectl-testkube'
RUN ln -s /bin/kubectl-testkube /bin/testkube
RUN ln -s /bin/kubectl-testkube /bin/tk

# Create and set permissions for /.testkube directory
RUN mkdir /.testkube && echo "{}" > /.testkube/config.json && chmod -R 755 /.testkube && chown -R 1001:1001 /.testkube && chmod 660 /.testkube/config.json

USER 1001
ENTRYPOINT ["/bin/kubectl-testkube"]

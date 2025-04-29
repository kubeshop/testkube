ARG GO_VERSION=1.23.5
FROM golang:${GO_VERSION}
ARG TARGETOS
ARG TARGETARCH
ARG PROTOC_VERSION=3.19.4
ARG PROTOC_GEN_GO_VERSION=1.28.1
ARG PROTOC_GEN_GRPC_VERSION=1.2.0

# We need zip to decompress the protoc binary.
RUN apt-get update && apt-get install -y zip

# Because protoc uses odd architecture specs we have to modify for anyone using, for example, Apple Silicon.
RUN if [ "$TARGETARCH" = "arm64" ]; then export TARGETARCH=aarch_64; fi && \
    PROTOC_ZIP_FILE=protoc-${PROTOC_VERSION}-${TARGETOS}-${TARGETARCH}.zip && \
    curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP_FILE} && \
    unzip -o ${PROTOC_ZIP_FILE} -d /usr

# Install Go protobuf tools.
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v${PROTOC_GEN_GO_VERSION}
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${PROTOC_GEN_GRPC_VERSION}

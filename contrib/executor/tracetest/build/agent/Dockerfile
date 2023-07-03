# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates git curl bash
RUN curl -L https://raw.githubusercontent.com/kubeshop/tracetest/main/install-cli.sh | bash
WORKDIR /root/
COPY tracetest /bin/runner
ENTRYPOINT ["/bin/runner"]
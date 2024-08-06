# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates libssl3 git openssh-client
COPY testworkflow-toolkit /toolkit
COPY testworkflow-init /init
RUN adduser --disabled-password --home / --no-create-home --uid 1001 default
USER 1001
ENTRYPOINT ["/toolkit"]

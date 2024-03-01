# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
COPY testworkflow-init /init
USER 1001
ENTRYPOINT ["/init"]

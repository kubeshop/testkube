# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
COPY testworkflow-toolkit /toolkit
USER 1001
ENTRYPOINT ["/toolkit"]

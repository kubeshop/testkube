# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates libssl3 git skopeo
COPY testworkflow-toolkit /toolkit
USER 1001
ENTRYPOINT ["/toolkit"]

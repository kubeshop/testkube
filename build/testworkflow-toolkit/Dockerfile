# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates libssl1.1 git
COPY testworkflow-toolkit /toolkit
USER 1001
ENTRYPOINT ["/toolkit"]

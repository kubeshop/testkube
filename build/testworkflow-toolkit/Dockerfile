# syntax=docker/dockerfile:1
ARG BUSYBOX_IMAGE
ARG ALPINE_IMAGE

FROM ${BUSYBOX_IMAGE} AS busybox
FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates libssl3 git openssh-client
COPY --from=busybox /bin /.tktw-bin
COPY testworkflow-toolkit /toolkit
COPY testworkflow-init /init
RUN adduser --disabled-password --home / --no-create-home --uid 1001 default
USER 1001
ENTRYPOINT ["/toolkit"]

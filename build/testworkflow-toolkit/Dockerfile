# syntax=docker/dockerfile:1
ARG BUSYBOX_IMAGE
ARG ALPINE_IMAGE

FROM ${BUSYBOX_IMAGE} AS busybox
FROM ${ALPINE_IMAGE}
RUN apk --no-cache upgrade && apk --no-cache add ca-certificates libssl3 "git>=2.54.1-r0" "libexpat>=2.10.0-r0" "openssh-client>=10.3_p1-r2"
COPY --from=busybox /bin /.tktw-bin
COPY testworkflow-toolkit /toolkit
COPY testworkflow-init /init
RUN adduser --disabled-password --home / --no-create-home --uid 1001 default
USER 1001
ENTRYPOINT ["/toolkit"]

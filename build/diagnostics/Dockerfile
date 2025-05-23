# syntax=docker/dockerfile:1
ARG BUSYBOX_IMAGE="busybox:1.36.1"
FROM ${BUSYBOX_IMAGE}
COPY testkube-diagnostics /
USER 1001
ENTRYPOINT ["/testkube-diagnostics"]

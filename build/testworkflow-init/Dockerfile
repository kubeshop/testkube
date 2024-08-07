# syntax=docker/dockerfile:1
ARG BUSYBOX_IMAGE
FROM ${BUSYBOX_IMAGE}
RUN cp -rf /bin /.tktw-bin
COPY testworkflow-init /init
USER 1001
ENTRYPOINT ["/init"]

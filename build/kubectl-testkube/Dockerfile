# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
COPY kubectl-testkube /bin/kubectl-testkube
RUN ["/bin/sh","-c", "mkdir /.testkube"]
RUN ["/bin/sh", "-c", "echo \"{}\" > /.testkube/config.json"]
RUN ["/bin/sh", "-c", "chmod -R 755 /.testkube"]
USER 1001
ENTRYPOINT ["/bin/kubectl-testkube"]

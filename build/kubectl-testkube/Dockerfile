# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
COPY testkube-cli /bin/testkube-cli
RUN mkdir .testkube && echo "{}" > .testkube/config.json \
    && chmod -R 755 .testkube
USER 1001
ENTRYPOINT ["/bin/testkube-cli"]

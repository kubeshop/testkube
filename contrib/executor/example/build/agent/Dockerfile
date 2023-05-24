# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
FROM $ALPINE_IMAGE
COPY example /bin/runner

RUN apk --no-cache add ca-certificates git
WORKDIR /root/
ENTRYPOINT ["/bin/runner"]

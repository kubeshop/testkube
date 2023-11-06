# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
COPY scraper /bin/runner
RUN apk --no-cache add ca-certificates git curl
WORKDIR /root/
ENTRYPOINT ["/bin/runner"]

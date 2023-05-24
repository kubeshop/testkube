# syntax=docker/dockerfile:1
FROM ${ALPINE_IMAGE}
COPY scraper /bin/runner
RUN apk --no-cache add ca-certificates git
WORKDIR /root/
ENTRYPOINT ["/bin/runner"]

# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
RUN apk --no-cache upgrade && apk --no-cache add ca-certificates libssl3 "git>=2.54.1-r0" "libexpat>=2.10.0-r0"
WORKDIR /root/
COPY testkube-api-server /bin/app
USER 1001
EXPOSE 8088
ENTRYPOINT ["/bin/app"]
